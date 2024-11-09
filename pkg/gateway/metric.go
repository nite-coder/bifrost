package gateway

import (
	"bytes"
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PromMetricMiddleware struct {
	path []byte
}

func newPromMetricMiddleware(path string) *PromMetricMiddleware {
	return &PromMetricMiddleware{
		path: []byte(path),
	}
}

func (m *PromMetricMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	if bytes.Equal(c.Request.Method(), httpGET) && bytes.Equal(c.Request.Path(), m.path) {
		httpReq, _ := adaptor.GetCompatRequest(&c.Request)
		httpResp := adaptor.GetCompatResponseWriter(&c.Response)

		h := promhttp.Handler()
		h.ServeHTTP(httpResp, httpReq)
		c.Abort()
	}
}
