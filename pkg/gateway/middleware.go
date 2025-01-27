package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/internal/pkg/runtime"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
)

type initMiddleware struct {
	logger   *slog.Logger
	serverID string
}

func newInitMiddleware(serverID string, logger *slog.Logger) *initMiddleware {
	return &initMiddleware{
		logger:   logger,
		serverID: serverID,
	}
}

func (m *initMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := m.logger
	ctx = log.NewContext(ctx, logger)

	defer func() {
		if r := recover(); r != nil {
			stackTrace := runtime.StackTrace()
			logger.Error("error recovered", slog.Any("unhandled error", r), slog.String("stack", stackTrace))
			c.Abort()
		}
	}()

	// create request info
	host := make([]byte, len(c.Request.Host()))
	copy(host, c.Request.Host())

	path := make([]byte, len(c.Request.Path()))
	copy(path, c.Request.Path())

	reqOrig := &variable.RequestOriginal{
		ServerID: m.serverID,
		Scheme:   c.Request.Scheme(),
		Host:     host,
		Path:     path,
		Protocol: c.Request.Header.GetProtocol(),
		Method:   c.Request.Method(),
		Query:    c.Request.QueryString(),
	}

	c.Set(variable.RequestOrig, reqOrig)

	c.Next(ctx)
}

type initRouteMiddleware struct {
	options *initRouteMiddlewareOptions
}

type initRouteMiddlewareOptions struct {
	routeID   string
	route     string
	serviceID string
}

func newInitRouteMiddleware(options *initRouteMiddlewareOptions) *initRouteMiddleware {
	return &initRouteMiddleware{
		options: options,
	}
}

func (m *initRouteMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	c.Set(variable.RouteID, m.options.routeID)
	c.Set(variable.HTTPRoute, m.options.route)
	c.Set(variable.ServiceID, m.options.serviceID)
}

func loadMiddlewares(middlewareOptions map[string]config.MiddlwareOptions) (map[string]app.HandlerFunc, error) {

	middlewares := map[string]app.HandlerFunc{}
	for id, middlewareOpts := range middlewareOptions {

		if len(id) == 0 {
			return nil, errors.New("middleware id can't be empty")
		}

		middlewareOpts.ID = id

		if len(middlewareOpts.Type) == 0 {
			return nil, fmt.Errorf("middleware type can't be empty in middleware id: '%s'", middlewareOpts.ID)
		}

		handler := middleware.FindHandlerByType(middlewareOpts.Type)

		if handler == nil {
			return nil, fmt.Errorf("middleware type '%s' was not found in middleware id: '%s'", middlewareOpts.Type, middlewareOpts.ID)
		}

		m, err := handler(middlewareOpts.Params)
		if err != nil {
			return nil, fmt.Errorf("middleware type '%s' params is invalid in middleware id: '%s'. error: %w", middlewareOpts.Type, middlewareOpts.ID, err)
		}

		middlewares[middlewareOpts.ID] = m
	}

	return middlewares, nil
}
