package requesttransformer

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRemove(t *testing.T) {
	h := middleware.FindHandlerByType("request_transformer")

	params := map[string]any{
		"remove": map[string]any{
			"headers":     []string{"x-user-id"},
			"querystring": []string{"mode"},
		},
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo?mode=1")
	hzCtx.Request.Header.Set("x-user-id", "1")
	m(ctx, hzCtx)

	userID := hzCtx.Request.Header.Get("x-user-id")
	assert.Empty(t, userID)

	mode := hzCtx.Request.URI().QueryArgs().Has("mode")
	assert.False(t, mode)
}

func TestAdd(t *testing.T) {
	h := middleware.FindHandlerByType("request_transformer")

	params := map[string]any{
		"add": map[string]any{
			"headers": map[string]string{
				"x-source":     "web",
				"x-http-start": "$var.http_start",
			},
			"querystring": map[string]string{
				"mode": "1",
			},
		},
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Set("http_start", "12345678")
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m(ctx, hzCtx)

	assert.Equal(t, "web", hzCtx.Request.Header.Get("x-source"))
	assert.Equal(t, "12345678", hzCtx.Request.Header.Get("x-http-start"))

	mode := hzCtx.Query("mode")
	assert.Equal(t, "1", mode)
}
