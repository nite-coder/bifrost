package requesttermination

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

type RequestTerminationMiddleware struct {
	options *Options
}

type Options struct {
	StatusCode  int
	ContentType string
	Body        string
}

func NewMiddleware(options Options) *RequestTerminationMiddleware {
	return &RequestTerminationMiddleware{
		options: &options,
	}
}

func (m *RequestTerminationMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {

	if m.options.StatusCode > 0 {
		c.Response.SetStatusCode(m.options.StatusCode)
	}

	if m.options.ContentType != "" {
		c.Response.Header.SetContentType(m.options.ContentType)
	}

	if m.options.Body != "" {
		c.Response.SetBodyString(m.options.Body)
	}

	c.Abort()
}
