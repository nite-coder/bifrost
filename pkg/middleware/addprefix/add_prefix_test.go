package addprefix

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestAddPrefixMiddleware(t *testing.T) {
	m := NewMiddleware("/api/v1")

	ctx := context.Background()

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m.ServeHTTP(ctx, hzCtx)

	assert.Equal(t, "/api/v1/foo", string(hzCtx.Request.Path()))
}
