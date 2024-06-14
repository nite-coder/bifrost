package gateway

import (
	"context"
	"http-benchmark/pkg/domain"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestRouters(t *testing.T) {
	r := NewRouter()

	err := r.add(POST, "/spot/orders", nil)
	assert.NoError(t, err)

	err = r.add(POST, "/futures/acc*", nil)
	assert.NoError(t, err)

	m := r.find(POST, "/spot/orders")
	assert.Len(t, m, 1)

	m = r.find(POST, "/futures/account")
	assert.Len(t, m, 1)
}

// dummyHandler is a placeholder handler function
func dummyHandler(c context.Context, ctx *app.RequestContext) {
	ctx.String(200, "OK")
}

// BenchmarkFind benchmarks the find function
func BenchmarkFind(b *testing.B) {
	router := NewRouter()
	router.add(POST, "/spot/orders", dummyHandler)
	router.add(POST, "/spot2/orders", dummyHandler)
	router.add(POST, "/spot3/orders", dummyHandler)
	router.add(POST, "/spot4/orders", dummyHandler)
	router.add(POST, "/spot/5orders", dummyHandler)

	tests := []struct {
		method string
		path   string
	}{
		{POST, "/spot/orders"},
	}

	req := app.NewContext(1)
	req.Request.SetMethod("POST")
	req.URI().SetPath("/spot/orders")

	for _, tt := range tests {
		b.Run(tt.method+" "+tt.path, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				router.ServeHTTP(context.Background(), req)
			}
		})
	}
}

// Mock handler function for testing
func mockHandler(c context.Context, ctx *app.RequestContext) {
	ctx.WriteString("btc mock handler")
}

// Test prefix matching
func TestPrefixMatching(t *testing.T) {
	router := NewRouter()

	// Add prefix route
	router.add("GET", "/market/btc*", mockHandler)

	testCases := []struct {
		path           string
		expectedResult bool
	}{
		{"/market/btcusdt/cool", true},
		{"/market/btc/usdt/caaa", true},
		{"/market/btc_usdt/hey", true},
		{"/market/btc", true},
		{"/market/eth", false},
	}

	for _, tc := range testCases {
		handlers := router.find("GET", tc.path)
		if (handlers != nil) != tc.expectedResult {
			t.Errorf("Expected %v for path %s, but got %v", tc.expectedResult, tc.path, handlers != nil)
		}
	}
}

// Test exact matching
func TestExactMatching(t *testing.T) {
	router := NewRouter()

	// Add exact route
	router.add("GET", "/market/btc", mockHandler)

	testCases := []struct {
		path           string
		expectedResult bool
	}{
		{"/market/btc", true},
		{"/market/btcusdt/cool", false},
		{"/market/btc/usdt/caaa", false},
		{"/market/btc_usdt/hey", false},
		{"/market/eth", false},
	}

	for _, tc := range testCases {
		handlers := router.find("GET", tc.path)
		if (handlers != nil) != tc.expectedResult {
			t.Errorf("Expected %v for path %s, but got %v", tc.expectedResult, tc.path, handlers != nil)
		}
	}
}

// Test prefix and exact matching priority
func TestPrefixAndExactMatchingPriority(t *testing.T) {
	router := NewRouter()

	// Add prefix and exact routes
	router.AddRoute(domain.RouteOptions{
		Paths: []string{"/market/btc*"},
	}, mockHandler)

	router.AddRoute(domain.RouteOptions{
		Paths: []string{"/market/usdt_hello*"},
	}, mockHandler)

	router.AddRoute(domain.RouteOptions{
		Paths: []string{"/market/eth_usdt*"},
	}, mockHandler)

	router.AddRoute(domain.RouteOptions{
		Paths: []string{"/market/btc"},
	}, func(c context.Context, ctx *app.RequestContext) {
		ctx.WriteString("exact handler")
	})

	testCases := []struct {
		path           string
		expectedResult string
	}{
		{"/market/btc", "exact handler"},
		{"/market/btcusdt/cool", "btc mock handler"},
		{"/market/btc/usdt/caaa", "btc mock handler"},
		{"/market/btc_usdt/hey", "btc mock handler"},
		{"/market/eth", ""},
	}

	for _, tc := range testCases {
		req := &app.RequestContext{}
		req.URI().SetPath(tc.path)
		router.ServeHTTP(context.Background(), req)
		result := string(req.Response.Body())

		if result != tc.expectedResult {
			t.Errorf("Expected %v for path %s, but got %v", tc.expectedResult, tc.path, result)
		}

	}
}
