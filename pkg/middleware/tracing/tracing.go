package tracing

import (
	"context"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/blackbear/pkg/cast"
	"go.opentelemetry.io/otel"
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

	method := cast.B2S(ctx.Method())
	path := cast.B2S(ctx.Request.Path())

	spanOptions := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithNewRoot(),
	}

	c, span := m.tracer.Start(c, method+" "+path, spanOptions...)
	defer span.End()

	ctx.Next(c)
}
