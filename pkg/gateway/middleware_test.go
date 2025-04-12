package gateway

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/stretchr/testify/assert"
)

func BenchmarkSaveByte(b *testing.B) {

	path := []byte(`/spot/orders/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`)

	c := app.NewContext(0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b := []byte{}
		copy(b, path)
		c.Set(variable.HTTPRequestPath, b)
	}
}

func BenchmarkSaveString(b *testing.B) {

	path := []byte(`/spot/orders/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`)

	c := app.NewContext(0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := string(path)
		c.Set(variable.HTTPRequestPath, p)
	}
}

func TestPanicMiddleware(t *testing.T) {
	ctx := context.Background()

	hzCtx := app.NewContext(0)

	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/test")

	logger := log.FromContext(ctx)
	m := newInitMiddleware("test", logger)

	hzCtx.SetHandlers([]app.HandlerFunc{func(ctx context.Context, c *app.RequestContext) {
		panic("test panic")
	}})

	m.ServeHTTP(ctx, hzCtx)

	assert.Equal(t, 500, hzCtx.Response.StatusCode())
}
