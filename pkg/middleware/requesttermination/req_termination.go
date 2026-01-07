package requesttermination

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

type RequestTerminationMiddleware struct {
	options *Options
}

type Options struct {
	ContentType string `mapstructure:"content_type"`
	Body        string `mapstructure:"body"`
	StatusCode  int    `mapstructure:"status_code"`
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

func init() {
	_ = middleware.RegisterTyped([]string{"request_termination"}, func(opts Options) (app.HandlerFunc, error) {
		if opts.StatusCode == 0 {
			return nil, errors.New("request_termination: status_code cannot be empty")
		}

		m := NewMiddleware(opts)
		return m.ServeHTTP, nil
	})
}
