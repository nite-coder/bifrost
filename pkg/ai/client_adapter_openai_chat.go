package ai

import (
	"io"

	"github.com/bytedance/sonic"

	"github.com/nite-coder/bifrost/internal/pkg/optional"
)

func init() {
	RegisterClientAdapter("openai-chat", func() ClientAdapter {
		return NewOpenAIChatClientAdapter()
	})
}

// OpenAIChatClientAdapter implements ClientAdapter for the OpenAI chat protocol.
type OpenAIChatClientAdapter struct{}

// NewOpenAIChatClientAdapter creates a new OpenAIChatClientAdapter instance.
func NewOpenAIChatClientAdapter() *OpenAIChatClientAdapter {
	return &OpenAIChatClientAdapter{}
}

// Name returns the client protocol name.
func (a *OpenAIChatClientAdapter) Name() string {
	return "openai-chat"
}

// ToChatRequest translates raw client JSON body into a canonical ChatRequest.
func (a *OpenAIChatClientAdapter) ToChatRequest(body []byte) (*ChatRequest, error) {
	var req ChatRequest
	if err := sonic.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// ToResponsesRequest translates raw client JSON body into a canonical ResponsesRequest.
func (a *OpenAIChatClientAdapter) ToResponsesRequest(body []byte) (*ResponsesRequest, error) {
	var req ResponsesRequest
	if err := sonic.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// ToClientChatResponse translates canonical ChatResponse back to client format.
func (a *OpenAIChatClientAdapter) ToClientChatResponse(resp *ChatResponse) (any, error) {
	return resp, nil
}

// ToClientResponsesResponse translates canonical ResponsesResponse back to client format.
func (a *OpenAIChatClientAdapter) ToClientResponsesResponse(resp *ResponsesResponse) (any, error) {
	return resp, nil
}

// StreamConverter returns the stream unchanged, since the canonical SSE stream already uses OpenAI's format.
func (a *OpenAIChatClientAdapter) StreamConverter(stream io.ReadCloser) io.ReadCloser {
	return stream
}

// OpenAIErrorResponse represents the standard error response returned by OpenAI APIs.
type OpenAIErrorResponse struct {
	Error OpenAIErrorDetail `json:"error"`
}

// OpenAIErrorDetail represents details of the OpenAI error.
type OpenAIErrorDetail struct {
	Message string                  `json:"message"`
	Type    string                  `json:"type"`
	Param   optional.Option[string] `json:"param"`
	Code    optional.Option[string] `json:"code"`
}

// ToClientError translates a canonical AIError into the client's expected format.
func (a *OpenAIChatClientAdapter) ToClientError(err *AIError) (any, error) {
	return &OpenAIErrorResponse{
		Error: OpenAIErrorDetail{
			Message: err.Message,
			Type:    err.Type,
			Param:   err.Param,
			Code:    err.Code,
		},
	}, nil
}
