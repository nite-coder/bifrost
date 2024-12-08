package requesttermination

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

type RequestTerminationMiddleware struct {
	options *Options
}

type Options struct {
	StatusCode  int
	ContentType string
	Body        string
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
	_ = middleware.RegisterMiddleware("request-termination", func(params map[string]any) (app.HandlerFunc, error) {
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

		if opts.StatusCode == 0 {
			return nil, errors.New("request-termination: status_code can't be empty")
		}

		m := NewMiddleware(*opts)

		return m.ServeHTTP, nil
	})
}
