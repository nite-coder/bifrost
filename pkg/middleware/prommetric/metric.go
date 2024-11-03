package prommetric

import (
	"bytes"
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	_ = middleware.RegisterMiddleware("prom_metric", func(param map[string]any) (app.HandlerFunc, error) {
		path, ok := param["path"].(string)
		if !ok {
			return nil, errors.New("path is not set or path is invalid")
		}

		m := New(path)
		return m.ServeHTTP, nil
	})

}

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

func (m *PromMetricMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	if bytes.Equal(c.Request.Method(), httpGET) && bytes.Equal(c.Request.Path(), m.path) {
		httpReq, _ := adaptor.GetCompatRequest(&c.Request)
		httpResp := adaptor.GetCompatResponseWriter(&c.Response)

		h := promhttp.Handler()
		h.ServeHTTP(httpResp, httpReq)
		c.Abort()
	}
}
