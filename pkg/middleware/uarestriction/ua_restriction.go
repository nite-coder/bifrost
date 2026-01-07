package uarestriction

import (
	"context"
	"errors"
	"regexp"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/blackbear/pkg/cast"
)

type Options struct {
	RejectedHTTPContentType  string   `mapstructure:"rejected_http_content_type"`
	RejectedHTTPResponseBody string   `mapstructure:"rejected_http_response_body"`
	Allow                    []string `mapstructure:"allow"`
	Deny                     []string `mapstructure:"deny"`
	RejectedHTTPStatusCode   int      `mapstructure:"rejected_http_status_code"`
	BypassMissing            bool     `mapstructure:"bypass_missing"`
}
type UARestriction struct {
	options         *Options
	allowRegexpList []*regexp.Regexp
	denyRegexpList  []*regexp.Regexp
}

func NewMiddleware(options Options) (*UARestriction, error) {
	if len(options.Allow) == 0 && len(options.Deny) == 0 {
		return nil, errors.New("allow and deny cannot be empty")
	} else if len(options.Allow) > 0 && len(options.Deny) > 0 {
		return nil, errors.New("allow and deny cannot be set at the same time")
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
	_ = middleware.RegisterTyped([]string{"ua_restriction"}, func(option Options) (app.HandlerFunc, error) {
		m, err := NewMiddleware(option)
		if err != nil {
			return nil, err
		}
		return m.ServeHTTP, nil
	})
}
