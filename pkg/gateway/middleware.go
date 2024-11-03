package gateway

import (
	"context"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
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
	ctx.Set(config.SERVER_ID, m.serverID)

	// save original host
	host := make([]byte, len(ctx.Request.Host()))
	copy(host, ctx.Request.Host())
	ctx.Set(config.HOST, host)

	if len(ctx.Request.Header.Get("X-Forwarded-For")) > 0 {
		ctx.Set("X-Forwarded-For", ctx.Request.Header.Get("X-Forwarded-For"))
	}

	// save original path
	path := make([]byte, len(ctx.Request.Path()))
	copy(path, ctx.Request.Path())
	ctx.Set(config.REQUEST_PATH, path)

	// add trace_id to logger
	spanCtx := trace.SpanContextFromContext(c)
	if spanCtx.HasTraceID() {
		traceID := spanCtx.TraceID().String()
		ctx.Set(config.TRACE_ID, traceID)

		logger = logger.With(slog.String("trace_id", traceID))
	}
	c = log.NewContext(c, logger)

	ctx.Next(c)
}
