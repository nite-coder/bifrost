package tracing

import (
	"context"
	"log/slog"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/tracing"
	"github.com/nite-coder/bifrost/pkg/variable"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

const traceIDKey = "trace_id"

func init() {
	_ = middleware.RegisterMiddleware("tracing", func(params map[string]any) (app.HandlerFunc, error) {
		m := NewMiddleware()
		return m.ServeHTTP, nil
	})
}

type TracingMiddleware struct {
	tracer trace.Tracer
}

func NewMiddleware() *TracingMiddleware {
	return &TracingMiddleware{
		tracer: otel.Tracer("bifrost"),
	}
}

func (m *TracingMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)

	if m.tracer == nil {
		c.Next(ctx)
		return
	}

	hostname, _ := os.Hostname()

	reqMethod := variable.GetString(variable.HTTPRequestMethod, c)

	spanOptions := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
	}

	ctx = tracing.Extract(ctx, &c.Request.Header)
	ctx, span := m.tracer.Start(ctx, "", spanOptions...)

	httpHost := variable.GetString(variable.HTTPRequestHost, c)
	reqScheme := variable.GetString(variable.HTTPRequestScheme, c)
	reqPath := variable.GetString(variable.HTTPRequestPath, c)
	reqQuery := variable.GetString(variable.HTTPRequestQuery, c)
	networkProtocol := variable.GetString(variable.HTTPRequestProtocol, c)
	clientIP := c.ClientIP()

	defer func() {
		serverID := variable.GetString(variable.ServerID, c)
		routeID := variable.GetString(variable.RouteID, c)
		serviceID := variable.GetString(variable.ServiceID, c)
		remoteAddr := variable.GetString(variable.NetworkPeerAddress, c)

		span.SetName(reqMethod + " " + routeID)

		// please refer to doc
		// https://github.com/open-telemetry/semantic-conventions/blob/v1.27.0/docs/http/http-spans.md#status

		labels := []attribute.KeyValue{
			attribute.String("bifrost.server_id", serverID),
			attribute.String("bifrost.route_id", routeID),
			attribute.String("bifrost.service_id", serviceID),
			semconv.HTTPRoute(routeID),
			semconv.HTTPRequestMethodOriginal(reqMethod),
			semconv.ServerAddress(httpHost),
			semconv.URLScheme(reqScheme),
			semconv.URLPath(reqPath),
			semconv.URLQuery(reqQuery),
			semconv.NetworkProtocolName(networkProtocol),
			semconv.ClientAddress(clientIP),
			semconv.NetworkLocalAddress(hostname),
			semconv.NetworkPeerAddress(remoteAddr),
			semconv.HTTPResponseStatusCode(c.Response.StatusCode()),
		}

		if c.Response.StatusCode() >= 500 {
			span.SetStatus(codes.Error, string(c.Response.Body()))
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
