package prommetric

import (
	"bytes"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpGET = []byte("GET")
)

type PromMetricMiddleware struct {
	path []byte
}

func New(path string) *PromMetricMiddleware {
	return &PromMetricMiddleware{
		path: []byte(path),
	}
}

func (m *PromMetricMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	if bytes.Equal(ctx.Request.Method(), httpGET) && bytes.Equal(ctx.Request.Path(), m.path) {
		httpReq, _ := adaptor.GetCompatRequest(&ctx.Request)
		httpResp := adaptor.GetCompatResponseWriter(&ctx.Response)

		h := promhttp.Handler()
		h.ServeHTTP(httpResp, httpReq)
	}
}
