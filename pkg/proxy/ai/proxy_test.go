package ai

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
	prom "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/ai"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/telemetry/metrics"
)

var (
	mockLL      *MockLLMAdapter
	mockLLMOnce sync.Once
	mockLLMMu   sync.Mutex
)

// MockLLMAdapter mocks the ai.LLMAdapter interface.
type MockLLMAdapter struct {
	chatFunc       func(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error)
	streamChatFunc func(ctx context.Context, req *ai.ChatRequest) (io.ReadCloser, error)
	responsesFunc  func(ctx context.Context, req *ai.ResponsesRequest) (*ai.ResponsesResponse, error)
}

func (m *MockLLMAdapter) Name() string { return "mock" }
func (m *MockLLMAdapter) Chat(ctx context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &ai.ChatResponse{}, nil
}

func (m *MockLLMAdapter) StreamChat(ctx context.Context, req *ai.ChatRequest) (io.ReadCloser, error) {
	if m.streamChatFunc != nil {
		return m.streamChatFunc(ctx, req)
	}
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (m *MockLLMAdapter) Responses(ctx context.Context, req *ai.ResponsesRequest) (*ai.ResponsesResponse, error) {
	if m.responsesFunc != nil {
		return m.responsesFunc(ctx, req)
	}
	return &ai.ResponsesResponse{}, nil
}

func (m *MockLLMAdapter) StreamResponses(_ context.Context, _ *ai.ResponsesRequest) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

// MockClientAdapter mocks the ai.ClientAdapter interface.
type MockClientAdapter struct {
	toChatRequestFunc             func(body []byte) (*ai.ChatRequest, error)
	toClientChatResponseFunc      func(resp *ai.ChatResponse) (any, error)
	streamConverterFunc           func(stream io.ReadCloser) io.ReadCloser
	toClientErrorFunc             func(err *ai.AIError) (any, error)
	toClientResponsesResponseFunc func(resp *ai.ResponsesResponse) (any, error)
}

func (m *MockClientAdapter) Name() string { return "mock-client" }
func (m *MockClientAdapter) ToChatRequest(body []byte) (*ai.ChatRequest, error) {
	if m.toChatRequestFunc != nil {
		return m.toChatRequestFunc(body)
	}
	return &ai.ChatRequest{}, nil
}

func (m *MockClientAdapter) ToResponsesRequest(_ []byte) (*ai.ResponsesRequest, error) {
	return &ai.ResponsesRequest{}, nil
}

func (m *MockClientAdapter) ToClientChatResponse(resp *ai.ChatResponse) (any, error) {
	if m.toClientChatResponseFunc != nil {
		return m.toClientChatResponseFunc(resp)
	}
	return map[string]any{}, nil
}

func (m *MockClientAdapter) ToClientResponsesResponse(resp *ai.ResponsesResponse) (any, error) {
	if m.toClientResponsesResponseFunc != nil {
		return m.toClientResponsesResponseFunc(resp)
	}
	return map[string]any{}, nil
}

func (m *MockClientAdapter) StreamConverter(stream io.ReadCloser) io.ReadCloser {
	if m.streamConverterFunc != nil {
		return m.streamConverterFunc(stream)
	}
	return stream
}

func (m *MockClientAdapter) ToClientError(err *ai.AIError) (any, error) {
	if m.toClientErrorFunc != nil {
		return m.toClientErrorFunc(err)
	}
	return map[string]any{"error": err.Message}, nil
}

func setupMockAdapter(t *testing.T) {
	t.Helper()
	mockLLMOnce.Do(func() {
		ai.RegisterLLMAdapter("mock", func(_ ai.LLMAdapterOptions) (ai.LLMAdapter, error) {
			return mockLL, nil
		})
		metrics.InitAI(nil, nil)
	})
	mockLL = &MockLLMAdapter{}
}

func TestAIProxy_ServeHTTP_UnarySuccess(t *testing.T) {
	mockLLMMu.Lock()
	defer mockLLMMu.Unlock()
	setupMockAdapter(t)

	aiOpts := &config.AIOptions{
		Providers: map[string]*config.AIProvider{
			"p1": {
				Handler: "mock",
				BaseURL: "http://localhost",
				APIKey:  "key",
			},
		},
	}

	proxy := NewProxy(ProxyOptions{
		ID:             "id1",
		Target:         "p1/gpt-4",
		Weight:         1,
		AIOptions:      aiOpts,
		MetricsEnabled: true,
	})

	mockLL.chatFunc = func(_ context.Context, req *ai.ChatRequest) (*ai.ChatResponse, error) {
		assert.Equal(t, "gpt-4", req.Model)
		return &ai.ChatResponse{
			ID:      "chat-123",
			Created: 12345,
			Model:   "gpt-4",
			Usage: ai.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}, nil
	}

	clientAdapter := &MockClientAdapter{
		toClientChatResponseFunc: func(resp *ai.ChatResponse) (any, error) {
			assert.Equal(t, "gpt-4o", resp.Model) // Should be masked with virtual model
			return map[string]any{"choices": []any{}}, nil
		},
	}

	hzCtx := app.NewContext(0)
	hzCtx.Set(ai.ContextKeyClientAdapter, clientAdapter)
	hzCtx.Set(ai.ContextKeyAIFamily, ai.FamilyChat)
	hzCtx.Set(ai.ContextKeyVirtualModelName, "gpt-4o")
	hzCtx.Set(ai.ContextKeyChatRequest, &ai.ChatRequest{Model: "gpt-4o"})

	// Reset metrics before recording
	metrics.AIInputTokens.Reset()
	metrics.AIOutputTokens.Reset()

	proxy.ServeHTTP(context.Background(), hzCtx)

	assert.Equal(t, http.StatusOK, hzCtx.Response.StatusCode())
	var body map[string]any
	err := sonic.Unmarshal(hzCtx.Response.Body(), &body)
	require.NoError(t, err)
	assert.Contains(t, body, "choices")

	assert.InDelta(t, float64(10), getCounterValue(metrics.AIInputTokens, "gpt-4o", "p1/gpt-4"), 0.0001)
	assert.InDelta(t, float64(20), getCounterValue(metrics.AIOutputTokens, "gpt-4o", "p1/gpt-4"), 0.0001)
}

func TestAIProxy_ServeHTTP_UnaryError(t *testing.T) {
	mockLLMMu.Lock()
	defer mockLLMMu.Unlock()
	setupMockAdapter(t)

	aiOpts := &config.AIOptions{
		Providers: map[string]*config.AIProvider{
			"p1": {
				Handler: "mock",
				BaseURL: "http://localhost",
				APIKey:  "key",
			},
		},
	}

	proxy := NewProxy(ProxyOptions{
		ID:             "id1",
		Target:         "p1/gpt-4",
		Weight:         1,
		AIOptions:      aiOpts,
		MetricsEnabled: true,
	})

	expectedErr := &ai.AIError{
		Type:       "invalid_request_error",
		Message:    "invalid parameters",
		StatusCode: 400,
	}

	mockLL.chatFunc = func(_ context.Context, _ *ai.ChatRequest) (*ai.ChatResponse, error) {
		return nil, expectedErr
	}

	clientAdapter := &MockClientAdapter{}

	hzCtx := app.NewContext(0)
	hzCtx.Set(ai.ContextKeyClientAdapter, clientAdapter)
	hzCtx.Set(ai.ContextKeyAIFamily, ai.FamilyChat)
	hzCtx.Set(ai.ContextKeyVirtualModelName, "gpt-4o")
	hzCtx.Set(ai.ContextKeyChatRequest, &ai.ChatRequest{Model: "gpt-4o"})

	proxy.ServeHTTP(context.Background(), hzCtx)

	// In unary failure, AIProxy must call c.Error(err) instead of writing directly.
	assert.Len(t, hzCtx.Errors, 1)
	assert.Equal(t, expectedErr, hzCtx.Errors[0].Err)
}

func TestAIProxy_ServeHTTP_StreamSuccess(t *testing.T) {
	mockLLMMu.Lock()
	defer mockLLMMu.Unlock()
	setupMockAdapter(t)

	aiOpts := &config.AIOptions{
		Providers: map[string]*config.AIProvider{
			"p1": {
				Handler: "mock",
				BaseURL: "http://localhost",
				APIKey:  "key",
			},
		},
	}

	proxy := NewProxy(ProxyOptions{
		ID:             "id1",
		Target:         "p1/gpt-4",
		Weight:         1,
		AIOptions:      aiOpts,
		MetricsEnabled: true,
	})

	canonicalChunks := "data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: [DONE]\n\n"
	mockLL.streamChatFunc = func(_ context.Context, _ *ai.ChatRequest) (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(canonicalChunks)), nil
	}

	clientAdapter := &MockClientAdapter{
		streamConverterFunc: func(stream io.ReadCloser) io.ReadCloser {
			// Simply returns stream unchanged for testing
			return stream
		},
	}

	hzCtx := app.NewContext(0)
	hzCtx.Set(ai.ContextKeyClientAdapter, clientAdapter)
	hzCtx.Set(ai.ContextKeyAIFamily, ai.FamilyChat)
	hzCtx.Set(ai.ContextKeyVirtualModelName, "gpt-4o")
	hzCtx.Set(ai.ContextKeyChatRequest, &ai.ChatRequest{Model: "gpt-4o", Stream: true})

	proxy.ServeHTTP(context.Background(), hzCtx)

	assert.Equal(t, "text/event-stream", string(hzCtx.Response.Header.ContentType()))
	assert.Equal(t, canonicalChunks, string(hzCtx.Response.Body()))
}

