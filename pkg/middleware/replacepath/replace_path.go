package replacepath

import (
	"context"
	"errors"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
)

type Config struct {
	Path string `mapstructure:"path"`
}

func init() {
	_ = middleware.RegisterTyped([]string{"replace_path"}, func(cfg Config) (app.HandlerFunc, error) {
		if cfg.Path == "" {
			return nil, errors.New("path field is not set")
		}
		m := NewMiddleware(cfg.Path)
		return m.ServeHTTP, nil
	})
}

type ReplacePathMiddleware struct {
	newPathStr string
	directives []string
	newPath    []byte
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
