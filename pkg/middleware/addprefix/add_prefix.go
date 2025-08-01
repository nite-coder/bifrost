package addprefix

import (
	"context"
	"errors"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/nite-coder/blackbear/pkg/cast"
)

func init() {
	_ = middleware.Register([]string{"add_prefix"}, func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("prefix is not set or prefix is invalid")
		}
		var prefix string
		if val, ok := params.(map[string]any); ok {
			prefix, ok = val["prefix"].(string)
			if !ok {
				return nil, errors.New("prefix is not set or prefix is invalid")
			}
		}
		m := NewMiddleware(prefix)
		return m.ServeHTTP, nil
	})
}

type AddPrefixMiddleware struct {
	prefixStr  string
	directives []string
	prefix     []byte
}

func NewMiddleware(prefix string) *AddPrefixMiddleware {
	return &AddPrefixMiddleware{
		prefix:     []byte(prefix),
		prefixStr:  prefix,
		directives: variable.ParseDirectives(prefix),
	}
}
func (m *AddPrefixMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	if len(m.directives) > 0 {
		replacements := make([]string, 0, len(m.directives)*2)
		for _, key := range m.directives {
			val := variable.GetString(key, c)
			replacements = append(replacements, key, val)
		}
		replacer := strings.NewReplacer(replacements...)
		result := replacer.Replace(m.prefixStr)
		c.Request.URI().SetPathBytes(append(cast.S2B(result), c.Request.Path()...))
	} else {
		c.Request.URI().SetPathBytes(append(m.prefix, c.Request.Path()...))
	}
	c.Next(ctx)
}
