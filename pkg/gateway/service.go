package gateway

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/google/uuid"
	"github.com/nite-coder/blackbear/pkg/cast"
	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/nite-coder/bifrost/internal/pkg/optional"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/ai"
	"github.com/nite-coder/bifrost/pkg/ai/pricing"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/proxy"
	aiproxy "github.com/nite-coder/bifrost/pkg/proxy/ai"
	grpcproxy "github.com/nite-coder/bifrost/pkg/proxy/grpc"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
	"github.com/nite-coder/bifrost/pkg/target"
	"github.com/nite-coder/bifrost/pkg/telemetry/metrics"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const statusClientClosedRequest = 499

// Service represents a backend service that can have multiple upstreams and middlewares.
type Service struct {
	bifrost           *Bifrost
	options           *config.ServiceOptions
	upstream          *Upstream
	dynamicUpstream   string
	middlewares       []app.HandlerFunc
	mu                sync.RWMutex
	proxyByAddress    sync.Map
	upstreamAddresses map[string]map[string]bool
	subscriptions     map[string]<-chan []*target.Endpoint
	cancelFuncs       []context.CancelFunc
}

// Close releases resources used by the service and its upstreams.
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, cancel := range s.cancelFuncs {
		cancel()
	}
	s.cancelFuncs = nil

	for id, ch := range s.subscriptions {
		if s.bifrost != nil && s.bifrost.upstreamManager != nil {
			if u, found := s.bifrost.upstreamManager.Get(id); found {
				u.Unsubscribe(ch)
				continue
			}
		}
		if s.upstream != nil && s.upstream.options.ID == id {
			s.upstream.Unsubscribe(ch)
		}
	}
	s.subscriptions = make(map[string]<-chan []*target.Endpoint)

	if s.upstream != nil && s.upstream.isExclusive.Load() {
		_ = s.upstream.Close()
	}

	s.proxyByAddress.Range(func(_, value any) bool {
		if p, ok := value.(proxy.Proxy); ok {
			_ = p.Close()
		}
		return true
	})

	return nil
}

func loadServices(bifrost *Bifrost) (map[string]*Service, error) {
	services := make(map[string]*Service)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errCh := make(chan error, len(bifrost.options.Services))

	for id, serviceOptions := range bifrost.options.Services {
		if len(id) == 0 {
			return nil, errors.New("service ID cannot be empty")
		}

		serviceOptions.ID = id
		currentServiceOptions := serviceOptions
		currentID := id

		wg.Add(1)
		go safety.Go(context.Background(), func() {
			defer wg.Done()

			service, err := newService(bifrost, currentServiceOptions)
			if err != nil {
				errCh <- err
				return
			}

			mu.Lock()
			if _, found := services[currentServiceOptions.ID]; found {
				errCh <- fmt.Errorf("service '%s' already exists", currentServiceOptions.ID)
				mu.Unlock()
				return
			}
			services[currentID] = service
			mu.Unlock()
		})
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			for _, svc := range services {
				_ = svc.Close()
			}
			return nil, err
		}
	}

	return services, nil
}

func newService(bifrost *Bifrost, serviceOptions config.ServiceOptions) (*Service, error) {
	svc := &Service{
		bifrost:           bifrost,
		options:           &serviceOptions,
		upstreamAddresses: make(map[string]map[string]bool),
		subscriptions:     make(map[string]<-chan []*target.Endpoint),
	}

	svc.applyProtocolDefaults()

	if err := svc.initMiddlewares(); err != nil {
		return nil, err
	}

	if serviceOptions.Type == config.ServiceTypeAI {
		svc.dynamicUpstream = variable.Model
		if svc.bifrost.options.Models != nil && svc.bifrost.upstreamManager != nil {
			for modelID := range svc.bifrost.options.Models {
				upstreamID := "ai:" + modelID
				if u, found := svc.bifrost.upstreamManager.Get(upstreamID); found {
					svc.subscribeToUpstream(u)
				}
			}
		}
	} else {
		if err := svc.resolveUpstreamStrategy(); err != nil {
			return nil, err
		}
		if len(svc.dynamicUpstream) > 0 && svc.bifrost.upstreamManager != nil {
			for id := range svc.bifrost.options.Upstreams {
				if u, found := svc.bifrost.upstreamManager.Get(id); found {
					svc.subscribeToUpstream(u)
				}
			}
		}
	}

	return svc, nil
}

