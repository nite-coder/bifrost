package opentelemetry

import (
	"context"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
)

func newTraceHTTPExporter(opts config.TracingOptions) (trace.SpanExporter, error) {

	if opts.OTLP.HTTP.Endpoint == "" {
		opts.OTLP.HTTP.Endpoint = "localhost:4318"
	}

	tracingOptions := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(opts.OTLP.HTTP.Endpoint),
	}

	if opts.OTLP.HTTP.Insecure {
		tracingOptions = append(tracingOptions, otlptracehttp.WithInsecure())
	}

	ctx := context.Background()
	exporter, err := otlptracehttp.New(ctx, tracingOptions...)
	if err != nil {
		return nil, err
	}

	return exporter, nil
}

func newTraceGRPCExporter(opts config.TracingOptions) (trace.SpanExporter, error) {

	if opts.OTLP.GRPC.Endpoint == "" {
		opts.OTLP.GRPC.Endpoint = "localhost:4318"
	}

	tracingOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(opts.OTLP.GRPC.Endpoint),
	}

	if opts.OTLP.GRPC.Insecure {
		tracingOptions = append(tracingOptions, otlptracegrpc.WithInsecure())
	}

	ctx := context.Background()
	exporter, err := otlptracegrpc.New(ctx, tracingOptions...)
	if err != nil {
		return nil, err
	}

	return exporter, nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
	)
}

func newTraceProvider(traceExporter []trace.SpanExporter) *trace.TracerProvider {

	tracerOptions := []trace.TracerProviderOption{}

	for _, exporter := range traceExporter {
		tracerOptions = append(tracerOptions, trace.WithBatcher(exporter, trace.WithBatchTimeout(time.Second)))
	}

	traceProvider := trace.NewTracerProvider(tracerOptions...)
	return traceProvider
}
