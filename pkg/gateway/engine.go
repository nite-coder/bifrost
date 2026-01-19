package gateway

import (
	"context"
	"fmt"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/telemetry/metrics"

	"github.com/cloudwego/hertz/pkg/app"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
)

type Engine struct {
	ID              string
	bifrost         *Bifrost
	handlers        app.HandlersChain
	middlewares     app.HandlersChain
	notFoundHandler app.HandlerFunc
	hzOptions       []hzconfig.Option
}

func newEngine(bifrost *Bifrost, serverOptions config.ServerOptions) (*Engine, error) {

	// routes
	route, err := loadRoutes(bifrost, serverOptions, bifrost.services)
	if err != nil {
		return nil, err
	}

	// engine
	engine := &Engine{
		bifrost:         bifrost,
		handlers:        make([]app.HandlerFunc, 0),
		notFoundHandler: nil,
		hzOptions:       make([]hzconfig.Option, 0),
	}

	// init middlewares
	logger, err := log.NewLogger(serverOptions.Logging)
	if err != nil {
		return nil, err
	}
	initMiddleware := newInitMiddleware(serverOptions.ID, logger)
	engine.Use(initMiddleware.ServeHTTP)

	if bifrost.options.Tracing.Enabled && serverOptions.Observability.Tracing.IsEnabled() {
		tracingMiddleware := newTracingMiddleware(serverOptions.Observability.Tracing)
		engine.Use(tracingMiddleware.ServeHTTP)
	}

	// set server's middlewares
	for _, m := range serverOptions.Middlewares {

		if len(m.Use) > 0 {
			val, found := bifrost.middlewares[m.Use]
			if !found {
				return nil, fmt.Errorf("middleware '%s' was not found in server id: '%s'", m.Use, serverOptions.ID)
			}

			engine.Use(val)
			continue
		}

		if len(m.Type) == 0 {
			return nil, fmt.Errorf("middleware type cannot be empty for server ID: %s", serverOptions.ID)
		}

		handler := middleware.Factory(m.Type)
		if handler == nil {
			return nil, fmt.Errorf("middleware type '%s' was not found in server id: '%s'", m.Type, serverOptions.ID)
		}

		apphandler, err := handler(m.Params)
		if err != nil {
			return nil, fmt.Errorf("middleware type '%s' params is invalid in server id: '%s'. error: %w", m.Type, serverOptions.ID, err)
		}

		engine.Use(apphandler)
	}

	// set prom metric middleware
	if (bifrost.options.Metrics.Prometheus.Enabled || bifrost.options.Metrics.OTLP.Enabled) && serverOptions.ID == bifrost.options.Metrics.Prometheus.ServerID {
		m := metrics.NewMetricMiddleware(bifrost.options.Metrics.Prometheus.Path, bifrost.metricsProvider)
		engine.Use(m.ServeHTTP)
	}

	engine.Use(route.ServeHTTP)

	return engine, nil
}

func (e *Engine) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	c.SetIndex(-1)
	c.SetHandlers(e.middlewares)
	c.Next(ctx)
	c.Abort()
}

func (e *Engine) OnShutdown() {

}

func (e *Engine) Use(middleware ...app.HandlerFunc) {
	e.handlers = append(e.handlers, middleware...)
	e.middlewares = e.handlers

	if e.notFoundHandler != nil {
		e.middlewares = append(e.handlers, e.notFoundHandler) // nolint
	}
}
