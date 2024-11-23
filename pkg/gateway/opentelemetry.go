package gateway

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTracerProvider(opts config.TracingOptions) (*sdktrace.TracerProvider, error) {
	if !opts.OTLP.Enabled {
		return nil, nil
	}

	if opts.OTLP.Endpoint == "" {
		// default we use grpc
		opts.OTLP.Endpoint = "localhost:4317"
	}

	if opts.OTLP.BatchSize <= 0 {
		opts.OTLP.BatchSize = 100
	}

	if opts.OTLP.Flush.Seconds() <= 0 {
		opts.OTLP.Flush = 5 * time.Second
	}

	if opts.OTLP.QueueSize <= 0 {
		opts.OTLP.QueueSize = 10000
	}

	if opts.OTLP.Timeout.Seconds() <= 0 {
		opts.OTLP.Timeout = 10 * time.Second
	}

	addr, err := url.Parse(opts.OTLP.Endpoint)
	if err != nil {
		return nil, err
	}

	addr.Scheme = strings.ToLower(addr.Scheme)

	var exporter sdktrace.SpanExporter

	if strings.EqualFold(addr.Scheme, "https") || strings.EqualFold(addr.Scheme, "http") {
		tracingOptions := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(opts.OTLP.Endpoint),
			otlptracehttp.WithTimeout(opts.OTLP.Timeout),
		}

		if opts.OTLP.Insecure {
			tracingOptions = append(tracingOptions, otlptracehttp.WithInsecure())
		}

		exporter, err = otlptracehttp.New(context.Background(), tracingOptions...)
		if err != nil {
			return nil, err
		}
	} else {
		// grpc
		tracingOptions := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(opts.OTLP.Endpoint),
			otlptracegrpc.WithTimeout(opts.OTLP.Timeout),
		}

		if opts.OTLP.Insecure {
			tracingOptions = append(tracingOptions, otlptracegrpc.WithInsecure())
		}

		ctx := context.Background()
		exporter, err = otlptracegrpc.New(ctx, tracingOptions...)
		if err != nil {
			return nil, err
		}
	}

	tracerProvider := newTraceProvider(exporter, opts.OTLP)
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

	return tracerProvider, nil
}

func newTraceProvider(exporter sdktrace.SpanExporter, options config.OTLPOptions) *sdktrace.TracerProvider {

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
	}

	traceProvider := sdktrace.NewTracerProvider(tracerOptions...)
	return traceProvider
}
