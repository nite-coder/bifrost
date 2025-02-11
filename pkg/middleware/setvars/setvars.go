package setvars

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
	_ = middleware.RegisterMiddleware("setvars", func(param map[string]any) (app.HandlerFunc, error) {
		if len(param) == 0 {
			return nil, errors.New("setvars middleware params is empty or invalid")
		}

		m := NewMiddleware(param)
		return m.ServeHTTP, nil
	})
}

type varInfo struct {
	key        string
	value      any
	valueStr   string
	directives []string
}

type SetVarsMiddlware struct {
	varinfos []*varInfo
}

func NewMiddleware(vars map[string]any) *SetVarsMiddlware {
	varinfos := []*varInfo{}

	for k, v := range vars {
		if k == "" {
			continue
		}

		val, _ := cast.ToString(v)

		varinfos = append(varinfos, &varInfo{
			key:        k,
			value:      v,
			valueStr:   val,
			directives: variable.ParseDirectives(val),
		})
	}

	return &SetVarsMiddlware{
		varinfos: varinfos,
	}
}

func (m *SetVarsMiddlware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	for _, varinfo := range m.varinfos {
		if len(varinfo.directives) > 0 {
			replacements := make([]string, 0, len(varinfo.directives)*2)

			for _, key := range varinfo.directives {
				val := variable.GetString(key, c)
				replacements = append(replacements, key, val)
			}

			replacer := strings.NewReplacer(replacements...)
			result := replacer.Replace(varinfo.valueStr)
			c.Set(varinfo.key, result)
		} else {
			c.Set(varinfo.key, varinfo.value)
		}
	}
}
