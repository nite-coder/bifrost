package buffering

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

// Config holds the configuration for the buffering middleware.
type Config struct {
	// MaxRequestBodySize limits the maximum number of bytes for the request body.
	// Default 4194304 (4MB) if not specified or set to 0.
	MaxRequestBodySize int64 `mapstructure:"max_request_body_size" json:"max_request_body_size"`
}

type BufferingMiddleware struct {
	config *Config
}

func NewMiddleware(config Config) *BufferingMiddleware {
	if config.MaxRequestBodySize <= 0 {
		config.MaxRequestBodySize = 4 * 1024 * 1024 // 4MB default
	}
	return &BufferingMiddleware{
		config: &config,
	}
}

func (m *BufferingMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	// Check content length header if present
	contentLength := c.Request.Header.ContentLength()
	if contentLength > 0 && int64(contentLength) > m.config.MaxRequestBodySize {
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		return
	}

	// Read the entire body to buffer it
	body := c.Request.Body()

	// Verify the actual body size (especially important if Content-Length was missing or incorrect)
	if int64(len(body)) > m.config.MaxRequestBodySize {
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		return
	}

	c.Next(ctx)
}

func init() {
	_ = middleware.RegisterTyped([]string{"buffering"}, func(cfg Config) (app.HandlerFunc, error) {
		m := NewMiddleware(cfg)
		return m.ServeHTTP, nil
	})
}