type errorReader struct {
	data []byte
	err  error
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	if len(r.data) > 0 {
		n = copy(p, r.data)
		r.data = r.data[n:]
		return n, nil
	}
	return 0, r.err
}

func (r *errorReader) Close() error {
	return nil
}

func TestAIProxy_ServeHTTP_StreamMidError(t *testing.T) {
	mockLLMMu.Lock()
	defer mockLLMMu.Unlock()
	setupMockAdapter(t)

	aiOpts := &config.AIOptions{
		Providers: map[string]*config.AIProvider{
			"p1": {
				Handler: "mock",
				BaseURL: "http://localhost",
				APIKey:  "key",
			},
		},
	}

	proxy := NewProxy(ProxyOptions{
		ID:             "id1",
		Target:         "p1/gpt-4",
		Weight:         1,
		AIOptions:      aiOpts,
		MetricsEnabled: true,
	})

	mockLL.streamChatFunc = func(_ context.Context, _ *ai.ChatRequest) (io.ReadCloser, error) {
		return &errorReader{
			data: []byte(
				"data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n",
			),
			err: errors.New("network failure mid-stream"),
		}, nil
	}

	clientAdapter := &MockClientAdapter{
		toClientErrorFunc: func(err *ai.AIError) (any, error) {
			return map[string]any{"error": map[string]any{"message": err.Message}}, nil
		},
		streamConverterFunc: func(stream io.ReadCloser) io.ReadCloser {
			return stream
		},
	}

	hzCtx := app.NewContext(0)
	hzCtx.Set(ai.ContextKeyClientAdapter, clientAdapter)
	hzCtx.Set(ai.ContextKeyAIFamily, ai.FamilyChat)
	hzCtx.Set(ai.ContextKeyVirtualModelName, "gpt-4o")
	hzCtx.Set(ai.ContextKeyChatRequest, &ai.ChatRequest{Model: "gpt-4o", Stream: true})

	proxy.ServeHTTP(context.Background(), hzCtx)

	bodyStr := string(hzCtx.Response.Body())
	// SECURITY: Error details should NOT be leaked to client - only generic message
	assert.Contains(t, bodyStr, "Internal server error")
	assert.Contains(t, bodyStr, "data: [DONE]\n\n")
	assert.NotContains(t, bodyStr, "network failure mid-stream")
}

