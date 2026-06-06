package ai

import (
	"context"
	"fmt"
	"io"

	"github.com/cloudwego/hertz/pkg/app/client"
)

func init() {
	RegisterAdapter("openai-chat", func(opts LLMAdapterOptions) (LLMAdapter, error) {
		return NewOpenAIChatAdapter(opts), nil
	})
}

// OpenAIChatAdapter implements LLMAdapter for OpenAI's Chat Completions API.
type OpenAIChatAdapter struct {
	client  *client.Client
	apiKey  string
	baseURL string
}

// NewOpenAIChatAdapter creates a new instance of OpenAIChatAdapter.
func NewOpenAIChatAdapter(opts LLMAdapterOptions) *OpenAIChatAdapter {
	return &OpenAIChatAdapter{
		client:  opts.HTTPClient,
		apiKey:  opts.APIKey,
		baseURL: opts.BaseURL,
	}
}

func (a *OpenAIChatAdapter) Name() string { return "openai-chat" }

func (a *OpenAIChatAdapter) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 1. Translate ChatRequest to OpenAI JSON
	// 2. a.client.Do(ctx, req, resp)
	// 3. Translate OpenAI JSON back to ChatResponse
	return nil, fmt.Errorf("not implemented")
}

func (a *OpenAIChatAdapter) StreamChat(ctx context.Context, req *ChatRequest) (io.ReadCloser, error) {
	// 1. Translate ChatRequest to OpenAI JSON
	// 2. a.client.Do(ctx, req, resp) with ResponseBodyStream=true
	// 3. Return a reader that yields canonical SSE chunks
	return nil, fmt.Errorf("not implemented")
}

func (a *OpenAIChatAdapter) Responses(ctx context.Context, req *ResponsesRequest) (*ResponsesResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (a *OpenAIChatAdapter) StreamResponses(ctx context.Context, req *ResponsesRequest) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}
