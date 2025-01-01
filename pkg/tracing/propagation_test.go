package tracing

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestExtract(t *testing.T) {
	ctx := context.Background()

	var propagators []propagation.TextMapPropagator
	propagators = append(propagators, propagation.TraceContext{})
	propagators = append(propagators, propagation.Baggage{})
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))

	headers := &protocol.RequestHeader{}
	headers.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	ctx = Extract(ctx, headers)
	span := trace.SpanFromContext(ctx)

	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", span.SpanContext().TraceID().String())
}

func TestInject(t *testing.T) {
	ctx := context.Background()

	var propagators []propagation.TextMapPropagator
	propagators = append(propagators, propagation.TraceContext{})
	propagators = append(propagators, propagation.Baggage{})
	propagators = append(propagators, b3.New())
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    [16]byte{1},
		SpanID:     [8]byte{2},
		TraceFlags: 0,
		TraceState: trace.TraceState{},
		Remote:     false,
	})

	ctx = trace.ContextWithSpanContext(ctx, spanContext)
	md := &protocol.RequestHeader{}

	type args struct {
		ctx      context.Context
		metadata *protocol.RequestHeader
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "inject valid",
			args: args{
				ctx:      ctx,
				metadata: md,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Inject(tt.args.ctx, tt.args.metadata)
			assert.NotEmpty(t, tt.args.metadata)
			assert.Equal(t, "01000000000000000000000000000000-0200000000000000-0", md.Get("b3"))
			assert.Equal(t, "00-01000000000000000000000000000000-0200000000000000-00", md.Get("traceparent"))
		})
	}
}