func TestAIProxy_ServeHTTP_InvalidTarget(t *testing.T) {
	mockLLMMu.Lock()
	defer mockLLMMu.Unlock()
	setupMockAdapter(t)

	aiOpts := &config.AIOptions{
		Providers: map[string]*config.AIProvider{
			"p1": {
				Handler: "mock",
				BaseURL: "http://localhost",
				APIKey:  "key",
			},
		},
	}

	proxy := NewProxy(ProxyOptions{
		ID:             "id1",
		Target:         "invalid_target_no_slash",
		Weight:         1,
		AIOptions:      aiOpts,
		MetricsEnabled: true,
	})

	clientAdapter := &MockClientAdapter{}

	hzCtx := app.NewContext(0)
	hzCtx.Set(ai.ContextKeyClientAdapter, clientAdapter)
	hzCtx.Set(ai.ContextKeyAIFamily, ai.FamilyChat)
	hzCtx.Set(ai.ContextKeyVirtualModelName, "gpt-4o")

	proxy.ServeHTTP(context.Background(), hzCtx)

	assert.Len(t, hzCtx.Errors, 1)
	var aiErr *ai.AIError
	require.ErrorAs(t, hzCtx.Errors[0].Err, &aiErr)
	assert.Equal(t, 500, aiErr.StatusCode)
	assert.Equal(t, "invalid target format", aiErr.Message)
}

