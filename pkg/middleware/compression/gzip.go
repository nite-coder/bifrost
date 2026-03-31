package compression

import (
	"bytes"
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/compress"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/nite-coder/blackbear/pkg/cast"

	"github.com/nite-coder/bifrost/pkg/middleware"
)

const (
	encodingGzip          = "gzip"
	headerAcceptEncoding  = "Accept-Encoding"
	headerContentEncoding = "Content-Encoding"
	headerContentType     = "Content-Type"
	headerVary            = "Vary"
)

// Options defines the configuration for the compression middleware.
type Options struct {
	Level         int      `mapstructure:"level"`
	ExcludedPaths []string `mapstructure:"excluded_paths"`
}

// CompressionMiddleware is a middleware that compresses the response body.
type CompressionMiddleware struct {
	options *Options
}

// NewMiddleware creates a new CompressionMiddleware instance.
func NewMiddleware(options Options) *CompressionMiddleware {
	return &CompressionMiddleware{
		options: &options,
	}
}

func (m *CompressionMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	if !m.shouldCompress(&c.Request) {
		return
	}

	c.Next(ctx)

	// Skip compression if already compressed
	if len(c.Response.Header.Peek(headerContentEncoding)) > 0 {
		return
	}

	// compress response body
	c.Header(headerContentEncoding, encodingGzip)
	c.Header(headerVary, headerAcceptEncoding)
	if len(c.Response.Body()) > 0 {
		gzipBytes := compress.AppendGzipBytesLevel(nil, c.Response.Body(), m.options.Level)
		c.Response.SetBodyStream(bytes.NewBuffer(gzipBytes), len(gzipBytes))
	}
}

func (m *CompressionMiddleware) shouldCompress(req *protocol.Request) bool {
	if (!strings.Contains(req.Header.Get(headerAcceptEncoding), encodingGzip) &&
		strings.TrimSpace(req.Header.Get(headerAcceptEncoding)) != "*") ||
		strings.Contains(req.Header.Get("Connection"), "Upgrade") ||
		strings.Contains(req.Header.Get("Accept"), "text/event-stream") {
		return false
	}

	// Check if the request path is excluded
	for _, excludedPath := range m.options.ExcludedPaths {
		path := cast.B2S(req.URI().RequestURI())
		if strings.EqualFold(path, excludedPath) {
			return false
		}
	}

	return true
}

// Init registers the compression middleware.
func Init() error {
	return middleware.RegisterTyped([]string{"compression"}, func(opts Options) (app.HandlerFunc, error) {
		if opts.Level == 0 {
			opts.Level = compress.CompressDefaultCompression
		}

		m := NewMiddleware(opts)
		return m.ServeHTTP, nil
	})
}
