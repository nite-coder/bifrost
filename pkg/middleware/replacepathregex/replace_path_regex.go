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

func (m *ReplacePathRegexMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	newPath := m.regex.ReplaceAll(ctx.Request.Path(), m.replacement)

	ctx.Request.Header.Set("X-Replaced-Path", string(ctx.Request.Path()))

	ctx.Request.URI().SetPathBytes(newPath)

	ctx.Next(c)
}
