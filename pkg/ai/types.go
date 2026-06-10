package ai

import (
	"encoding/json"
	"maps"

	"github.com/bytedance/sonic"

	"github.com/nite-coder/bifrost/internal/pkg/optional"
	"github.com/nite-coder/bifrost/pkg/config"
)

// --- Constants & Context Keys ---

const (
	// ContextKeyAIFamily is the context key for communicating the AI family between middleware and proxy.
	ContextKeyAIFamily = "ai_family"
	// ContextKeyChatRequest is the context key for the ChatRequest object.
	ContextKeyChatRequest = "ai_chat_request"
	// ContextKeyResponsesRequest is the context key for the ResponsesRequest object.
	ContextKeyResponsesRequest = "ai_responses_request"
	// ContextKeyClientAdapter is the context key for the client translator adapter.
	ContextKeyClientAdapter = "ai_client_adapter"
	// ContextKeyVirtualModelName is the context key for the original model name from the client.
	ContextKeyVirtualModelName = "ai_virtual_model_name"
	// ContextKeyChatResponse is the context key for the ChatResponse object.
	ContextKeyChatResponse = "ai_chat_response"
	// ContextKeyResponsesResponse is the context key for the ResponsesResponse object.
	ContextKeyResponsesResponse = "ai_responses_response"
	// ContextKeyResponseStream is the context key for the response stream.
	ContextKeyResponseStream = "ai_response_stream"

	// FamilyChat is the chat API family identifier.
	FamilyChat = "chat"
	// FamilyResponses is the responses API family identifier.
	FamilyResponses = "responses"
)

// --- Chat Request (Stateless) ---

// ChatRequest represents a canonical chat completion request,
// heavily aligned with the OpenAI Chat Completion API.
type ChatRequest struct {
	Model             string         `json:"model"`
	Messages          []Message      `json:"messages"`
	Stream            bool           `json:"stream,omitempty"`
	StreamOptions     *StreamOptions `json:"stream_options,omitempty"`
	Temperature       *float64       `json:"temperature,omitempty"`
	TopP              *float64       `json:"top_p,omitempty"`
	MaxTokens         *int           `json:"max_tokens,omitempty"`
	Stop              []string       `json:"stop,omitempty"`
	Tools             []Tool         `json:"tools,omitempty"`
	ToolChoice        any            `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool          `json:"parallel_tool_calls,omitempty"`
	Reasoning         *Reasoning     `json:"reasoning,omitempty"`
	ResponseFormat    any            `json:"response_format,omitempty"`
	UnknownFields     map[string]any `json:"-"`               // Collects unmapped fields for passthrough
	Extra             *ExtraOptions  `json:"extra,omitempty"` // Internal metadata
}

// UnmarshalJSON implements custom unmarshaling to capture unknown fields.
// IMPORTANT: When adding new fields to ChatRequest, add a corresponding delete(raw, "field_name") below.
func (r *ChatRequest) UnmarshalJSON(data []byte) error {
	type Alias ChatRequest
	var aux Alias
	if err := sonic.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = ChatRequest(aux)

	var raw map[string]any
	if err := sonic.Unmarshal(data, &raw); err != nil {
		return err
	}
	delete(raw, "model")
	delete(raw, "messages")
	delete(raw, "stream")
	delete(raw, "stream_options")
	delete(raw, "temperature")
	delete(raw, "top_p")
	delete(raw, "max_tokens")
	delete(raw, "stop")
	delete(raw, "tools")
	delete(raw, "tool_choice")
	delete(raw, "parallel_tool_calls")
	delete(raw, "reasoning")
	delete(raw, "response_format")
	delete(raw, "extra")

	r.UnknownFields = raw
	return nil
}

// MarshalJSON implements custom marshaling to flatten unknown fields.
func (r *ChatRequest) MarshalJSON() ([]byte, error) {
	type Alias ChatRequest
	aux := (Alias)(*r)

	b, err := sonic.Marshal(aux)
	if err != nil {
		return nil, err
	}

	if len(r.UnknownFields) == 0 {
		return b, nil
	}

	var m map[string]any
	if err := sonic.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	maps.Copy(m, r.UnknownFields)

	return sonic.Marshal(m)
}

// StreamOptions controls streaming behavior options.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// Reasoning configures reasoning behavior for models that support extended thinking (e.g., o1, o3, DeepSeek).
type Reasoning struct {
	Effort string `json:"effort,omitempty"` // "low", "medium", "high"
}

// Message represents a single turn in a conversation.
type Message struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"` // string OR []ContentPart for multi-modal
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // Required for "tool" role
}

