package replacepath

import (
	"context"
	"http-benchmark/pkg/domain"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

type ReplacePathMiddleware struct {
	newPath []byte
}

func NewMiddleware(newPath string) *ReplacePathMiddleware {

	if !strings.HasPrefix(newPath, "/") {
		newPath = "/" + newPath
	}

	return &ReplacePathMiddleware{
		newPath: []byte(newPath),
	}
}

func (m *ReplacePathMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	path := string(ctx.Request.Path())
	_, found := ctx.Get(domain.REQUEST_PATH)
	if !found {
		ctx.Set(domain.REQUEST_PATH, path)
	}

	ctx.Request.Header.Set("X-Replaced-Path", path)
	ctx.Request.URI().SetPathBytes(m.newPath)

	ctx.Next(c)
}
