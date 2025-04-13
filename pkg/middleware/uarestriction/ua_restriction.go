package uarestriction

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/blackbear/pkg/cast"
)

type Options struct {
	BypassMissing            bool     `mapstructure:"bypass_missing"`
	Allow                    []string `mapstructure:"allow"`
	Deny                     []string `mapstructure:"deny"`
	RejectedHTTPStatusCode   int      `mapstructure:"rejected_http_status_code"`
	RejectedHTTPContentType  string   `mapstructure:"rejected_http_content_type"`
	RejectedHTTPResponseBody string   `mapstructure:"rejected_http_response_body"`
}

type UARestriction struct {
	options         *Options
	allowRegexpList []*regexp.Regexp
	denyRegexpList  []*regexp.Regexp
}

func NewMiddleware(options Options) (*UARestriction, error) {
	if len(options.Allow) == 0 && len(options.Deny) == 0 {
		return nil, errors.New("allow and deny can't be empty")
	} else if len(options.Allow) > 0 && len(options.Deny) > 0 {
		return nil, errors.New("allow and deny can't be set at the same time")
	}

	if options.RejectedHTTPStatusCode == 0 {
		options.RejectedHTTPStatusCode = 403
	}

	if len(options.RejectedHTTPContentType) == 0 {
		options.RejectedHTTPContentType = "application/json"
	}

	m := &UARestriction{options: &options}

	for _, allowUA := range options.Allow {
		allowRegexp, err := regexp.Compile(allowUA)
		if err != nil {
			return nil, err
		}
		m.allowRegexpList = append(m.allowRegexpList, allowRegexp)
	}

	for _, denyUA := range options.Deny {
		denyRegexp, err := regexp.Compile(denyUA)
		if err != nil {
			return nil, err
		}
		m.denyRegexpList = append(m.denyRegexpList, denyRegexp)
	}

	return m, nil
}

func (m *UARestriction) ServeHTTP(ctx context.Context, c *app.RequestContext) {

	userAgent := cast.B2S(c.UserAgent())

	if userAgent == "" && m.options.BypassMissing {
		c.Next(ctx)
		return
	}

	if len(m.allowRegexpList) > 0 {
		for _, allowRegexp := range m.allowRegexpList {
			if allowRegexp.MatchString(userAgent) {
				c.Next(ctx)
				return
			}
		}

		c.SetStatusCode(m.options.RejectedHTTPStatusCode)
		if len(m.options.RejectedHTTPContentType) > 0 {
			c.SetContentType(m.options.RejectedHTTPContentType)
		}
		if len(m.options.RejectedHTTPResponseBody) > 0 {
			c.SetBodyString(m.options.RejectedHTTPResponseBody)
		}

		c.Abort()
		return
	}

	if len(m.denyRegexpList) > 0 {
		for _, denyRegexp := range m.denyRegexpList {
			if denyRegexp.MatchString(userAgent) {
				c.SetStatusCode(m.options.RejectedHTTPStatusCode)

				if len(m.options.RejectedHTTPContentType) > 0 {
					c.SetContentType(m.options.RejectedHTTPContentType)
				}
				if len(m.options.RejectedHTTPResponseBody) > 0 {
					c.SetBodyString(m.options.RejectedHTTPResponseBody)
				}

				c.Abort()
				return
			}
		}
	}

}

func init() {
	_ = middleware.RegisterMiddleware("ua_restriction", func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("ua_restriction middleware params is empty or invalid")
		}

		option := Options{}

		decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
			WeaklyTypedInput: true,
			Result:           &option,
		})

		err := decoder.Decode(params)
		if err != nil {
			return nil, fmt.Errorf("ua_restriction middleware params is invalid: %w", err)
		}

		m, err := NewMiddleware(option)
		if err != nil {
			return nil, err
		}
		return m.ServeHTTP, nil
	})
}
