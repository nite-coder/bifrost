package response

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

type ResponseMiddleware struct {
	statusCode  int
	contentType string
	content     string
}

type ResponseOptions struct {
	StatusCode  int
	ContentType string
	Content     string
}

func NewMiddleware(options ResponseOptions) *ResponseMiddleware {
	return &ResponseMiddleware{
		statusCode:  options.StatusCode,
		contentType: options.ContentType,
		content:     options.Content,
	}
}

func (m *ResponseMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {

	if m.statusCode > 0 {
		c.Response.SetStatusCode(m.statusCode)
	}

	if m.contentType != "" {
		c.Response.Header.SetContentType(m.contentType)
	}

	if m.content != "" {
		c.Response.SetBodyString(m.content)
	}

	c.Abort()
}
