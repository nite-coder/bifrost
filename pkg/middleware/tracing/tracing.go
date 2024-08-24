package tracing

import (
	"context"
	"http-benchmark/pkg/config"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type TracingMiddleware struct {
	once   sync.Once
	tracer trace.Tracer
}

func NewMiddleware() *TracingMiddleware {
	return &TracingMiddleware{}
}

func (m *TracingMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	m.once.Do(func() {
		m.tracer = otel.Tracer("bifrost")
	})

	if m.tracer == nil {
		ctx.Next(c)
		return
	}

	method := string(ctx.Method())
	path := string(ctx.Request.Path())

	spanOptions := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithNewRoot(),
	}

	c, span := m.tracer.Start(c, method+" "+path, spanOptions...)

	defer func() {
		if ctx.Response.StatusCode() >= 200 && ctx.Response.StatusCode() < 300 {
			span.SetStatus(codes.Ok, "OK")
		}

		span.End()
	}()

	labels := []attribute.KeyValue{
		attribute.String("http.host", string(ctx.Request.Host())),
		attribute.String("http.method", method),
		attribute.String("http.path", path),
		attribute.String("http.protocol", ctx.Request.Header.GetProtocol()),
	}
	span.SetAttributes(labels...)

	traceID := span.SpanContext().TraceID()
	ctx.Set(config.TRACE_ID, traceID.String())

	ctx.Next(c)
}
