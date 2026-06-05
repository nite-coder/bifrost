package ai

import (
	"context"
	"io"

	"github.com/cloudwego/hertz/pkg/app/client"
)

func init() {
	RegisterAdapter("anthropic", func(opts LLMAdapterOptions) (LLMAdapter, error) {
		return NewAnthropicAdapter(opts), nil
	})
}

// AnthropicAdapter implements LLMAdapter for Anthropic's Messages API.
type AnthropicAdapter struct {
	client  *client.Client
	apiKey  string
	baseURL string
}

// NewAnthropicAdapter creates a new instance of AnthropicAdapter.
func NewAnthropicAdapter(opts LLMAdapterOptions) *AnthropicAdapter {
	return &AnthropicAdapter{
		client:  opts.HTTPClient,
		apiKey:  opts.APIKey,
		baseURL: opts.BaseURL,
	}
}

func (a *AnthropicAdapter) Name() string { return "anthropic" }

func (a *AnthropicAdapter) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 1. Translate ChatRequest to Anthropic JSON (extract system msg, handle tool_use)
	// 2. a.client.Do(ctx, req, resp) with Anthropic-Version header
	// 3. Translate Anthropic JSON back to ChatResponse
	return nil, nil
}

func (a *AnthropicAdapter) StreamChat(ctx context.Context, req *ChatRequest) (io.ReadCloser, error) {
	// 1. Translate ChatRequest to Anthropic JSON
	// 2. a.client.Do(ctx, req, resp)
	// 3. Use an internal translator reader to map Anthropic SSE events to Canonical chunks
	return nil, nil
}

func (a *AnthropicAdapter) Responses(ctx context.Context, req *ResponsesRequest) (*ResponsesResponse, error) {
	return nil, nil
}

func (a *AnthropicAdapter) StreamResponses(ctx context.Context, req *ResponsesRequest) (io.ReadCloser, error) {
	return nil, nil
}
