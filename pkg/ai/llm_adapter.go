package ai

import (
	"context"
	"fmt"
	"io"

	"github.com/cloudwego/hertz/pkg/app/client"
)

// LLMAdapterOptions defines dependencies required to initialize an LLMAdapter.
type LLMAdapterOptions struct {
	HTTPClient *client.Client
	APIKey     string
	BaseURL    string
}

// AdapterFactory is a function type that creates a specific LLMAdapter instance.
type AdapterFactory func(opts LLMAdapterOptions) (LLMAdapter, error)

var adapterFactories = make(map[string]AdapterFactory)

// RegisterLLMAdapter registers a new adapter factory with a unique name.
func RegisterLLMAdapter(name string, factory AdapterFactory) {
	adapterFactories[name] = factory
}

// GetAdapter creates an instance of the requested adapter using the provided options.
func GetAdapter(name string, opts LLMAdapterOptions) (LLMAdapter, error) {
	factory, found := adapterFactories[name]
	if !found {
		return nil, fmt.Errorf("ai: adapter factory '%s' not found", name)
	}
	return factory(opts)
}

// LLMAdapter defines the functional contract for interacting with an LLM provider.
// It is a stateful object that encapsulates the HTTP client, API credentials,
// and base URL. Each adapter is responsible for bidirectional translation
// between the canonical format and the provider's native protocol.
type LLMAdapter interface {
	// Name returns the unique identifier for this adapter (e.g., "openai-chat", "anthropic").
	Name() string

	// --- Chat Completion (Stateless) ---

	// Chat executes a non-streaming chat completion request.
	// It handles the translation from canonical ChatRequest to native payload,
	// performs the network request, and translates the native response back to ChatResponse.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// StreamChat executes a streaming chat completion request.
	// It returns an io.ReadCloser that yields a stream of translated canonical SSE chunks.
	// The caller is responsible for closing the stream.
	StreamChat(ctx context.Context, req *ChatRequest) (io.ReadCloser, error)

	// --- Responses API (Stateful, Phase 1.5) ---

	// Responses executes a non-streaming OpenAI-compatible Responses API request.
	Responses(ctx context.Context, req *ResponsesRequest) (*ResponsesResponse, error)

	// StreamResponses executes a streaming OpenAI-compatible Responses API request.
	// It returns an io.ReadCloser yielding canonical SSE chunks for the Responses family.
	StreamResponses(ctx context.Context, req *ResponsesRequest) (io.ReadCloser, error)
}
