package ai

import (
	"context"
	"errors"
	"io"

	"github.com/cloudwego/hertz/pkg/app/client"
)

func init() {
	RegisterLLMAdapter("anthropic", func(opts LLMAdapterOptions) (LLMAdapter, error) {
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

// Name returns the name of the adapter.
func (a *AnthropicAdapter) Name() string { return "anthropic" }

// Chat sends a unary chat completion request to Anthropic.
func (a *AnthropicAdapter) Chat(_ context.Context, _ *ChatRequest) (*ChatResponse, error) {
	// 1. Translate ChatRequest to Anthropic JSON (extract system msg, handle tool_use)
	// 2. a.client.Do(ctx, req, resp) with Anthropic-Version header
	// 3. Translate Anthropic JSON back to ChatResponse
	return nil, errors.New("not implemented")
}

// StreamChat sends a streaming chat completion request to Anthropic.
func (a *AnthropicAdapter) StreamChat(_ context.Context, _ *ChatRequest) (io.ReadCloser, error) {
	// 1. Translate ChatRequest to Anthropic JSON
	// 2. a.client.Do(ctx, req, resp)
	// 3. Use an internal translator reader to map Anthropic SSE events to Canonical chunks
	return nil, errors.New("not implemented")
}

// Responses sends a batch responses request to Anthropic.
func (a *AnthropicAdapter) Responses(_ context.Context, _ *ResponsesRequest) (*ResponsesResponse, error) {
	return nil, errors.New("not implemented")
}

// StreamResponses sends a streaming responses request to Anthropic.
func (a *AnthropicAdapter) StreamResponses(_ context.Context, _ *ResponsesRequest) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}