// Upstream returns the primary upstream associated with this service.
func (s *Service) Upstream() *Upstream {
	return s.upstream
}

// Middlewares returns the list of middlewares associated with this service.
func (s *Service) Middlewares() []app.HandlerFunc {
	return s.middlewares
}

func (s *Service) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)

	defer func() {
		if r := recover(); r != nil {
			stackTrace := cast.B2S(debug.Stack())
			logger.ErrorContext(ctx, "service panic recovered", slog.Any("panic", r), slog.String("stack", stackTrace))
			c.SetStatusCode(http.StatusInternalServerError)
			c.Abort()
		}
	}()

	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			fullURI := fullURI(&c.Request)
			routeID := variable.GetString(variable.RouteID, c)

			httpStart, _ := variable.Get(variable.HTTPStart, c)
			var duration time.Duration
			if httpStart != nil {
				if t, ok := httpStart.(time.Time); ok {
					duration = time.Since(t)
				}
			}

			logger.InfoContext(ctx, "client cancel the request",
				slog.String("route_id", routeID),
				slog.String("client_ip", c.ClientIP()),
				slog.String("full_uri", fullURI),
				slog.Duration("duration", duration),
			)

			c.Response.SetStatusCode(statusClientClosedRequest)
			return
		}
	}

	var myEndpoint *target.Endpoint
	var err error

	if len(s.dynamicUpstream) > 0 {
		upstreamID := variable.GetString(s.dynamicUpstream, c)

		if len(upstreamID) == 0 {
			if s.options.Type == config.ServiceTypeAI {
				_ = c.Error(&ai.AIError{
					Type:       "invalid_request_error",
					Message:    "The model is missing or empty in the request",
					StatusCode: http.StatusNotFound,
					Code:       optional.Some("model_not_found"),
				})
			} else {
				logger.Warn("upstream is empty",
					slog.String("path", string(c.Request.Path())),
				)
				c.Abort()
			}
			return
		}

		if s.options.Type == config.ServiceTypeAI {
			upstreamID = "ai:" + upstreamID
		}

		bal, found := s.getBalancer(upstreamID)

		if !found {
			if s.options.Type == config.ServiceTypeAI {
				virtualModel := strings.TrimPrefix(upstreamID, "ai:")
				_ = c.Error(&ai.AIError{
					Type:       "invalid_request_error",
					Message:    fmt.Sprintf("The model `%s` does not exist", virtualModel),
					StatusCode: http.StatusNotFound,
					Code:       optional.Some("model_not_found"),
				})
			} else {
				logger.Warn("upstream is not found",
					slog.String("name", upstreamID),
				)
				c.Abort()
			}
			return
		}

		c.Set(variable.UpstreamID, upstreamID)
		myEndpoint, err = bal.Select(ctx, c)
	} else if s.upstream != nil {
		c.Set(variable.UpstreamID, s.upstream.options.ID)

		bal := s.upstream.Balancer()
		if bal == nil {
			logger.Warn("balancer is nil, upstream may not be initialized",
				"upstream_id", s.upstream.options.ID,
				"service_id", s.options.ID,
			)
			c.SetStatusCode(http.StatusServiceUnavailable)
			return
		}
		myEndpoint, err = bal.Select(ctx, c)
	}

	if myEndpoint == nil || err != nil {
		c.SetStatusCode(http.StatusServiceUnavailable)

		if !errors.Is(err, balancer.ErrNotAvailable) {
			_ = c.Error(err)
		}
		return
	}

	myProxy := s.findProxyByEndpoint(myEndpoint)
	if myProxy == nil {
		upstreamID := variable.GetString(variable.UpstreamID, c)
		if p := s.buildProxy(myEndpoint, upstreamID); p != nil {
			actual, _ := s.proxyByAddress.LoadOrStore(myEndpoint.Address, p)
			if pp, ok := actual.(proxy.Proxy); ok {
				myProxy = pp
			}
		}
	}
	if myProxy == nil {
		c.SetStatusCode(http.StatusServiceUnavailable)
		return
	}

	startTime := timecache.Now()
	myProxy.ServeHTTP(ctx, c)
	endTime := timecache.Now()

	dur := endTime.Sub(startTime)
	c.Set(variable.UpstreamDuration, dur)

	if c.GetBool(variable.TargetTimeout) {
		c.Response.SetStatusCode(http.StatusGatewayTimeout)
	} else {
		c.Set(variable.UpstreamResponoseStatusCode, c.Response.StatusCode())
	}
}

