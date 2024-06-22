package gateway

import (
	"context"
	"fmt"
	bifrostConfig "http-benchmark/pkg/config"
	"http-benchmark/pkg/log"
	"slices"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/hertz-contrib/obs-opentelemetry/provider"
	"github.com/hertz-contrib/obs-opentelemetry/tracing"
)

type Engine struct {
	ID              string
	opts            bifrostConfig.Options
	handlers        app.HandlersChain
	middlewares     app.HandlersChain
	notFoundHandler app.HandlerFunc

	options []config.Option
}

func newEngine(bifrost *Bifrost, entry bifrostConfig.EntryOptions) (*Engine, error) {

	// middlewares
	middlewares, err := loadMiddlewares(bifrost.opts.Middlewares)
	if err != nil {
		return nil, err
	}

	// services
	services, err := loadServices(bifrost, middlewares)
	if err != nil {
		return nil, err
	}

	// routes
	isHostEnabled := false
	for _, routeOpts := range bifrost.opts.Routes {
		if len(routeOpts.Hosts) > 0 {
			isHostEnabled = true
			break
		}
	}
	router := newRouter(isHostEnabled)

	for routeID, routeOpts := range bifrost.opts.Routes {

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
		opts:            *bifrost.opts,
		handlers:        make([]app.HandlerFunc, 0),
		notFoundHandler: nil,
		options:         make([]config.Option, 0),
	}

	// tracing
	if bifrost.opts.Observability.Tracing.Enabled {

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
