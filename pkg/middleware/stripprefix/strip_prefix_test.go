package stripprefix

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestStripMiddleware(t *testing.T) {
	m := NewMiddleware([]string{"/api/v1"})

	ctx := context.Background()

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("POST")
	hzCtx.Request.URI().SetPath("/api/v1/hello")
	m.ServeHTTP(ctx, hzCtx)

	assert.Equal(t, "/hello", string(hzCtx.Request.Path()))
}