func (s *Service) applyProtocolDefaults() {
	if len(s.options.Protocol) == 0 && len(s.bifrost.options.Default.Service.Protocol) > 0 {
		s.options.Protocol = s.bifrost.options.Default.Service.Protocol
	} else if len(s.options.Protocol) == 0 && len(s.bifrost.options.Default.Service.Protocol) == 0 {
		s.options.Protocol = config.ProtocolHTTP
	}
}

func (s *Service) initMiddlewares() error {
	for _, middlewareOpts := range s.options.Middlewares {
		if len(middlewareOpts.Use) > 0 {
			m, found := s.bifrost.middlewares[middlewareOpts.Use]
			if !found {
				return fmt.Errorf("middleware '%s' not found in service: '%s'", middlewareOpts.Use, s.options.ID)
			}
			s.middlewares = append(s.middlewares, m)
			continue
		}

		if len(middlewareOpts.Type) == 0 {
			return fmt.Errorf("middleware type cannot be empty for service: %s", s.options.ID)
		}

		handler := middleware.Factory(middlewareOpts.Type)
		if handler == nil {
			return fmt.Errorf(
				"middleware handler '%s' was not found in service: '%s'",
				middlewareOpts.Type,
				s.options.ID,
			)
		}

		appHandler, e := handler(middlewareOpts.Params)
		if e != nil {
			return fmt.Errorf(
				"failed to create middleware '%s' in route: '%s', error: %w",
				middlewareOpts.Type,
				s.options.ID,
				e,
			)
		}

		s.middlewares = append(s.middlewares, appHandler)
	}
	return nil
}

func (s *Service) resolveUpstreamStrategy() error {
	addr, e := url.Parse(s.options.URL)
	if e != nil {
		return e
	}

	hostname := addr.Hostname()

	if len(hostname) == 0 {
		return fmt.Errorf("hostname is invalid in service URL for service ID: %s", s.options.ID)
	}

	if hostname[0] == '$' {
		s.dynamicUpstream = hostname
		return nil
	}

	var upstream *Upstream
	var found bool
	if s.bifrost.upstreamManager != nil {
		upstream, found = s.bifrost.upstreamManager.Get(hostname)
	}
	if found {
		s.upstream = upstream
		s.subscribeToUpstream(upstream)
		return nil
	}

	upstreamOptions := config.UpstreamOptions{
		ID: "direct:" + uuid.NewString(),
		Balancer: config.BalancerOptions{
			Type: "round_robin",
		},
		Targets: []config.TargetOptions{
			{
				Target: hostname,
				Weight: 1,
			},
		},
	}

	upstream, err := newUpstream(s.bifrost, upstreamOptions)
	if err != nil {
		return err
	}
	s.upstream = upstream
	upstream.isExclusive.Store(true)
	s.subscribeToUpstream(upstream)

	return nil
}

