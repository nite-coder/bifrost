package prometheus

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

func NewMetricMiddleware(path string) *PromMetricMiddleware {
	if path == "" {
		path = "/metrics"
	}

	return &PromMetricMiddleware{
		path: []byte(path),
	}
}

func (m *PromMetricMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	if bytes.Equal(c.Request.Method(), httpGET) && bytes.Equal(c.Request.Path(), m.path) {
		h := promhttp.Handler()
		hzHandler := adaptor.HertzHandler(h)
		hzHandler(ctx, c)
		c.Abort()
	}
}
