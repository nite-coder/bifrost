package stripprefix

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestStripMiddleware(t *testing.T) {
	h := middleware.Factory("strip_prefix")

	params := map[string]any{
		"prefixes": []string{"/api/v1"},
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("POST")
	hzCtx.Request.URI().SetPath("/api/v1/hello")
	m(ctx, hzCtx)

	assert.Equal(t, "/hello", string(hzCtx.Request.Path()))
}

func TestStripSlash(t *testing.T) {
	h := middleware.Factory("strip_prefix")

	params := map[string]any{
		"prefixes": []any{"/api/v1/"},
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("POST")
	hzCtx.Request.URI().SetPath("/api/v1/hello")
	m(ctx, hzCtx)

	assert.Equal(t, "/hello", string(hzCtx.Request.Path()))
}
