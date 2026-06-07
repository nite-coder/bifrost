package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"github.com/nite-coder/blackbear/pkg/cast"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/proxy"
	aiproxy "github.com/nite-coder/bifrost/pkg/proxy/ai"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const statusClientClosedRequest = 499

// Service represents a backend service that can have multiple upstreams and middlewares.
type Service struct {
	bifrost         *Bifrost
	options         *config.ServiceOptions
	upstreams       map[string]*Upstream
	upstream        *Upstream
	dynamicUpstream string
	middlewares     []app.HandlerFunc
}

// Close releases resources used by the service and its upstreams.
func (s *Service) Close() error {
	for _, upstream := range s.upstreams {
		_ = upstream.Close()
	}

	if s.upstream != nil {
		found := false
		for _, u := range s.upstreams {
			if u == s.upstream {
				found = true
				break
			}
		}
		if !found {
			_ = s.upstream.Close()
		}
	}

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
		currentServiceOptions := serviceOptions // Create a new variable for the goroutine
		currentID := id                         // Create a new variable for the goroutine

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
			// cleanup
			for _, svc := range services {
				_ = svc.Close()
			}
			return nil, err // Return the first error encountered
		}
	}

	return services, nil
}

func newService(bifrost *Bifrost, serviceOptions config.ServiceOptions) (*Service, error) {
	svc := &Service{
		bifrost: bifrost,
		options: &serviceOptions,
	}

	svc.applyProtocolDefaults()

	upstreams, err := loadUpstreams(bifrost, serviceOptions)
	if err != nil {
		return nil, err
	}
	svc.upstreams = upstreams

	if err := svc.initMiddlewares(); err != nil {
		return nil, err
	}

	if err := svc.loadModels(); err != nil {
		return nil, err
	}

	if serviceOptions.Type == "ai" {
		svc.dynamicUpstream = variable.AIModelName
	} else {
		if err := svc.resolveUpstreamStrategy(); err != nil {
			return nil, err
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
			// the request is canceled by client
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

			// client canceled the request
			c.Response.SetStatusCode(statusClientClosedRequest)
			return
		}
	}

	targetUpstream := s.upstream
	if len(s.dynamicUpstream) > 0 {
		upstreamName := variable.GetString(s.dynamicUpstream, c)

		if len(upstreamName) == 0 {
			logger.Warn("upstream is empty",
				slog.String("path", string(c.Request.Path())),
			)
			c.Abort()
			return
		}

		var found bool
		targetUpstream, found = s.upstreams[upstreamName]
		if !found {
			logger.Warn("upstream is not found",
				slog.String("name", upstreamName),
			)
			c.Abort()
			return
		}
		targetUpstream.watch()
	}

	var myProxy proxy.Proxy
	var err error
	if targetUpstream != nil {
		c.Set(variable.UpstreamID, targetUpstream.options.ID)

		balaner := targetUpstream.Balancer()
		if balaner == nil {
			logger.Warn("balancer is nil, upstream may not be initialized",
				"upstream_id", targetUpstream.options.ID,
				"service_id", s.options.ID,
			)
			c.SetStatusCode(http.StatusServiceUnavailable)
			return
		}

		myProxy, err = balaner.Select(ctx, c)
	}

	if myProxy == nil || err != nil {
		// no live upstream
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

	// the upstream target timeout and we need to response http status 504 back to client
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

	// validate
	if len(hostname) == 0 {
		return fmt.Errorf("hostname is invalid in service URL for service ID: %s", s.options.ID)
	}

	// dynamic upstream
	if hostname[0] == '$' {
		s.dynamicUpstream = hostname
		return nil
	}

	// exist upstream
	// the hostname cannot be found in upstreams because user can use real domain name like www.google.com
	upstream, found := s.upstreams[hostname]
	if found {
		s.upstream = upstream
		upstream.watch()
		return nil
	}

	// direct proxy
	upstreamOptions := config.UpstreamOptions{
		ID: uuid.NewString(),
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

	upstream, err := newUpstream(s.bifrost, *s.options, upstreamOptions)
	if err != nil {
		return err
	}
	s.upstream = upstream
	upstream.watch()

	return nil
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

	// Create a new slice to avoid modifying the original service.middlewares
	middlewares := make([]app.HandlerFunc, len(service.middlewares)+1)
	copy(middlewares, service.middlewares)
	middlewares[len(service.middlewares)] = service.ServeHTTP
	c.SetIndex(-1)
	c.SetHandlers(middlewares)
	c.Next(ctx)
}

func (s *Service) loadModels() error {
	if s.upstreams == nil {
		s.upstreams = make(map[string]*Upstream)
	}

	if s.bifrost.options.Models == nil {
		return nil
	}

	for modelID, modelOpts := range s.bifrost.options.Models {
		if len(modelID) == 0 {
			continue
		}

		var targets []config.TargetOptions
		for _, t := range modelOpts.Targets {
			var weight uint32
			if t.Weight > 0 && t.Weight <= 4294967295 {
				weight = uint32(t.Weight)
			}
			if weight <= 0 {
				weight = 1
			}
			targets = append(targets, config.TargetOptions{
				Target: t.Target,
				Weight: weight,
			})
		}

		balancerType := "weighted"
		if modelOpts.Balancer != nil && modelOpts.Balancer.Type != "" {
			balancerType = modelOpts.Balancer.Type
		}

		upstreamOpts := config.UpstreamOptions{
			ID: "ai:" + modelID,
			Balancer: config.BalancerOptions{
				Type: balancerType,
			},
			Targets: targets,
		}

		var proxies []proxy.Proxy
		metricsEnabled := s.bifrost.options.Metrics.Prometheus.Enabled || s.bifrost.options.Metrics.OTLP.Enabled
		for _, t := range targets {
			p := aiproxy.NewProxy(t.Target, t.Target, t.Weight, s.bifrost.options.AI, metricsEnabled)
			proxies = append(proxies, p)
		}

		factory := balancer.Factory(balancerType)
		if factory == nil {
			return fmt.Errorf("unsupported balancer strategy '%s' for virtual model: %s", balancerType, modelID)
		}

		bal, err := factory(proxies, nil)
		if err != nil {
			return fmt.Errorf("failed to create balancer for virtual model %s: %w", modelID, err)
		}

		u := &Upstream{
			bifrost:        s.bifrost,
			options:        &upstreamOpts,
			serviceOptions: s.options,
		}
		u.balancer.Store(bal)

		s.upstreams["ai:"+modelID] = u
	}

	return nil
}
