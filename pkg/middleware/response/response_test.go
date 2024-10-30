package response

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestResponseMiddleware(t *testing.T) {
	m := NewMiddleware(ResponseOptions{
		StatusCode:  404,
		ContentType: "text/plain charset=utf-8",
		Content:     "not found",
	})

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m.ServeHTTP(ctx, hzCtx)


	assert.Equal(t, 404, hzCtx.Response.StatusCode())
	assert.Equal(t, "text/plain charset=utf-8", hzCtx.Response.Header.Get("Content-Type"))
	assert.Equal(t, "not found", string(hzCtx.Response.Body()))
}
