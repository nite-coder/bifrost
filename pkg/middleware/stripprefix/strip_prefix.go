package stripprefix

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func init() {
	_ = middleware.RegisterMiddleware("strip_prefix", func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("strip_prefix middleware params is empty or invalid")
		}

		prefixes := make([]string, 0)
		if val, ok := params.(map[string]any); ok {
			prefixVal, found := val["prefixes"]

			if !found {
				return nil, errors.New("prefixes params is not set or prefixes is invalid")
			}

			err := mapstructure.Decode(prefixVal, &prefixes)
			if err != nil {
				return nil, fmt.Errorf("prefixes params is invalid, error: %w", err)
			}
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
