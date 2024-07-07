package gateway

import (
	"context"
	"net/http"

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
