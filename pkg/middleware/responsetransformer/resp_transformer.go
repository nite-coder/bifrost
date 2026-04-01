package responsetransformer

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
)

// ResponseTransFormaterMiddleware is a middleware that transforms the response by adding, setting, or removing headers.
type ResponseTransFormaterMiddleware struct {
	options *Options
}

// RemoveOptions defines the headers to be removed from the response.
type RemoveOptions struct {
	Headers []string
}

// AddOptions defines the headers to be added to the response if they are not already present.
type AddOptions struct {
	Headers map[string]string
}

// SetOptions defines the headers to be set on the response, overwriting any existing values.
type SetOptions struct {
	Headers map[string]string
}

// Options defines the total configuration for the response transformer middleware.
type Options struct {
	Add    AddOptions
	Set    SetOptions
	Remove RemoveOptions
}

// NewMiddleware creates a new ResponseTransFormaterMiddleware instance.
func NewMiddleware(opts Options) *ResponseTransFormaterMiddleware {
	return &ResponseTransFormaterMiddleware{
		options: &opts,
	}
}

func (m *ResponseTransFormaterMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	c.Next(ctx)
	if len(m.options.Remove.Headers) > 0 {
		for _, header := range m.options.Remove.Headers {
			if header == "" {
				continue
			}
			c.Response.Header.Del(header)
		}
	}
	if len(m.options.Add.Headers) > 0 {
		for k, v := range m.options.Add.Headers {
			if k == "" {
				continue
			}
			if c.Response.Header.Get(k) == "" {
				if variable.IsDirective(v) {
					v = variable.GetString(v, c)
				}
				c.Response.Header.Set(k, v)
			}
		}
	}
	if len(m.options.Set.Headers) > 0 {
		for k, v := range m.options.Set.Headers {
			if k == "" {
				continue
			}
			if variable.IsDirective(v) {
				v = variable.GetString(v, c)
			}
			c.Response.Header.Set(k, v)
		}
	}
}

// Init registers the response_transformer middleware.
func Init() error {
	return middleware.RegisterTyped([]string{"response_transformer"}, func(opts Options) (app.HandlerFunc, error) {
		m := NewMiddleware(opts)
		return m.ServeHTTP, nil
	})
}