func (s *Service) subscribeToUpstream(upstream *Upstream) {
	s.mu.Lock()
	if _, exists := s.subscriptions[upstream.options.ID]; exists {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	endpoints := upstream.Endpoints()
	s.updateEndpoints(upstream.options.ID, endpoints)

	ch := upstream.Subscribe()

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.subscriptions[upstream.options.ID]; exists {
		upstream.Unsubscribe(ch)
		return
	}
	s.subscriptions[upstream.options.ID] = ch

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFuncs = append(s.cancelFuncs, cancel)

	go safety.Go(ctx, func() {
		for {
			select {
			case <-ctx.Done():
				return
			case endpoints, ok := <-ch:
				if !ok {
					return
				}
				s.updateEndpoints(upstream.options.ID, endpoints)
			}
		}
	})
}

type proxyBuildTask struct {
	ep         *target.Endpoint
	upstreamID string
}

func (s *Service) updateEndpoints(upstreamID string, endpoints []*target.Endpoint) {
	toBuild := make([]proxyBuildTask, 0, len(endpoints))

	s.mu.Lock()
	oldAddresses := s.upstreamAddresses[upstreamID]
	newSet := make(map[string]bool, len(endpoints))

	for _, ep := range endpoints {
		newSet[ep.Address] = true
		if v, ok := s.proxyByAddress.Load(ep.Address); ok {
			if p, ok := v.(proxy.Proxy); ok {
				p.SetEndpoint(ep)
			}
			continue
		}
		toBuild = append(toBuild, proxyBuildTask{ep: ep, upstreamID: upstreamID})
	}
	s.upstreamAddresses[upstreamID] = newSet
	s.mu.Unlock()

	failedAddrs := make([]string, 0)
	for _, task := range toBuild {
		p := s.buildProxy(task.ep, task.upstreamID)
		if p == nil {
			failedAddrs = append(failedAddrs, task.ep.Address)
			continue
		}
		if actual, loaded := s.proxyByAddress.LoadOrStore(task.ep.Address, p); loaded {
			_ = p.Close()
			if existing, ok := actual.(proxy.Proxy); ok {
				existing.SetEndpoint(task.ep)
			}
		}
	}

	s.mu.Lock()
	for _, addr := range failedAddrs {
		delete(s.upstreamAddresses[upstreamID], addr)
	}
	for addr := range oldAddresses {
		if newSet[addr] || s.isAddressUsedByAnyUpstream(addr) {
			continue
		}
		v, ok := s.proxyByAddress.LoadAndDelete(addr)
		if !ok {
			continue
		}
		p, ok := v.(proxy.Proxy)
		if !ok {
			continue
		}
		_ = p.Close()
	}
	s.mu.Unlock()
}

func (s *Service) isAddressUsedByAnyUpstream(addr string) bool {
	for _, addrs := range s.upstreamAddresses {
		if addrs[addr] {
			return true
		}
	}
	return false
}

func (s *Service) buildProxy(ep *target.Endpoint, upstreamID string) proxy.Proxy {
	modelID, isAI := strings.CutPrefix(upstreamID, "ai:")
	if isAI {
		if s.bifrost.options.Models == nil {
			return nil
		}
		modelOpts, found := s.bifrost.options.Models[modelID]
		if !found {
			return nil
		}

		parts := strings.SplitN(ep.Address, "/", aiproxy.TargetPartsCount)
		if len(parts) != aiproxy.TargetPartsCount {
			slog.Error("invalid target format in AI model", "target", ep.Address)
			return nil
		}

		handler := ""
		if s.bifrost.options.AI != nil && s.bifrost.options.AI.Providers != nil {
			if prov, ok := s.bifrost.options.AI.Providers[parts[0]]; ok {
				handler = prov.Handler
			}
		}

		var targetPricing *config.AIPricingOptions
		for _, t := range modelOpts.Targets {
			if t.Target == ep.Address {
				targetPricing = t.Pricing
				break
			}
		}

		metricsEnabled := s.bifrost.options.Metrics.Prometheus.Enabled || s.bifrost.options.Metrics.OTLP.Enabled
		p, pErr := aiproxy.NewProxy(aiproxy.ProxyOptions{
			ID:             ep.Address,
			Target:         ep.Address,
			AIOptions:      s.bifrost.options.AI,
			MetricsEnabled: metricsEnabled,
			Pricing:        pricing.Resolve(handler, parts[1], targetPricing),
			Endpoint:       ep,
		})
		if pErr != nil {
			slog.Error("failed to create AI proxy", "error", pErr)
			return nil
		}
		return p
	}

	serverName := ep.Tags["server_name"]

	targetHost, targetPort, splitErr := net.SplitHostPort(ep.Address)
	if splitErr != nil {
		targetHost = ep.Address
		targetPort = "0"
	}

	addr, parseErr := url.Parse(s.options.URL)
	if parseErr != nil {
		slog.Error("failed to parse service URL", "url", s.options.URL, "error", parseErr)
		return nil
	}

	port := addr.Port()
	myURL := fmt.Sprintf("%s://%s%s", addr.Scheme, targetHost, addr.Path)
	if port != "" {
		myURL = fmt.Sprintf("%s://%s:%s%s", addr.Scheme, targetHost, port, addr.Path)
	} else if targetPort != "" && targetPort != "0" {
		myURL = fmt.Sprintf("%s://%s:%s%s", addr.Scheme, targetHost, targetPort, addr.Path)
	}

	switch s.options.Protocol {
	case "", config.ProtocolHTTP, config.ProtocolHTTP2:
		clientOpts := s.getClientOptions()

		if strings.EqualFold(addr.Scheme, "https") {
			clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
				ServerName:         serverName,
				InsecureSkipVerify: !s.options.TLSVerify, //nolint:gosec
			}))
		}
		if s.bifrost.options.Metrics.Prometheus.Enabled {
			clientOpts = append(clientOpts, client.WithConnStateObserve(func(hcs hzconfig.HostClientState) {
				labels := make(prom.Labels)
				labels["service_id"] = s.options.ID
				labels["target"] = hcs.ConnPoolState().Addr
				metrics.HTTPServiceOpenConnections.With(labels).Set(float64(hcs.ConnPoolState().TotalConnNum))
			}))
		}

		httpClient, clientErr := httpproxy.NewClient(httpproxy.ClientOptions{
			IsHTTP2:   s.options.Protocol == config.ProtocolHTTP2,
			HZOptions: clientOpts,
		})
		if clientErr != nil {
			slog.Error("failed to create http client", "error", clientErr)
			return nil
		}

		p, pErr := httpproxy.New(httpproxy.Options{
			Target:           myURL,
			Protocol:         s.options.Protocol,
			IsTracingEnabled: s.bifrost.options.Tracing.Enabled,
			ServiceID:        s.options.ID,
			TargetHostHeader: serverName,
			PassHostHeader:   s.options.IsPassHostHeader(),
			Endpoint:         ep,
		}, httpClient)
		if pErr != nil {
			slog.Error("failed to create http proxy", "error", pErr)
			return nil
		}
		return p

	case config.ProtocolGRPC:
		p, pErr := grpcproxy.New(grpcproxy.Options{
			Target:           myURL,
			TLSVerify:        s.options.TLSVerify,
			IsTracingEnabled: s.bifrost.options.Tracing.Enabled,
			Timeout:          s.options.Timeout.GRPC,
			Endpoint:         ep,
		})
		if pErr != nil {
			slog.Error("failed to create grpc proxy", "error", pErr)
			return nil
		}
		return p

	default:
		slog.Error("unsupported protocol", "protocol", s.options.Protocol)
		return nil
	}
}

