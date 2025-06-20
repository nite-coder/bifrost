package gateway

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/tracing"
	"github.com/nite-coder/bifrost/pkg/variable"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	"go.opentelemetry.io/otel/trace"
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
			otlptracehttp.WithEndpointURL(opts.Endpoint),
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
		if setting.Key == "vcs.revision" {
			attrs = append(attrs, attribute.String("vcs.revision", setting.Value))
		}
	}

	res, err := resource.New(
		context.Background(),
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

type TracingMiddleware struct {
	options  *config.ServerTracingOptions
	tracer   trace.Tracer
	hostname string
}

func newTracingMiddleware(options config.ServerTracingOptions) *TracingMiddleware {
	hostname, _ := os.Hostname()

	return &TracingMiddleware{
		options:  &options,
		tracer:   otel.Tracer("bifrost"),
		hostname: hostname,
	}
}

func (m *TracingMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)

	if m.tracer == nil {
		c.Next(ctx)
		return
	}

	reqMethod := variable.GetString(variable.HTTPRequestMethod, c)

	spanOptions := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
	}

	ctx = tracing.Extract(ctx, &c.Request.Header)
	ctx, span := m.tracer.Start(ctx, "", spanOptions...)

	reqScheme := variable.GetString(variable.HTTPRequestScheme, c)
	reqPath := variable.GetString(variable.HTTPRequestPath, c)
	reqQuery := variable.GetString(variable.HTTPRequestQuery, c)

	defer func() {
		routeID := variable.GetString(variable.RouteID, c)
		httpRoute := variable.GetString(variable.HTTPRoute, c)
		errType := variable.GetString(variable.ErrorType, c)

		if len(httpRoute) > 0 {
			span.SetName(reqMethod + " " + httpRoute)
		} else {
			span.SetName(reqMethod + " " + routeID)
		}

		// please refer to doc
		// https://github.com/open-telemetry/semantic-conventions/blob/v1.32.0/docs/http/http-spans.md

		labels := []attribute.KeyValue{
			semconv.HTTPRequestMethodKey.String(reqMethod),
			semconv.URLPath(reqPath),
			semconv.URLScheme(reqScheme),
		}

		if len(httpRoute) > 0 {
			labels = append(labels, semconv.HTTPRoute(httpRoute))
		}

		if len(reqQuery) > 0 {
			labels = append(labels, semconv.URLQuery(reqQuery))
		}

		if len(errType) > 0 {
			labels = append(labels, semconv.ErrorTypeKey.String(errType))
		}

		if c.Response.StatusCode() > 0 {
			labels = append(labels, semconv.HTTPResponseStatusCode(c.Response.StatusCode()))
		}

		if c.Response.StatusCode() >= 500 {
			span.SetStatus(codes.Error, "")
		}

		if c.GetBool(variable.TargetTimeout) {
			span.SetStatus(codes.Error, "timeout")
		}

		for k, v := range m.options.Attributes {
			if k == "" || v == "" {
				continue
			}

			val := variable.GetString(v, c)

			if len(val) == 0 && variable.IsDirective(v) {
				continue
			}

			labels = append(labels, attribute.String(k, val))
		}

		span.SetAttributes(labels...)
		span.End()
	}()

	traceID := span.SpanContext().TraceID().String()

	if len(traceID) > 0 {
		// add trace_id to logger
		logger = logger.With(slog.String("trace_id", traceID))
		ctx = log.NewContext(ctx, logger)

		c.Set(variable.TraceID, traceID)
	}

	c.Next(ctx)
}
