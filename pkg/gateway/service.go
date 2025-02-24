package gateway

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/nite-coder/bifrost/internal/pkg/runtime"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/proxy"
	grpcproxy "github.com/nite-coder/bifrost/pkg/proxy/grpc"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/nite-coder/blackbear/pkg/cast"
)

type Service struct {
	bifrost         *Bifrost
	options         *config.ServiceOptions
	upstreams       map[string]*Upstream
	proxy           proxy.Proxy
	upstream        *Upstream
	dynamicUpstream string
	middlewares     []app.HandlerFunc
}

func loadServices(bifrost *Bifrost) (map[string]*Service, error) {
	services := map[string]*Service{}

	for id, serviceOptions := range bifrost.options.Services {
		if len(id) == 0 {
			return nil, errors.New("service id can't be empty")
		}

		serviceOptions.ID = id

		_, found := services[serviceOptions.ID]
		if found {
			return nil, fmt.Errorf("service '%s' already exists", serviceOptions.ID)
		}

		service, err := newService(bifrost, serviceOptions)
		if err != nil {
			return nil, err
		}
		services[serviceOptions.ID] = service
	}

	return services, nil
}

func newService(bifrost *Bifrost, serviceOptions config.ServiceOptions) (*Service, error) {

	if len(serviceOptions.Protocol) == 0 {
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
			return nil, fmt.Errorf("middleware kind can't be empty in service: '%s'", serviceOptions.ID)
		}

		handler := middleware.FindHandlerByType(middlewareOpts.Type)
		if handler == nil {
			return nil, fmt.Errorf("middleware handler '%s' was not found in service: '%s'", middlewareOpts.Type, serviceOptions.ID)
		}

		appHandler, err := handler(middlewareOpts.Params)
		if err != nil {
			return nil, fmt.Errorf("fail to create middleware '%s' in route: '%s', error: %w", middlewareOpts.Type, serviceOptions.ID, err)
		}

		svc.middlewares = append(svc.middlewares, appHandler)
	}

	addr, err := url.Parse(serviceOptions.Url)
	if err != nil {
		return nil, err
	}

	hostname := addr.Hostname()

	// validate
	if len(hostname) == 0 {
		return nil, fmt.Errorf("the hostname is invalid in service url. service_id: %s", serviceOptions.ID)
	}

	// dynamic upstream
	if hostname[0] == '$' {
		svc.dynamicUpstream = hostname
		return svc, nil
	}

	// exist upstream
	// the hostname can be not found in upstreams because user can use real domain name like www.google.com
	upstream, found := svc.upstreams[hostname]
	if found {
		svc.upstream = upstream
		return svc, nil
	}

	// direct proxy
	switch serviceOptions.Protocol {
	case config.ProtocolHTTP, config.ProtocolHTTP2:
		httpProxy, err := initHTTPProxy(bifrost, serviceOptions, addr)
		if err != nil {
			return nil, err
		}
		svc.proxy = httpProxy
	case config.ProtocolGRPC:
		grpcProxy, err := initGRPCProxy(bifrost, serviceOptions, addr)
		if err != nil {
			return nil, err
		}
		svc.proxy = grpcProxy
	}

	return svc, nil
}

func (svc *Service) Middlewares() []app.HandlerFunc {
	return svc.middlewares
}

func (svc *Service) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)
	done := make(chan bool)

	runTask(ctx, func() {
		defer func() {
			done <- true
			if r := recover(); r != nil {
				stackTrace := runtime.StackTrace()
				logger.ErrorContext(ctx, "service panic recovered", slog.Any("panic", r), slog.String("stack", stackTrace))
				c.Abort()
			}
		}()

		if len(svc.dynamicUpstream) > 0 {
			upstreamName := c.GetString(svc.dynamicUpstream)

			if len(upstreamName) == 0 {
				logger.Warn("upstream is empty", slog.String("path", cast.B2S(c.Request.Path())))
				c.Abort()
				return
			}

			var found bool
			svc.upstream, found = svc.upstreams[upstreamName]
			if !found {
				logger.Warn("upstream is not found", slog.String("name", upstreamName))
				c.Abort()
				return
			}
		}

		proxy := svc.proxy
		if svc.upstream != nil && proxy == nil {
			c.Set(variable.UpstreamID, svc.upstream.opts.ID)

			switch svc.upstream.opts.Strategy {
			case config.RoundRobinStrategy, "":
				proxy = svc.upstream.roundRobin()
			case config.WeightedStrategy:
				proxy = svc.upstream.weighted()
			case config.RandomStrategy:
				proxy = svc.upstream.random()
			case config.HashingStrategy:
				hashon := svc.upstream.opts.HashOn
				val := variable.GetString(hashon, c)
				proxy = svc.upstream.hasing(val)
			}
		}

		if proxy == nil {
			reqMethod := cast.B2S(c.Request.Method())
			reqPath := cast.B2S(c.Request.Path())
			reqProtocol := c.Request.Header.GetProtocol()

			logger.ErrorContext(ctx, ErrNoLiveUpstream.Error(),
				"request_uri", reqMethod+" "+reqPath+" "+reqProtocol,
				"upstream_uri", reqPath,
				"host", cast.B2S(c.Request.Host()))

			// no live upstream
			c.SetStatusCode(503)
			return
		}

		startTime := timecache.Now()
		proxy.ServeHTTP(ctx, c)
		endTime := timecache.Now()

		dur := endTime.Sub(startTime)
		mic := dur.Microseconds()
		duration := float64(mic) / 1e6
		responseTime := strconv.FormatFloat(duration, 'f', -1, 64)
		c.Set(variable.UpstreamDuration, responseTime)

		// the upstream target timeout and we need to response http status 504 back to client
		if c.GetBool(variable.TargetTimeout) {
			c.Response.SetStatusCode(504)
		} else {
			c.Set(variable.UpstreamResponoseStatusCode, c.Response.StatusCode())
		}
	})

	select {
	case <-ctx.Done():
		fullURI := fullURI(&c.Request)
		logger.WarnContext(ctx, "client cancel the request",
			slog.String("full_uri", fullURI),
		)

		// The client canceled the request
		c.Response.SetStatusCode(499)
	case <-done:
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

func (svc *DynamicService) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	logger := log.FromContext(c)
	serviceName := ctx.GetString(svc.name)

	if len(serviceName) == 0 {
		logger.Error("service name is empty", slog.String("path", cast.B2S(ctx.Request.Path())))
		ctx.Abort()
		return
	}

	service, found := svc.services[serviceName]
	if !found {
		logger.Warn("service is not found", slog.String("name", serviceName))
		ctx.Abort()
		return
	}
	ctx.Set(variable.ServiceID, serviceName)

	service.middlewares = append(service.middlewares, service.ServeHTTP)
	ctx.SetIndex(-1)
	ctx.SetHandlers(service.middlewares)
	ctx.Next(c)
}