func (s *Service) getClientOptions() []hzconfig.ClientOption {
	clientOpts := httpproxy.DefaultClientOptions()

	dialTimeout := s.options.Timeout.Dail
	if dialTimeout <= 0 {
		dialTimeout = s.bifrost.options.Default.Service.Timeout.Dail
	}
	if dialTimeout > 0 {
		clientOpts = append(clientOpts, client.WithDialTimeout(dialTimeout))
	}

	readTimeout := s.options.Timeout.Read
	if readTimeout <= 0 {
		readTimeout = s.bifrost.options.Default.Service.Timeout.Read
	}
	if readTimeout > 0 {
		clientOpts = append(clientOpts, client.WithClientReadTimeout(readTimeout))
	}

	writeTimeout := s.options.Timeout.Write
	if writeTimeout <= 0 {
		writeTimeout = s.bifrost.options.Default.Service.Timeout.Write
	}
	if writeTimeout > 0 {
		clientOpts = append(clientOpts, client.WithWriteTimeout(writeTimeout))
	}

	maxConnWait := s.options.Timeout.MaxConnWait
	if maxConnWait <= 0 {
		maxConnWait = s.bifrost.options.Default.Service.Timeout.MaxConnWait
	}
	if maxConnWait > 0 {
		clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(maxConnWait))
	}

	maxConns := s.options.MaxConnsPerHost
	if maxConns == nil {
		maxConns = s.bifrost.options.Default.Service.MaxConnsPerHost
	}
	if maxConns != nil {
		clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*maxConns))
	}

	return clientOpts
}

