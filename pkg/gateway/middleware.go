package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/nite-coder/blackbear/pkg/cast"
)

var (
	abortMiiddleware = &AbortMiddleware{}
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
			var err error
			switch v := r.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("%v", v)
			}

			stackTrace := cast.B2S(debug.Stack())
			fullURI := fullURI(&c.Request)
			routeID := variable.GetString(variable.RouteID, c)

			logger.Error("http request recovered",
				slog.String("route_id", routeID),
				slog.String("client_ip", c.ClientIP()),
				slog.String("full_uri", fullURI),
				slog.String("error", err.Error()),
				slog.String("stack", stackTrace),
			)
			c.SetStatusCode(500)
			c.Abort()
		}
	}()

	// create request info
	host := make([]byte, len(c.Request.Host()))
	copy(host, c.Request.Host())

	path := make([]byte, len(c.Request.Path()))
	copy(path, c.Request.Path())

	schema := make([]byte, len(c.Request.Scheme()))
	copy(schema, c.Request.Scheme())

	querystring := make([]byte, len(c.Request.QueryString()))
	if len(querystring) > 0 {
		copy(querystring, c.Request.QueryString())
	}

	reqOrig := &variable.RequestOriginal{
		ServerID: m.serverID,
		Scheme:   schema,
		Host:     host,
		Path:     path,
		Protocol: c.Request.Header.GetProtocol(),
		Method:   variable.MethodToString(c.Request.Method()),
		Query:    querystring,
	}

	c.Set(variable.RequestOrig, reqOrig)

	c.Next(ctx)
}

type FirstRouteMiddleware struct {
	options *variable.RequestRoute
}

func newFirstRouteMiddleware(options *variable.RequestRoute) *FirstRouteMiddleware {
	return &FirstRouteMiddleware{
		options: options,
	}
}

func (m *FirstRouteMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	c.Set(variable.BifrostRoute, m.options)
}

func loadMiddlewares(middlewareOptions map[string]config.MiddlwareOptions) (map[string]app.HandlerFunc, error) {

	middlewares := map[string]app.HandlerFunc{}
	for id, middlewareOpts := range middlewareOptions {

		if len(id) == 0 {
			return nil, errors.New("middleware ID cannot be empty")
		}

		middlewareOpts.ID = id

		if len(middlewareOpts.Type) == 0 {
			return nil, fmt.Errorf("middleware type cannot be empty for middleware ID: %s", middlewareOpts.ID)
		}

		handler := middleware.Factory(middlewareOpts.Type)

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

type AbortMiddleware struct{}

func (m *AbortMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	c.Abort()
}
