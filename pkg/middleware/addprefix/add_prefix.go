package addprefix

import (
	"context"
	"http-benchmark/pkg/config"

	"github.com/cloudwego/hertz/pkg/app"
)

type AddPrefixMiddleware struct {
	prefix []byte
}

func NewMiddleware(prefix string) *AddPrefixMiddleware {
	return &AddPrefixMiddleware{
		prefix: []byte(prefix),
	}
}

func (m *AddPrefixMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	_, found := ctx.Get(config.REQUEST_PATH)
	if !found {
		ctx.Set(config.REQUEST_PATH, string(ctx.Request.Path()))
	}

	newPath := append(m.prefix, ctx.Request.Path()...)
	ctx.Request.URI().SetPathBytes(newPath)
	ctx.Next(c)
}
