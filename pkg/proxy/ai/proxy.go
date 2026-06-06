package ai

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"

	coreai "github.com/nite-coder/bifrost/pkg/ai"
)

// Proxy implements the proxy.Proxy interface for AI LLM requests.
// It acts as the bridge between Bifrost's core routing/balancing logic
// and the specialized LLM adapters.
type Proxy struct {
	id         string
	target     string // Format: provider_id/actual-model-name
	weight     uint32
	adapter    coreai.LLMAdapter
	httpClient *client.Client
	observer   coreai.UsageObserver
}

// ID returns the unique identifier for this proxy target.
func (p *Proxy) ID() string {
	return p.id
}

// Target returns the backend model identifier.
func (p *Proxy) Target() string {
	return p.target
}

// Weight returns the load balancing weight.
func (p *Proxy) Weight() uint32 {
	return p.weight
}

// IsAvailable reports whether the provider is currently healthy.
func (p *Proxy) IsAvailable() bool {
	// Implementation will involve circuit breaker or health check status.
	return true
}

// AddFailedCount increments the failure metrics for this target.
func (p *Proxy) AddFailedCount(count uint) error {
	// Implementation will trigger circuit breaking if threshold is reached.
	return nil
}

// ServeHTTP handles the incoming LLM request by delegating to the adapter.
func (p *Proxy) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	// 1. Determine the API family (injected by ai_transformer)
	family := c.GetString(coreai.ContextKeyAIFamily)

	// 2. Prepare metadata using the original virtual model name
	virtualModel := c.GetString(coreai.ContextKeyVirtualModelName)
	metadata := coreai.UsageMetadata{
		Model:    virtualModel,
		Provider: p.id,
		// ... StartTime etc to be set here
	}

	// 3. Extract actual model name from p.target ("provider_id/actual-model-name")
	parts := strings.SplitN(p.target, "/", 2)
	if len(parts) != 2 {
		// Log error and return
		return
	}
	actualModel := parts[1]

	// 4. Branch based on family and handle request
	switch family {
	case coreai.FamilyChat:
		req := c.MustGet(coreai.ContextKeyChatRequest).(*coreai.ChatRequest)
		
		// 🚨 CRITICAL: Override the client's requested model with the actual backend model
		req.Model = actualModel

		if req.Stream {
			p.handleChatStream(ctx, c, req, metadata)
		} else {
			p.handleChatUnary(ctx, c, req, metadata)
		}
	case coreai.FamilyResponses:
		req := c.MustGet(coreai.ContextKeyResponsesRequest).(*coreai.ResponsesRequest)
		req.Model = actualModel
		// ... handle Responses family
	}
}

// handleChatUnary performs a standard request-response interaction.
func (p *Proxy) handleChatUnary(ctx context.Context, c *app.RequestContext, req *coreai.ChatRequest, meta coreai.UsageMetadata) {
	// - Call p.adapter.Chat()
	// - Call p.observer.OnUsage()
	// - Write JSON response
}

// handleChatStream performs a zero-buffered SSE interaction with mid-stream error handling.
func (p *Proxy) handleChatStream(ctx context.Context, c *app.RequestContext, req *coreai.ChatRequest, meta coreai.UsageMetadata) {
	// - Call p.adapter.StreamChat()
	// - c.Response.HijackWriter(...)
	
	// - 🚨 FIX 2.1: Mid-stream error handling loop
	// for {
	//     Read chunk -> error?
	//     if error {
	//         Write SSE error event: data: {"error":{...}}\n\n
	//         Write data: [DONE]\n\n
	//         return
	//     }
	//     Write chunk + Flush
	// }
}

// Tag returns metadata associated with this proxy.
func (p *Proxy) Tag(key string) (value string, exist bool) {
	return "", false
}

// Tags returns all metadata tags.
func (p *Proxy) Tags() map[string]string {
	return nil
}

// Close releases resources like idle connections in the HTTP client.
func (p *Proxy) Close() error {
	return nil
}
