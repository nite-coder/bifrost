package replacepathregex

import (
	"context"
	"regexp"

	"github.com/cloudwego/hertz/pkg/app"
)

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
