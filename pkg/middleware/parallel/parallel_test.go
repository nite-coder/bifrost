package parallel

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/middleware/setvars"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/stretchr/testify/assert"
)

func TestParallelMiddleware(t *testing.T) {
	_ = Init()
	_ = setvars.Init()
	h := middleware.Factory("parallel")

	params := []any{
		config.MiddlwareOptions{
			Type: "setvars",
			Params: []any{
				map[string]any{
					"Key":   variable.HTTPRoute,
					"Value": "/orders/{order_id}",
				},
			},
		},
		config.MiddlwareOptions{
			Type: "setvars",
			Params: []any{
				map[string]any{
					"Key":   "user_id",
					"Value": "123456",
				},
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

	assert.Equal(t, "/orders/{order_id}", hzCtx.GetString(variable.HTTPRoute))
	assert.Equal(t, "123456", variable.GetString("$var.user_id", hzCtx))
}
