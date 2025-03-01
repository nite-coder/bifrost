package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitingMiddleware(t *testing.T) {
	options := Options{
		LimitBy:          "$client_ip",
		Limit:            3,
		WindowSize:       10 * time.Second,
		HTTPResponseBody: "too many requests",
		Strategy:         "local",
	}

	t.Run("Basic functionality", func(t *testing.T) {
		m, err := NewMiddleware(options)
		assert.NoError(t, err)

		ctx := context.Background()

		hzCtx := app.NewContext(0)
		hzCtx.Request.SetMethod("GET")
		hzCtx.Request.URI().SetPath("/foo")

		m.ServeHTTP(ctx, hzCtx) // first request
		assert.Equal(t, 200, hzCtx.Response.StatusCode())
		assert.Equal(t, "3", hzCtx.Response.Header.Get("X-RateLimit-Limit"))
		assert.Equal(t, "2", hzCtx.Response.Header.Get("X-RateLimit-Remaining"))

		m.ServeHTTP(ctx, hzCtx) // secord request
		m.ServeHTTP(ctx, hzCtx) // third request
		assert.Equal(t, 200, hzCtx.Response.StatusCode())
		assert.Equal(t, "3", hzCtx.Response.Header.Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", hzCtx.Response.Header.Get("X-RateLimit-Remaining"))

		// blocked
		m.ServeHTTP(ctx, hzCtx)
		assert.Equal(t, 429, hzCtx.Response.StatusCode())
		assert.Equal(t, "3", hzCtx.Response.Header.Get("X-RateLimit-Limit"))
		assert.Equal(t, "0", hzCtx.Response.Header.Get("X-RateLimit-Remaining"))
		assert.Equal(t, "too many requests", string(hzCtx.Response.Body()))
	})

}
