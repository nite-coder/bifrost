package trafficsplitter

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestSplitter(t *testing.T) {

	options := &Options{
		Key: "$my_order",
		Destinations: []*Destination{
			{
				To:     "old_server",
				Weight: 90,
			},
			{
				To:     "new_server",
				Weight: 10,
			},
		},
	}

	m := NewMiddleware(options)

	hits := map[string]int{"old_server": 0, "new_server": 0}

	for i := 0; i < 1000; i++ {
		ctx := context.Background()
		hzCtx := app.NewContext(0)
		hzCtx.Request.SetMethod("POST")
		hzCtx.Request.URI().SetPath("/api/v1/hello")
		m.ServeHTTP(ctx, hzCtx)

		val := hzCtx.GetString(m.options.Key)
		hits[val]++
	}

	assert.InDelta(t, 900, hits["old_server"], 50)
	assert.InDelta(t, 100, hits["new_server"], 50)
	t.Log("old_server", hits["old_server"])
	t.Log("new_server", hits["new_server"])
}

func TestFactory_Errors(t *testing.T) {
	_ = Init()
	h := middleware.Factory("traffic_splitter")

	tests := []struct {
		name        string
		params      any
		expectedErr string
	}{
		{
			name:        "invalid params structure",
			params:      "invalid-string",
			expectedErr: "failed to decode middleware params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := h(tt.params)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectedErr)
			}
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestZeroWeight(t *testing.T) {
	// Test that 0 weight is automatically promoted to 1
	options := &Options{
		Key: "dest",
		Destinations: []*Destination{
			{To: "a", Weight: 0},
			{To: "b", Weight: 0},
		},
	}

	m := NewMiddleware(options)
	assert.Equal(t, int64(2), m.totalWeight)
	assert.Equal(t, int64(1), options.Destinations[0].Weight)
	assert.Equal(t, int64(1), options.Destinations[1].Weight)
}

func TestNoDestinations(t *testing.T) {
	options := &Options{
		Key:          "dest",
		Destinations: []*Destination{},
	}

	m := NewMiddleware(options)
	assert.Equal(t, int64(0), m.totalWeight)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	m.ServeHTTP(ctx, hzCtx)
	// Should not panic, just proceed
	assert.Empty(t, hzCtx.GetString("dest"))
}

func TestServeHTTP_RandomError(t *testing.T) {
	// Mock getRandomNumber to return error
	originalGetRandomNumber := getRandomNumber
	defer func() { getRandomNumber = originalGetRandomNumber }()

	getRandomNumber = func(max int64) (int64, error) {
		return 0, errors.New("random error")
	}

	options := &Options{
		Key: "dest",
		Destinations: []*Destination{
			{To: "default", Weight: 100},
			{To: "other", Weight: 100},
		},
	}

	m := NewMiddleware(options)
	ctx := context.Background()
	hzCtx := app.NewContext(0)

	m.ServeHTTP(ctx, hzCtx)

	// Should fallback to first destination
	val := hzCtx.GetString("dest")
	assert.Equal(t, "default", val)
}
