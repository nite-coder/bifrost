package replacepath

import (
	"context"
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
	ctx.Request.Header.Set("X-Replaced-Path", string(ctx.Request.Path()))
	ctx.Request.URI().SetPathBytes(m.newPath)

	ctx.Next(c)
}
