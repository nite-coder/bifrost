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

type Config struct {
	Prefix string `mapstructure:"prefix"`
}

func Init() error {
	return middleware.RegisterTyped([]string{"add_prefix"}, func(cfg Config) (app.HandlerFunc, error) {
		if cfg.Prefix == "" {
			return nil, errors.New("prefix is required and must be a string")
		}

		m := NewMiddleware(cfg.Prefix)
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
