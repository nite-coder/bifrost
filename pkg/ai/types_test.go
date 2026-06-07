package ai

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatRequestPassthrough(t *testing.T) {
	inputJSON := []byte(`{
		"model": "gpt-4o",
		"stream": true,
		"temperature": 0.7,
		"custom_field_1": "hello",
		"custom_field_2": {
			"key": "value"
		}
	}`)

	var req ChatRequest
	err := sonic.Unmarshal(inputJSON, &req)
	require.NoError(t, err)

	// Verify known fields are parsed correctly
	assert.Equal(t, "gpt-4o", req.Model)
	assert.True(t, req.Stream)
	assert.NotNil(t, req.Temperature)
	assert.InDelta(t, 0.7, *req.Temperature, 0.0001)

	// Verify unknown fields are captured
	require.NotNil(t, req.UnknownFields)
	assert.Equal(t, "hello", req.UnknownFields["custom_field_1"])
	assert.Equal(t, map[string]any{"key": "value"}, req.UnknownFields["custom_field_2"])

	// Marshal back to JSON
	outputJSON, err := sonic.Marshal(&req)
	require.NoError(t, err)

	// Unmarshal outputJSON into a map to verify it is flattened
	var outputMap map[string]any
	err = sonic.Unmarshal(outputJSON, &outputMap)
	require.NoError(t, err)

	assert.Equal(t, "gpt-4o", outputMap["model"])
	assert.Equal(t, true, outputMap["stream"])
	assert.InDelta(t, 0.7, outputMap["temperature"], 0.0001)
	assert.Equal(t, "hello", outputMap["custom_field_1"])
	assert.Equal(t, map[string]any{"key": "value"}, outputMap["custom_field_2"])
}
