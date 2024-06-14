package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/log"
	"io"
	"log/slog"
	"net"
	"os"
	"slices"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"

	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/tracer"

	configHTTP2 "github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
	"github.com/hertz-contrib/pprof"
	"golang.org/x/sys/unix"

	hertzslog "github.com/hertz-contrib/logger/slog"
)

type HTTPServer struct {
	entryOpts domain.EntryOptions
	switcher  *switcher
	server    *server.Hertz
}

func newHTTPServer(bifrost *Bifrost, entryOpts domain.EntryOptions, opts domain.Options, tracers []tracer.Tracer) (*HTTPServer, error) {

	//gopool.SetCap(20000)
	engine, err := NewEngine(bifrost, entryOpts, opts)
	if err != nil {
		return nil, err
	}

	switcher := newSwitcher(engine)

	// hertz server
	logger := hertzslog.NewLogger(hertzslog.WithOutput(io.Discard))
	hlog.SetLevel(hlog.LevelError)
	hlog.SetLogger(logger)
	hlog.SetSilentMode(true)

	hzOpts := []config.Option{
		server.WithHostPorts(entryOpts.Bind),
		server.WithIdleTimeout(entryOpts.IdleTimeout),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithSenseClientDisconnection(true),
		withDefaultServerHeader(true),
		server.WithALPN(true),
	}

	for _, tracer := range tracers {
		hzOpts = append(hzOpts, server.WithTracer(tracer))
	}

	if entryOpts.ReusePort {
		hzOpts = append(hzOpts, server.WithListenConfig(&net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				return c.Control(func(fd uintptr) {
					err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
					if err != nil {
						return
					}
				})
			},
		}))
	}

	var tlsConfig *tls.Config
	if entryOpts.TLS.Enabled {
		tlsConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
		}

		if entryOpts.TLS.CertPEM == "" {
			return nil, fmt.Errorf("cert_pem can't be empty")
		}

		if entryOpts.TLS.KeyPEM == "" {
			return nil, fmt.Errorf("key_pem can't be empty")
		}

		certPEM, err := os.ReadFile(entryOpts.TLS.CertPEM)
		if err != nil {
			return nil, err
		}

		keyPEM, err := os.ReadFile(entryOpts.TLS.KeyPEM)
		if err != nil {
			return nil, err
		}

		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
		hzOpts = append(hzOpts, server.WithTLS(tlsConfig))
	}

	httpServer := &HTTPServer{
		entryOpts: entryOpts,
	}

	h := server.Default(hzOpts...)

	if entryOpts.TLS.Enabled && entryOpts.TLS.HTTP2 {
		// register http2 server factory
		h.AddProtocol("h2", factory.NewServerFactory(
			configHTTP2.WithReadTimeout(time.Minute),
			configHTTP2.WithDisableKeepAlive(false)))

		tlsConfig.NextProtos = append(tlsConfig.NextProtos, "h2")
	}

	h.OnShutdown = append(h.OnShutdown, func(ctx context.Context) {
		// if accessLogTracer != nil {
		// 	accessLogTracer.Shutdown()
		// }

	})

	pprof.Register(h)

	h.Use(switcher.ServeHTTP)

	httpServer.switcher = switcher
	httpServer.server = h

	return httpServer, nil
}

func (s *HTTPServer) Run() {
	slog.Info("starting entry", "id", s.entryOpts.ID, "bind", s.entryOpts.Bind)
	s.server.Spin()
}

type Engine struct {
	ID              string
	opts            domain.Options
	handlers        app.HandlersChain
	middlewares     app.HandlersChain
	notFoundHandler app.HandlerFunc
}

