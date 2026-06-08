package ai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
)

func init() {
	RegisterLLMAdapter("openai-chat", func(opts LLMAdapterOptions) (LLMAdapter, error) {
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

// Name returns the name of the adapter.
func (a *OpenAIChatAdapter) Name() string { return "openai-chat" }

// Chat sends a unary chat completion request to OpenAI.
func (a *OpenAIChatAdapter) Chat(ctx context.Context, chatReq *ChatRequest) (*ChatResponse, error) {
	req := protocol.AcquireRequest()
	defer protocol.ReleaseRequest(req)
	resp := protocol.AcquireResponse()
	defer protocol.ReleaseResponse(resp)

	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI(a.baseURL + "/chat/completions")
	req.Header.SetContentTypeBytes([]byte("application/json"))
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	body, err := sonic.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("openai-chat: failed to marshal request: %w", err)
	}
	req.SetBody(body)

	err = a.client.Do(ctx, req, resp)
	if err != nil {
		return nil, fmt.Errorf("openai-chat: request failed: %w", err)
	}

	var respBody []byte
	if resp.IsBodyStream() {
		respBody, err = io.ReadAll(resp.BodyStream())
		if err != nil {
			return nil, fmt.Errorf("openai-chat: failed to read response body stream: %w", err)
		}
	} else {
		respBody = resp.Body()
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, parseError(resp.StatusCode(), respBody)
	}

	var chatResp ChatResponse
	if err := sonic.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("openai-chat: failed to unmarshal response: %w", err)
	}

	return &chatResp, nil
}

// responseStreamCloser wraps a stream reader and handles close operations safely.
type responseStreamCloser struct {
	reader io.Reader
	req    *protocol.Request
	resp   *protocol.Response
}

func (r *responseStreamCloser) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *responseStreamCloser) Close() error {
	var err error
	if closer, ok := r.reader.(io.ReadCloser); ok {
		err = closer.Close()
	}
	if r.req != nil {
		r.req.Reset()
	}
	if r.resp != nil {
		r.resp.Reset()
	}
	return err
}

// StreamChat sends a streaming chat completion request to OpenAI.
func (a *OpenAIChatAdapter) StreamChat(ctx context.Context, chatReq *ChatRequest) (io.ReadCloser, error) {
	// We allocate directly to avoid pool reuse issues since the response is read asynchronously.
	req := &protocol.Request{}
	resp := &protocol.Response{}

	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI(a.baseURL + "/chat/completions")
	req.Header.SetContentTypeBytes([]byte("application/json"))

	if len(a.apiKey) > 0 {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	body, err := sonic.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("openai-chat: failed to marshal request: %w", err)
	}
	req.SetBody(body)

	err = a.client.Do(ctx, req, resp)
	if err != nil {
		return nil, fmt.Errorf("openai-chat: request failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		var respBody []byte
		if resp.IsBodyStream() {
			respBody, _ = io.ReadAll(resp.BodyStream())
		} else {
			respBody = resp.Body()
		}
		return nil, parseError(resp.StatusCode(), respBody)
	}

	if resp.IsBodyStream() {
		return &responseStreamCloser{
			reader: resp.BodyStream(),
			req:    req,
			resp:   resp,
		}, nil
	}

	bodyBytes := resp.Body()
	req.Reset()
	resp.Reset()
	return io.NopCloser(bytes.NewReader(bodyBytes)), nil
}

// Responses sends a batch responses request to OpenAI.
func (a *OpenAIChatAdapter) Responses(_ context.Context, _ *ResponsesRequest) (*ResponsesResponse, error) {
	return nil, &AIError{
		Type:       "invalid_request_error",
		Message:    "Responses API is not supported by openai-chat adapter",
		StatusCode: http.StatusNotImplemented,
		Provider:   "openai-chat",
	}
}

// StreamResponses sends a streaming responses request to OpenAI.
func (a *OpenAIChatAdapter) StreamResponses(_ context.Context, _ *ResponsesRequest) (io.ReadCloser, error) {
	return nil, &AIError{
		Type:       "invalid_request_error",
		Message:    "Responses API streaming is not supported by openai-chat adapter",
		StatusCode: http.StatusNotImplemented,
		Provider:   "openai-chat",
	}
}

type openAIErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

type openAIErrorResponse struct {
	Error openAIErrorDetail `json:"error"`
}

func parseError(statusCode int, body []byte) error {
	var errResp openAIErrorResponse

	if err := sonic.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return &AIError{
			Type:       errResp.Error.Type,
			Message:    errResp.Error.Message,
			StatusCode: statusCode,
			Provider:   "openai-chat",
			Param:      errResp.Error.Param,
			Code:       errResp.Error.Code,
		}
	}

	// SECURITY: Log full upstream error details internally, return generic error to prevent leakage
	slog.ErrorContext(context.Background(), "upstream returned non-standard error",
		"status_code", statusCode,
		"body", string(body),
		"provider", "openai-chat",
	)

	return fmt.Errorf("upstream error: status %d", statusCode)
}
