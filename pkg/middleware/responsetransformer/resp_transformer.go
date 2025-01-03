package responsetransformer

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
)

type ResponseTransFormaterMiddleware struct {
	options *Options
}

type RemoveOptions struct {
	Headers []string
}

type AddOptions struct {
	Headers map[string]string
}

type Options struct {
	Remove RemoveOptions
	Add    AddOptions
}

func NewMiddleware(opts Options) *ResponseTransFormaterMiddleware {
	return &ResponseTransFormaterMiddleware{
		options: &opts,
	}
}

func (m *ResponseTransFormaterMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {

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

			if variable.IsDirective(v) {
				v = variable.GetString(v, c)
			}
			c.Response.Header.Set(k, v)
		}
	}

}

func init() {
	_ = middleware.RegisterMiddleware("response-transformer", func(params map[string]any) (app.HandlerFunc, error) {
		opts := &Options{}

		config := &mapstructure.DecoderConfig{
			Metadata: nil,
			Result:   opts,
			TagName:  "mapstructure",
		}

		decoder, err := mapstructure.NewDecoder(config)
		if err != nil {
			return nil, err
		}

		if err := decoder.Decode(params); err != nil {
			return nil, err
		}

		m := NewMiddleware(*opts)

		return m.ServeHTTP, nil
	})
}