func NewEngine(bifrost *Bifrost, entry domain.EntryOptions, opts domain.Options) (*Engine, error) {

	// middlewares
	middlewares := map[string]app.HandlerFunc{}
	for id, middlewareOpts := range opts.Middlewares {

		if len(id) == 0 {
			return nil, fmt.Errorf("middleware id can't be empty")
		}

		middlewareOpts.ID = id

		if len(middlewareOpts.Kind) == 0 {
			return nil, fmt.Errorf("middleware kind can't be empty")
		}

		handler, found := middlewareFactory[middlewareOpts.Kind]
		if !found {
			return nil, fmt.Errorf("middleware handler '%s' was not found", middlewareOpts.Kind)
		}

		m, err := handler(middlewareOpts.Params)
		if err != nil {
			return nil, err
		}

		middlewares[middlewareOpts.ID] = m
	}

	// services
	services := map[string]*Service{}
	for id, serviceOpts := range opts.Services {

		if len(id) == 0 {
			return nil, fmt.Errorf("service id can't be empty")
		}

		serviceOpts.ID = id

		_, found := services[serviceOpts.ID]
		if found {
			return nil, fmt.Errorf("service '%s' already exists", serviceOpts.ID)
		}

		service, err := newService(bifrost, &serviceOpts, opts.Upstreams)
		if err != nil {
			return nil, err
		}
		services[serviceOpts.ID] = service
	}

	// routes
	router := NewRouter()

	for routeID, routeOpts := range opts.Routes {

		routeOpts.ID = routeID

		if len(routeOpts.Entries) > 0 && !slices.Contains(routeOpts.Entries, entry.ID) {
			continue
		}

		if len(routeOpts.Paths) == 0 {
			return nil, fmt.Errorf("route match can't be empty")
		}

		if len(routeOpts.ServiceID) == 0 {
			return nil, fmt.Errorf("route service_id can't be empty")
		}

		service, found := services[routeOpts.ServiceID]
		if !found {
			return nil, fmt.Errorf("service_id '%s' was not found in route: %s", routeOpts.ServiceID, routeOpts.ID)
		}

		routeMiddlewares := make([]app.HandlerFunc, 0)

		for _, middleware := range routeOpts.Middlewares {
			if len(middleware.Link) > 0 {
				val, found := middlewares[middleware.Link]
				if !found {
					return nil, fmt.Errorf("middleware '%s' was not found in route id: '%s'", middleware.Link, routeOpts.ID)
				}

				routeMiddlewares = append(routeMiddlewares, val)
				continue
			}

			if len(middleware.Kind) == 0 {
				return nil, fmt.Errorf("middleware kind can't be empty in route: '%s'", routeOpts.Paths)
			}

			handler, found := middlewareFactory[middleware.Kind]
			if !found {
				return nil, fmt.Errorf("middleware handler '%s' was not found in route: '%s'", middleware.Kind, routeOpts.Paths)
			}

			m, err := handler(middleware.Params)
			if err != nil {
				return nil, fmt.Errorf("create middleware handler '%s' failed in route: '%s'", middleware.Kind, routeOpts.Paths)
			}

			routeMiddlewares = append(routeMiddlewares, m)
		}

		routeMiddlewares = append(routeMiddlewares, service.ServeHTTP)

		err := router.AddRoute(routeOpts, routeMiddlewares...)
		if err != nil {
			return nil, err
		}
	}

	// engine
	engine := &Engine{
		opts:            opts,
		handlers:        make([]app.HandlerFunc, 0),
		notFoundHandler: nil,
	}

	// init middlewares
	logger, err := log.NewLogger(entry.Logging)
	if err != nil {
		return nil, err
	}
	initMiddleware := newInitMiddleware(entry.ID, logger)
	engine.Use(initMiddleware.ServeHTTP)

	for _, middleware := range entry.Middlewares {

		if len(middleware.Link) > 0 {
			val, found := middlewares[middleware.Link]
			if !found {
				return nil, fmt.Errorf("middleware '%s' was not found in entry id: '%s'", middleware.Link, entry.ID)
			}

			engine.Use(val)
			continue
		}

		if len(middleware.Kind) == 0 {
			return nil, fmt.Errorf("middleware kind can't be empty in entry id: '%s'", entry.ID)
		}

		handler, found := middlewareFactory[middleware.Kind]
		if !found {
			return nil, fmt.Errorf("middleware handler '%s' was not found in entry id: '%s'", middleware.Kind, entry.ID)
		}

		m, err := handler(middleware.Params)
		if err != nil {
			return nil, err
		}

		engine.Use(m)
	}

	engine.Use(router.ServeHTTP)

	return engine, nil
}

func (e *Engine) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	ctx.SetIndex(-1)
	ctx.SetHandlers(e.middlewares)
	ctx.Next(c)
	ctx.Abort()
}

func (e *Engine) OnShutdown() {

}

func (e *Engine) Use(middleware ...app.HandlerFunc) {
	e.handlers = append(e.handlers, middleware...)
	e.middlewares = e.handlers

	if e.notFoundHandler != nil {
		e.middlewares = append(e.handlers, e.notFoundHandler)
	}
}

type switcher struct {
	engine atomic.Value
}

func newSwitcher(engine *Engine) *switcher {
	s := &switcher{}
	s.SetEngine(engine)
	return s
}

func (s *switcher) Engine() *Engine {
	return s.engine.Load().(*Engine)
}

func (s *switcher) SetEngine(engine *Engine) {
	s.engine.Store(engine)
}

func (s *switcher) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	s.Engine().ServeHTTP(c, ctx)
	ctx.Abort()
}

func withDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}
