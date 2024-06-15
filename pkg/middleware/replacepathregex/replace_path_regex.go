package replacepathregex

import (
	"context"
	"http-benchmark/pkg/config"
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
	originalPath := string(ctx.Request.Path())
	_, found := ctx.Get(config.REQUEST_PATH)
	if !found {
		ctx.Set(config.REQUEST_PATH, originalPath)
	}

	newPath := m.regex.ReplaceAll(ctx.Request.Path(), m.replacement)

	ctx.Request.Header.Set("X-Replaced-Path", originalPath)

	ctx.Request.URI().SetPathBytes(newPath)

	ctx.Next(c)
}
