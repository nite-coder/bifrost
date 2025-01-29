package tracing

import (
	"context"
	"log/slog"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/tracing"
	"github.com/nite-coder/bifrost/pkg/variable"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

const traceIDKey = "trace_id"

func init() {
	_ = middleware.RegisterMiddleware("tracing", func(params map[string]any) (app.HandlerFunc, error) {
		opts := &Options{}

		config := &mapstructure.DecoderConfig{
			Metadata: nil,
			Result:   opts,
			TagName:  "mapstructure",
		}

		decoder, err := mapstructure.NewDecoder(config)
		if err != nil {
			return nil, err
		}

		if err := decoder.Decode(params); err != nil {
			return nil, err
		}

		m := NewMiddleware(opts)

		return m.ServeHTTP, nil
	})
}

type TracingMiddleware struct {
	options  *Options
	tracer   trace.Tracer
	hostname string
}

type Options struct {
	Attributes map[string]string `mapstructure:"attributes"`
}

func NewMiddleware(options *Options) *TracingMiddleware {
	hostname, _ := os.Hostname()

	return &TracingMiddleware{
		options:  options,
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

	defer func() {
		routeID := variable.GetString(variable.RouteID, c)
		httpRoute := variable.GetString(variable.HTTPRoute, c)

		if len(httpRoute) > 0 {
			span.SetName(reqMethod + " " + httpRoute)
		} else {
			span.SetName(reqMethod + " " + routeID)
		}

		// please refer to doc
		// https://github.com/open-telemetry/semantic-conventions/blob/v1.27.0/docs/http/http-spans.md#status

		labels := []attribute.KeyValue{
			semconv.HTTPRequestMethodKey.String(reqMethod),
			semconv.URLPath(reqPath),
			semconv.URLScheme(reqScheme),
		}

		for k, v := range m.options.Attributes {
			if k == "" {
				continue
			}

			val := variable.GetString(v, c)
			labels = append(labels, attribute.String(k, val))
		}

		span.SetAttributes(labels...)
		span.End()
	}()

	traceID := span.SpanContext().TraceID().String()

	if len(traceID) > 0 {
		// add trace_id to logger
		logger = logger.With(slog.String(traceIDKey, traceID))
		ctx = log.NewContext(ctx, logger)

		c.Set(traceIDKey, traceID)
	}

	c.Next(ctx)
}
