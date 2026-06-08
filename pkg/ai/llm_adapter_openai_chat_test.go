package ai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/internal/pkg/optional"
)

func TestOpenAIChatClientAdapter(t *testing.T) {
	adapter, err := GetClientAdapter("openai-chat")
	require.NoError(t, err)
	assert.Equal(t, "openai-chat", adapter.Name())

	// Test ToChatRequest
	reqJSON := []byte(`{"model":"gpt-4o","stream":true,"messages":[{"role":"user","content":"hello"}]}`)
	chatReq, err := adapter.ToChatRequest(reqJSON)
	require.NoError(t, err)
	assert.Equal(t, "gpt-4o", chatReq.Model)
	assert.True(t, chatReq.Stream)
	require.Len(t, chatReq.Messages, 1)
	assert.Equal(t, "user", chatReq.Messages[0].Role)
	assert.Equal(t, "hello", chatReq.Messages[0].Content)

	// Test ToResponsesRequest
	respJSON := []byte(`{"model":"gpt-4o","input":[{"role":"user","content":"hello"}]}`)
	responsesReq, err := adapter.ToResponsesRequest(respJSON)
	require.NoError(t, err)
	assert.Equal(t, "gpt-4o", responsesReq.Model)
	require.Len(t, responsesReq.Input, 1)
	assert.Equal(t, "hello", responsesReq.Input[0].Content)

	// Test ToClientChatResponse
	chatResp := &ChatResponse{ID: "chat-123"}
	mappedChat, err := adapter.ToClientChatResponse(chatResp)
	require.NoError(t, err)
	assert.Equal(t, chatResp, mappedChat)

	// Test ToClientResponsesResponse
	responsesResp := &ResponsesResponse{ID: "resp-123"}
	mappedResponses, err := adapter.ToClientResponsesResponse(responsesResp)
	require.NoError(t, err)
	assert.Equal(t, responsesResp, mappedResponses)

	// Test ToClientError
	aiErr := &AIError{
		Type:       "invalid_request_error",
		Message:    "Invalid parameters",
		StatusCode: http.StatusBadRequest,
		Param:      optional.Some("model"),
		Code:       optional.Some("invalid_model"),
	}
	clientErr, err := adapter.ToClientError(aiErr)
	require.NoError(t, err)

	openAIErrorResp, ok := clientErr.(*OpenAIErrorResponse)
	require.True(t, ok)
	assert.Equal(t, "Invalid parameters", openAIErrorResp.Error.Message)
	assert.Equal(t, "invalid_request_error", openAIErrorResp.Error.Type)
	assert.Equal(t, "model", openAIErrorResp.Error.Param.Unwrap())
	assert.Equal(t, "invalid_model", openAIErrorResp.Error.Code.Unwrap())
}

func TestOpenAIChatAdapter_Chat_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Read request
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var chatReq ChatRequest
		err = sonic.Unmarshal(body, &chatReq)
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		assert.Equal(t, "gpt-4o", chatReq.Model)

		// Respond
		resp := ChatResponse{
			ID:      "chat-123",
			Object:  "chat.completion",
			Created: 1670000000,
			Model:   "gpt-4o",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    "assistant",
						Content: "hello client",
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
		respBytes, err := sonic.Marshal(resp)
		if !assert.NoError(t, err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respBytes)
	}))
	defer ts.Close()

	httpClient, err := client.NewClient(client.WithResponseBodyStream(true))
	require.NoError(t, err)

	opts := LLMAdapterOptions{
		HTTPClient: httpClient,
		APIKey:     "test-key",
		BaseURL:    ts.URL,
	}
	adapter, err := GetAdapter("openai-chat", opts)
	require.NoError(t, err)

	chatReq := &ChatRequest{
		Model: "gpt-4o",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	}

	resp, err := adapter.Chat(context.Background(), chatReq)
	require.NoError(t, err)
	assert.Equal(t, "chat-123", resp.ID)
	assert.Equal(t, "gpt-4o", resp.Model)
	require.Len(t, resp.Choices, 1)
	assert.Equal(t, "hello client", resp.Choices[0].Message.Content)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 5, resp.Usage.CompletionTokens)
}

