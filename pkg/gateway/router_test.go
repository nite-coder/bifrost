package gateway

import (
	"context"
	"http-benchmark/pkg/config"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

// Mock handler function for testing
func mockHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetStatusCode(201)
}

func exactkHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetStatusCode(204)
}

// Test exact matching
func TestRoutes(t *testing.T) {
	router := newRouter()

	err := router.AddRoute(config.RouteOptions{
		Paths: []string{"/market/btc*"},
	}, mockHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths:   []string{"/spot/order", "/market/btc"},
		Methods: []string{"POST", "GET"},
	}, exactkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"~/futures/(btc|usdt|eth)"},
	}, mockHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"orders"},
	}, mockHandler)
	assert.Error(t, err)

	testCases := []struct {
		method         string
		path           string
		expectedResult int
	}{
		{"POST", "/spot/order", 204},
		{"GET", "/market/btc", 204},

		{"PUT", "/market/btcusdt/cool", 201},
		{"GET", "/market/btc/", 201},

		{"DELETE", "/futures/eth/orders", 201},

		{"PATCH", "/market/eth", 200},
	}

	for _, tc := range testCases {
		c := &app.RequestContext{}
		c.Request.SetMethod(tc.method)
		c.Request.URI().SetPath(tc.path)

		router.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != tc.expectedResult {
			t.Errorf("Expected %v for path %s, but got %v", tc.expectedResult, tc.path, statusCode)
		}
	}
}
