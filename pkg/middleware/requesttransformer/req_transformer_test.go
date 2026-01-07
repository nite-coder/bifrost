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
			"headers":     []string{"x-user-id", "", "$var.header_key"},
			"querystring": []string{"mode", "", "$var.query_key"},
		},
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Set("header_key", "x-remove-me")
	hzCtx.Set("query_key", "remove_query")

	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo?mode=1&remove_query=true")
	hzCtx.Request.Header.Set("x-user-id", "1")
	hzCtx.Request.Header.Set("x-remove-me", "true")
	m(ctx, hzCtx)

	userID := hzCtx.Request.Header.Get("x-user-id")
	assert.Empty(t, userID)

	removedHeader := hzCtx.Request.Header.Get("x-remove-me")
	assert.Empty(t, removedHeader)

	mode := hzCtx.Request.URI().QueryArgs().Has("mode")
	assert.False(t, mode)

	removedQuery := hzCtx.Request.URI().QueryArgs().Has("remove_query")
	assert.False(t, removedQuery)
}

func TestAdd(t *testing.T) {
	h := middleware.Factory("request_transformer")

	params := map[string]any{
		"add": map[string]any{
			"headers": map[string]string{
				"x-source":         "web",
				"x-http-start":     "$var.http_start",
				"x-existing-value": "hello",
				"":                 "skip-me",
			},
			"querystring": map[string]string{
				"mode": "1",
				"foo":  "bar",
				"dyn":  "$var.dyn_val",
				"":     "skip-me",
			},
		},
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Set("http_start", "12345678")
	hzCtx.Set("dyn_val", "dynamic")
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

	dyn := hzCtx.Query("dyn")
	assert.Equal(t, "dynamic", dyn)
}

func TestSet(t *testing.T) {
	h := middleware.Factory("request_transformer")

	params := map[string]any{
		"set": map[string]any{
			"headers": map[string]string{
				"x-existing-value": "hello",
				"x-dyn":            "$var.dyn_header",
				"":                 "skip-me",
			},
			"querystring": map[string]string{
				"foo":   "bar",
				"q-dyn": "$var.dyn_query",
				"":      "skip-me",
			},
		},
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Set("dyn_header", "h-val")
	hzCtx.Set("dyn_query", "q-val")

	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	hzCtx.Request.Header.Add("x-existing-value", "world")
	hzCtx.Request.URI().QueryArgs().Add("foo", "foo1")
	m(ctx, hzCtx)

	assert.Equal(t, "hello", hzCtx.Request.Header.Get("x-existing-value"))
	assert.Equal(t, "h-val", hzCtx.Request.Header.Get("x-dyn"))
	assert.Equal(t, "bar", hzCtx.Query("foo"))
	assert.Equal(t, "q-val", hzCtx.Query("q-dyn"))
}

func TestFactory_Errors(t *testing.T) {
	h := middleware.Factory("request_transformer")

	tests := []struct {
		name        string
		params      any
		expectedErr string
	}{
		{
			name:        "nil params",
			params:      nil,
			expectedErr: "request_transformer middleware params is empty or invalid",
		},
		{
			name:        "invalid params structure",
			params:      "invalid-string",
			expectedErr: "request_transformer middleware params is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h(tt.params)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}