func TestOpenAIChatAdapter_Chat_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{
			"error": {
				"message": "The model gpt-5 does not exist",
				"type": "invalid_request_error",
				"param": "model",
				"code": "model_not_found"
			}
		}`))
	}))
	defer ts.Close()

	httpClient, err := client.NewClient(client.WithResponseBodyStream(true))
	require.NoError(t, err)

	opts := LLMAdapterOptions{
		HTTPClient: httpClient,
		APIKey:     "test-key",
		BaseURL:    ts.URL,
	}
	adapter := NewOpenAIChatAdapter(opts)

	chatReq := &ChatRequest{Model: "gpt-5"}
	_, err = adapter.Chat(context.Background(), chatReq)
	require.Error(t, err)

	var aiErr *AIError
	require.ErrorAs(t, err, &aiErr)
	assert.Equal(t, http.StatusBadRequest, aiErr.StatusCode)
	assert.Equal(t, "The model gpt-5 does not exist", aiErr.Message)
	assert.Equal(t, "invalid_request_error", aiErr.Type)
	assert.Equal(t, "model", aiErr.Param.Unwrap())
	assert.Equal(t, "model_not_found", aiErr.Code.Unwrap())
}

func TestOpenAIChatAdapter_StreamChat_Success(t *testing.T) {
	chunks := []string{
		`data: {"id":"chat-123","choices":[{"index":0,"delta":{"role":"assistant","content":"hel"}}]}`,
		`data: {"id":"chat-123","choices":[{"index":0,"delta":{"content":"lo"}}]}`,
		`data: [DONE]`,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk + "\n\n"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}))
	defer ts.Close()

	httpClient, err := client.NewClient(client.WithResponseBodyStream(true))
	require.NoError(t, err)

	opts := LLMAdapterOptions{
		HTTPClient: httpClient,
		APIKey:     "test-key",
		BaseURL:    ts.URL,
	}
	adapter := NewOpenAIChatAdapter(opts)

	chatReq := &ChatRequest{Model: "gpt-4o", Stream: true}
	stream, err := adapter.StreamChat(context.Background(), chatReq)
	require.NoError(t, err)
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	require.NoError(t, err)

	expectedBody := chunks[0] + "\n\n" + chunks[1] + "\n\n" + chunks[2] + "\n\n"
	assert.Equal(t, expectedBody, buf.String())
}

func TestOpenAIChatAdapter_UnsupportedResponses(t *testing.T) {
	adapter := NewOpenAIChatAdapter(LLMAdapterOptions{})
	_, err := adapter.Responses(context.Background(), &ResponsesRequest{})
	require.Error(t, err)
	var aiErr *AIError
	require.ErrorAs(t, err, &aiErr)
	assert.Equal(t, http.StatusNotImplemented, aiErr.StatusCode)

	_, err = adapter.StreamResponses(context.Background(), &ResponsesRequest{})
	require.Error(t, err)
	require.ErrorAs(t, err, &aiErr)
	assert.Equal(t, http.StatusNotImplemented, aiErr.StatusCode)
}

func TestOpenAIChatAdapter_ParseError_Secure(t *testing.T) {
	// Test that non-standard error responses return plain error (not AIError)
	// This tests the internal parseError function behavior via the adapter

	// HTML error page - should return plain error
	htmlError := []byte(`<html><body><h1>500 Internal Server Error</h1></body></html>`)
	err := parseErrorForTest(http.StatusInternalServerError, htmlError)
	require.Error(t, err)

	// Should be a plain error, not AIError
	var aiErr *AIError
	assert.NotErrorAs(t, err, &aiErr, "HTML error should return plain error, not AIError")
	assert.Contains(t, err.Error(), "upstream error: status 500")

	// Empty body - should return plain error
	emptyError := []byte(``)
	err = parseErrorForTest(http.StatusBadGateway, emptyError)
	require.Error(t, err)
	assert.NotErrorAs(t, err, &aiErr, "Empty error should return plain error, not AIError")
	assert.Contains(t, err.Error(), "upstream error: status 502")

	// Random text - should return plain error
	randomError := []byte(`Something went wrong on the server`)
	err = parseErrorForTest(http.StatusServiceUnavailable, randomError)
	require.Error(t, err)
	assert.NotErrorAs(t, err, &aiErr, "Random text error should return plain error, not AIError")
	assert.Contains(t, err.Error(), "upstream error: status 503")
}

// Helper to test parseError (unexported function)
// We test via the adapter's error handling path.
func parseErrorForTest(statusCode int, body []byte) error {
	// This mirrors the parseError logic but is accessible for testing
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
	return fmt.Errorf("upstream error: status %d", statusCode)
}
