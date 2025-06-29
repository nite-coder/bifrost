package compression

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func newRequestContext(method, path string, headers map[string]string, body []byte) *app.RequestContext {
	c := app.NewContext(0)
	c.Request.SetMethod(method)
	c.Request.URI().SetPath(path)
	for k, v := range headers {
		c.Request.Header.Set(k, v)
	}
	if body != nil {
		c.Response.SetBody(body)
	}
	return c
}

func TestCompressesResponse(t *testing.T) {
	h := middleware.FindHandlerByType("compression")
	mw, err := h(nil)
	assert.NoError(t, err)

	ctx := context.Background()

	headers := map[string]string{
		"Accept-Encoding": "gzip",
	}
	c := newRequestContext("POST", "/test", headers, []byte("hello"))

	// Set a response body before calling ServeHTTP
	c.Response.SetBody([]byte("hello world"))

	mw(ctx, c)

	assert.Equal(t, "gzip", string(c.Response.Header.Peek(headerContentEncoding)))
	assert.Equal(t, "Accept-Encoding", string(c.Response.Header.Peek(headerVary)))
	assert.NotEqual(t, []byte("hello world"), c.Response.Body())
	assert.True(t, len(c.Response.Body()) > 0)
}

func TestSkipsIfAlreadyCompressed(t *testing.T) {
	opts := Options{Level: 1}
	mw := NewMiddleware(opts)
	ctx := context.Background()

	headers := map[string]string{
		"Accept-Encoding": "gzip",
	}
	c := newRequestContext("GET", "/test", headers, []byte("hello world"))

	// Set a response body before calling ServeHTTP
	c.Response.SetBody([]byte("hello world"))
	c.Response.Header.Set(headerContentEncoding, "gzip")

	mw.ServeHTTP(ctx, c)

	// Should not double compress, so body remains unchanged
	assert.Equal(t, "gzip", string(c.Response.Header.Peek(headerContentEncoding)))
	assert.Equal(t, []byte("hello world"), c.Response.Body())
}

func TestSkipsIfNotAcceptedEncoding(t *testing.T) {
	opts := Options{Level: 1}
	mw := NewMiddleware(opts)
	ctx := context.Background()

	headers := map[string]string{
		"Accept-Encoding": "deflate",
	}
	c := newRequestContext("GET", "/test", headers, nil)

	c.Response.SetBody([]byte("hello world"))

	mw.ServeHTTP(ctx, c)

	assert.Empty(t, c.Response.Header.Peek(headerContentEncoding))
	assert.Equal(t, []byte("hello world"), c.Response.Body())
}

func TestSkipsIfConnectionUpgrade(t *testing.T) {
	opts := Options{Level: 1}
	mw := NewMiddleware(opts)
	ctx := context.Background()

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Connection":      "Upgrade",
	}
	c := newRequestContext("GET", "/test", headers, nil)

	c.Response.SetBody([]byte("hello world"))

	mw.ServeHTTP(ctx, c)

	assert.Empty(t, c.Response.Header.Peek(headerContentEncoding))
	assert.Equal(t, []byte("hello world"), c.Response.Body())
}

func TestSkipsIfEventStream(t *testing.T) {
	opts := Options{Level: 1}
	mw := NewMiddleware(opts)
	ctx := context.Background()

	headers := map[string]string{
		"Accept-Encoding": "gzip",
		"Accept":          "text/event-stream",
	}
	c := newRequestContext("GET", "/test", headers, nil)

	c.Response.SetBody([]byte("hello world"))

	mw.ServeHTTP(ctx, c)

	assert.Empty(t, c.Response.Header.Peek(headerContentEncoding))
	assert.Equal(t, []byte("hello world"), c.Response.Body())
}

func TestSkipsIfExcludedPath(t *testing.T) {
	opts := Options{
		Level:         1,
		ExcludedPaths: []string{"/excluded"},
	}
	mw := NewMiddleware(opts)
	ctx := context.Background()

	headers := map[string]string{
		"Accept-Encoding": "gzip",
	}
	c := newRequestContext("GET", "/excluded", headers, nil)

	c.Response.SetBody([]byte("hello world"))

	mw.ServeHTTP(ctx, c)

	assert.Empty(t, c.Response.Header.Peek(headerContentEncoding))
	assert.Equal(t, []byte("hello world"), c.Response.Body())
}

func TestNoBody(t *testing.T) {
	opts := Options{Level: 1}
	mw := NewMiddleware(opts)
	ctx := context.Background()

	headers := map[string]string{
		"Accept-Encoding": "gzip",
	}
	c := newRequestContext("POST", "/test", headers, nil)

	mw.ServeHTTP(ctx, c)

	assert.Equal(t, "gzip", string(c.Response.Header.Peek(headerContentEncoding)))
	assert.Equal(t, "Accept-Encoding", string(c.Response.Header.Peek(headerVary)))
	assert.Equal(t, 0, len(c.Response.Body()))
}

func TestWildcardAcceptEncoding(t *testing.T) {
	opts := Options{Level: 1}
	mw := NewMiddleware(opts)
	ctx := context.Background()

	headers := map[string]string{
		"Accept-Encoding": "*",
	}
	c := newRequestContext("GET", "/test", headers, nil)

	c.Response.SetBody([]byte("hello world"))

	mw.ServeHTTP(ctx, c)

	assert.Equal(t, "gzip", string(c.Response.Header.Peek(headerContentEncoding)))
	assert.Equal(t, "Accept-Encoding", string(c.Response.Header.Peek(headerVary)))
	assert.NotEqual(t, []byte("hello world"), c.Response.Body())
	assert.True(t, len(c.Response.Body()) > 0)
}
