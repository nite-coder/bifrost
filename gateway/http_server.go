package gateway

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	hertzslog "github.com/hertz-contrib/logger/slog"
)

type HTTPServer struct {
	ID       string
	switcher *Switcher
	server   *server.Hertz
}

func NewHTTPServer(entry EntryOptions, opts Options) (*HTTPServer, error) {

	//gopool.SetCap(20000)

	engine, err := NewEngine(entry.ID, opts)
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

func NewEngine(id string, opts Options) (*Engine, error) {

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

		if len(routeOpts.Entries) > 0 && !slices.Contains(routeOpts.Entries, id) {
			continue
		}

		upstream, ok := upstreams[routeOpts.Upstream]
		if !ok {
			return nil, fmt.Errorf("upstream '%s' was not found in '%s' route", routeOpts.Upstream, routeOpts.Match)
		}

		err := router.AddRoute(routeOpts, upstream.ServeHTTP)
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