func initHTTPProxy(bifrost *Bifrost, serviceOptions config.ServiceOptions, addr *url.URL) (proxy.Proxy, error) {
	clientOpts := httpproxy.DefaultClientOptions()

	if serviceOptions.Timeout.Dail > 0 {
		clientOpts = append(clientOpts, client.WithDialTimeout(serviceOptions.Timeout.Dail))
	}

	if serviceOptions.Timeout.Read > 0 {
		clientOpts = append(clientOpts, client.WithClientReadTimeout(serviceOptions.Timeout.Read))
	}

	if serviceOptions.Timeout.Write > 0 {
		clientOpts = append(clientOpts, client.WithWriteTimeout(serviceOptions.Timeout.Write))
	}

	if serviceOptions.Timeout.MaxConnWait > 0 {
		clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(serviceOptions.Timeout.MaxConnWait))
	}

	if serviceOptions.MaxConnsPerHost != nil {
		clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*serviceOptions.MaxConnsPerHost))
	}

	hostname := addr.Hostname()

	if strings.EqualFold(addr.Scheme, "https") {
		clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
			ServerName:         hostname,
			InsecureSkipVerify: !serviceOptions.TLSVerify, //nolint:gosec
		}))
	}

	ips, err := bifrost.dnsResolver.Lookup(context.Background(), hostname)
	if err != nil {
		return nil, fmt.Errorf("lookup host '%s' error: %w", hostname, err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("lookup host '%s' error: no ip found", hostname)
	}

	url := fmt.Sprintf("%s://%s%s", addr.Scheme, ips[0], addr.Path)
	if addr.Port() != "" {
		url = fmt.Sprintf("%s://%s:%s%s", addr.Scheme, ips[0], addr.Port(), addr.Path)
	}

	clientOptions := httpproxy.ClientOptions{
		IsHTTP2:   serviceOptions.Protocol == config.ProtocolHTTP2,
		HZOptions: clientOpts,
	}

	client, err := httpproxy.NewClient(clientOptions)
	if err != nil {
		return nil, err
	}

	proxyOptions := httpproxy.Options{
		Target:           url,
		Protocol:         serviceOptions.Protocol,
		Weight:           0,
		HeaderHost:       hostname,
		IsTracingEnabled: bifrost.options.Tracing.Enabled,
		ServiceID:        serviceOptions.ID,
	}

	httpProxy, err := httpproxy.New(proxyOptions, client)
	if err != nil {
		return nil, err
	}

	return httpProxy, nil
}

func initGRPCProxy(bifrost *Bifrost, serviceOptions config.ServiceOptions, addr *url.URL) (proxy.Proxy, error) {
	hostname := addr.Hostname()

	url := fmt.Sprintf("grpc://%s%s", hostname, addr.Path)
	if addr.Port() != "" {
		url = fmt.Sprintf("grpc://%s:%s%s", hostname, addr.Port(), addr.Path)
	}

	grpcOptions := grpcproxy.Options{
		Target:    url,
		TLSVerify: serviceOptions.TLSVerify,
		Weight:    1,
		Timeout:   serviceOptions.Timeout.GRPC,
	}

	grpcProxy, err := grpcproxy.New(grpcOptions)
	if err != nil {
		return nil, err
	}

	return grpcProxy, nil
}
