package tracing

import (
	"context"

	"github.com/cloudwego/hertz/pkg/protocol"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

var _ propagation.TextMapCarrier = &httpHeaderCarrier{}
var _ propagation.TextMapCarrier = &grpcMetadataCarrier{}

type httpHeaderCarrier struct {
	metadata map[string]string
	headers  *protocol.RequestHeader
}

type grpcMetadataCarrier struct {
	md metadata.MD
}

// Get a value from metadata by key
func (g *grpcMetadataCarrier) Get(key string) string {
	values := g.md.Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// Set a value to metadata by k/v
func (g *grpcMetadataCarrier) Set(key, value string) {
	g.md.Set(key, value)
}

// Keys Iteratively get all keys of metadata
func (g *grpcMetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(g.md))
	for k := range g.md {
		keys = append(keys, k)
	}
	return keys
}

// Get a value from metadata by key
func (m *httpHeaderCarrier) Get(key string) string {
	return m.headers.Get(key)
}

// Set a value to metadata by k/v
func (m *httpHeaderCarrier) Set(key, value string) {
	m.headers.Set(key, value)
}

// Keys Iteratively get all keys of metadata
func (m *httpHeaderCarrier) Keys() []string {
	out := make([]string, 0, len(m.metadata))

	m.headers.VisitAll(func(key, value []byte) {
		out = append(out, string(key))
	})

	return out
}

// InjectHTTPHeader injects span context into the hertz metadata info
func InjectHTTPHeader(ctx context.Context, headers *protocol.RequestHeader) {
	otel.GetTextMapPropagator().Inject(ctx, &httpHeaderCarrier{headers: headers})
}

// ExtractHTTPHeader returns the baggage and span context
func ExtractHTTPHeader(ctx context.Context, headers *protocol.RequestHeader) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, &httpHeaderCarrier{headers: headers})
}

// InjectGRPCMetadata injects span context into the grpc metadata
func InjectGRPCMetadata(ctx context.Context, md metadata.MD) {
	otel.GetTextMapPropagator().Inject(ctx, &grpcMetadataCarrier{md: md})
}

// ExtractGRPCMetadata returns the baggage and span context from grpc metadata
func ExtractGRPCMetadata(ctx context.Context, md metadata.MD) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, &grpcMetadataCarrier{md: md})
}
