package requesttermination

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestRequestTermination(t *testing.T) {

	options := Options{
		StatusCode:  200,
		ContentType: "application/json",
		Body:        "[]",
	}

	m := NewMiddleware(options)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m.ServeHTTP(ctx, hzCtx)

	assert.Equal(t, options.StatusCode, hzCtx.Response.StatusCode())
	assert.Equal(t, options.ContentType, hzCtx.Response.Header.Get("Content-Type"))
	assert.Equal(t, options.Body, string(hzCtx.Response.Body()))
}
