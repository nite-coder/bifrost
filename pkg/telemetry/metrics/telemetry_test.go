package metrics

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestTracer(t *testing.T) {

	opts := config.MetricsOptions{
		Prometheus: config.PrometheusOptions{
			Enabled: true,
		},
	}
	provider, err := NewProvider(context.Background(), opts)
	assert.NoError(t, err)
	defer provider.Shutdown(context.Background())

	promOpts := []Option{}
	promOpts = append(promOpts, WithHistogramBuckets(defaultBuckets))
	promTracer := NewTracer(promOpts...)

	h := server.Default(
		server.WithHostPorts("127.0.0.1:6666"),
		server.WithTracer(promTracer),
	)

	// Use the provider with the middleware to enable OTel metrics in the /metrics endpoint
	h.Use(NewMetricMiddleware("/metrics", provider).ServeHTTP)

	h.GET("/test_get", func(c context.Context, ctx *app.RequestContext) {
		ctx.String(200, "hello get")
	})

	h.POST("/test_post", func(ctx context.Context, c *app.RequestContext) {
		c.Set(variable.GRPCStatusCode, codes.OK)
		c.String(200, "hello post")
	})

	// Record a custom OTel metric
	meter := provider.MeterProvider().Meter("test-meter")
	counter, err := meter.Int64Counter("otel_custom_counter")
	assert.NoError(t, err)
	counter.Add(context.Background(), 5)

	go h.Spin()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = h.Shutdown(ctx)
	}()

	assert.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:6666", 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Server failed to start")

	for i := 0; i < 10; i++ {
		_, err := http.Get("http://127.0.0.1:6666/test_get")
		assert.NoError(t, err)
		_, err = http.Post("http://127.0.0.1:6666/test_post", "application/json", strings.NewReader(""))
		assert.NoError(t, err)
	}

	metricsRes, err := http.Get("http://127.0.0.1:6666/metrics")

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, metricsRes.StatusCode)

	defer metricsRes.Body.Close()

	metricsResBytes, err := io.ReadAll(metricsRes.Body)

	assert.NoError(t, err)

	metricsResStr := string(metricsResBytes)

	// Verify legacy tracer metrics
	assert.True(t, strings.Contains(metricsResStr, `http_server_requests{grpc_status_code="",method="GET",path="/test_get",route_id="unknown",server_id="unknown",service_id="unknown",status_code="200"} 10`))
	assert.True(t, strings.Contains(metricsResStr, `http_server_requests{grpc_status_code="OK",method="POST",path="/test_post",route_id="unknown",server_id="unknown",service_id="unknown",status_code="200"} 10`))
	assert.True(t, strings.Contains(metricsResStr, `http_bifrost_request_duration_bucket{method="GET",path="/test_get",route_id="unknown",server_id="unknown",service_id="unknown",status_code="200",le="0.005"} 10`))
	assert.True(t, strings.Contains(metricsResStr, `http_bifrost_request_duration_bucket{method="POST",path="/test_post",route_id="unknown",server_id="unknown",service_id="unknown",status_code="200",le="0.05"} 10`))
	assert.True(t, strings.Contains(metricsResStr, `http_bifrost_request_duration_count{method="GET",path="/test_get",route_id="unknown",server_id="unknown",service_id="unknown",status_code="200"} 10`))

	// Verify custom OTel metric (converted to Prometheus format)
	// OTel metrics might have scope labels
	assert.True(t, strings.Contains(metricsResStr, `otel_custom_counter_total{otel_scope_name="test-meter",otel_scope_version=""} 5`))
}
