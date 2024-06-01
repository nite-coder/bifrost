package gateway

import (
	"context"
	"http-benchmark/pkg/log"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
)

type initMiddleware struct {
	logger *slog.Logger
}

func newInitMiddleware(logger *slog.Logger) *initMiddleware {
	return &initMiddleware{
		logger: logger,
	}
}

func (m *initMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	ctx.Set("X-Forwarded-For", ctx.Request.Header.Get("X-Forwarded-For"))

	c = log.NewContext(c, m.logger)
	ctx.Next(c)
}