package gateway

import (
	"context"
	"runtime/debug"
	"strings"
	"time"

	"github.com/henrylee2cn/goutil/errors"
	"github.com/nite-coder/bifrost/pkg/config"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

func initTracerProvider(opts config.TracingOptions) (*sdktrace.TracerProvider, error) {
	if !opts.Enabled {
		return nil, nil
	}

	if opts.Endpoint == "" {
		// use grpc as default
		opts.Endpoint = "localhost:4317"
	}

	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}

	if opts.Flush.Seconds() <= 0 {
		opts.Flush = 5 * time.Second
	}

	if opts.QueueSize <= 0 {
		opts.QueueSize = 10000
	}

	if opts.Timeout.Seconds() <= 0 {
		opts.Timeout = 10 * time.Second
	}

	opts.Endpoint = strings.ToLower(opts.Endpoint)

	var exporter sdktrace.SpanExporter
	var err error

	if strings.HasPrefix(opts.Endpoint, "https") || strings.HasPrefix(opts.Endpoint, "http") {
		tracingOptions := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(opts.Endpoint),
			otlptracehttp.WithTimeout(opts.Timeout),
		}

		if opts.Insecure {
			tracingOptions = append(tracingOptions, otlptracehttp.WithInsecure())
		}

		exporter, err = otlptracehttp.New(context.Background(), tracingOptions...)
		if err != nil {
			return nil, err
		}
	} else {
		// grpc
		tracingOptions := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(opts.Endpoint),
			otlptracegrpc.WithTimeout(opts.Timeout),
		}

		if opts.Insecure {
			tracingOptions = append(tracingOptions, otlptracegrpc.WithInsecure())
		}

		ctx := context.Background()
		exporter, err = otlptracegrpc.New(ctx, tracingOptions...)
		if err != nil {
			return nil, err
		}
	}

	tracerProvider, err := newTraceProvider(exporter, opts)
	if err != nil {
		return nil, err
	}
	otel.SetTracerProvider(tracerProvider)

	var propagators []propagation.TextMapPropagator

	for _, p := range opts.Propagators {
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

	return tracerProvider, nil
}

func newTraceProvider(exporter sdktrace.SpanExporter, options config.TracingOptions) (*sdktrace.TracerProvider, error) {

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, errors.New("failed to read build info")
	}

	if options.ServiceName == "" {
		options.ServiceName = "bifrost"
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceName(options.ServiceName),
	}

	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			attrs = append(attrs, attribute.String("vcs.revision", setting.Value))
		case "vcs.time":
			attrs = append(attrs, attribute.String("vcs.time", setting.Value))
		}
	}

	res, err := resource.New(
		context.Background(),
		resource.WithFromEnv(), // Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		resource.WithProcessPID(),
		resource.WithProcessRuntimeVersion(),
		resource.WithOSType(),
		resource.WithContainer(),
		resource.WithHost(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(options.SamplingRate),
	)

	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exporter,
		sdktrace.WithMaxQueueSize(int(options.QueueSize)),
		sdktrace.WithMaxExportBatchSize(int(options.BatchSize)),
		sdktrace.WithBatchTimeout(options.Flush),
	)

	tracerOptions := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sampler),
		sdktrace.WithSpanProcessor(batchSpanProcessor),
		sdktrace.WithResource(res),
	}

	traceProvider := sdktrace.NewTracerProvider(tracerOptions...)
	return traceProvider, nil
}
