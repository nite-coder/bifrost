package addprefix

import (
	"context"

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
	newPath := append(m.prefix, ctx.Request.Path()...)
	ctx.Request.URI().SetPathBytes(newPath)
	ctx.Next(c)
}
