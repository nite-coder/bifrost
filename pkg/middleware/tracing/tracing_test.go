package tracing

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestServeHTTP(t *testing.T) {
	tracerProvider := trace.NewTracerProvider()
	otel.SetTracerProvider(tracerProvider)
	defer func() { _ = tracerProvider.Shutdown(context.Background()) }()

	middleware := NewMiddleware()

	h := server.New()
	h.Use(middleware.ServeHTTP)

	h.GET("/test", func(ctx context.Context, c *app.RequestContext) {
		c.String(http.StatusOK, "OK")
	})

	req := app.NewContext(0)
	req.Request.URI().SetPath("/test")
	req.Request.SetMethod("GET")
	req.Request.Header.SetUserAgentBytes([]byte("Go-http-client/1.1"))
	req.Request.SetRequestURI("http://127.0.0.1:8001/test")

	middleware.ServeHTTP(context.Background(), req)

	traceID, found := variable.Get(variable.TRACE_ID, req)
	assert.True(t, found)
	assert.NotEmpty(t, traceID)
}
