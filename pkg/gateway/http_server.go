package gateway

import (
	"context"
	"fmt"
	"http-benchmark/pkg/domain"
	"io"
	"log/slog"
	"net"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/pprof"
	"golang.org/x/sys/unix"

	hertzslog "github.com/hertz-contrib/logger/slog"
)

type HTTPServer struct {
	ID       string
	switcher *switcher
	server   *server.Hertz
}

func NewHTTPServer(entry domain.EntryOptions, opts domain.Options) (*HTTPServer, error) {

	//gopool.SetCap(20000)

	engine, err := NewEngine(entry, opts)
	if err != nil {
		return nil, err
	}

	switcher := newSwitcher(engine)

	// hertz server
	logger := hertzslog.NewLogger(hertzslog.WithOutput(os.Stderr))
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
		// server.WithTracer(prometheus.NewServerTracer("", "/metrics",
		// 	prometheus.WithEnableGoCollector(true),
		// 	prometheus.WithDisableServer(false),
		// )),
		withDefaultServerHeader(true),
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

	var accessLogTracer *LoggerTracer
	if entry.AccessLog.Enabled {
		accessLogTracer, err = NewLoggerTracer(entry.AccessLog)
		if err != nil {
			return nil, err
		}
		hzOpts = append(hzOpts, server.WithTracer(accessLogTracer))
	}

	h := server.Default(hzOpts...)
	h.OnShutdown = append(h.OnShutdown, func(ctx context.Context) {
		if accessLogTracer != nil {
			accessLogTracer.Shutdown()
		}

	})

	pprof.Register(h)

	h.Use(switcher.ServeHTTP)

	httpServer := &HTTPServer{
		ID:       entry.ID,
		switcher: switcher,
		server:   h,
	}

	return httpServer, nil
}

func (s *HTTPServer) Run() {
	s.server.Spin()
}

type Engine struct {
	ID              string
	opts            domain.Options
	handlers        app.HandlersChain
	middlewares     app.HandlersChain
	notFoundHandler app.HandlerFunc
}

func NewEngine(entry domain.EntryOptions, opts domain.Options) (*Engine, error) {

	// middlewares
	middlewares := map[string]app.HandlerFunc{}
	for _, middlewareOpts := range opts.Middlewares {

		if len(middlewareOpts.ID) == 0 {
			return nil, fmt.Errorf("middleware id can't be empty")
		}

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
	for _, transportOpts := range opts.Transports {

		if len(transportOpts.ID) == 0 {
			return nil, fmt.Errorf("transport id can't be empty")
		}

		_, found := transports[transportOpts.ID]
		if found {
			return nil, fmt.Errorf("transport '%s' already exists", transportOpts.ID)
		}

		transports[transportOpts.ID] = &transportOpts
	}

	// upstreams
	upstreams := map[string]*Upstream{}
	for _, upstreamOpts := range opts.Upstreams {

		if len(upstreamOpts.ID) == 0 {
			return nil, fmt.Errorf("upstream id can't be empty")
		}

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

		upstream, err := NewUpstream(upstreamOpts, transportOpts)
		if err != nil {
			return nil, err
		}
		upstreams[upstreamOpts.ID] = upstream
	}

	// routes
	router := NewRouter()

	for _, routeOpts := range opts.Routes {

		if len(routeOpts.Entries) > 0 && !slices.Contains(routeOpts.Entries, entry.ID) {
			continue
		}

		if len(routeOpts.Match) == 0 {
			return nil, fmt.Errorf("route match can't be empty")
		}

		upstream, ok := upstreams[routeOpts.Upstream]
		if !ok {
			return nil, fmt.Errorf("upstream '%s' was not found in '%s' route", routeOpts.Upstream, routeOpts.Match)
		}

		routeMiddlewares := make([]app.HandlerFunc, 0)

		for _, middleware := range routeOpts.Middlewares {
			if len(middleware.Link) > 0 {
				val, found := middlewares[middleware.Link]
				if !found {
					return nil, fmt.Errorf("middleware '%s' was not found in route: '%s'", middleware.Link, routeOpts.Match)
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

		routeMiddlewares = append(routeMiddlewares, upstream.ServeHTTP)

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
	logger, err := newLogger(entry.Logging)
	if err != nil {
		return nil, err
	}
	initMiddleware := newInitMiddleware(logger)
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

func newLogger(opts domain.LoggingOtions) (*slog.Logger, error) {
	var err error

	logOptions := &slog.HandlerOptions{}

	level := strings.TrimSpace(opts.Level)
	level = strings.ToLower(level)

	switch level {
	case "debug":
		logOptions.Level = slog.LevelDebug
	case "info":
		logOptions.Level = slog.LevelInfo
	case "warn":
		logOptions.Level = slog.LevelWarn
	case "error":
		logOptions.Level = slog.LevelError
	default:
		logOptions.Level = slog.LevelError
	}

	var writer io.Writer

	output := strings.TrimSpace(opts.Output)
	output = strings.ToLower(output)

	switch output {
	case "":
	case "stderr":
		writer = os.Stderr
	default:
		writer, err = os.OpenFile(opts.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	}

	if !opts.Enabled {
		logOptions.Level = slog.LevelError
		writer = io.Discard
	}

	var logHandler slog.Handler

	handler := strings.TrimSpace(opts.Handler)
	handler = strings.ToLower(handler)

	switch handler {
	case "text":
		logHandler = slog.NewTextHandler(writer, logOptions)
	default:
		logHandler = slog.NewJSONHandler(writer, logOptions)
	}

	logger := slog.New(logHandler)
	return logger, nil
}
