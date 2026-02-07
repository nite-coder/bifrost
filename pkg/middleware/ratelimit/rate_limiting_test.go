package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitMiddleware(t *testing.T) {
	_ = Init()
	h := middleware.Factory("rate_limit")

	params := map[string]any{
		"strategy":                    "local",
		"limit":                       3,
		"window_size":                 10 * time.Second,
		"limit_by":                    "ip:$client_ip",
		"header_limit":                "X-RateLimit-Limit",
		"header_remaining":            "X-RateLimit-Remaining",
		"rejected_http_status_code":   429,
		"rejected_http_response_body": "too many requests",
	}

	t.Run("local strategy", func(t *testing.T) {
		m, err := h(params)
		assert.NoError(t, err)

		ctx := context.Background()

		hzCtx := app.NewContext(0)
		hzCtx.Request.SetMethod("GET")
		hzCtx.Request.URI().SetPath("/foo")

		m(ctx, hzCtx) // first request
		assert.Equal(t, 200, hzCtx.Response.StatusCode())
		assert.Equal(t, "3", hzCtx.Response.Header.Get("X-RateLimit-Limit"))
		assert.Equal(t, "2", hzCtx.Response.Header.Get("X-RateLimit-Remaining"))

		m(ctx, hzCtx) // secord request
		m(ctx, hzCtx) // third request
		assert.Equal(t, 200, hzCtx.Response.StatusCode())
		assert.Equal(t, "3", hzCtx.Response.Header.Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", hzCtx.Response.Header.Get("X-RateLimit-Remaining"))

		// blocked
		m(ctx, hzCtx)
		assert.Equal(t, 429, hzCtx.Response.StatusCode())
		assert.Equal(t, "3", hzCtx.Response.Header.Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", hzCtx.Response.Header.Get("X-RateLimit-Remaining"))
		assert.Equal(t, "too many requests", string(hzCtx.Response.Body()))
	})
}
