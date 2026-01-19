package metrics

import (
	"bytes"
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	promhttp "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpGET = []byte("GET")
)

// MetricMiddleware serves the /metrics endpoint for Prometheus scraping.
type MetricMiddleware struct {
	path    []byte
	handler http.Handler
}

// NewMetricMiddleware creates a new middleware that serves the /metrics endpoint.
// If provider has a MetricsHandler (OTel Prometheus Exporter), it uses that.
// Otherwise, falls back to the default promhttp.Handler() for compatibility.
func NewMetricMiddleware(path string, provider *Provider) *MetricMiddleware {
	if path == "" {
		path = "/metrics"
	}

	var handler http.Handler

	// Use provider's handler if available (OTel Prometheus Exporter with isolated registry)
	if provider != nil && provider.MetricsHandler() != nil {
		handler = provider.MetricsHandler()
	} else {
		// Fallback to default Prometheus handler (uses global registry)
		handler = promhttp.Handler()
	}

	return &MetricMiddleware{
		path:    []byte(path),
		handler: handler,
	}
}

// ServeHTTP handles the /metrics endpoint requests.
func (m *MetricMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	if bytes.Equal(c.Request.Method(), httpGET) && bytes.Equal(c.Request.Path(), m.path) {
		hzHandler := adaptor.HertzHandler(m.handler)
		hzHandler(ctx, c)
		c.Abort()
	}
}
