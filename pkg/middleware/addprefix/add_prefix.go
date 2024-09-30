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
	c.Request.URI().SetPathBytes(append(m.prefix, c.Request.Path()...))
	c.Next(ctx)
}
