package gateway

import (
	"context"
	"fmt"
	"net"
	"os"
	"slices"
	"sync/atomic"
	"syscall"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	hertzslog "github.com/hertz-contrib/logger/slog"
	"golang.org/x/sys/unix"
)

type HTTPServer struct {
	ID       string
	switcher *Switcher
	server   *server.Hertz
}

func NewHTTPServer(entry EntryOptions, opts Options) (*HTTPServer, error) {

	//gopool.SetCap(20000)

	engine, err := NewEngine(entry, opts)
	if err != nil {
		return nil, err
	}

	switcher := NewSwitcher(engine)

	// hertz server
	logger := hertzslog.NewLogger(hertzslog.WithOutput(os.Stdout))
	hlog.SetLogger(logger)

	hzOpts := []config.Option{
		server.WithHostPorts(entry.Bind),
		server.WithIdleTimeout(entry.IdleTimeout),
		server.WithReadTimeout(entry.ReadTimeout),
		server.WithWriteTimeout(entry.WriteTimeout),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		WithDefaultServerHeader(true),
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

	h := server.Default(hzOpts...)
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
	opts            Options
	handlers        app.HandlersChain
	middlewares     app.HandlersChain
	notFoundHandler app.HandlerFunc
}

func NewEngine(entry EntryOptions, opts Options) (*Engine, error) {

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

	// upstreams
	upstreams := map[string]*Upstream{}
	for _, upstreamOpts := range opts.Upstreams {

		_, found := upstreams[upstreamOpts.ID]
		if found {
			return nil, fmt.Errorf("upstream '%s' already exists", upstreamOpts.ID)
		}

		upstream, err := NewUpstream(upstreamOpts, nil)
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
			if len(middleware.ID) > 0 {
				val, found := middlewares[middleware.ID]
				if !found {
					return nil, fmt.Errorf("middleware '%s' was not found in route: '%s'", middleware.ID, routeOpts.Match)
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
				return nil, err
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

	for _, middleware := range entry.Middlewares {

		if len(middleware.ID) > 0 {
			val, found := middlewares[middleware.ID]
			if !found {
				return nil, fmt.Errorf("middleware '%s' was not found in entry id: '%s'", middleware.ID, entry.ID)
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

func (e *Engine) Use(middleware ...app.HandlerFunc) {
	e.handlers = append(e.handlers, middleware...)
	e.middlewares = e.handlers

	if e.notFoundHandler != nil {
		e.middlewares = append(e.handlers, e.notFoundHandler)
	}
}

type Switcher struct {
	engine atomic.Value
}

func NewSwitcher(engine *Engine) *Switcher {
	s := &Switcher{}
	s.SetEngine(engine)
	return s
}

func (s *Switcher) Engine() *Engine {
	return s.engine.Load().(*Engine)
}

func (s *Switcher) SetEngine(engine *Engine) {
	s.engine.Store(engine)
}

func (s *Switcher) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	s.Engine().ServeHTTP(c, ctx)
	ctx.Abort()
}

func WithDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}
