package gateway

import (
	"context"
	"unsafe"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/cloudwego/netpoll"
)

var (
	spaceByte   = []byte{byte(' ')}
	httpMethods = []string{CONNECT, DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT, TRACE}
)

const (
	// CONNECT HTTP method
	CONNECT = "CONNECT"
	// DELETE HTTP method
	DELETE = "DELETE"
	// GET HTTP method
	GET = "GET"
	// HEAD HTTP method
	HEAD = "HEAD"
	// OPTIONS HTTP method
	OPTIONS = "OPTIONS"
	// PATCH HTTP method
	PATCH = "PATCH"
	// POST HTTP method
	POST = "POST"
	// PUT HTTP method
	PUT = "PUT"
	// TRACE HTTP method
	TRACE = "TRACE"
)

var runTask = gopool.CtxGo

func setRunner(runner func(ctx context.Context, f func())) {
	runTask = runner
}

func DisableGopool() error {
	_ = netpoll.DisableGopool()
	runTask = func(ctx context.Context, f func()) {
		go f()
	}
	return nil
}

// b2s converts byte slice to a string without memory allocation.
// See https://groups.google.com/forum/#!msg/Golang-Nuts/ENgbUzYvCuU/90yGx7GUAgAJ .
func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// s2b converts string to a byte slice without memory allocation.
func s2b(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
