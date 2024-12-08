package tracing

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestTracingMiddleware(t *testing.T) {
	h := middleware.FindHandlerByType("tracing")

	params := map[string]any{
		"response_header": "x-trace-id",
	}

	m, err := h(params)
	assert.NoError(t, err)

	tracerProvider := trace.NewTracerProvider()
	otel.SetTracerProvider(tracerProvider)
	defer func() { _ = tracerProvider.Shutdown(context.Background()) }()

	hzCtx := app.NewContext(0)
	hzCtx.Request.URI().SetPath("/test")
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.Header.SetUserAgentBytes([]byte("Go-http-client/1.1"))
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8001/test")

	m(context.Background(), hzCtx)

	traceID, found := variable.Get(variable.TraceID, hzCtx)
	assert.True(t, found)
	assert.NotEmpty(t, traceID)

	assert.Equal(t, traceID, hzCtx.Response.Header.Get("x-trace-id"))
}
