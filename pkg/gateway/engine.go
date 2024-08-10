package gateway

import (
	"context"
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/log"

	"github.com/cloudwego/hertz/pkg/app"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/hertz-contrib/obs-opentelemetry/provider"
	"github.com/hertz-contrib/obs-opentelemetry/tracing"
)

type Engine struct {
	ID              string
	opts            config.Options
	handlers        app.HandlersChain
	middlewares     app.HandlersChain
	notFoundHandler app.HandlerFunc

	options []hzconfig.Option
}

func newEngine(bifrost *Bifrost, serverOpts config.ServerOptions) (*Engine, error) {

	// middlewares
	middlewares, err := loadMiddlewares(bifrost.options.Middlewares)
	if err != nil {
		return nil, err
	}

	// services
	services, err := loadServices(bifrost, middlewares)
	if err != nil {
		return nil, err
	}

	// routes
	route, err := loadRoutes(bifrost, serverOpts, services, middlewares)
	if err != nil {
		return nil, err
	}

	// engine
	engine := &Engine{
		opts:            *bifrost.options,
		handlers:        make([]app.HandlerFunc, 0),
		notFoundHandler: nil,
		options:         make([]hzconfig.Option, 0),
	}

	// tracing
	if bifrost.options.Tracing.Enabled {

		provider.NewOpenTelemetryProvider(
			provider.WithEnableMetrics(false),
			provider.WithServiceName("bifrost"),
		)

		tracer, cfg := tracing.NewServerTracer()
		engine.options = append(engine.options, tracer)
		tracingServerMiddleware := tracing.ServerMiddleware(cfg)
		engine.Use(tracingServerMiddleware)
	}

	// init middlewares
	logger, err := log.NewLogger(serverOpts.Logging)
	if err != nil {
		return nil, err
	}
	initMiddleware := newInitMiddleware(serverOpts.ID, logger)
	engine.Use(initMiddleware.ServeHTTP)

	// set server's middlewares
	for _, middleware := range serverOpts.Middlewares {

		if len(middleware.Use) > 0 {
			val, found := middlewares[middleware.Use]
			if !found {
				return nil, fmt.Errorf("middleware '%s' was not found in server id: '%s'", middleware.Use, serverOpts.ID)
			}

			engine.Use(val)
			continue
		}

		if len(middleware.Type) == 0 {
			return nil, fmt.Errorf("middleware type can't be empty in server id: '%s'", serverOpts.ID)
		}

		handler, found := middlewareFactory[middleware.Type]
		if !found {
			return nil, fmt.Errorf("middleware type '%s' was not found in server id: '%s'", middleware.Type, serverOpts.ID)
		}

		m, err := handler(middleware.Params)
		if err != nil {
			return nil, fmt.Errorf("middleware type '%s' params is invalid in server id: '%s'. error: %w", middleware.Type, serverOpts.ID, err)
		}

		engine.Use(m)
	}

	engine.Use(route.ServeHTTP)

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
