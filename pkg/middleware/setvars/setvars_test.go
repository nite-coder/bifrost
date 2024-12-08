package setvars

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/stretchr/testify/assert"
)

func TestSetVarsMiddleware(t *testing.T) {

	vars := map[string]any{
		variable.REQUEST_PATH_ALIAS:  "/orders/{order_id}",
		variable.UPSTREAM_PATH_ALIAS: "/backend/orders/{order_id}",
	}

	m := NewMiddleware(vars)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/orders/123")
	m.ServeHTTP(ctx, hzCtx)

	assert.Equal(t, "/orders/{order_id}", hzCtx.GetString(variable.REQUEST_PATH_ALIAS))
	assert.Equal(t, "/backend/orders/{order_id}", hzCtx.GetString(variable.UPSTREAM_PATH_ALIAS))

}
