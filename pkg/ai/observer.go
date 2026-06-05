package ai

import (
	"context"
	"io"
)

// UsageObserver defines the contract for components that monitor AI usage.
// It is triggered whenever usage data (tokens) is successfully captured
// from either a unary response or a completed stream.
type UsageObserver interface {
	// OnUsage is called with the usage metrics and associated metadata.
	// Context is provided for potential database or remote logging operations.
	OnUsage(ctx context.Context, metadata UsageMetadata, usage Usage)
}

// ObservedStream is a decorator for io.ReadCloser that intercepts SSE chunks
// to extract usage data before passing them to the client.
type ObservedStream struct {
	io.ReadCloser
	observer UsageObserver
	metadata UsageMetadata
}

// NewObservedStream creates a new stream decorator that notifies the observer
// when usage data is detected in the stream.
func NewObservedStream(stream io.ReadCloser, observer UsageObserver, metadata UsageMetadata) io.ReadCloser {
	if observer == nil {
		return stream
	}
	return &ObservedStream{
		ReadCloser: stream,
		observer:   observer,
		metadata:   metadata,
	}
}

// Read implements the io.Reader interface. It intercepts data chunks and
// parses them to find usage information.
func (s *ObservedStream) Read(p []byte) (n int, err error) {
	n, err = s.ReadCloser.Read(p)
	if n > 0 {
		// HLD: 🚨 CRITICAL BOUNDARY HANDLING
		// Network reads (p[:n]) are not guaranteed to align with SSE event boundaries (\n\n).
		// Implementation MUST use a buffer (e.g., bufio.Scanner or a byte slice accumulator)
		// to safely split chunks by "\n\n" before attempting JSON parsing.
		// If a complete JSON payload contains a "usage" field, s.observer.OnUsage(...) will be called.
	}
	return n, err
}
