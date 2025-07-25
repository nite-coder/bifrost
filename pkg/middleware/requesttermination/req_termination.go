package requesttermination

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
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
	_ = middleware.Register([]string{"request_termination"}, func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("requesttermination middleware params is empty or invalid")
		}

		opts := &Options{}

		err := mapstructure.Decode(params, &opts)
		if err != nil {
			return nil, fmt.Errorf("requesttermination middleware params is invalid: %w", err)
		}

		if opts.StatusCode == 0 {
			return nil, errors.New("requesttermination: status_code can't be empty")
		}

		m := NewMiddleware(*opts)
		return m.ServeHTTP, nil
	})
}
