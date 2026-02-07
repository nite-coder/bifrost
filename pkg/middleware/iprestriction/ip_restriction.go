package iprestriction

import (
	"context"
	"errors"
	"net"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

type Options struct {
	RejectedHTTPContentType  string   `mapstructure:"rejected_http_content_type"`
	RejectedHTTPResponseBody string   `mapstructure:"rejected_http_response_body"`
	Allow                    []string `mapstructure:"allow"`
	Deny                     []string `mapstructure:"deny"`
	RejectedHTTPStatusCode   int      `mapstructure:"rejected_http_status_code"`
}
type IPRestriction struct {
	options *Options
}

func NewMiddleware(options Options) (*IPRestriction, error) {
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
	return &IPRestriction{options: &options}, nil
}
func (m *IPRestriction) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	clientIP := c.ClientIP()
	if len(m.options.Allow) > 0 {
		for _, allowIP := range m.options.Allow {
			_, ipNet, err := net.ParseCIDR(allowIP)
			if err != nil {
				// If not CIDR, check exact IP match
				if clientIP == allowIP {
					c.Next(ctx)
					return
				}
				continue
			}
			// Parse client IP
			ip := net.ParseIP(clientIP)
			if ip == nil {
				continue
			}
			// Check if client IP is in CIDR range
			if ipNet.Contains(ip) {
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
	if len(m.options.Deny) > 0 {
		for _, denyIP := range m.options.Deny {
			_, ipNet, err := net.ParseCIDR(denyIP)
			if err != nil {
				// If not CIDR, check exact IP match
				if clientIP == denyIP {
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
				continue
			}
			// Parse client IP
			ip := net.ParseIP(clientIP)
			if ip == nil {
				continue
			}
			// Check if client IP is in CIDR range
			if ipNet.Contains(ip) {
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
func Init() error {
	return middleware.RegisterTyped([]string{"ip_restriction"}, func(option Options) (app.HandlerFunc, error) {
		m, err := NewMiddleware(option)
		if err != nil {
			return nil, err
		}
		return m.ServeHTTP, nil
	})
}
