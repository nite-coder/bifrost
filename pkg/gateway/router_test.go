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

func generalkHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetStatusCode(204)
}

func registerByNodeType(router *Router, nodeType nodeType) *Router {
	switch nodeType {
	case nodeTypeExact:
		_ = router.AddRoute(config.RouteOptions{
			Paths: []string{"= /market/btc", "= /"},
		}, exactkHandler)
	case nodeTypePrefix:
		_ = router.AddRoute(config.RouteOptions{
			Paths: []string{"^= /market/btc", "^= /"},
		}, prefixHandler)
	case nodeTypeRegex:
		_ = router.AddRoute(config.RouteOptions{
			Paths: []string{"~ /market/(btc|usdt|eth)$", "~ ^/$"},
		}, regexkHandler)

		_ = router.AddRoute(config.RouteOptions{
			Paths: []string{"~ /market/(btc|usdt|eth)"},
		}, nil) // test two regexs router order
	case nodeTypeGeneral:
		_ = router.AddRoute(config.RouteOptions{
			Paths: []string{"/market/btc", "/"},
		}, generalkHandler)
	}

	return router
}

func TestRoutePriorityAndRoot(t *testing.T) {

	t.Run("exact match", func(t *testing.T) {
		nodeTypes := []nodeType{nodeTypeExact, nodeTypePrefix, nodeTypeGeneral, nodeTypeRegex}
		router := newRouter()

		for _, nodeType := range nodeTypes {
			router = registerByNodeType(router, nodeType)
		}

		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.URI().SetPath("/market/btc")

		router.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != 201 {
			t.Errorf("Expected %v for path %s, but got %v", 201, "/market/btc", statusCode)
		}

		c.Request.URI().SetPath("/")
		router.ServeHTTP(context.Background(), c)
		statusCode = c.Response.StatusCode()

		if statusCode != 201 {
			t.Errorf("Expected %v for path %s, but got %v", 201, "/", statusCode)
		}
	})

	t.Run("prefix match", func(t *testing.T) {
		nodeTypes := []nodeType{nodeTypePrefix, nodeTypeGeneral, nodeTypeRegex}
		router := newRouter()

		for _, nodeType := range nodeTypes {
			router = registerByNodeType(router, nodeType)
		}

		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.URI().SetPath("/market/btc")

		router.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != 202 {
			t.Errorf("Expected %v for path %s, but got %v", 202, "/market/btc", statusCode)
		}

		c.Request.URI().SetPath("/")
		router.ServeHTTP(context.Background(), c)
		statusCode = c.Response.StatusCode()

		if statusCode != 202 {
			t.Errorf("Expected %v for path %s, but got %v", 202, "/", statusCode)
		}
	})

	t.Run("regex match", func(t *testing.T) {
		nodeTypes := []nodeType{nodeTypeGeneral, nodeTypeRegex}
		router := newRouter()

		for _, nodeType := range nodeTypes {
			router = registerByNodeType(router, nodeType)
		}

		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.URI().SetPath("/market/btc")

		router.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != 203 {
			t.Errorf("Expected %v for path %s, but got %v", 203, "/market/btc", statusCode)
		}

		c.Request.URI().SetPath("/")
		router.ServeHTTP(context.Background(), c)
		statusCode = c.Response.StatusCode()

		if statusCode != 203 {
			t.Errorf("Expected %v for path %s, but got %v", 203, "/", statusCode)
		}
	})

	t.Run("general match", func(t *testing.T) {
		nodeTypes := []nodeType{nodeTypeGeneral}
		router := newRouter()

		for _, nodeType := range nodeTypes {
			router = registerByNodeType(router, nodeType)
		}

		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.URI().SetPath("/market/btc")

		router.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != 204 {
			t.Errorf("Expected %v for path %s, but got %v", 204, "/market/btc", statusCode)
		}

		c.Request.URI().SetPath("/")
		router.ServeHTTP(context.Background(), c)
		statusCode = c.Response.StatusCode()

		if statusCode != 204 {
			t.Errorf("Expected %v for path %s, but got %v", 204, "/", statusCode)
		}
	})

}

func TestRootRoute(t *testing.T) {

	router := newRouter()

	_ = router.AddRoute(config.RouteOptions{
		Paths: []string{"= /"},
		//Methods: []string{"POST"},
	}, exactkHandler)

	_ = router.AddRoute(config.RouteOptions{
		Paths: []string{"^= /"},
		//Methods: []string{"POST"},
	}, prefixHandler)

	_ = router.AddRoute(config.RouteOptions{
		Paths: []string{"/"},
		//Methods: []string{"POST"},
	}, generalkHandler)

	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.URI().SetPath("/")

	router.ServeHTTP(context.Background(), c)
	statusCode := c.Response.StatusCode()

	if statusCode != 201 {
		t.Errorf("Expected %v for path %s, but got %v", 201, "/", statusCode)
	}
}

func TestRoutes(t *testing.T) {
	router := newRouter()

	err := router.AddRoute(config.RouteOptions{
		Paths: []string{"^= /market/btc", "^= /spot"},
	}, prefixHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths:   []string{"= /spot/order"},
		Methods: []string{"POST", "GET"},
	}, exactkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"~/market/(btc|usdt|eth)$"},
	}, regexkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Paths: []string{"/market"},
	}, generalkHandler)
	assert.NoError(t, err)

	testCases := []struct {
		method         string
		host           string
		path           string
		expectedResult int
	}{
		{"POST", "abc.com", "/spot/order", 201},

		{"GET", "abc.com", "/spot/777", 202},
		{"PUT", "abc.com", "/market/btcusdt/cool", 202},

		{"GET", "abc.com", "/market/usdt", 203},
		{"DELETE", "abc.com", "/market/eth", 203},

		{"DELETE", "abc.com", "/market/eth/orders", 204},
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

func TestDuplicateHTTPMethods(t *testing.T) {
	router := newRouter()

	err := router.AddRoute(config.RouteOptions{
		Methods: []string{"GET", "POST"},
		Paths:   []string{"/foo"},
	}, exactkHandler)
	assert.NoError(t, err)

	err = router.AddRoute(config.RouteOptions{
		Methods: []string{"GET"},
		Paths:   []string{"/foo"},
	}, exactkHandler)
	assert.ErrorIs(t, err, ErrAlreadyExists)

	err = router.AddRoute(config.RouteOptions{
		Methods: []string{},
		Paths:   []string{"/foo"},
	}, exactkHandler)
	assert.NoError(t, err)
}

func loadStaticRouter() *Router {
	router := newRouter()
	_ = router.add("GET", "/", nodeTypeGeneral, exactkHandler)
	_ = router.add("GET", "/foo", nodeTypeGeneral, exactkHandler)
	_ = router.add("GET", "/foo/bar/baz/", nodeTypeGeneral, exactkHandler)
	_ = router.add("GET", "/foo/bar/baz/qux/quux", nodeTypeGeneral, exactkHandler)
	_ = router.add("GET", "/foo/bar/baz/qux/quux/corge/grault/garply/waldo/fred", nodeTypeGeneral, exactkHandler)
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
		_, _ = router.find(method, path)
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
