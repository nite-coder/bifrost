package addprefix

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestAddPrefixMiddleware(t *testing.T) {
	h := middleware.FindHandlerByType("add_prefix")

	params := map[string]any{
		"prefix": "/api/v1",
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m(ctx, hzCtx)

	assert.Equal(t, "/api/v1/foo", string(hzCtx.Request.Path()))
}

func TestAddPrefixMoreSlash(t *testing.T) {
	h := middleware.FindHandlerByType("add_prefix")

	params := map[string]any{
		"prefix": "/api/v1/",
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m(ctx, hzCtx)

	assert.Equal(t, "/api/v1/foo", string(hzCtx.Request.Path()))
}


func TestAddPrefixWithDirective(t *testing.T) {
	h := middleware.FindHandlerByType("add_prefix")

	params := map[string]any{
		"prefix": "/api/v1/$var.name",
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()

	hzCtx := app.NewContext(0)
	hzCtx.Set("name", "bifrost")
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m(ctx, hzCtx)

	assert.Equal(t, "/api/v1/bifrost/foo", string(hzCtx.Request.Path()))
}