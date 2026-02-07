package replacepath

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestReplacePath(t *testing.T) {
	_ = Init()
	h := middleware.Factory("replace_path")

	params := map[string]any{
		"path": "/api/v1/hello",
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m(ctx, hzCtx)

	assert.Equal(t, "/api/v1/hello", string(hzCtx.Request.Path()))
}

func TestReplacePathWithDirective(t *testing.T) {
	h := middleware.Factory("replace_path")

	params := map[string]any{
		"path": "/api/v1/hello/$var.name",
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()

	hzCtx := app.NewContext(0)
	hzCtx.Set("name", "bifrost")
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m(ctx, hzCtx)

	assert.Equal(t, "/api/v1/hello/bifrost", string(hzCtx.Request.Path()))
}
