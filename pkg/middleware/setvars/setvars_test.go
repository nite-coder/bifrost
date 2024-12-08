package setvars

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/stretchr/testify/assert"
)

func TestSetVarsMiddleware(t *testing.T) {
	h := middleware.FindHandlerByType("setvars")

	params := map[string]any{
		variable.REQUEST_PATH_ALIAS:  "/orders/{order_id}",
		variable.UPSTREAM_PATH_ALIAS: "/backend/orders/{order_id}",
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/orders/123")
	m(ctx, hzCtx)

	assert.Equal(t, "/orders/{order_id}", hzCtx.GetString(variable.REQUEST_PATH_ALIAS))
	assert.Equal(t, "/backend/orders/{order_id}", hzCtx.GetString(variable.UPSTREAM_PATH_ALIAS))
}