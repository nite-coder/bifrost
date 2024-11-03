package stripprefix

import (
	"bytes"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func init() {
	_ = middleware.RegisterMiddleware("strip_prefix", func(params map[string]any) (app.HandlerFunc, error) {
		val := params["prefixes"].([]any)

		prefixes := make([]string, 0)
		for _, v := range val {
			prefixes = append(prefixes, v.(string))
		}

		m := NewMiddleware(prefixes)
		return m.ServeHTTP, nil
	})
}

type StripPrefixMiddleware struct {
	prefixes [][]byte
}

func NewMiddleware(prefixs []string) *StripPrefixMiddleware {
	m := &StripPrefixMiddleware{
		prefixes: make([][]byte, 0),
	}
	for _, prefix := range prefixs {
		m.prefixes = append(m.prefixes, []byte(prefix))
	}

	return m
}

func (m *StripPrefixMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	for _, prefix := range m.prefixes {
		if bytes.HasPrefix(c.Request.Path(), prefix) {
			newPath := bytes.TrimPrefix(c.Request.Path(), prefix)
			c.Request.URI().SetPathBytes(newPath)
			break
		}
	}

	c.Next(ctx)
}