// ContentPart represents a single part of a multi-modal message.
type ContentPart struct {
	Type     string    `json:"type"` // "text" or "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL contains the URL or base64 data for an image.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// Tool represents a functional capability the model can invoke.
type Tool struct {
	Type     string       `json:"type"` // Currently only "function"
	Function FunctionDesc `json:"function"`
}

// FunctionDesc describes a single tool function.
type FunctionDesc struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"` // JSON Schema
}

// ToolCall represents a specific invocation of a tool by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall contains the name and arguments of a tool invocation.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ExtraOptions carries Bifrost-specific metadata or passthrough fields.
type ExtraOptions struct {
	AIFamily string `json:"ai_family"` // "chat" or "responses"
}

// --- Chat Response ---

// ChatResponse represents a canonical chat completion response.
type ChatResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"` // "chat.completion"
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
}

// Choice represents one potential completion generated by the model.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"` // "stop", "length", "tool_calls", etc.
	Logprobs     any     `json:"logprobs,omitempty"`
}

// UsageMetadata carries business-related context for usage tracking.
// This allows passing user information, route details, and other
// metadata to observers without changing interface signatures.
type UsageMetadata struct {
	Model    string `json:"model"`    // Logical model name (e.g., "gpt-4o")
	UserID   string `json:"user_id"`  // User identifier for billing/quota
	RouteID  string `json:"route_id"` // Bifrost route ID
	Provider string `json:"provider"` // Target provider ID (e.g., "openai-official")
}

// Usage provides token consumption metrics, including extended details for reasoning and caching.
type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
	InputCost               float64                  `json:"input_cost,omitempty"`
	OutputCost              float64                  `json:"output_cost,omitempty"`
}

const (
	// TokensPerMillion is the number of tokens in a million.
	TokensPerMillion = 1000000.0
)

// CalculateCost calculates the input and output costs based on the pricing options.
func (u *Usage) CalculateCost(p *config.AIPricingOptions) {
	if p == nil {
		return
	}

	promptTokens := float64(u.PromptTokens)
	cachedTokens := 0.0
	if u.PromptTokensDetails != nil {
		cachedTokens = float64(u.PromptTokensDetails.CachedTokens)
	}

	if cachedTokens > 0 && p.CachedInputPerMtok > 0 {
		nonCachedTokens := promptTokens - cachedTokens
		if nonCachedTokens < 0 {
			nonCachedTokens = 0
		}
		u.InputCost = (nonCachedTokens / TokensPerMillion * p.InputPerMtok) + (cachedTokens / TokensPerMillion * p.CachedInputPerMtok)
	} else {
		u.InputCost = promptTokens / TokensPerMillion * p.InputPerMtok
	}

	u.OutputCost = float64(u.CompletionTokens) / TokensPerMillion * p.OutputPerMtok
}

// PromptTokensDetails holds extended input token breakdown (caching).
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// CompletionTokensDetails holds extended output token breakdown (reasoning).
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

// --- Streaming (SSE) ---

// StreamChunk represents a single chunk in a streaming chat response.
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"` // "chat.completion.chunk"
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// StreamChoice represents a delta update in a streaming response.
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        StreamDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// StreamDelta contains the incremental content or tool calls.
type StreamDelta struct {
	Role             string     `json:"role,omitempty"`
	Content          string     `json:"content,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"` // For reasoning models
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

// --- Responses API (Phase 1.5) ---

// ResponsesRequest placeholder for stateful Responses API.
type ResponsesRequest struct {
	Model        string    `json:"model"`
	Instructions string    `json:"instructions,omitempty"`
	Input        []Message `json:"input"`
}

// ResponsesResponse placeholder for Responses API response.
type ResponsesResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Model  string `json:"model"`
	Usage  Usage  `json:"usage"`
}

// --- Unified Error ---

// AIError defines a standardized error object for the AI Gateway.
type AIError struct { //nolint:revive
	Type       string                  `json:"type"`    // "invalid_request_error", "authentication_error", etc.
	Message    string                  `json:"message"` // Human-readable error message
	StatusCode int                     `json:"-"`       // HTTP status code
	Provider   string                  `json:"provider,omitempty"`
	Param      optional.Option[string] `json:"param"`
	Code       optional.Option[string] `json:"code"`
}

func (e *AIError) Error() string {
	return e.Message
}
