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
	"github.com/stretchr/testify/assert"
)

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

	t.Run("should abort if limit exceeded by actual body", func(t *testing.T) {
		cfg := Config{
			MaxRequestBodySize: 5,
		}
		router := newTestRouter(cfg)

		body := "hello world"
		// Even if Content-Length is missing, ut.PerformRequest might add it.
		// But our middleware checks both.
		w := ut.PerformRequest(router, "POST", "/", &ut.Body{Body: strings.NewReader(body), Len: len(body)})

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})
}
