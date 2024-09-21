package headers

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

type HeadersMiddleware struct {
	requestHeaders  map[string]string
	responseHeaders map[string]string
}

func NewMiddleware(requestHeaders map[string]string, responseHeaders map[string]string) *HeadersMiddleware {
	return &HeadersMiddleware{
		requestHeaders:  requestHeaders,
		responseHeaders: responseHeaders,
	}
}

func (m *HeadersMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {

	if len(m.requestHeaders) > 0 {
		for k, v := range m.requestHeaders {
			c.Request.Header.Set(k, v)
		}
	}

	c.Next(ctx)

	if len(m.responseHeaders) > 0 {
		for k, v := range m.responseHeaders {
			c.Response.Header.Set(k, v)
		}
	}
}
