package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestTracer(t *testing.T) {

	options := config.TracingOptions{
		Enabled:      true,
		ServiceName:  "bifrost",
		Propagators:  []string{"tracecontext", "baggage"},
		Endpoint:     "http://localhost:4317",
		Insecure:     false,
		SamplingRate: 1,
		BatchSize:    500,
		QueueSize:    50000,
		Flush:        10 * time.Second,
	}

	tracer, err := initTracerProvider(options)
	assert.NoError(t, err)
	assert.NotNil(t, tracer)
}

func TestTracingMiddleware(t *testing.T) {
	attrs := map[string]string{
		"response_header": "x-trace-id",
	}

	flag := true
	options := config.ServerTracingOptions{
		Enabled:    &flag,
		Attributes: attrs,
	}

	m := newTracingMiddleware(options)

	tracerProvider := trace.NewTracerProvider()
	otel.SetTracerProvider(tracerProvider)
	defer func() { _ = tracerProvider.Shutdown(context.Background()) }()

	hzCtx := app.NewContext(0)
	hzCtx.Request.URI().SetPath("/test")
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.Header.SetUserAgentBytes([]byte("Go-http-client/1.1"))
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8001/test")

	m.ServeHTTP(context.Background(), hzCtx)

	traceID := variable.GetString(variable.TraceID, hzCtx)
	assert.NotEmpty(t, traceID)
}
