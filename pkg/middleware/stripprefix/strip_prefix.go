package stripprefix

import (
	"bytes"
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

type Config struct {
	Prefixes []string `mapstructure:"prefixes"`
}

func Init() error {
	return middleware.RegisterTyped([]string{"strip_prefix"}, func(cfg Config) (app.HandlerFunc, error) {
		if len(cfg.Prefixes) == 0 {
			return nil, errors.New("prefixes parameter is missing or invalid")
		}

		m := NewMiddleware(cfg.Prefixes)
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
