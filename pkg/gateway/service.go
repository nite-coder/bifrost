package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"runtime/debug"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/nite-coder/blackbear/pkg/cast"
)

type Service struct {
	bifrost         *Bifrost
	options         *config.ServiceOptions
	upstreams       map[string]*Upstream
	upstream        *Upstream
	dynamicUpstream string
	middlewares     []app.HandlerFunc
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
			return nil, err // Return the first error encountered
		}
	}

	return services, nil
}

func newService(bifrost *Bifrost, serviceOptions config.ServiceOptions) (*Service, error) {

	if len(serviceOptions.Protocol) == 0 && len(bifrost.options.Default.Service.Protocol) > 0 {
		serviceOptions.Protocol = bifrost.options.Default.Service.Protocol
	} else if len(serviceOptions.Protocol) == 0 && len(bifrost.options.Default.Service.Protocol) == 0 {
		serviceOptions.Protocol = config.ProtocolHTTP
	}

	upstreams, err := loadUpstreams(bifrost, serviceOptions)
	if err != nil {
		return nil, err
	}

	svc := &Service{
		bifrost:     bifrost,
		options:     &serviceOptions,
		upstreams:   upstreams,
		middlewares: make([]app.HandlerFunc, 0),
	}

	for _, middlewareOpts := range serviceOptions.Middlewares {

		if len(middlewareOpts.Use) > 0 {
			m, found := bifrost.middlewares[middlewareOpts.Use]
			if !found {
				return nil, fmt.Errorf("middleware '%s' not found in service: '%s'", middlewareOpts, serviceOptions.ID)
			}
			svc.middlewares = append(svc.middlewares, m)
			continue
		}

		if len(middlewareOpts.Type) == 0 {
			return nil, fmt.Errorf("middleware type cannot be empty for service: %s", serviceOptions.ID)
		}

		handler := middleware.Factory(middlewareOpts.Type)
		if handler == nil {
			return nil, fmt.Errorf("middleware handler '%s' was not found in service: '%s'", middlewareOpts.Type, serviceOptions.ID)
		}

		appHandler, err := handler(middlewareOpts.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to create middleware '%s' in route: '%s', error: %w", middlewareOpts.Type, serviceOptions.ID, err)
		}

		svc.middlewares = append(svc.middlewares, appHandler)
	}

	addr, err := url.Parse(serviceOptions.URL)
	if err != nil {
		return nil, err
	}

	hostname := addr.Hostname()

	// validate
	if len(hostname) == 0 {
		return nil, fmt.Errorf("hostname is invalid in service URL for service ID: %s", serviceOptions.ID)
	}

	// dynamic upstream
	if hostname[0] == '$' {
		svc.dynamicUpstream = hostname
		return svc, nil
	}

	// exist upstream
	// the hostname cannot be found in upstreams because user can use real domain name like www.google.com
	upstream, found := svc.upstreams[hostname]
	if found {
		svc.upstream = upstream
		upstream.watch()
		return svc, nil
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

	upstream, err = newUpstream(bifrost, serviceOptions, upstreamOptions)
	if err != nil {
		return nil, err
	}
	svc.upstream = upstream
	upstream.watch()

	return svc, nil
}

func (svc *Service) Upstream() *Upstream {
	return svc.upstream
}

func (svc *Service) Middlewares() []app.HandlerFunc {
	return svc.middlewares
}

func (svc *Service) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)

	defer func() {
		if r := recover(); r != nil {
			stackTrace := cast.B2S(debug.Stack())
			logger.ErrorContext(ctx, "service panic recovered", slog.Any("panic", r), slog.String("stack", stackTrace))
			c.SetStatusCode(500)
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
				duration = time.Since(httpStart.(time.Time))
			}

			logger.InfoContext(ctx, "client cancel the request",
				slog.String("route_id", routeID),
				slog.String("client_ip", c.ClientIP()),
				slog.String("full_uri", fullURI),
				slog.Duration("duration", duration),
			)

			// client canceled the request
			c.Response.SetStatusCode(499)
			return
		}
	}

	if len(svc.dynamicUpstream) > 0 {
		upstreamName := variable.GetString(svc.dynamicUpstream, c)

		if len(upstreamName) == 0 {
			logger.Warn("upstream is empty",
				slog.String("path", cast.B2S(c.Request.Path())),
			)
			c.Abort()
			return
		}

		var found bool
		svc.upstream, found = svc.upstreams[upstreamName]
		if !found {
			logger.Warn("upstream is not found",
				slog.String("name", upstreamName),
			)
			c.Abort()
			return
		}
		svc.upstream.watch()
	}

	var proxy proxy.Proxy
	var err error
	if svc.upstream != nil {
		c.Set(variable.UpstreamID, svc.upstream.options.ID)

		balaner := svc.upstream.Balancer()
		if balaner == nil {
			logger.Warn("balancer is nil, upstream may not be initialized",
				"upstream_id", svc.upstream.options.ID,
				"service_id", svc.options.ID,
			)
			c.SetStatusCode(503)
			return
		}

		proxy, err = balaner.Select(ctx, c)
	}

	if proxy == nil || err != nil {
		// no live upstream
		c.SetStatusCode(503)

		if !errors.Is(err, balancer.ErrNotAvailable) {
			_ = c.Error(err)
		}
		return
	}

	startTime := timecache.Now()
	proxy.ServeHTTP(ctx, c)
	endTime := timecache.Now()

	dur := endTime.Sub(startTime)
	c.Set(variable.UpstreamDuration, dur)

	// the upstream target timeout and we need to response http status 504 back to client
	if c.GetBool(variable.TargetTimeout) {
		c.Response.SetStatusCode(504)
	} else {
		c.Set(variable.UpstreamResponoseStatusCode, c.Response.StatusCode())
	}
}

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

func (svc *DynamicService) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)
	serviceName := variable.GetString(svc.name, c)

	if len(serviceName) == 0 {
		routeID := variable.GetString(variable.RouteID, c)
		fullURI := fullURI(&c.Request)
		logger.Error("service name is empty",
			slog.String("route_id", routeID),
			slog.String("client_ip", c.ClientIP()),
			slog.String("full_uri", fullURI),
			slog.String("dynamic_service_name", svc.name),
		)
		c.Abort()
		return
	}

	service, found := svc.services[serviceName]
	if !found {
		routeID := variable.GetString(variable.RouteID, c)
		fullURI := fullURI(&c.Request)
		logger.Warn("service name is not found",
			slog.String("route_id", routeID),
			slog.String("client_ip", c.ClientIP()),
			slog.String("full_uri", fullURI),
			slog.String("service_name", svc.name),
		)
		c.Abort()
		return
	}
	c.Set(variable.ServiceID, serviceName)

	// Create a new slice to avoid modifying the original service.middlewares
	middlewares := append(service.middlewares, service.ServeHTTP) // nolint
	c.SetIndex(-1)
	c.SetHandlers(middlewares)
	c.Next(ctx)
}
