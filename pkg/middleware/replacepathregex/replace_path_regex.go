package replacepathregex

import (
	"context"
	"errors"
	"regexp"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func init() {
	_ = middleware.RegisterMiddleware("replace_path_regex", func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("replace_path_regex middleware params is not set or invalid")
		}

		if val, ok := params.(map[string]interface{}); ok {
			regexVal, found := val["regex"]
			if !found {
				return nil, errors.New("regex field is not set")
			}

			regex, ok := regexVal.(string)
			if !ok {
				return nil, errors.New("regex field  is invalid")
			}

			replacementVal, found := val["replacement"]
			if !found {
				return nil, errors.New("replacement field is not set")
			}

			replacement, ok := replacementVal.(string)
			if !ok {
				return nil, errors.New("replacement field is invalid")
			}

			m := NewMiddleware(regex, replacement)
			return m.ServeHTTP, nil
		}

		return nil, errors.New("replace_path_regex is not set or regex is invalid")
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
