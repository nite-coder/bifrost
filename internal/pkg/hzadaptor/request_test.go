package hzadaptor

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToHTTPRequest_Basic(t *testing.T) {
	t.Run("GET request without body", func(t *testing.T) {
		req := &protocol.Request{}
		req.SetMethod("GET")
		req.SetRequestURI("http://example.com/api/test?foo=bar")
		req.Header.Set("X-Custom-Header", "custom-value")

		httpReq, err := ToHTTPRequest(context.Background(), req)
		require.NoError(t, err)

		assert.Equal(t, "GET", httpReq.Method)
		assert.Equal(t, "/api/test", httpReq.URL.Path)
		assert.Equal(t, "foo=bar", httpReq.URL.RawQuery)
		assert.Equal(t, "example.com", httpReq.Host)
		assert.Equal(t, "custom-value", httpReq.Header.Get("X-Custom-Header"))
		assert.Nil(t, httpReq.Body)
		assert.Equal(t, int64(0), httpReq.ContentLength)
	})

	t.Run("POST request with body", func(t *testing.T) {
		req := &protocol.Request{}
		req.SetMethod("POST")
		req.SetRequestURI("http://example.com/api/submit")
		req.SetBody([]byte(`{"key":"value"}`))
		req.Header.Set("Content-Type", "application/json")

		httpReq, err := ToHTTPRequest(context.Background(), req)
		require.NoError(t, err)

		assert.Equal(t, "POST", httpReq.Method)
		assert.Equal(t, "/api/submit", httpReq.URL.Path)
		assert.Equal(t, "application/json", httpReq.Header.Get("Content-Type"))
		assert.Equal(t, int64(15), httpReq.ContentLength)

		body, err := io.ReadAll(httpReq.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"key":"value"}`, string(body))
	})

	t.Run("request with multiple headers", func(t *testing.T) {
		req := &protocol.Request{}
		req.SetMethod("GET")
		req.SetRequestURI("http://example.com/")
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Accept", "text/plain")
		req.Header.Set("Authorization", "Bearer token123")

		httpReq, err := ToHTTPRequest(context.Background(), req)
		require.NoError(t, err)

		accepts := httpReq.Header.Values("Accept")
		assert.Len(t, accepts, 2)
		assert.Contains(t, accepts, "application/json")
		assert.Contains(t, accepts, "text/plain")
		assert.Equal(t, "Bearer token123", httpReq.Header.Get("Authorization"))
	})
}

func TestToHTTPRequest_StreamBody(t *testing.T) {
	t.Run("streaming body is preserved", func(t *testing.T) {
		req := &protocol.Request{}
		req.SetMethod("POST")
		req.SetRequestURI("http://example.com/upload")
		req.Header.SetContentLength(100)

		streamData := strings.Repeat("x", 100)
		req.SetBodyStream(strings.NewReader(streamData), 100)

		httpReq, err := ToHTTPRequest(context.Background(), req)
		require.NoError(t, err)

		assert.True(t, req.IsBodyStream())
		assert.Equal(t, int64(100), httpReq.ContentLength)

		body, err := io.ReadAll(httpReq.Body)
		require.NoError(t, err)
		assert.Equal(t, streamData, string(body))
	})
}

func TestToHTTPRequest_Context(t *testing.T) {
	t.Run("context is passed through", func(t *testing.T) {
		type contextKey string
		key := contextKey("test-key")
		ctx := context.WithValue(context.Background(), key, "test-value")

		req := &protocol.Request{}
		req.SetMethod("GET")
		req.SetRequestURI("http://example.com/")

		httpReq, err := ToHTTPRequest(ctx, req)
		require.NoError(t, err)

		assert.Equal(t, "test-value", httpReq.Context().Value(key))
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		req := &protocol.Request{}
		req.SetMethod("GET")
		req.SetRequestURI("http://example.com/")

		httpReq, err := ToHTTPRequest(ctx, req)
		require.NoError(t, err)

		assert.Error(t, httpReq.Context().Err())
	})
}

func TestToHTTPRequest_Protocol(t *testing.T) {
	t.Run("HTTP/1.1", func(t *testing.T) {
		req := &protocol.Request{}
		req.SetMethod("GET")
		req.SetRequestURI("http://example.com/")
		req.Header.SetProtocol("HTTP/1.1")

		httpReq, err := ToHTTPRequest(context.Background(), req)
		require.NoError(t, err)

		assert.Equal(t, "HTTP/1.1", httpReq.Proto)
		assert.Equal(t, 1, httpReq.ProtoMajor)
		assert.Equal(t, 1, httpReq.ProtoMinor)
	})

	t.Run("HTTP/2.0", func(t *testing.T) {
		req := &protocol.Request{}
		req.SetMethod("GET")
		req.SetRequestURI("http://example.com/")
		req.Header.SetProtocol("HTTP/2.0")

		httpReq, err := ToHTTPRequest(context.Background(), req)
		require.NoError(t, err)

		assert.Equal(t, "HTTP/2.0", httpReq.Proto)
		assert.Equal(t, 2, httpReq.ProtoMajor)
		assert.Equal(t, 0, httpReq.ProtoMinor)
	})
}

func TestParseHTTPVersion(t *testing.T) {
	tests := []struct {
		name      string
		proto     string
		wantMajor int
		wantMinor int
	}{
		{"HTTP/1.0", "HTTP/1.0", 1, 0},
		{"HTTP/1.1", "HTTP/1.1", 1, 1},
		{"HTTP/2.0", "HTTP/2.0", 2, 0},
		{"HTTP/3.0", "HTTP/3.0", 3, 0},
		{"empty string defaults to 1.1", "", 1, 1},
		{"short string defaults to 1.1", "HTTP", 1, 1},
		{"invalid format defaults to 1.1", "invalid", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor := parseHTTPVersion(tt.proto)
			assert.Equal(t, tt.wantMajor, major)
			assert.Equal(t, tt.wantMinor, minor)
		})
	}
}

func TestBytesReader(t *testing.T) {
	t.Run("read all at once", func(t *testing.T) {
		data := []byte("hello world")
		r := &bytesReader{b: data}

		buf := make([]byte, 100)
		n, err := r.Read(buf)

		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf[:n])
	})

	t.Run("read in chunks", func(t *testing.T) {
		data := []byte("hello world")
		r := &bytesReader{b: data}

		var result bytes.Buffer
		buf := make([]byte, 3)

		for {
			n, err := r.Read(buf)
			if n > 0 {
				result.Write(buf[:n])
			}
			if err == io.EOF {
				break
			}
			assert.NoError(t, err)
		}

		assert.Equal(t, data, result.Bytes())
	})

	t.Run("EOF on empty reader", func(t *testing.T) {
		r := &bytesReader{b: []byte{}}
		buf := make([]byte, 10)

		n, err := r.Read(buf)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})

	t.Run("subsequent reads return EOF", func(t *testing.T) {
		r := &bytesReader{b: []byte("hi")}
		buf := make([]byte, 10)

		// First read
		n, err := r.Read(buf)
		assert.Equal(t, 2, n)
		assert.NoError(t, err)

		// Second read should be EOF
		n, err = r.Read(buf)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
	})
}
