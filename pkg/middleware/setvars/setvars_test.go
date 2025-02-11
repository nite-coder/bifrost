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
		variable.HTTPRoute: "/orders/{order_id}",
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/orders/123")
	m(ctx, hzCtx)

	assert.Equal(t, "/orders/{order_id}", hzCtx.GetString(variable.HTTPRoute))
}


func TestSetVarsWithDirectivesMiddleware(t *testing.T) {
	h := middleware.FindHandlerByType("setvars")

	params := map[string]any{
		variable.HTTPRoute: "/orders/{order_id}",
		"user_id": "$http.request.header.user_id",
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/orders/123")
	hzCtx.Request.Header.Set("user_id", "996")
	m(ctx, hzCtx)

	assert.Equal(t, "/orders/{order_id}", hzCtx.GetString(variable.HTTPRoute))
	assert.Equal(t, "996", variable.GetString("$var.user_id", hzCtx))
}