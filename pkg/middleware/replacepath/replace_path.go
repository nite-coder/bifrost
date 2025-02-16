package replacepath

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
	_ = middleware.RegisterMiddleware("replace_path", func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("replace_path middleware params is not set or invalid")
		}

		var path string
		var err error
		if val, ok := params.(map[string]any); ok {
			pathVal, found := val["path"]
			if !found {
				return nil, errors.New("path field is not set")
			}

			path, err = cast.ToString(pathVal)
			if err != nil {
				return nil, errors.New("path field is invalid")
			}

			m := NewMiddleware(path)
			return m.ServeHTTP, nil
		}

		return nil, errors.New("replace_path middleware params is invalid")
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
