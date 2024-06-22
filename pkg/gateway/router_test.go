package gateway

import (
	"context"
	"fmt"
	"http-benchmark/pkg/config"
	"slices"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func exactkHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetStatusCode(201)
}

func prefixHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetStatusCode(202)
}

func regexkHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetStatusCode(203)
}

func notFoundHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetStatusCode(404)
}

func TestRoutes(t *testing.T) {
	router := newRouter(false)

	err := router.AddRoute(config.RouteOptions{
		Paths: []string{"/market/btc*"},
	}, prefixHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths:   []string{"/spot/order", "/market/btc"},
		Methods: []string{"POST", "GET"},
	}, exactkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"~/market/(btc|usdt|eth)$"},
		Hosts: []string{"abc.com"},
	}, regexkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"orders"},
	}, nil)
	assert.Error(t, err) // invalid path

	testCases := []struct {
		method         string
		host           string
		path           string
		expectedResult int
	}{
		{"POST", "abc.com", "/spot/order", 201},
		{"GET", "abc.com", "/market/btc", 201},

		{"PUT", "abc.com", "/market/btcusdt/cool", 202},
		{"GET", "abc.com", "/market/btc/", 202},

		{"DELETE", "abc.com", "/market/eth", 203},

		{"DELETE", "abc.com", "/market/eth/orders", 200}, // not found
	}

	for _, tc := range testCases {
		c := &app.RequestContext{}
		c.Request.SetMethod(tc.method)
		c.Request.URI().SetPath(tc.path)
		c.Request.SetHost(tc.host)

		router.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != tc.expectedResult {
			t.Errorf("Expected %v for path %s, but got %v", tc.expectedResult, tc.path, statusCode)
		}
	}
}

func TestHostRoutes(t *testing.T) {
	router := newRouter(true)

	err := router.AddRoute(config.RouteOptions{
		Paths: []string{"/market/btc*"},
	}, prefixHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"/market/usdt*"},
		Hosts: []string{"abc.xyz"},
	}, prefixHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths:   []string{"/spot/order"},
		Methods: []string{"POST"},
	}, exactkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths:   []string{"/market/btc"},
		Methods: []string{"GET"},
		Hosts:   []string{"abc.com"},
	}, exactkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"~/market/(btc|usdt|eth)$"},
		Hosts: []string{"abc.com"},
		Headers: map[string][]string{
			"X-User-ID": {"666"},
		},
	}, regexkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"~/regex/(btc|usdt|eth)$"},
		Hosts: []string{"abc.xyz"},
	}, regexkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"~/regex/headers"},
		Headers: map[string][]string{
			"X-User-ID": {"6667"},
		},
	}, regexkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"orders"},
	}, nil)
	assert.Error(t, err)

	testCases := []struct {
		method         string
		host           string
		path           string
		expectedResult int
	}{
		{"POST", "abc.com", "/:host/spot/order", 201},
		{"GET", "abc.com", "/abc.com/market/btc", 201},

		{"PUT", "abc.com", "/market/btcusdt/cool", 202},
		{"GET", "abc.com", "/market/btc/", 202},        // not found
		{"GET", "abc.com", "/market/usdt/orders", 200}, // host not match

		{"DELETE", "abc.com", "/market/eth", 203},
		{"POST", "abc.com", "/regex/headers", 203},
		{"DELETE", "abc.com", "/regex/btc", 200}, // not found

		{"DELETE", "abc.com", "/market/eth/orders", 200}, // not found
	}

	for _, tc := range testCases {
		c := &app.RequestContext{}
		c.Request.SetMethod(tc.method)
		c.Request.URI().SetPath(tc.path)
		c.Request.SetHost(tc.host)
		c.Request.Header.Set("X-User-ID", "666")

		router.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != tc.expectedResult {
			t.Errorf("Expected %v for path %s, but got %v", tc.expectedResult, tc.path, statusCode)
		}
	}
}

func loadStaticRouter() *Router {
	router := newRouter(false)
	_ = router.add("GET", "/", exactkHandler)
	_ = router.add("GET", "/foo", exactkHandler)
	_ = router.add("GET", "/foo/bar/baz/", exactkHandler)
	_ = router.add("GET", "/foo/bar/baz/qux/quux", exactkHandler)
	_ = router.add("GET", "/foo/bar/baz/qux/quux/corge/grault/garply/waldo/fred", exactkHandler)
	return router
}

func BenchmarkStaticRoot(b *testing.B) {
	router := loadStaticRouter()

	b.ReportAllocs()
	b.ResetTimer()

	benchmark(b, router, "GET", "/foo")
}

func BenchmarkStatic1(b *testing.B) {
	router := loadStaticRouter()

	b.ReportAllocs()
	b.ResetTimer()

	benchmark(b, router, "GET", "/foo")
}

func BenchmarkStatic3(b *testing.B) {
	router := loadStaticRouter()

	b.ReportAllocs()
	b.ResetTimer()

	benchmark(b, router, "GET", "/foo/bar/baz")
}

func BenchmarkStatic5(b *testing.B) {
	router := loadStaticRouter()

	b.ReportAllocs()
	b.ResetTimer()

	benchmark(b, router, "GET", "/foo/bar/baz/qux/quux")
}

func BenchmarkCode(b *testing.B) {
	method := "GET"
	//path1 := "/foo"
	//path5 := "/foo/bar/baz/qux/quux"
	path10 := "/foo/bar/baz/qux/quux/corge/grault/garply/waldo/fred"
	//prefix := "/foo/bar/baz/qux/quux"

	routeSetting := config.RouteOptions{
		Methods: []string{method},
		Paths:   []string{path10},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		func() app.HandlerFunc {

			isFound := false
			if slices.Contains(routeSetting.Paths, path10) {
				isFound = true
			}

			if slices.Contains(routeSetting.Methods, method) {
				isFound = true
			}

			if isFound {
				return exactkHandler
			}

			return nil
		}()
	}
}

func BenchmarkStatic10(b *testing.B) {
	router := loadStaticRouter()

	b.ReportAllocs()
	b.ResetTimer()

	benchmark(b, router, "GET", "/foo/bar/baz/qux/quux/corge/grault/garply/waldo/fred")
}

func benchmark(b *testing.B, router *Router, method, path string) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = router.find(method, path)
		// if len(handlers) != 1 {
		// 	b.Errorf("Expected 1 handler, got %d", len(handlers))
		// }
	}
}

func setupMap() map[string]*node {
	m := make(map[string]*node)
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("futures%d", i)
		m[key] = &node{}
	}

	return m
}

func BenchmarkMapLookup(b *testing.B) {
	m := setupMap()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, found := m[""]
		if !found {
			b.Errorf("key not found")
		}
	}
}