func (s *Service) getBalancer(upstreamID string) (balancer.Balancer, bool) {
	if s.bifrost == nil || s.bifrost.upstreamManager == nil {
		return nil, false
	}
	u, found := s.bifrost.upstreamManager.Get(upstreamID)
	if !found {
		return nil, false
	}
	b := u.Balancer()
	if b == nil {
		return nil, false
	}
	return b, true
}

func (s *Service) findProxyByEndpoint(ep *target.Endpoint) proxy.Proxy {
	if ep == nil {
		return nil
	}
	v, ok := s.proxyByAddress.Load(ep.Address)
	if !ok {
		return nil
	}
	p, ok := v.(proxy.Proxy)
	if !ok {
		return nil
	}
	return p
}

// DynamicService handles requests for services determined dynamically at runtime.
type DynamicService struct {
	services map[string]*Service
	name     string
}

func newDynamicService(name string, services map[string]*Service) *DynamicService {
	return &DynamicService{
		services: services,
		name:     name,
	}
}

func (s *DynamicService) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)
	serviceName := variable.GetString(s.name, c)

	if len(serviceName) == 0 {
		routeID := variable.GetString(variable.RouteID, c)
		fullURI := fullURI(&c.Request)
		logger.Error("service name is empty",
			slog.String("route_id", routeID),
			slog.String("client_ip", c.ClientIP()),
			slog.String("full_uri", fullURI),
			slog.String("dynamic_service_name", s.name),
		)
		c.Abort()
		return
	}

	service, found := s.services[serviceName]
	if !found {
		routeID := variable.GetString(variable.RouteID, c)
		fullURI := fullURI(&c.Request)
		logger.Warn("service name is not found",
			slog.String("route_id", routeID),
			slog.String("client_ip", c.ClientIP()),
			slog.String("full_uri", fullURI),
			slog.String("service_name", s.name),
		)
		c.Abort()
		return
	}
	c.Set(variable.ServiceID, serviceName)

	middlewares := make([]app.HandlerFunc, len(service.middlewares)+1)
	copy(middlewares, service.middlewares)
	middlewares[len(service.middlewares)] = service.ServeHTTP
	c.SetIndex(-1)
	c.SetHandlers(middlewares)
	c.Next(ctx)
}
