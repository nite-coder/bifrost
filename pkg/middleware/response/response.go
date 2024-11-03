package response

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/blackbear/pkg/cast"
)

func init() {
	_ = middleware.RegisterMiddleware("response", func(param map[string]any) (app.HandlerFunc, error) {
		options := ResponseOptions{}
		val, found := param["status_code"]
		if found {
			statusCode, err := cast.ToInt(val)
			if err != nil {
				return nil, errors.New("status_code is invalid in response middleware")
			}
			options.StatusCode = statusCode
		}

		val, found = param["content_type"]
		if found {
			contentType, err := cast.ToString(val)
			if err != nil {
				return nil, errors.New("content_type is invalid in response middleware")
			}
			options.ContentType = contentType
		}

		val, found = param["content"]
		if found {
			content, err := cast.ToString(val)
			if err != nil {
				return nil, errors.New("content is invalid in response middleware")
			}
			options.Content = content
		}

		m := NewMiddleware(options)
		return m.ServeHTTP, nil
	})
}

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
