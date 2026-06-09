package ai

import (
	"bytes"
	"context"
	"io"

	"github.com/bytedance/sonic"
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
		buf:        nil,
	}
}

// Read implements the io.Reader interface. It intercepts data chunks and
// parses them to find usage information.
func (s *ObservedStream) Read(p []byte) (n int, err error) {
	const eventDelimiterLen = 2
	n, err = s.ReadCloser.Read(p)
	if n > 0 {
		s.buf = append(s.buf, p[:n]...)
		for {
			idx := bytes.Index(s.buf, []byte("\n\n"))
			if idx == -1 {
				break
			}
			event := s.buf[:idx]
			s.processEvent(event)
			nextIdx := idx + eventDelimiterLen
			copy(s.buf, s.buf[nextIdx:])
			s.buf = s.buf[:len(s.buf)-nextIdx]
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
		if err := sonic.Unmarshal(data, &chunk); err == nil && chunk.Usage != nil {
			s.observer.OnUsage(context.Background(), s.metadata, *chunk.Usage)
		}
	}
}
