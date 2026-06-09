package ai

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUsageObserver struct {
	usages []Usage
	metas  []UsageMetadata
}

func (m *mockUsageObserver) OnUsage(_ context.Context, metadata UsageMetadata, usage Usage) {
	m.usages = append(m.usages, usage)
	m.metas = append(m.metas, metadata)
}

func TestObservedStream(t *testing.T) {
	rawData := []byte(
		"data: {\"id\":\"chunk1\",\"model\":\"gpt-4\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n" +
			"data: {\"id\":\"chunk2\",\"model\":\"gpt-4\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":10,\"total_tokens\":20}}\n\n" +
			"data: [DONE]\n\n",
	)

	// Read in different chunk sizes to simulate TCP segmentation
	chunkSizes := []int{5, 10, 50, 3, 200}

	observer := &mockUsageObserver{}
	metadata := UsageMetadata{
		Model:    "gpt-4",
		Provider: "openai",
	}

	rawStream := io.NopCloser(bytes.NewReader(rawData))
	observedStream := NewObservedStream(rawStream, observer, metadata)

	var output bytes.Buffer
	buf := make([]byte, 256)

	chunkIdx := 0
	for {
		size := chunkSizes[chunkIdx%len(chunkSizes)]
		chunkIdx++
		if size > len(buf) {
			size = len(buf)
		}

		n, err := observedStream.Read(buf[:size])
		if n > 0 {
			_, _ = output.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	// Verify all raw bytes are correctly returned without loss or corruption
	assert.Equal(t, rawData, output.Bytes())

	// Verify usage observer was called for each event containing usage
	require.Len(t, observer.usages, 2)
	assert.Equal(t, 10, observer.usages[0].PromptTokens)
	assert.Equal(t, 5, observer.usages[0].CompletionTokens)
	assert.Equal(t, 15, observer.usages[0].TotalTokens)
	assert.Equal(t, "gpt-4", observer.metas[0].Model)
	assert.Equal(t, "openai", observer.metas[0].Provider)

	assert.Equal(t, 10, observer.usages[1].PromptTokens)
	assert.Equal(t, 10, observer.usages[1].CompletionTokens)
	assert.Equal(t, 20, observer.usages[1].TotalTokens)
}
