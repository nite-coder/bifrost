package gateway

import (
	"context"
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/log"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
)

type initMiddleware struct {
	logger  *slog.Logger
	entryID string
}

func newInitMiddleware(entryID string, logger *slog.Logger) *initMiddleware {
	return &initMiddleware{
		logger:  logger,
		entryID: entryID,
	}
}

func (m *initMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	if len(ctx.Request.Header.Get("X-Forwarded-For")) > 0 {
		ctx.Set("X-Forwarded-For", ctx.Request.Header.Get("X-Forwarded-For"))
	}

	ctx.Set(domain.ENTRY_ID, m.entryID)

	c = log.NewContext(c, m.logger)
	ctx.Next(c)
}
