package iprestriction

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestIPRestriction(t *testing.T) {
	tests := []struct {
		name           string
		options        Options
		clientIP       string
		wantStatus     int
		wantAborted    bool
		wantContinue   bool
		wantContentLen int
	}{
		{
			name: "allow single IP - matched",
			options: Options{
				Allow: []string{"192.168.1.1"},
			},
			clientIP:     "192.168.1.1",
			wantStatus:   200,
			wantAborted:  false,
			wantContinue: true,
		},
		{
			name: "allow single IP - not matched",
			options: Options{
				Allow:                    []string{"192.168.1.1"},
				RejectedHTTPStatusCode:   403,
				RejectedHTTPContentType:  "application/json",
				RejectedHTTPResponseBody: "forbidden",
			},
			clientIP:       "192.168.1.2",
			wantStatus:     403,
			wantAborted:    true,
			wantContinue:   false,
			wantContentLen: len("forbidden"),
		},
		{
			name: "allow CIDR - matched",
			options: Options{
				Allow: []string{"192.168.1.0/24"},
			},
			clientIP:     "192.168.1.100",
			wantStatus:   200,
			wantAborted:  false,
			wantContinue: true,
		},
		{
			name: "allow CIDR - not matched",
			options: Options{
				Allow:                    []string{"192.168.1.0/24"},
				RejectedHTTPStatusCode:   403,
				RejectedHTTPContentType:  "application/json",
				RejectedHTTPResponseBody: "forbidden",
			},
			clientIP:       "192.168.2.1",
			wantStatus:     403,
			wantAborted:    true,
			wantContinue:   false,
			wantContentLen: len("forbidden"),
		},
		{
			name: "deny single IP - matched",
			options: Options{
				Deny:                     []string{"192.168.1.1"},
				RejectedHTTPStatusCode:   403,
				RejectedHTTPContentType:  "application/json",
				RejectedHTTPResponseBody: "forbidden",
			},
			clientIP:       "192.168.1.1",
			wantStatus:     403,
			wantAborted:    true,
			wantContinue:   false,
			wantContentLen: len("forbidden"),
		},
		{
			name: "deny single IP - not matched",
			options: Options{
				Deny: []string{"192.168.1.1"},
			},
			clientIP:     "192.168.1.2",
			wantStatus:   200,
			wantAborted:  false,
			wantContinue: true,
		},
		{
			name: "deny CIDR - matched",
			options: Options{
				Deny:                     []string{"192.168.1.0/24"},
				RejectedHTTPStatusCode:   403,
				RejectedHTTPContentType:  "application/json",
				RejectedHTTPResponseBody: "forbidden",
			},
			clientIP:       "192.168.1.100",
			wantStatus:     403,
			wantAborted:    true,
			wantContinue:   false,
			wantContentLen: len("forbidden"),
		},
		{
			name: "deny CIDR - not matched",
			options: Options{
				Deny: []string{"192.168.1.0/24"},
			},
			clientIP:     "192.168.2.1",
			wantStatus:   200,
			wantAborted:  false,
			wantContinue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware, err := NewMiddleware(tt.options)
			assert.NoError(t, err)

			ctx := context.Background()
			hzctx := app.RequestContext{
				Request:  protocol.Request{},
				Response: protocol.Response{},
			}

			hzctx.Request.Header.Set("X-Forwarded-For", tt.clientIP)

			var continueExec bool

			hzctx.SetIndex(-1)
			hzctx.SetHandlers([]app.HandlerFunc{middleware.ServeHTTP, func(ctx context.Context, c *app.RequestContext) {
				continueExec = true
				c.Response.SetStatusCode(200)
			}})
			hzctx.Next(ctx)

			assert.Equal(t, tt.wantStatus, hzctx.Response.StatusCode())
			assert.Equal(t, tt.wantAborted, hzctx.IsAborted())
			assert.Equal(t, tt.wantContinue, continueExec)
			if tt.wantContentLen > 0 {
				assert.Equal(t, tt.wantContentLen, len(hzctx.Response.Body()))
			}
		})
	}
}

func TestNewMiddleware(t *testing.T) {
	_ = Init()
	h := middleware.Factory("ip_restriction")

	params := map[string]any{
		"deny":                        []string{"192.16.8.0/24", "192.168.1.1"},
		"rejected_http_status_code":   403,
		"rejected_http_content_type":  "application/json",
		"rejected_http_response_body": "forbidden",
	}

	_, err := h(params)
	assert.NoError(t, err)
}
