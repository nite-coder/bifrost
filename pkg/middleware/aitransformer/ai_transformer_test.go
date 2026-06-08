package aitransformer

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/ai"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const testRequestBody = `{"model":"gpt-4o"}`

var (
	mockClientAdapter *MockClientAdapter
	mockOnce          sync.Once
)

// MockClientAdapter mocks the ai.ClientAdapter interface.
type MockClientAdapter struct {
	toChatRequestFunc func(body []byte) (*ai.ChatRequest, error)
	toClientErrorFunc func(err *ai.AIError) (any, error)
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

func (m *MockClientAdapter) ToClientChatResponse(_ *ai.ChatResponse) (any, error) {
	return map[string]any{}, nil
}

func (m *MockClientAdapter) ToClientResponsesResponse(_ *ai.ResponsesResponse) (any, error) {
	return map[string]any{}, nil
}

func (m *MockClientAdapter) StreamConverter(stream io.ReadCloser) io.ReadCloser {
	return stream
}

func (m *MockClientAdapter) ToClientError(err *ai.AIError) (any, error) {
	if m.toClientErrorFunc != nil {
		return m.toClientErrorFunc(err)
	}
	return map[string]any{"error": err.Message}, nil
}

func setupMockClient(t *testing.T) {
	t.Helper()
	mockOnce.Do(func() {
		ai.RegisterClientAdapter("mock-client", func() ai.ClientAdapter {
			return mockClientAdapter
		})
	})
	mockClientAdapter = &MockClientAdapter{}
}

func TestAITransformer_Ingress(t *testing.T) {
	setupMockClient(t)
	m := NewMiddleware(Options{Format: "mock-client"})

	mockClientAdapter.toChatRequestFunc = func(body []byte) (*ai.ChatRequest, error) {
		assert.JSONEq(t, testRequestBody, string(body))
		return &ai.ChatRequest{Model: "gpt-4o"}, nil
	}

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetBody([]byte(testRequestBody))
	hzCtx.Request.SetRequestURI("/v1/chat/completions")

	m.ServeHTTP(context.Background(), hzCtx)

	// Verify Context Keys
	adapterVal, exists := hzCtx.Get(ai.ContextKeyClientAdapter)
	require.True(t, exists)
	assert.Equal(t, mockClientAdapter, adapterVal)

	chatReqVal, exists := hzCtx.Get(ai.ContextKeyChatRequest)
	require.True(t, exists)
	chatReq, ok := chatReqVal.(*ai.ChatRequest)
	require.True(t, ok)
	assert.Equal(t, "gpt-4o", chatReq.Model)

	virtualModelVal, exists := hzCtx.Get(ai.ContextKeyVirtualModelName)
	require.True(t, exists)
	assert.Equal(t, "gpt-4o", virtualModelVal)

	modelNameVar := hzCtx.GetString(variable.Model)
	assert.Equal(t, "gpt-4o", modelNameVar)
}

func TestAITransformer_FamilyDetection(t *testing.T) {
	setupMockClient(t)
	m := NewMiddleware(Options{Format: "mock-client"})

	mockClientAdapter.toChatRequestFunc = func(_ []byte) (*ai.ChatRequest, error) {
		return &ai.ChatRequest{Model: "gpt-4o"}, nil
	}

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetBody([]byte(testRequestBody))
	// Path contains /responses but does not end with it.
	hzCtx.Request.SetRequestURI("/v1/responses/chat/completions")

	m.ServeHTTP(context.Background(), hzCtx)

	familyVal, exists := hzCtx.Get(ai.ContextKeyAIFamily)
	require.True(t, exists)
	assert.Equal(t, ai.FamilyChat, familyVal)
}

func TestAITransformer_EgressError(t *testing.T) {
	setupMockClient(t)
	m := NewMiddleware(Options{Format: "mock-client"})

	mockClientAdapter.toChatRequestFunc = func(_ []byte) (*ai.ChatRequest, error) {
		return &ai.ChatRequest{Model: "gpt-4o"}, nil
	}
	mockClientAdapter.toClientErrorFunc = func(err *ai.AIError) (any, error) {
		return map[string]any{"client_error": err.Message}, nil
	}

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetBody([]byte(testRequestBody))
	hzCtx.Request.SetRequestURI("/v1/chat/completions")

	// Mock dynamic handlers calling Next and throwing an error
	hzCtx.SetHandlers([]app.HandlerFunc{
		func(_ context.Context, c *app.RequestContext) {
			_ = c.Error(&ai.AIError{
				Type:       "invalid_request_error",
				Message:    "bad prompt",
				StatusCode: 400,
			})
		},
	})

	m.ServeHTTP(context.Background(), hzCtx)

	assert.Equal(t, 400, hzCtx.Response.StatusCode())
	var res map[string]any
	err := sonic.Unmarshal(hzCtx.Response.Body(), &res)
	require.NoError(t, err)
	assert.Equal(t, "bad prompt", res["client_error"])
}

func TestAITransformer_EgressPlainError(t *testing.T) {
	setupMockClient(t)
	m := NewMiddleware(Options{Format: "mock-client"})

	mockClientAdapter.toChatRequestFunc = func(_ []byte) (*ai.ChatRequest, error) {
		return &ai.ChatRequest{Model: "gpt-4o"}, nil
	}
	mockClientAdapter.toClientErrorFunc = func(err *ai.AIError) (any, error) {
		return map[string]any{"error": map[string]any{"message": err.Message}}, nil
	}

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetBody([]byte(testRequestBody))
	hzCtx.Request.SetRequestURI("/v1/chat/completions")

	// Mock dynamic handlers calling Next and throwing a PLAIN error (not AIError)
	hzCtx.SetHandlers([]app.HandlerFunc{
		func(_ context.Context, c *app.RequestContext) {
			_ = c.Error(errors.New("upstream returned HTML error page"))
		},
	})

	m.ServeHTTP(context.Background(), hzCtx)

	// Should return 502 Bad Gateway with generic error message
	assert.Equal(t, 502, hzCtx.Response.StatusCode())
	var res map[string]any
	err := sonic.Unmarshal(hzCtx.Response.Body(), &res)
	require.NoError(t, err)

	// Verify generic error is returned (not the original error message)
	errMap, ok := res["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Internal server error", errMap["message"])

	// Original error should NOT be in response
	bodyStr := string(hzCtx.Response.Body())
	assert.NotContains(t, bodyStr, "upstream returned HTML error page")
}
