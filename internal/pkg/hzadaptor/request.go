// Package hzadaptor provides utilities for adapting between Hertz protocol types
// and Go's standard library types.
package hzadaptor

import (
	"context"
	"io"
	"net/http"

	"github.com/cloudwego/hertz/pkg/protocol"
)

// ToHTTPRequest converts a Hertz protocol.Request to a standard library http.Request.
// This is designed for client-side use where we need to send a request using net/http.
//
// Note: The body is read from the request, so for streaming bodies this should
// be called only once. The returned request will have the body set appropriately.
func ToHTTPRequest(ctx context.Context, req *protocol.Request) (*http.Request, error) {
	// Create base request with method, URL, and nil body initially
	httpReq, err := http.NewRequestWithContext(
		ctx,
		string(req.Method()),
		req.URI().String(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Handle body: prefer streaming if available, otherwise use buffered body
	if req.IsBodyStream() {
		httpReq.Body = io.NopCloser(req.BodyStream())
		httpReq.ContentLength = int64(req.Header.ContentLength())
	} else {
		body := req.Body()
		if len(body) > 0 {
			httpReq.Body = io.NopCloser(&bytesReader{b: body})
			httpReq.ContentLength = int64(len(body))
		}
	}

	// Copy headers
	req.Header.VisitAll(func(k, v []byte) {
		httpReq.Header.Add(string(k), string(v))
	})

	// Set protocol version if available
	if proto := req.Header.GetProtocol(); proto != "" {
		httpReq.Proto = proto
		major, minor := parseHTTPVersion(proto)
		httpReq.ProtoMajor = major
		httpReq.ProtoMinor = minor
	}

	// Set Host header explicitly (net/http uses this field separately)
	httpReq.Host = string(req.Host())

	return httpReq, nil
}

// bytesReader is a simple io.Reader that reads from a byte slice.
// Unlike bytes.Reader, it doesn't implement Seek, which prevents
// http.Client from attempting to retry requests by seeking.
type bytesReader struct {
	b   []byte
	off int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.off >= len(r.b) {
		return 0, io.EOF
	}
	n = copy(p, r.b[r.off:])
	r.off += n
	return n, nil
}

// parseHTTPVersion parses an HTTP version string like "HTTP/1.1" or "HTTP/2.0"
func parseHTTPVersion(proto string) (major, minor int) {
	// Default to HTTP/1.1
	major, minor = 1, 1

	if len(proto) < 8 { // "HTTP/X.Y" minimum length
		return
	}

	// Parse "HTTP/X.Y" format
	if proto[0:5] == "HTTP/" && len(proto) >= 8 {
		if proto[5] >= '0' && proto[5] <= '9' {
			major = int(proto[5] - '0')
		}
		if proto[6] == '.' && len(proto) > 7 && proto[7] >= '0' && proto[7] <= '9' {
			minor = int(proto[7] - '0')
		}
	}

	return
}
