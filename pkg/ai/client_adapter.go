package ai

import (
	"fmt"
	"io"
)

// ClientAdapter defines the contract for translating between the client's SDK format
// and Bifrost's canonical internal AI formats.
type ClientAdapter interface {
	// Name returns the unique identifier for this client protocol (e.g., "openai-chat", "anthropic").
	Name() string

	// --- Ingress (Client -> Canonical) ---

	// ToChatRequest translates a raw client JSON body into a canonical ChatRequest.
	ToChatRequest(body []byte) (*ChatRequest, error)

	// ToResponsesRequest translates a raw client JSON body into a canonical ResponsesRequest.
	ToResponsesRequest(body []byte) (*ResponsesRequest, error)

	// --- Egress Unary (Canonical -> Client) ---

	// ToClientChatResponse translates a canonical ChatResponse back into the client's expected format.
	ToClientChatResponse(resp *ChatResponse) (any, error)

	// ToClientResponsesResponse translates a canonical ResponsesResponse back into the client's expected format.
	ToClientResponsesResponse(resp *ResponsesResponse) (any, error)

	// --- Egress Streaming (Canonical SSE -> Client SSE) ---

	// WrapEgressStream wraps a canonical SSE stream with a protocol-specific translator
	// to re-encode chunks into the format expected by the client SDK.
	WrapEgressStream(stream io.ReadCloser) io.ReadCloser

	// ToClientError translates a canonical AIError into the client's expected format.
	ToClientError(err *AIError) (any, error)
}

// ClientAdapterFactory is a function type that creates a specific ClientAdapter instance.
type ClientAdapterFactory func() ClientAdapter

var clientAdapterFactories = make(map[string]ClientAdapterFactory)

// RegisterClientAdapter registers a new client adapter factory.
func RegisterClientAdapter(name string, factory ClientAdapterFactory) {
	clientAdapterFactories[name] = factory
}

// GetClientAdapter retrieves a client adapter by its format name.
func GetClientAdapter(name string) (ClientAdapter, error) {
	factory, found := clientAdapterFactories[name]
	if !found {
		return nil, fmt.Errorf("ai: client adapter factory '%s' not found", name)
	}
	return factory(), nil
}
