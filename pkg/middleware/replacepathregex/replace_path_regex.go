package replacepathregex

import (
	"context"
	"errors"
	"regexp"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func init() {
	_ = middleware.RegisterMiddleware("replace_path_regex", func(params map[string]any) (app.HandlerFunc, error) {
		regex, ok := params["regex"].(string)
		if !ok {
			return nil, errors.New("regex is not set or regex is invalid")
		}
		replacement, ok := params["replacement"].(string)
		if !ok {
			return nil, errors.New("replacement is not set or replacement is invalid")
		}
		m := NewMiddleware(regex, replacement)
		return m.ServeHTTP, nil
	})

}

type ReplacePathRegexMiddleware struct {
	regex       *regexp.Regexp
	replacement []byte
}

func NewMiddleware(regex, replacement string) *ReplacePathRegexMiddleware {

	re := regexp.MustCompile(regex)

	return &ReplacePathRegexMiddleware{
		regex:       re,
		replacement: []byte(replacement),
	}
}

func (m *ReplacePathRegexMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	newPath := m.regex.ReplaceAll(c.Request.Path(), m.replacement)
	c.Request.URI().SetPathBytes(newPath)
	c.Next(ctx)
}
