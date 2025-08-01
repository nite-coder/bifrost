package requesttransformer

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
)

type RequestTransFormaterMiddleware struct {
	options *Options
}
type RemoveOptions struct {
	Headers     []string
	Querystring []string
}
type AddOptions struct {
	Headers     map[string]string
	Querystring map[string]string
}
type SetOptions struct {
	Headers     map[string]string
	Querystring map[string]string
}
type Options struct {
	Add    AddOptions
	Set    SetOptions
	Remove RemoveOptions
}

func NewMiddleware(opts Options) *RequestTransFormaterMiddleware {
	return &RequestTransFormaterMiddleware{
		options: &opts,
	}
}
func (m *RequestTransFormaterMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
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
			c.Request.SetQueryString(k + "=" + v)
		}
	}
}
func init() {
	_ = middleware.Register([]string{"request_transformer"}, func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("request_transformer middleware params is empty or invalid")
		}
		opts := &Options{}
		err := mapstructure.Decode(params, &opts)
		if err != nil {
			return nil, fmt.Errorf("request_transformer middleware params is invalid: %w", err)
		}
		m := NewMiddleware(*opts)
		return m.ServeHTTP, nil
	})
}
