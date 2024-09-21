package opentelemetry

import (
	"context"

	"github.com/nite-coder/bifrost/pkg/config"

	"github.com/cloudwego/hertz/pkg/app"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	tracer trace.Tracer
}

func NewTracer(opts config.TracingOptions) (*Tracer, error) {

	exporters := []sdktrace.SpanExporter{}

	if len(opts.OTLP.HTTP.Endpoint) > 0 {
		httpTraceExporter, err := newTraceHTTPExporter(opts)
		if err != nil {
			return nil, err
		}

		exporters = append(exporters, httpTraceExporter)
	}

	if len(opts.OTLP.GRPC.Endpoint) > 0 {
		grpcTraceExporter, err := newTraceGRPCExporter(opts)
		if err != nil {
			return nil, err
		}

		exporters = append(exporters, grpcTraceExporter)
	}

	tracerProvider := newTraceProvider(exporters)
	otel.SetTracerProvider(tracerProvider)

	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	tracer := otel.Tracer("bifrost")

	t := &Tracer{tracer: tracer}

	return t, nil
}

func (t *Tracer) Start(ctx context.Context, c *app.RequestContext) context.Context {
	return ctx
}

func (t *Tracer) Finish(ctx context.Context, c *app.RequestContext) {
}
