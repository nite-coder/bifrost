package headers

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/blackbear/pkg/cast"
)

func init() {
	_ = middleware.RegisterMiddleware("headers", func(param map[string]any) (app.HandlerFunc, error) {
		requestHeader := map[string]string{}
		val, found := param["request_headers"]
		if found {
			headers, ok := val.(map[string]any)
			if !ok {
				return nil, errors.New("request_headers is not set or request_headers is invalid")
			}

			for k, v := range headers {
				val, err := cast.ToString(v)
				if err != nil {
					continue
				}
				requestHeader[k] = val
			}
		}

		respHeader := map[string]string{}
		val, found = param["response_headers"]
		if found {
			headers, ok := val.(map[string]any)
			if !ok {
				return nil, errors.New("response_headers is not set or response_headers is invalid")
			}

			for k, v := range headers {
				val, err := cast.ToString(v)
				if err != nil {
					continue
				}
				respHeader[k] = val
			}
		}

		m := NewMiddleware(requestHeader, respHeader)
		return m.ServeHTTP, nil
	})

}

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
