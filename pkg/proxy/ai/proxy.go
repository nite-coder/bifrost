package ai

import (
	"context"

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
	// family := c.GetString(coreai.ContextKeyAIFamily)
	// metadata := p.buildMetadata(c)

	// 2. Extract actual model name from p.target ("provider_id/actual-model-name")
	// parts := strings.SplitN(p.target, "/", 2)
	// actualModel := parts[1]

	// 3. Branch based on family and handle request
	// switch family {
	// case coreai.FamilyChat:
	//     req := c.MustGet(coreai.ContextKeyChatRequest).(*coreai.ChatRequest)
	//     
	//     // 🚨 CRITICAL: Override the client's requested model with the actual backend model
	//     // Example: "gpt-4o" (virtual) -> "claude-3-5-sonnet" (actual target)
	//     req.Model = actualModel
	//
	//     if req.Stream {
	//         p.handleChatStream(ctx, c, req, metadata)
	//     } else {
	//         p.handleChatUnary(ctx, c, req, metadata)
	//     }
	// case coreai.FamilyResponses:
	//     req := c.MustGet(coreai.ContextKeyResponsesRequest).(*coreai.ResponsesRequest)
	//     req.Model = actualModel
	//     // ... handle Responses family
	// }
}

// handleChatUnary performs a standard request-response interaction.
func (p *Proxy) handleChatUnary(ctx context.Context, c *app.RequestContext, req *coreai.ChatRequest, meta coreai.UsageMetadata) {
	// - Call p.adapter.Chat()
	// - Call p.observer.OnUsage()
	// - Write JSON response
}

// handleChatStream performs a zero-buffered SSE interaction using HijackWriter.
func (p *Proxy) handleChatStream(ctx context.Context, c *app.RequestContext, req *coreai.ChatRequest, meta coreai.UsageMetadata) {
	// - Call p.adapter.StreamChat()
	// - Set SSE headers
	// - c.Response.HijackWriter(...)
	// - Use coreai.NewObservedStream(stream, p.observer, meta)
	// - io.Copy to c.GetWriter()
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
