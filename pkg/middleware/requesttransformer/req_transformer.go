package requesttransformer

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
)

// RequestTransFormaterMiddleware is a middleware that transforms the request by adding, setting, or removing headers and query parameters.
type RequestTransFormaterMiddleware struct {
	options *Options
}

// RemoveOptions defines the headers and query parameters to be removed from the request.
type RemoveOptions struct {
	Headers     []string
	Querystring []string
}

// AddOptions defines the headers and query parameters to be added to the request if they are not already present.
type AddOptions struct {
	Headers     map[string]string
	Querystring map[string]string
}

// SetOptions defines the headers and query parameters to be set on the request, overwriting any existing values.
type SetOptions struct {
	Headers     map[string]string
	Querystring map[string]string
}

// Options defines the total configuration for the request transformer middleware.
type Options struct {
	Add    AddOptions
	Set    SetOptions
	Remove RemoveOptions
}

// NewMiddleware creates a new RequestTransFormaterMiddleware instance.
func NewMiddleware(opts Options) *RequestTransFormaterMiddleware {
	return &RequestTransFormaterMiddleware{
		options: &opts,
	}
}

func (m *RequestTransFormaterMiddleware) ServeHTTP(_ context.Context, c *app.RequestContext) {
	if len(m.options.Remove.Headers) > 0 {
		for _, header := range m.options.Remove.Headers {
			if header == "" {
				continue
			}
			if variable.IsDirective(header) {
				header = variable.GetString(header, c)
			}
			c.Request.Header.Del(header)
		}
	}
	if len(m.options.Remove.Querystring) > 0 {
		for _, qs := range m.options.Remove.Querystring {
			if qs == "" {
				continue
			}
			c.Request.URI().QueryArgs().Del(qs)
		}
	}
	if len(m.options.Add.Headers) > 0 {
		for k, v := range m.options.Add.Headers {
			if k == "" {
				continue
			}
			if c.Request.Header.Get(k) == "" {
				if variable.IsDirective(v) {
					v = variable.GetString(v, c)
				}
				c.Request.Header.Set(k, v)
			}
		}
	}
	if len(m.options.Add.Querystring) > 0 {
		for k, v := range m.options.Add.Querystring {
			if k == "" {
				continue
			}
			if c.Query(k) == "" {
				if variable.IsDirective(v) {
					v = variable.GetString(v, c)
				}
				c.Request.URI().QueryArgs().Add(k, v)
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
			c.Request.Header.Set(k, v)
		}
	}
	if len(m.options.Set.Querystring) > 0 {
		for k, v := range m.options.Set.Querystring {
			if k == "" {
				continue
			}
			if variable.IsDirective(v) {
				v = variable.GetString(v, c)
			}
			c.Request.URI().QueryArgs().Set(k, v)
		}
	}
}

// Init registers the request_transformer middleware.
func Init() error {
	return middleware.RegisterTyped([]string{"request_transformer"}, func(opts Options) (app.HandlerFunc, error) {
		m := NewMiddleware(opts)
		return m.ServeHTTP, nil
	})
}
