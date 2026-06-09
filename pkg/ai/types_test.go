package ai

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/config"
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

func TestUsage_CalculateCost(t *testing.T) {
	p := &config.AIPricingOptions{
		InputPerMtok:       2.0,
		OutputPerMtok:      10.0,
		CachedInputPerMtok: 1.0,
	}
	u := &Usage{
		PromptTokens:     1000000,
		CompletionTokens: 1000000,
		PromptTokensDetails: &PromptTokensDetails{
			CachedTokens: 500000,
		},
	}
	u.CalculateCost(p)
	if u.InputCost != 1.5 { // (0.5 * 2) + (0.5 * 1)
		t.Errorf("expected input cost 1.5, got %f", u.InputCost)
	}
	if u.OutputCost != 10.0 {
		t.Errorf("expected output cost 10.0, got %f", u.OutputCost)
	}
}
