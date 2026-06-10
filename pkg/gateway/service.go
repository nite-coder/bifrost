package gateway

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
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
	"github.com/nite-coder/bifrost/pkg/telemetry/metrics"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const statusClientClosedRequest = 499

// Service represents a backend service that can have multiple upstreams and middlewares.
type Service struct {
	bifrost         *Bifrost
	options         *config.ServiceOptions
	upstream        *Upstream
	dynamicUpstream string
	middlewares     []app.HandlerFunc
	balancer        atomic.Value
	mu              sync.RWMutex
	balancers       map[string]balancer.Balancer
	activeProxies   map[string]proxy.Proxy
	upstreamProxies map[string]map[string]proxy.Proxy
	subscriptions   map[string]<-chan []*proxy.Endpoint
	cancelFuncs     []context.CancelFunc
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
	s.subscriptions = make(map[string]<-chan []*proxy.Endpoint)

	if s.upstream != nil && s.upstream.isExclusive {
		_ = s.upstream.Close()
	}

	for _, p := range s.activeProxies {
		_ = p.Close()
	}
	s.activeProxies = make(map[string]proxy.Proxy)
	s.upstreamProxies = make(map[string]map[string]proxy.Proxy)

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
		bifrost:         bifrost,
		options:         &serviceOptions,
		balancers:       make(map[string]balancer.Balancer),
		activeProxies:   make(map[string]proxy.Proxy),
		upstreamProxies: make(map[string]map[string]proxy.Proxy),
		subscriptions:   make(map[string]<-chan []*proxy.Endpoint),
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

func generateServiceProxyHash(target string, tags map[string]string) string {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var builder strings.Builder
	_, _ = builder.WriteString(target)
	for _, k := range keys {
		_, _ = builder.WriteString(";")
		_, _ = builder.WriteString(k)
		_, _ = builder.WriteString("=")
		_, _ = builder.WriteString(tags[k])
	}

	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
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

	var myProxy proxy.Proxy
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
		myProxy, err = bal.Select(ctx, c)
	} else if s.upstream != nil {
		c.Set(variable.UpstreamID, s.upstream.options.ID)

		val := s.balancer.Load()
		if val == nil {
			logger.Warn("balancer is nil, upstream may not be initialized",
				"upstream_id", s.upstream.options.ID,
				"service_id", s.options.ID,
			)
			c.SetStatusCode(http.StatusServiceUnavailable)
			return
		}
		bal, ok := val.(balancer.Balancer)
		if !ok {
			logger.Warn("balancer type assertion failed",
				"upstream_id", s.upstream.options.ID,
				"service_id", s.options.ID,
			)
			c.SetStatusCode(http.StatusServiceUnavailable)
			return
		}
		myProxy, err = bal.Select(ctx, c)
	}

	if myProxy == nil || err != nil {
		c.SetStatusCode(http.StatusServiceUnavailable)

		if !errors.Is(err, balancer.ErrNotAvailable) {
			_ = c.Error(err)
		}
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
	upstream.isExclusive = true
	s.subscribeToUpstream(upstream)

	return nil
}

func (s *Service) subscribeToUpstream(upstream *Upstream) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subscriptions[upstream.options.ID]; exists {
		return
	}

	ch := upstream.Subscribe()
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

func (s *Service) updateEndpoints(upstreamID string, endpoints []*proxy.Endpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var upstreamOptions *config.UpstreamOptions
	var modelOpts *config.AIModelOptions
	if after, ok := strings.CutPrefix(upstreamID, "ai:"); ok {
		modelID := after
		var found bool
		modelOpts, found = s.bifrost.options.Models[modelID]
		if !found {
			return
		}
		balancerType := "weighted"
		if modelOpts.Balancer != nil && modelOpts.Balancer.Type != "" {
			balancerType = modelOpts.Balancer.Type
		}
		upstreamOptions = &config.UpstreamOptions{
			ID: upstreamID,
			Balancer: config.BalancerOptions{
				Type: balancerType,
			},
		}
	} else {
		var u *Upstream
		var found bool
		if s.bifrost.upstreamManager != nil {
			u, found = s.bifrost.upstreamManager.Get(upstreamID)
		}
		switch {
		case found:
			upstreamOptions = u.options
		case s.upstream != nil && s.upstream.options.ID == upstreamID:
			upstreamOptions = s.upstream.options
		default:
			return
		}
	}

	newProxies := make([]proxy.Proxy, 0, len(endpoints))
	newProxyHashes := make(map[string]proxy.Proxy)

	for _, ep := range endpoints {
		var targetHost, targetPort string
		var splitErr error
		if strings.HasPrefix(upstreamID, "ai:") {
			targetHost = ep.Address
			targetPort = ""
		} else {
			targetHost, targetPort, splitErr = net.SplitHostPort(ep.Address)
			if splitErr != nil {
				targetHost = ep.Address
				targetPort = "0"
			}
		}

		var myURL string
		if !strings.HasPrefix(upstreamID, "ai:") {
			addr, parseErr := url.Parse(s.options.URL)
			if parseErr != nil {
				slog.Error("failed to parse service URL", "url", s.options.URL, "error", parseErr)
				continue
			}
			port := ""
			if len(addr.Port()) > 0 {
				port = addr.Port()
			} else if targetPort != "" && targetPort != "0" {
				port = targetPort
			}
			myURL = fmt.Sprintf("%s://%s%s", addr.Scheme, targetHost, addr.Path)
			if port != "" {
				myURL = fmt.Sprintf("%s://%s:%s%s", addr.Scheme, targetHost, port, addr.Path)
			}
		} else {
			myURL = targetHost
		}

		hash := generateServiceProxyHash(myURL, ep.Tags)

		if existing, found := s.activeProxies[hash]; found {
			newEp := &proxy.Endpoint{
				Address:     ep.Address,
				Weight:      ep.Weight,
				Tags:        ep.Tags,
				HealthState: ep.HealthState,
			}
			existing.SetEndpoint(newEp)
			newProxies = append(newProxies, existing)
			newProxyHashes[hash] = existing
			continue
		}

		var p proxy.Proxy
		if strings.HasPrefix(upstreamID, "ai:") {
			parts := strings.SplitN(ep.Address, "/", aiproxy.TargetPartsCount)
			if len(parts) != aiproxy.TargetPartsCount {
				slog.Error("invalid target format in AI model", "target", ep.Address)
				continue
			}
			providerID := parts[0]
			actualModel := parts[1]

			handler := ""
			if s.bifrost.options.AI != nil && s.bifrost.options.AI.Providers != nil {
				if prov, ok := s.bifrost.options.AI.Providers[providerID]; ok {
					handler = prov.Handler
				}
			}

			var targetPricing *config.AIPricingOptions
			if modelOpts != nil {
				for _, t := range modelOpts.Targets {
					if t.Target == ep.Address {
						targetPricing = t.Pricing
						break
					}
				}
			}
			finalPricing := pricing.Resolve(handler, actualModel, targetPricing)

			metricsEnabled := s.bifrost.options.Metrics.Prometheus.Enabled || s.bifrost.options.Metrics.OTLP.Enabled
			var pErr error
			p, pErr = aiproxy.NewProxy(aiproxy.ProxyOptions{
				ID:             ep.Address,
				Target:         ep.Address,
				AIOptions:      s.bifrost.options.AI,
				MetricsEnabled: metricsEnabled,
				Pricing:        finalPricing,
				Endpoint:       ep,
			})
			if pErr != nil {
				slog.Error("failed to create AI proxy", "error", pErr)
				continue
			}
		} else {
			serverName := ep.Tags["server_name"]
			switch s.options.Protocol {
			case "", config.ProtocolHTTP, config.ProtocolHTTP2:
				clientOpts := httpproxy.DefaultClientOptions()
				if s.options.Timeout.Dail > 0 {
					clientOpts = append(clientOpts, client.WithDialTimeout(s.options.Timeout.Dail))
				} else if s.bifrost.options.Default.Service.Timeout.Dail > 0 {
					clientOpts = append(clientOpts, client.WithDialTimeout(s.bifrost.options.Default.Service.Timeout.Dail))
				}
				if s.options.Timeout.Read > 0 {
					clientOpts = append(clientOpts, client.WithClientReadTimeout(s.options.Timeout.Read))
				} else if s.bifrost.options.Default.Service.Timeout.Read > 0 {
					clientOpts = append(
						clientOpts,
						client.WithClientReadTimeout(s.bifrost.options.Default.Service.Timeout.Read),
					)
				}
				if s.options.Timeout.Write > 0 {
					clientOpts = append(clientOpts, client.WithWriteTimeout(s.options.Timeout.Write))
				} else if s.bifrost.options.Default.Service.Timeout.Write > 0 {
					clientOpts = append(clientOpts, client.WithWriteTimeout(s.bifrost.options.Default.Service.Timeout.Write))
				}
				if s.options.Timeout.MaxConnWait > 0 {
					clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(s.options.Timeout.MaxConnWait))
				} else if s.bifrost.options.Default.Service.Timeout.MaxConnWait > 0 {
					clientOpts = append(
						clientOpts,
						client.WithMaxConnWaitTimeout(s.bifrost.options.Default.Service.Timeout.MaxConnWait),
					)
				}
				if s.options.MaxConnsPerHost != nil {
					clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*s.options.MaxConnsPerHost))
				} else if s.bifrost.options.Default.Service.MaxConnsPerHost != nil {
					clientOpts = append(
						clientOpts,
						client.WithMaxConnsPerHost(*s.bifrost.options.Default.Service.MaxConnsPerHost),
					)
				}

				addr, _ := url.Parse(s.options.URL)
				if addr != nil && strings.EqualFold(addr.Scheme, "https") {
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

				clientOptions := httpproxy.ClientOptions{
					IsHTTP2:   s.options.Protocol == config.ProtocolHTTP2,
					HZOptions: clientOpts,
				}
				httpClient, clientErr := httpproxy.NewClient(clientOptions)
				if clientErr != nil {
					slog.Error("failed to create http client", "error", clientErr)
					continue
				}

				proxyOptions := httpproxy.Options{
					Target:           myURL,
					Protocol:         s.options.Protocol,
					IsTracingEnabled: s.bifrost.options.Tracing.Enabled,
					ServiceID:        s.options.ID,
					TargetHostHeader: serverName,
					PassHostHeader:   s.options.IsPassHostHeader(),
					Endpoint:         ep,
				}
				var pErr error
				p, pErr = httpproxy.New(proxyOptions, httpClient)
				if pErr != nil {
					slog.Error("failed to create http proxy", "error", pErr)
					continue
				}
			case config.ProtocolGRPC:
				grpcOptions := grpcproxy.Options{
					Target:           myURL,
					TLSVerify:        s.options.TLSVerify,
					IsTracingEnabled: s.bifrost.options.Tracing.Enabled,
					Timeout:          s.options.Timeout.GRPC,
					Endpoint:         ep,
				}
				var pErr error
				p, pErr = grpcproxy.New(grpcOptions)
				if pErr != nil {
					slog.Error("failed to create grpc proxy", "error", pErr)
					continue
				}
			default:
				slog.Error("unsupported protocol", "protocol", s.options.Protocol)
				continue
			}
		}

		if p != nil {
			newProxies = append(newProxies, p)
			newProxyHashes[hash] = p
			s.activeProxies[hash] = p
		}
	}

	factory := balancer.Factory(upstreamOptions.Balancer.Type)
	if factory == nil {
		slog.Error("unsupported balancer type", "type", upstreamOptions.Balancer.Type)
		return
	}

	bal, balancerErr := factory(newProxies, upstreamOptions.Balancer.Params)
	if balancerErr != nil {
		slog.Error("failed to create balancer", "upstream_id", upstreamID, "error", balancerErr)
		return
	}

	if s.upstream != nil && s.upstream.options.ID == upstreamID {
		s.balancer.Store(bal)
	}
	s.balancers[upstreamID] = bal

	oldProxies := s.upstreamProxies[upstreamID]
	s.upstreamProxies[upstreamID] = newProxyHashes

	for hash, p := range oldProxies {
		if _, found := newProxyHashes[hash]; !found {
			// Check if the proxy is still used by any other upstream of this service
			inUse := false
			for id, proxies := range s.upstreamProxies {
				if id != upstreamID {
					if _, exists := proxies[hash]; exists {
						inUse = true
						break
					}
				}
			}
			if !inUse {
				_ = p.Close()
				delete(s.activeProxies, hash)
			}
		}
	}
}

func (s *Service) getBalancer(upstreamName string) (balancer.Balancer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bal, found := s.balancers[upstreamName]
	return bal, found
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
