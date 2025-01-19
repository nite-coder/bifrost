package requesttermination

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRequestTermination(t *testing.T) {
	h := middleware.FindHandlerByType("request_termination")

	params := map[string]any{
		"status_code":  200,
		"content_type": "application/json",
		"body":         "[]",
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m(ctx, hzCtx)

	assert.Equal(t, params["status_code"], hzCtx.Response.StatusCode())
	assert.Equal(t, params["content_type"], hzCtx.Response.Header.Get("Content-Type"))
	assert.Equal(t, params["body"], string(hzCtx.Response.Body()))
}
