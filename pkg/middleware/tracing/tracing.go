package tracing

import (
	"context"

	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"

	"github.com/cloudwego/hertz/pkg/app"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	_ = middleware.RegisterMiddleware("tracing", func(param map[string]any) (app.HandlerFunc, error) {
		m := NewMiddleware()
		return m.ServeHTTP, nil
	})
}

type TracingMiddleware struct {
	tracer trace.Tracer
}

func NewMiddleware() *TracingMiddleware {
	return &TracingMiddleware{
		tracer: otel.Tracer("bifrost"),
	}
}

func (m *TracingMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	if m.tracer == nil {
		c.Next(ctx)
		return
	}

	method := string(c.Method())
	path := string(c.Request.Path())

	spanOptions := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithNewRoot(),
	}

	ctx, span := m.tracer.Start(ctx, method+" "+path, spanOptions...)

	defer func() {
		labels := []attribute.KeyValue{
			attribute.String("http.scheme", string(c.Request.Scheme())),
			attribute.String("http.host", string(c.Request.Host())),
			attribute.String("http.method", method),
			attribute.String("http.path", path),
			attribute.String("http.user_agent", string(c.Request.Header.UserAgent())),
			attribute.String("protocol", c.Request.Header.GetProtocol()),
			attribute.String("client_ip", c.ClientIP()),
			attribute.Int("http.status", c.Response.StatusCode()),
		}

		if c.Response.StatusCode() >= 200 && c.Response.StatusCode() < 300 {
			span.SetStatus(codes.Ok, "")
		} else if c.Response.StatusCode() >= 400 {
			span.SetStatus(codes.Error, "")
		}

		span.SetAttributes(labels...)
		span.End()
	}()

	traceID := span.SpanContext().TraceID()
	c.Set(variable.TRACE_ID, traceID.String())

	c.Next(ctx)
}
