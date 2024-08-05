package gateway

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/cloudwego/netpoll"
)

var (
	spaceByte   = []byte{byte(' ')}
	httpMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodConnect}
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

func isValidHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodConnect:
		return true
	default:
		return false
	}
}

func getStackTrace() string {
	stackBuf := make([]uintptr, 50)
	length := runtime.Callers(3, stackBuf)
	stack := stackBuf[:length]

	var b strings.Builder
	frames := runtime.CallersFrames(stack)

	for {
		frame, more := frames.Next()

		if !strings.Contains(frame.File, "runtime/") {
			_, _ = b.WriteString(fmt.Sprintf("\n\tFile: %s, Line: %d. Function: %s", frame.File, frame.Line, frame.Function))
		}

		if !more {
			break
		}
	}
	return b.String()
}
