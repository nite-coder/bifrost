package setvars

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
)

func init() {
	_ = middleware.RegisterMiddleware("setvars", func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("setvars middleware params is empty or invalid")
		}

		options := []*Options{}

		paramsSlice, ok := params.([]interface{})
		if !ok {
			return nil, errors.New("setvars middleware params is invalid")
		}

		err := mapstructure.Decode(paramsSlice, &options)
		if err != nil {
			return nil, fmt.Errorf("setvars middleware params is invalid: %w", err)
		}

		for _, opt := range options {
			if opt.Key == "" {
				return nil, errors.New("the key can't be empty in setvars middleware params")
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
