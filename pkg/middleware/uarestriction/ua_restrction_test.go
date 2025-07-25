package uarestriction

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestUARestriction(t *testing.T) {
	tests := []struct {
		name            string
		options         Options
		userAgent       string
		wantStatusCode  int
		wantContentType string
		wantBody        string
		wantNext        bool
	}{
		{
			name: "allow empty UA with bypass",
			options: Options{
				BypassMissing: true,
				Allow:         []string{"test-agent"},
			},
			userAgent:      "",
			wantStatusCode: 200,
			wantNext:       true,
		},
		{
			name: "allow matching UA",
			options: Options{
				Allow: []string{"test-agent"},
			},
			userAgent:      "test-agent",
			wantStatusCode: 200,
			wantNext:       true,
		},
		{
			name: "allow non-matching UA",
			options: Options{
				Allow:                    []string{"test-agent"},
				RejectedHTTPStatusCode:   403,
				RejectedHTTPContentType:  "application/json",
				RejectedHTTPResponseBody: "forbidden",
			},
			userAgent:       "bad-agent",
			wantStatusCode:  403,
			wantContentType: "application/json",
			wantBody:        "forbidden",
			wantNext:        false,
		},
		{
			name: "deny matching UA",
			options: Options{
				Deny:                     []string{"bad-agent"},
				RejectedHTTPStatusCode:   403,
				RejectedHTTPContentType:  "application/json",
				RejectedHTTPResponseBody: "forbidden",
			},
			userAgent:       "bad-agent",
			wantStatusCode:  403,
			wantContentType: "application/json",
			wantBody:        "forbidden",
			wantNext:        false,
		},
		{
			name: "deny non-matching UA",
			options: Options{
				Deny: []string{"bad-agent"},
			},
			userAgent:      "good-agent",
			wantStatusCode: 200,
			wantNext:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware, err := NewMiddleware(tt.options)
			assert.NoError(t, err)

			ctx := context.Background()
			hzctx := app.NewContext(0)

			if tt.userAgent != "" {
				hzctx.Request.Header.SetUserAgentBytes([]byte(tt.userAgent))
			}

			called := false
			hzctx.SetIndex(-1)
			hzctx.SetHandlers([]app.HandlerFunc{middleware.ServeHTTP, func(ctx context.Context, c *app.RequestContext) {
				called = true
				c.Response.SetStatusCode(200)
			}})
			hzctx.Next(ctx)

			assert.Equal(t, tt.wantStatusCode, hzctx.Response.StatusCode())
			if tt.wantContentType != "" {
				assert.Equal(t, tt.wantContentType, string(hzctx.Response.Header.ContentType()))
			}

			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, string(hzctx.Response.Body()))
			}

			assert.Equal(t, tt.wantNext, called)
		})
	}
}

func TestNewMiddleware(t *testing.T) {
	h := middleware.Factory("ua_restriction")

	params := map[string]any{
		"bypass_missing":              true,
		"deny":                        []string{"test-agent"},
		"rejected_http_status_code":   403,
		"rejected_http_content_type":  "application/json",
		"rejected_http_response_body": "forbidden",
	}

	_, err := h(params)
	assert.NoError(t, err)
}
