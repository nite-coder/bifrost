package gateway

import (
	"context"
	"fmt"
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/tracer/accesslog"
	"io"
	"log/slog"
	"net"
	"slices"
	"sync/atomic"
	"syscall"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/hertz-contrib/pprof"
	"golang.org/x/sys/unix"

	hertzslog "github.com/hertz-contrib/logger/slog"
)

type HTTPServer struct {
	entryOpts       domain.EntryOptions
	switcher        *switcher
	server          *server.Hertz
	accesslogTracer *accesslog.LoggerTracer
}

func NewHTTPServer(bifrost *Bifrost, entry domain.EntryOptions, opts domain.Options, tracers []tracer.Tracer) (*HTTPServer, error) {

	//gopool.SetCap(20000)

	engine, err := NewEngine(bifrost, entry, opts)
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
		server.WithHostPorts(entry.Bind),
		server.WithIdleTimeout(entry.IdleTimeout),
		server.WithReadTimeout(entry.ReadTimeout),
		server.WithWriteTimeout(entry.WriteTimeout),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithSenseClientDisconnection(true),
		withDefaultServerHeader(true),
	}

	for _, tracer := range tracers {
		hzOpts = append(hzOpts, server.WithTracer(tracer))
	}

	if entry.ReusePort {
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

	httpServer := &HTTPServer{
		entryOpts: entry,
	}

	var accessLogTracer *accesslog.LoggerTracer
	if entry.AccessLog.Enabled {
		accessLogTracer, err = accesslog.NewTracer(entry.AccessLog)
		if err != nil {
			return nil, err
		}
		hzOpts = append(hzOpts, server.WithTracer(accessLogTracer))

		httpServer.accesslogTracer = accessLogTracer
	}

	h := server.Default(hzOpts...)
	h.OnShutdown = append(h.OnShutdown, func(ctx context.Context) {
		if accessLogTracer != nil {
			accessLogTracer.Shutdown()
		}

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

	// transports
	transports := map[string]*domain.TransportOptions{}
	for id, transportOpts := range opts.Transports {

		if len(id) == 0 {
			return nil, fmt.Errorf("transport id can't be empty")
		}

		transportOpts.ID = id

		_, found := transports[transportOpts.ID]
		if found {
			return nil, fmt.Errorf("transport '%s' already exists", transportOpts.ID)
		}

		transports[transportOpts.ID] = &transportOpts
	}

	// upstreams
	upstreams := map[string]*Upstream{}
	for id, upstreamOpts := range opts.Upstreams {

		if len(id) == 0 {
			return nil, fmt.Errorf("upstream id can't be empty")
		}

		upstreamOpts.ID = id

		_, found := upstreams[upstreamOpts.ID]
		if found {
			return nil, fmt.Errorf("upstream '%s' already exists", upstreamOpts.ID)
		}

		var transportOpts *domain.TransportOptions
		if len(upstreamOpts.ClientTransport) > 0 {

			transportOpts, found = transports[upstreamOpts.ClientTransport]
			if !found {
				return nil, fmt.Errorf("transport '%s' was not found in '%s' upstream", upstreamOpts.ClientTransport, upstreamOpts.ID)
			}
		}

		upstream, err := NewUpstream(bifrost, upstreamOpts, transportOpts)
		if err != nil {
			return nil, err
		}
		upstreams[upstreamOpts.ID] = upstream
	}

	// routes
	router := NewRouter()

	for routeID, routeOpts := range opts.Routes {

		routeOpts.ID = routeID

		if len(routeOpts.Entries) > 0 && !slices.Contains(routeOpts.Entries, entry.ID) {
			continue
		}

		if len(routeOpts.Match) == 0 {
			return nil, fmt.Errorf("route match can't be empty")
		}

		if len(routeOpts.Upstream) == 0 {
			return nil, fmt.Errorf("route upstream can't be empty")
		}

		var upstreamFunc app.HandlerFunc

		if routeOpts.Upstream[0] == '$' {
			dynamicUpstream := &DynamicUpstream{
				upstreams: upstreams,
				name:      routeOpts.Upstream,
			}

			upstreamFunc = dynamicUpstream.ServeHTTP
		} else {
			upstream, ok := upstreams[routeOpts.Upstream]
			if !ok {
				return nil, fmt.Errorf("upstream '%s' was not found in '%s' route", routeOpts.Upstream, routeOpts.ID)
			}

			upstreamFunc = upstream.ServeHTTP
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
				return nil, fmt.Errorf("middleware kind can't be empty in route: '%s'", routeOpts.Match)
			}

			handler, found := middlewareFactory[middleware.Kind]
			if !found {
				return nil, fmt.Errorf("middleware handler '%s' was not found in route: '%s'", middleware.Kind, routeOpts.Match)
			}

			m, err := handler(middleware.Params)
			if err != nil {
				return nil, fmt.Errorf("create middleware handler '%s' failed in route: '%s'", middleware.Kind, routeOpts.Match)
			}

			routeMiddlewares = append(routeMiddlewares, m)
		}

		routeMiddlewares = append(routeMiddlewares, upstreamFunc)

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
