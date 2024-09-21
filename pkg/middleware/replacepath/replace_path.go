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

func (m *ReplacePathMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	c.Request.Header.Set("X-Replaced-Path", string(c.Request.Path()))
	c.Request.URI().SetPathBytes(m.newPath)

	c.Next(ctx)
}
