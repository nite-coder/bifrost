package requesttransformer

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRemove(t *testing.T) {
	h := middleware.Factory("request_transformer")

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
	h := middleware.Factory("request_transformer")

	params := map[string]any{
		"add": map[string]any{
			"headers": map[string]string{
				"x-source":         "web",
				"x-http-start":     "$var.http_start",
				"x-existing-value": "hello",
			},
			"querystring": map[string]string{
				"mode": "1",
				"foo":  "bar",
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
	hzCtx.Request.Header.Add("x-existing-value", "world")
	hzCtx.Request.URI().QueryArgs().Add("foo", "foo1")
	m(ctx, hzCtx)

	assert.Equal(t, "web", hzCtx.Request.Header.Get("x-source"))
	assert.Equal(t, "12345678", hzCtx.Request.Header.Get("x-http-start"))
	assert.Equal(t, "world", hzCtx.Request.Header.Get("x-existing-value")) // cannot overwrite
	assert.Equal(t, "foo1", hzCtx.Query("foo"))                            // cannot overwrite

	mode := hzCtx.Query("mode")
	assert.Equal(t, "1", mode)
}

func TestSet(t *testing.T) {
	h := middleware.Factory("request_transformer")

	params := map[string]any{
		"set": map[string]any{
			"headers": map[string]string{
				"x-existing-value": "hello",
			},
			"querystring": map[string]string{
				"foo": "bar",
			},
		},
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	hzCtx.Request.Header.Add("x-existing-value", "world")
	hzCtx.Request.URI().QueryArgs().Add("foo", "foo1")
	m(ctx, hzCtx)

	assert.Equal(t, "hello", hzCtx.Request.Header.Get("x-existing-value"))
	assert.Equal(t, "bar", hzCtx.Query("foo"))
}
