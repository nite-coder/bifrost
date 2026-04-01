package stripprefix

import (
	"bytes"
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/middleware"
)

// Config defines the configuration for the strip_prefix middleware.
type Config struct {
	Prefixes []string `mapstructure:"prefixes"`
}

// Init registers the strip_prefix middleware.
func Init() error {
	return middleware.RegisterTyped([]string{"strip_prefix"}, func(cfg Config) (app.HandlerFunc, error) {
		if len(cfg.Prefixes) == 0 {
			return nil, errors.New("prefixes parameter is missing or invalid")
		}

		m := NewMiddleware(cfg.Prefixes)
		return m.ServeHTTP, nil
	})
}

// Middleware is a middleware that removes prefixes from the request path.
type Middleware struct {
	prefixes [][]byte
}

// NewMiddleware creates a new StripPrefixMiddleware instance.
func NewMiddleware(prefixs []string) *Middleware {
	m := &Middleware{
		prefixes: make([][]byte, 0),
	}
	for _, prefix := range prefixs {
		m.prefixes = append(m.prefixes, []byte(prefix))
	}

	return m
}

func (m *Middleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	for _, prefix := range m.prefixes {
		if after, ok := bytes.CutPrefix(c.Request.Path(), prefix); ok {
			newPath := after
			c.Request.URI().SetPathBytes(newPath)
			break
		}
	}

	c.Next(ctx)
}
