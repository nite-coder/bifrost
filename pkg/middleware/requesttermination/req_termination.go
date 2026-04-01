package requesttermination

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/middleware"
)

// Middleware is a middleware that terminates the request with a specified status code and body.
type Middleware struct {
	options *Options
}

// Options defines the configuration for the request termination middleware.
type Options struct {
	ContentType string `mapstructure:"content_type"`
	Body        string `mapstructure:"body"`
	StatusCode  int    `mapstructure:"status_code"`
}

// NewMiddleware creates a new RequestTerminationMiddleware instance.
func NewMiddleware(options Options) *Middleware {
	return &Middleware{
		options: &options,
	}
}

func (m *Middleware) ServeHTTP(_ context.Context, c *app.RequestContext) {
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

// Init registers the request_termination middleware.
func Init() error {
	return middleware.RegisterTyped([]string{"request_termination"}, func(opts Options) (app.HandlerFunc, error) {
		if opts.StatusCode == 0 {
			return nil, errors.New("request_termination: status_code cannot be empty")
		}

		m := NewMiddleware(opts)
		return m.ServeHTTP, nil
	})
}
