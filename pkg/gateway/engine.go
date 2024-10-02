package gateway

import (
	"context"
	"fmt"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"

	"github.com/cloudwego/hertz/pkg/app"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
)

type Engine struct {
	ID              string
	bifrost         *Bifrost
	handlers        app.HandlersChain
	middlewares     app.HandlersChain
	notFoundHandler app.HandlerFunc

	hzOptions []hzconfig.Option
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
		bifrost:         bifrost,
		handlers:        make([]app.HandlerFunc, 0),
		notFoundHandler: nil,
		hzOptions:       make([]hzconfig.Option, 0),
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
