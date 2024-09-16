package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/proxy"
	grpcproxy "http-benchmark/pkg/proxy/grpc"
	httpproxy "http-benchmark/pkg/proxy/http"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/rs/dnscache"
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

func loadServices(bifrost *Bifrost, middlewares map[string]app.HandlerFunc) (map[string]*Service, error) {
	services := map[string]*Service{}
	for id, serviceOpts := range bifrost.options.Services {

		if len(id) == 0 {
			return nil, fmt.Errorf("service id can't be empty")
		}

		serviceOpts.ID = id

		_, found := services[serviceOpts.ID]
		if found {
			return nil, fmt.Errorf("service '%s' already exists", serviceOpts.ID)
		}

		service, err := newService(bifrost, serviceOpts)
		if err != nil {
			return nil, err
		}
		services[serviceOpts.ID] = service

		if len(serviceOpts.Middlewares) > 0 {
			for _, middlewareOpts := range serviceOpts.Middlewares {
				middleware, found := middlewares[middlewareOpts.ID]
				if !found {
					return nil, fmt.Errorf("middleware '%s' not found", middlewareOpts)
				}
				service.middlewares = append(service.middlewares, middleware)
			}
		}
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

func (svc *Service) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	logger := log.FromContext(c)
	defer ctx.Abort()
	done := make(chan bool)

	runTask(c, func() {
		defer func() {
			done <- true
			if r := recover(); r != nil {
				stackTrace := getStackTrace()
				logger.ErrorContext(c, "proxy panic recovered", slog.Any("panic", r), "stack", stackTrace)
				ctx.Abort()
			}
		}()

		if len(svc.dynamicUpstream) > 0 {
			upstreamName := ctx.GetString(svc.dynamicUpstream)

			if len(upstreamName) == 0 {
				logger.Warn("upstream is empty", slog.String("path", cast.B2S(ctx.Request.Path())))
				ctx.Abort()
				return
			}

			var found bool
			svc.upstream, found = svc.upstreams[upstreamName]
			if !found {
				logger.Warn("upstream is not found", slog.String("name", upstreamName))
				ctx.Abort()
				return
			}
		}

		proxy := svc.proxy
		if svc.upstream != nil && proxy == nil {
			ctx.Set(config.UPSTREAM, svc.upstream.opts.ID)

			switch svc.upstream.opts.Strategy {
			case config.RoundRobinStrategy, "":
				proxy = svc.upstream.roundRobin()
			case config.WeightedStrategy:
				proxy = svc.upstream.weighted()
			case config.RandomStrategy:
				proxy = svc.upstream.random()
			case config.HashingStrategy:
				hashon := svc.upstream.opts.HashOn
				val := ctx.GetString(hashon)
				proxy = svc.upstream.hasing(val)
			}
		}

		if proxy == nil {
			reqMethod := cast.B2S(ctx.Request.Method())
			reqPath := ctx.Request.Path()
			reqProtocol := ctx.Request.Header.GetProtocol()

			logger.ErrorContext(c, ErrNoLiveUpstream.Error(),
				"request_uri", fmt.Sprintf("%s %s %s", reqMethod, reqPath, reqProtocol),
				"upstream_uri", reqPath,
				"host", cast.B2S(ctx.Request.Host()))

			// no live upstream
			ctx.SetStatusCode(503)
			return
		}

		startTime := time.Now()
		proxy.ServeHTTP(c, ctx)

		dur := time.Since(startTime)
		mic := dur.Microseconds()
		duration := float64(mic) / 1e6
		responseTime := strconv.FormatFloat(duration, 'f', -1, 64)
		ctx.Set(config.UPSTREAM_DURATION, responseTime)

		// the upstream target timeout and we need to response http status 504 back to client
		if ctx.GetBool("target_timeout") {
			ctx.Response.SetStatusCode(504)
		} else {
			ctx.Set(config.UPSTREAM_STATUS, ctx.Response.StatusCode())
		}

		// check upstream health
		// TODO: move to http proxy
		if ctx.Response.StatusCode() >= 500 {
			err := proxy.AddFailedCount(1)
			if err != nil {
				slog.Warn("upstream server temporarily disabled")
			}
		}
	})

	select {
	case <-c.Done():
		time := time.Now()
		ctx.Set(config.CLIENT_CANCELED_AT, time)

		fullURI := fullURI(&ctx.Request)
		logger.WarnContext(c, "client cancel the request",
			slog.String("full_uri", fullURI),
		)

		// The client canceled the request
		ctx.Response.SetStatusCode(499)
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

	service.ServeHTTP(c, ctx)
}

func initHTTPProxy(bifrost *Bifrost, opts config.ServiceOptions, addr *url.URL) (proxy.Proxy, error) {
	clientOpts := httpproxy.DefaultClientOptions()

	if opts.Timeout.Dail > 0 {
		clientOpts = append(clientOpts, client.WithDialTimeout(opts.Timeout.Dail))
	}

	if opts.Timeout.Read > 0 {
		clientOpts = append(clientOpts, client.WithClientReadTimeout(opts.Timeout.Read))
	}

	if opts.Timeout.Write > 0 {
		clientOpts = append(clientOpts, client.WithWriteTimeout(opts.Timeout.Write))
	}

	if opts.Timeout.MaxConnWait > 0 {
		clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(opts.Timeout.MaxConnWait))
	}

	if opts.MaxConnsPerHost != nil {
		clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*opts.MaxConnsPerHost))
	}

	hostname := addr.Hostname()

	var dnsResolver dnscache.DNSResolver
	if allowDNS(hostname) {
		_, err := bifrost.resolver.LookupHost(context.Background(), hostname)
		if err != nil {
			return nil, fmt.Errorf("lookup service host error: %v", err)
		}
		dnsResolver = bifrost.resolver
	}

	switch strings.ToLower(addr.Scheme) {
	case "http":
		if dnsResolver != nil {
			clientOpts = append(clientOpts, client.WithDialer(newHTTPDialer(dnsResolver)))
		}
	case "https":
		if dnsResolver != nil {
			clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
				InsecureSkipVerify: !opts.TLSVerify,
			}))
			clientOpts = append(clientOpts, client.WithDialer(newHTTPSDialer(dnsResolver)))
		}
	}

	url := fmt.Sprintf("%s://%s%s", addr.Scheme, hostname, addr.Path)
	if addr.Port() != "" {
		url = fmt.Sprintf("%s://%s:%s%s", addr.Scheme, hostname, addr.Port(), addr.Path)
	}

	clientOptions := httpproxy.ClientOptions{
		IsTracingEnabled: bifrost.options.Tracing.OTLP.Enabled,
		IsHTTP2:          opts.Protocol == config.ProtocolHTTP2,
		HZOptions:        clientOpts,
	}

	client, err := httpproxy.NewClient(clientOptions)
	if err != nil {
		return nil, err
	}

	proxyOptions := httpproxy.Options{
		Target:   url,
		Protocol: opts.Protocol,
		Weight:   0,
	}

	httpProxy, err := httpproxy.New(proxyOptions, client)
	if err != nil {
		return nil, err
	}

	return httpProxy, nil
}

func initGRPCProxy(bifrost *Bifrost, opts config.ServiceOptions, addr *url.URL) (proxy.Proxy, error) {
	hostname := addr.Hostname()

	url := fmt.Sprintf("grpc://%s%s", hostname, addr.Path)
	if addr.Port() != "" {
		url = fmt.Sprintf("grpc://%s:%s%s", hostname, addr.Port(), addr.Path)
	}

	grpcOptions := grpcproxy.Options{
		Target:    url,
		TLSVerify: opts.TLSVerify,
		Weight:    1,
	}

	grpcProxy, err := grpcproxy.New(grpcOptions)
	if err != nil {
		return nil, err
	}

	return grpcProxy, nil
}
