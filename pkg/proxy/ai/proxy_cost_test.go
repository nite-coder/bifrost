package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/ai"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/target"
	"github.com/nite-coder/bifrost/pkg/telemetry/metrics"
)

func TestAIProxy_CostCalculation(t *testing.T) {
	mockLLMMu.Lock()
	defer mockLLMMu.Unlock()
	setupMockAdapter(t)

	aiOpts := &config.AIOptions{
		Providers: map[string]*config.AIProvider{
			"p1": {
				Handler: "mock",
				BaseURL: "http://localhost",
				APIKey:  "key",
			},
		},
	}

	pricing := &config.AIPricingOptions{
		InputPerMtok:  10.0, // $10 per 1M tokens
		OutputPerMtok: 20.0, // $20 per 1M tokens
	}

	p, err := NewProxy(ProxyOptions{
		ID:             "id1",
		Target:         "p1/gpt-4",
		AIOptions:      aiOpts,
		MetricsEnabled: true,
		Pricing:        pricing,
		Endpoint: &target.Endpoint{
			Address: "p1/gpt-4",
			Weight:  1,
			State:   target.NewState(0, 0),
		},
	})
	require.NoError(t, err)

	mockLL.chatFunc = func(_ context.Context, _ *ai.ChatRequest) (*ai.ChatResponse, error) {
		return &ai.ChatResponse{
			Usage: ai.Usage{
				PromptTokens:     1000000, // 1M tokens = $10
				CompletionTokens: 500000,  // 0.5M tokens = $10
				TotalTokens:      1500000,
			},
		}, nil
	}

	clientAdapter := &MockClientAdapter{
		toClientChatResponseFunc: func(_ *ai.ChatResponse) (any, error) {
			return map[string]any{}, nil
		},
	}

	hzCtx := app.NewContext(0)
	hzCtx.Set(ai.ContextKeyClientAdapter, clientAdapter)
	hzCtx.Set(ai.ContextKeyAIFamily, ai.FamilyChat)
	hzCtx.Set(ai.ContextKeyVirtualModelName, "gpt-4-virtual")
	hzCtx.Set(ai.ContextKeyChatRequest, &ai.ChatRequest{Model: "gpt-4-virtual"})

	metrics.AIRequestCost.Reset()

	p.ServeHTTP(context.Background(), hzCtx)

	assert.Equal(t, http.StatusOK, hzCtx.Response.StatusCode())
	// Total cost should be $10 (input) + $10 (output) = $20
	assert.InDelta(t, 20.0, getCounterValue(metrics.AIRequestCost, "gpt-4-virtual", "p1/gpt-4"), 0.0001)
}

func TestAIProxy_CostCalculation_Stream(t *testing.T) {
	mockLLMMu.Lock()
	defer mockLLMMu.Unlock()
	setupMockAdapter(t)

	aiOpts := &config.AIOptions{
		Providers: map[string]*config.AIProvider{
			"p1": {
				Handler: "mock",
				BaseURL: "http://localhost",
				APIKey:  "key",
			},
		},
	}

	pricing := &config.AIPricingOptions{
		InputPerMtok:  10.0, // $10 per 1M tokens
		OutputPerMtok: 20.0, // $20 per 1M tokens
	}

	p, err := NewProxy(ProxyOptions{
		ID:             "id1",
		Target:         "p1/gpt-4",
		AIOptions:      aiOpts,
		MetricsEnabled: true,
		Pricing:        pricing,
		Endpoint: &target.Endpoint{
			Address: "p1/gpt-4",
			Weight:  1,
			State:   target.NewState(0, 0),
		},
	})
	require.NoError(t, err)

	// Mock stream with usage at the end
	usage := ai.Usage{
		PromptTokens:     1000000,
		CompletionTokens: 500000,
		TotalTokens:      1500000,
	}
	usageJSON, _ := sonic.Marshal(usage)
	canonicalChunks := fmt.Sprintf(
		"data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: {\"usage\":%s}\n\ndata: [DONE]\n\n",
		usageJSON,
	)

	mockLL.streamChatFunc = func(_ context.Context, _ *ai.ChatRequest) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(canonicalChunks)), nil
	}

	clientAdapter := &MockClientAdapter{
		streamConverterFunc: func(stream io.ReadCloser) io.ReadCloser {
			return stream
		},
	}

	hzCtx := app.NewContext(0)
	hzCtx.Set(ai.ContextKeyClientAdapter, clientAdapter)
	hzCtx.Set(ai.ContextKeyAIFamily, ai.FamilyChat)
	hzCtx.Set(ai.ContextKeyVirtualModelName, "gpt-4-virtual")
	hzCtx.Set(ai.ContextKeyChatRequest, &ai.ChatRequest{Model: "gpt-4-virtual", Stream: true})

	metrics.AIRequestCost.Reset()

	p.ServeHTTP(context.Background(), hzCtx)

	assert.Equal(t, http.StatusOK, hzCtx.Response.StatusCode())
	// Total cost should be $10 (input) + $10 (output) = $20
	assert.InDelta(t, 20.0, getCounterValue(metrics.AIRequestCost, "gpt-4-virtual", "p1/gpt-4"), 0.0001)
}
