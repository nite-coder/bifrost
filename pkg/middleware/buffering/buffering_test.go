package buffering

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestNewMiddleware(t *testing.T) {
	t.Run("should set default MaxRequestBodySize", func(t *testing.T) {
		m := NewMiddleware(Config{MaxRequestBodySize: 0})
		assert.Equal(t, int64(4*1024*1024), m.config.MaxRequestBodySize)
	})

	t.Run("should set custom MaxRequestBodySize", func(t *testing.T) {
		m := NewMiddleware(Config{MaxRequestBodySize: 100})
		assert.Equal(t, int64(100), m.config.MaxRequestBodySize)
	})
}

func TestBufferingMiddleware(t *testing.T) {
	newTestRouter := func(cfg Config) *route.Engine {
		router := route.NewEngine(config.NewOptions([]config.Option{}))
		m := NewMiddleware(cfg)
		router.Use(m.ServeHTTP)
		router.POST("/", func(ctx context.Context, c *app.RequestContext) {
			c.String(consts.StatusOK, "ok")
		})
		return router
	}

	t.Run("should pass within limit", func(t *testing.T) {
		cfg := Config{
			MaxRequestBodySize: 1024,
		}
		router := newTestRouter(cfg)

		body := "hello world"
		w := ut.PerformRequest(router, "POST", "/", &ut.Body{Body: strings.NewReader(body), Len: len(body)})

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "ok", w.Body.String())
	})

	t.Run("should abort if limit exceeded by Content-Length", func(t *testing.T) {
		cfg := Config{
			MaxRequestBodySize: 5,
		}
		router := newTestRouter(cfg)

		body := "hello world"
		w := ut.PerformRequest(router, "POST", "/", &ut.Body{Body: strings.NewReader(body), Len: len(body)})

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("should abort if limit exceeded by actual body (no Content-Length)", func(t *testing.T) {
		cfg := Config{
			MaxRequestBodySize: 5,
		}
		router := newTestRouter(cfg)

		body := "hello world"
		// passing -1 for Len to simulate no Content-Length
		w := ut.PerformRequest(router, "POST", "/", &ut.Body{Body: strings.NewReader(body), Len: -1})

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("should work when registered via factory", func(t *testing.T) {
		factory := middleware.Factory("buffering")
		assert.NotNil(t, factory)

		handler, err := factory(map[string]any{
			"max_request_body_size": 1024,
		})
		assert.NoError(t, err)
		assert.NotNil(t, handler)

		router := route.NewEngine(config.NewOptions([]config.Option{}))
		router.Use(handler)
		router.POST("/", func(ctx context.Context, c *app.RequestContext) {
			c.String(consts.StatusOK, "ok")
		})

		body := "hello"
		w := ut.PerformRequest(router, "POST", "/", &ut.Body{Body: strings.NewReader(body), Len: len(body)})
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
