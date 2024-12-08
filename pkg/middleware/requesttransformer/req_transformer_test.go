package requesttransformer

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRemove(t *testing.T) {
	h := middleware.FindHandlerByType("request-transformer")

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
	h := middleware.FindHandlerByType("request-transformer")

	params := map[string]any{
		"add": map[string]any{
			"headers": map[string]string{
				"source": "web",
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
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m(ctx, hzCtx)

	mode := hzCtx.Query("mode")
	assert.Equal(t, "1", mode)
}
