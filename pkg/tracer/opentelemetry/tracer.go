package opentelemetry

import (
	"context"
	"strings"

	"github.com/nite-coder/bifrost/pkg/config"

	"github.com/cloudwego/hertz/pkg/app"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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

	var propagators []propagation.TextMapPropagator

	for _, p := range opts.OTLP.Propagators {
		switch strings.TrimSpace(strings.ToLower(p)) {
		case "tracecontext":
			propagators = append(propagators, propagation.TraceContext{})
		case "baggage":
			propagators = append(propagators, propagation.Baggage{})
		case "b3":
			propagators = append(propagators, b3.New())
		case "jaeger":
			propagators = append(propagators, jaeger.Jaeger{})
		default:
		}
	}

	if len(propagators) == 0 {
		propagators = append(propagators, propagation.TraceContext{})
		propagators = append(propagators, propagation.Baggage{})
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagators...))

	tracer := otel.Tracer("bifrost")

	t := &Tracer{tracer: tracer}

	return t, nil
}

func (t *Tracer) Start(ctx context.Context, c *app.RequestContext) context.Context {
	return ctx
}

func (t *Tracer) Finish(ctx context.Context, c *app.RequestContext) {
}
