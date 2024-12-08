package setvars

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func init() {
	_ = middleware.RegisterMiddleware("setvars", func(param map[string]any) (app.HandlerFunc, error) {
		if len(param) == 0 {
			return nil, errors.New("setvars middleware params is empty or invalid")
		}

		m := NewMiddleware(param)
		return m.ServeHTTP, nil
	})
}

type SetVarsMiddlware struct {
	vars map[string]any
}

func NewMiddleware(vars map[string]any) *SetVarsMiddlware {
	return &SetVarsMiddlware{
		vars: vars,
	}
}

func (m *SetVarsMiddlware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	for k, v := range m.vars {
		c.Set(k, v)
	}
}