func TestAIProxy_ServeHTTP_Responses(t *testing.T) {
	mockLLMMu.Lock()
	defer mockLLMMu.Unlock()
	setupMockAdapter(t)

	aiOpts := &config.AIOptions{
		Providers: map[string]*config.AIProvider{
			"p1": {
				Handler: "mock",
				BaseURL: "http://localhost",
				APIKey:  "key",
			},
		},
	}

	proxy := NewProxy(ProxyOptions{
		ID:             "id1",
		Target:         "p1/claude-3-opus",
		Weight:         1,
		AIOptions:      aiOpts,
		MetricsEnabled: true,
	})

	mockLL.responsesFunc = func(_ context.Context, req *ai.ResponsesRequest) (*ai.ResponsesResponse, error) {
		assert.Equal(t, "claude-3-opus", req.Model)
		return &ai.ResponsesResponse{
			ID:    "resp-123",
			Model: "claude-3-opus",
			Usage: ai.Usage{
				PromptTokens:     5,
				CompletionTokens: 15,
				TotalTokens:      20,
			},
		}, nil
	}

	clientAdapter := &MockClientAdapter{
		toClientResponsesResponseFunc: func(resp *ai.ResponsesResponse) (any, error) {
			assert.Equal(t, "claude-3-opus-virtual", resp.Model)
			return map[string]any{"id": resp.ID}, nil
		},
	}

	hzCtx := app.NewContext(0)
	hzCtx.Set(ai.ContextKeyClientAdapter, clientAdapter)
	hzCtx.Set(ai.ContextKeyAIFamily, ai.FamilyResponses)
	hzCtx.Set(ai.ContextKeyVirtualModelName, "claude-3-opus-virtual")
	hzCtx.Set(ai.ContextKeyResponsesRequest, &ai.ResponsesRequest{Model: "claude-3-opus-virtual"})

	proxy.ServeHTTP(context.Background(), hzCtx)

	assert.Equal(t, http.StatusOK, hzCtx.Response.StatusCode())
	var body map[string]any
	err := sonic.Unmarshal(hzCtx.Response.Body(), &body)
	require.NoError(t, err)
	assert.Equal(t, "resp-123", body["id"])
}

func getCounterValue(counter *prom.CounterVec, labels ...string) float64 {
	var m dto.Metric
	if err := counter.WithLabelValues(labels...).Write(&m); err != nil {
		return 0
	}
	return m.GetCounter().GetValue()
}
