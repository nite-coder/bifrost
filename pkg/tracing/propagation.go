package tracing

import (
	"context"

	"github.com/cloudwego/hertz/pkg/protocol"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var _ propagation.TextMapCarrier = &metadataProvider{}

type metadataProvider struct {
	metadata map[string]string
	headers  *protocol.RequestHeader
}

// Get a value from metadata by key
func (m *metadataProvider) Get(key string) string {
	return m.headers.Get(key)
}

// Set a value to metadata by k/v
func (m *metadataProvider) Set(key, value string) {
	m.headers.Set(key, value)
}

// Keys Iteratively get all keys of metadata
func (m *metadataProvider) Keys() []string {
	out := make([]string, 0, len(m.metadata))

	m.headers.VisitAll(func(key, value []byte) {
		out = append(out, string(key))
	})

	return out
}

// Inject injects span context into the hertz metadata info
func Inject(ctx context.Context, headers *protocol.RequestHeader) {
	otel.GetTextMapPropagator().Inject(ctx, &metadataProvider{headers: headers})
}

// Extract returns the baggage and span context
func Extract(ctx context.Context, headers *protocol.RequestHeader) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, &metadataProvider{headers: headers})
}
