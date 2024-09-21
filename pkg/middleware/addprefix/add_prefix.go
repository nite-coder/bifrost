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

func (m *AddPrefixMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	newPath := append(m.prefix, c.Request.Path()...)
	c.Request.URI().SetPathBytes(newPath)
	c.Next(ctx)
}
