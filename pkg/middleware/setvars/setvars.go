package setvars

import (
	"context"
	"errors"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
)

func Init() error {
	return middleware.RegisterTyped([]string{"setvars"}, func(options []*Options) (app.HandlerFunc, error) {
		if len(options) == 0 {
			return nil, errors.New("setvars middleware params is invalid")
		}

		for _, opt := range options {
			if opt.Key == "" {
				return nil, errors.New("key cannot be empty in setvars middleware parameters")
			}
		}

		m := NewMiddleware(options)
		return m.ServeHTTP, nil
	})
}

type Options struct {
	Key     string
	Value   string
	Default string

	directives []string
}

type SetVarsMiddlware struct {
	options []*Options
}

func NewMiddleware(options []*Options) *SetVarsMiddlware {

	for _, v := range options {
		if v.Key == "" {
			continue
		}

		v.directives = variable.ParseDirectives(v.Value)
	}

	return &SetVarsMiddlware{
		options: options,
	}
}

func (m *SetVarsMiddlware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	for _, varinfo := range m.options {
		if len(varinfo.directives) > 0 {
			replacements := make([]string, 0, len(varinfo.directives)*2)

			for _, key := range varinfo.directives {
				val := variable.GetString(key, c)

				if val == "" {
					val = varinfo.Default
				}

				replacements = append(replacements, key, val)
			}

			replacer := strings.NewReplacer(replacements...)
			result := replacer.Replace(varinfo.Value)
			c.Set(varinfo.Key, result)
		} else {
			c.Set(varinfo.Key, varinfo.Value)
		}
	}
}
