package tracing

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

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

		m := NewMiddleware(*opts)
		return m.ServeHTTP, nil
	})
}

type TracingMiddleware struct {
	tracer  trace.Tracer
	options *Options
}

type Options struct {
	ResponseHeader string `mapstructure:"response_header"`
}

func NewMiddleware(options Options) *TracingMiddleware {
	return &TracingMiddleware{
		options: &options,
		tracer:  otel.Tracer("bifrost"),
	}
}

func (m *TracingMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	if m.tracer == nil {
		c.Next(ctx)
		return
	}

	method := string(c.Method())
	path := string(c.Request.Path())

	spanOptions := []trace.SpanStartOption{
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithNewRoot(),
	}

	ctx, span := m.tracer.Start(ctx, method+" "+path, spanOptions...)

	defer func() {
		serverID := variable.GetString(variable.ServerID, c)
		routeID := variable.GetString(variable.RouteID, c)
		serviceID := variable.GetString(variable.ServiceID, c)
		remoteAddr := variable.GetString(variable.RemoteAddr, c)

		span.SetName(routeID)

		labels := []attribute.KeyValue{
			attribute.String("server_id", serverID),
			attribute.String("route_id", routeID),
			attribute.String("service_id", serviceID),
			attribute.String("http.scheme", string(c.Request.Scheme())),
			attribute.String("http.host", string(c.Request.Host())),
			attribute.String("http.method", method),
			attribute.String("http.path", path),
			attribute.String("http.user_agent", string(c.Request.Header.UserAgent())),
			attribute.String("protocol", c.Request.Header.GetProtocol()),
			attribute.String("client_ip", c.ClientIP()),
			attribute.String("remote_addr", remoteAddr),
			attribute.Int("http.status", c.Response.StatusCode()),
		}

		if c.Response.StatusCode() >= 200 && c.Response.StatusCode() < 300 {
			span.SetStatus(codes.Ok, "")
		} else if c.Response.StatusCode() >= 400 {
			span.SetStatus(codes.Error, "")
		}

		span.SetAttributes(labels...)
		span.End()
	}()

	traceID := span.SpanContext().TraceID()
	c.Set(variable.TraceID, traceID.String())

	c.Next(ctx)

	if m.options.ResponseHeader != "" {
		c.Response.Header.Set(m.options.ResponseHeader, traceID.String())
	}
}
