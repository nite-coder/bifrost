package ai

import (
	"bytes"
	"context"
	"encoding/json"
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
	buf      []byte
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
		s.buf = append(s.buf, p[:n]...)
		for {
			idx := bytes.Index(s.buf, []byte("\n\n"))
			if idx == -1 {
				break
			}
			event := s.buf[:idx]
			s.buf = s.buf[idx+2:]
			s.processEvent(event)
		}
	}
	return n, err
}

func (s *ObservedStream) processEvent(event []byte) {
	lines := bytes.SplitSeq(event, []byte("\n"))
	for line := range lines {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			continue
		}

		var chunk StreamChunk
		if err := json.Unmarshal(data, &chunk); err == nil && chunk.Usage != nil {
			s.observer.OnUsage(context.Background(), s.metadata, *chunk.Usage)
		}
	}
}
