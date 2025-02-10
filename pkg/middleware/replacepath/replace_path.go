package replacepath

import (
	"context"
	"errors"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
)

func init() {
	_ = middleware.RegisterMiddleware("replace_path", func(params map[string]any) (app.HandlerFunc, error) {
		newPath, ok := params["path"].(string)
		if !ok {
			return nil, errors.New("path is not set or path is invalid")
		}
		m := NewMiddleware(newPath)
		return m.ServeHTTP, nil
	})
}

type ReplacePathMiddleware struct {
	directives []string
	newPath    []byte
	newPathStr string
}

func NewMiddleware(newPath string) *ReplacePathMiddleware {

	if !strings.HasPrefix(newPath, "/") {
		newPath = "/" + newPath
	}

	return &ReplacePathMiddleware{
		newPath:    []byte(newPath),
		newPathStr: newPath,
		directives: variable.ParseDirectives(newPath),
	}
}

func (m *ReplacePathMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	if len(m.directives) > 0 {
		replacements := make([]string, 0, len(m.directives)*2)

		for _, key := range m.directives {
			val := variable.GetString(key, c)
			replacements = append(replacements, key, val)
		}

		replacer := strings.NewReplacer(replacements...)
		result := replacer.Replace(m.newPathStr)

		c.Request.URI().SetPath(result)
	} else {
		c.Request.URI().SetPathBytes(m.newPath)
	}

	c.Next(ctx)
}
