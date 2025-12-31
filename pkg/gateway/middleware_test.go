package gateway

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	_ "github.com/nite-coder/bifrost/pkg/middleware/cors"
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

func TestLoadMiddlewares(t *testing.T) {
	t.Run("empty middleware ID", func(t *testing.T) {
		middlewareOptions := map[string]config.MiddlwareOptions{
			"": {
				Type: "cors",
			},
		}

		_, err := loadMiddlewares(middlewareOptions)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "middleware ID cannot be empty")
	})

	t.Run("empty middleware type", func(t *testing.T) {
		middlewareOptions := map[string]config.MiddlwareOptions{
			"test_middleware": {
				Type: "",
			},
		}

		_, err := loadMiddlewares(middlewareOptions)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "middleware type cannot be empty")
	})

	t.Run("unknown middleware type", func(t *testing.T) {
		middlewareOptions := map[string]config.MiddlwareOptions{
			"test_middleware": {
				Type: "unknown_type",
			},
		}

		_, err := loadMiddlewares(middlewareOptions)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "was not found")
	})

	t.Run("successful middleware loading", func(t *testing.T) {
		middlewareOptions := map[string]config.MiddlwareOptions{
			"cors_middleware": {
				Type: "cors",
			},
		}

		middlewares, err := loadMiddlewares(middlewareOptions)
		assert.NoError(t, err)
		assert.Len(t, middlewares, 1)
		assert.NotNil(t, middlewares["cors_middleware"])
	})
}

func TestAbortMiddleware(t *testing.T) {
	ctx := context.Background()
	hzCtx := app.NewContext(0)

	m := &AbortMiddleware{}
	m.ServeHTTP(ctx, hzCtx)

	assert.True(t, hzCtx.IsAborted())
}

func TestFirstRouteMiddleware(t *testing.T) {
	ctx := context.Background()
	hzCtx := app.NewContext(0)

	routeOpts := &variable.RequestRoute{
		RouteID:   "test_route",
		ServiceID: "test_service",
	}

	m := newFirstRouteMiddleware(routeOpts)
	m.ServeHTTP(ctx, hzCtx)

	storedRoute, exists := hzCtx.Get(variable.BifrostRoute)
	assert.True(t, exists)
	assert.Equal(t, routeOpts, storedRoute)
}
