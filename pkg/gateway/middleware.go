package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
	"go.opentelemetry.io/otel/trace"
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

func (m *initMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	logger := m.logger

	// save serverID for access log
	ctx.Set(variable.SERVER_ID, m.serverID)

	// save original host
	host := make([]byte, len(ctx.Request.Host()))
	copy(host, ctx.Request.Host())
	ctx.Set(variable.HOST, host)

	if len(ctx.Request.Header.Get("X-Forwarded-For")) > 0 {
		ctx.Set("X-Forwarded-For", ctx.Request.Header.Get("X-Forwarded-For"))
	}

	// save original path
	path := make([]byte, len(ctx.Request.Path()))
	copy(path, ctx.Request.Path())
	ctx.Set(variable.REQUEST_PATH, path)

	// add trace_id to logger
	spanCtx := trace.SpanContextFromContext(c)
	if spanCtx.HasTraceID() {
		traceID := spanCtx.TraceID().String()
		ctx.Set(variable.TRACE_ID, traceID)

		logger = logger.With(slog.String("trace_id", traceID))
	}
	c = log.NewContext(c, logger)

	ctx.Next(c)
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
