package gateway

import (
	"context"
	"regexp"
	"testing"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/router"

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

func registerByNodeType(route *Routes, nodeType router.NodeType) *Routes {
	switch nodeType {
	case router.Exact:
		_ = route.Add(config.RouteOptions{
			Paths: []string{"= /market/btc", "= /"},
		}, exactkHandler)
	case router.PreferentialPrefix:
		_ = route.Add(config.RouteOptions{
			Paths: []string{"^~ /market/btc", "^~ /"},
		}, prefixHandler)
	case router.Regex:
		_ = route.Add(config.RouteOptions{
			Paths: []string{"~ /market/(btc|usdt|eth)$", "~ ^/$"},
		}, regexkHandler)

		_ = route.Add(config.RouteOptions{
			Paths: []string{"~* /HELLO/WORLD/aaa/bbb"},
		}, regexkHandler)

		_ = route.Add(config.RouteOptions{
			Paths: []string{"~ /market/(btc|usdt|eth)"},
		}, nil) // test two regexs router order
	case router.Prefix:
		_ = route.Add(config.RouteOptions{
			Paths: []string{"/market/btc", "/"},
		}, generalkHandler)
	}

	return route
}

func TestRoutePriorityAndRoot(t *testing.T) {

	t.Run("exact match", func(t *testing.T) {
		nodeTypes := []router.NodeType{router.Exact, router.PreferentialPrefix, router.Prefix, router.Regex}
		route := newRoutes()

		for _, nodeType := range nodeTypes {
			route = registerByNodeType(route, nodeType)
		}

		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.URI().SetPath("/market/btc")

		route.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != 201 {
			t.Errorf("Expected %v for path %s, but got %v", 201, "/market/btc", statusCode)
		}

		c.Request.URI().SetPath("/")
		route.ServeHTTP(context.Background(), c)
		statusCode = c.Response.StatusCode()

		if statusCode != 201 {
			t.Errorf("Expected %v for path %s, but got %v", 201, "/", statusCode)
		}
	})

	t.Run("prefix match 1", func(t *testing.T) {
		nodeTypes := []router.NodeType{router.PreferentialPrefix, router.Prefix, router.Regex}
		route := newRoutes()

		for _, nodeType := range nodeTypes {
			route = registerByNodeType(route, nodeType)
		}

		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.URI().SetPath("/market/btc")

		route.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != 202 {
			t.Errorf("Expected %v for path %s, but got %v", 202, "/market/btc", statusCode)
		}

		c.Request.URI().SetPath("/")
		route.ServeHTTP(context.Background(), c)
		statusCode = c.Response.StatusCode()

		if statusCode != 202 {
			t.Errorf("Expected %v for path %s, but got %v", 202, "/", statusCode)
		}
	})

	t.Run("prefix match 2", func(t *testing.T) {
		route := newRoutes()

		_ = route.Add(config.RouteOptions{
			Methods: []string{"GET"},
			Paths:   []string{"^~ /"},
		}, generalkHandler)

		_ = route.Add(config.RouteOptions{
			Methods: []string{"GET"},
			Paths:   []string{"^~ /order_book"},
		}, prefixHandler)

		c := app.NewContext(0)
		c.Request.SetMethod("GET")
		c.Request.URI().SetPath("/order_book")

		route.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != 202 {
			t.Errorf("Expected %v for path %s, but got %v", 202, "/market/btc", statusCode)
		}
	})

	t.Run("regex match", func(t *testing.T) {
		nodeTypes := []router.NodeType{router.Prefix, router.Regex}
		route := newRoutes()

		for _, nodeType := range nodeTypes {
			route = registerByNodeType(route, nodeType)
		}

		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.URI().SetPath("/market/btc")

		route.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != 203 {
			t.Errorf("Expected %v for path %s, but got %v", 203, "/market/btc", statusCode)
		}

		c.Request.URI().SetPath("/")
		route.ServeHTTP(context.Background(), c)
		statusCode = c.Response.StatusCode()

		if statusCode != 203 {
			t.Errorf("Expected %v for path %s, but got %v", 203, "/", statusCode)
		}

		c.Request.URI().SetPath("/hello/world/aaa/bbb")
		route.ServeHTTP(context.Background(), c)
		statusCode = c.Response.StatusCode()

		if statusCode != 203 {
			t.Errorf("Expected %v for path %s, but got %v", 203, "/", statusCode)
		}
	})

	t.Run("general match", func(t *testing.T) {
		nodeTypes := []router.NodeType{router.Prefix}
		route := newRoutes()

		for _, nodeType := range nodeTypes {
			route = registerByNodeType(route, nodeType)
		}

		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.URI().SetPath("/market/btc")

		route.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != 204 {
			t.Errorf("Expected %v for path %s, but got %v", 204, "/market/btc", statusCode)
		}

		c.Request.URI().SetPath("/")
		route.ServeHTTP(context.Background(), c)
		statusCode = c.Response.StatusCode()

		if statusCode != 204 {
			t.Errorf("Expected %v for path %s, but got %v", 204, "/", statusCode)
		}
	})

	t.Run("general match 2", func(t *testing.T) {
		route := newRoutes()

		_ = route.Add(config.RouteOptions{
			Paths: []string{"/"},
		}, prefixHandler)

		_ = route.Add(config.RouteOptions{
			Paths: []string{"/mock"},
		}, generalkHandler)

		c := app.NewContext(0)
		c.Request.SetMethod("POST")
		c.Request.URI().SetPath("/mock")

		route.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != 204 {
			t.Errorf("Expected %v for path %s, but got %v", 204, "/market/btc", statusCode)
		}
	})

}

func TestRootRoute(t *testing.T) {
	route := newRoutes()

	_ = route.Add(config.RouteOptions{
		Paths: []string{"= /"},
	}, exactkHandler)

	_ = route.Add(config.RouteOptions{
		Paths: []string{"^~ /"},
	}, prefixHandler)

	_ = route.Add(config.RouteOptions{
		Paths: []string{"/"},
	}, generalkHandler)

	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.URI().SetPath("/any/subpath")

	route.ServeHTTP(context.Background(), c)
	statusCode := c.Response.StatusCode()

	if statusCode != 202 {
		t.Errorf("Expected %v for path %s, but got %v", 202, "/", statusCode)
	}

	c = app.NewContext(0)
	c.Request.SetMethod("GET")
	c.Request.URI().SetPath("/")

	route.ServeHTTP(context.Background(), c)
	statusCode = c.Response.StatusCode()

	if statusCode != 201 {
		t.Errorf("Expected %v for path %s, but got %v", 201, "/", statusCode)
	}
}

func TestRoutes(t *testing.T) {
	route := newRoutes()

	err := route.Add(config.RouteOptions{
		Paths: []string{"^~ /market/btc", "^~ /spot"},
	}, prefixHandler)
	assert.NoError(t, err)

	err = route.Add(config.RouteOptions{
		Paths:   []string{"= /spot/order"},
		Methods: []string{"POST", "GET"},
	}, exactkHandler)
	assert.NoError(t, err)

	err = route.Add(config.RouteOptions{
		Paths: []string{"~/market/(btc|usdt|eth)$"},
	}, regexkHandler)
	assert.NoError(t, err)

	err = route.Add(config.RouteOptions{
		Paths: []string{"/market"},
	}, generalkHandler)
	assert.NoError(t, err)

	err = route.Add(config.RouteOptions{
		Paths: []string{"/api/v1/"},
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

		{"GET", "abc.com", "/api/v1/orders", 204},
	}

	for _, tc := range testCases {
		c := &app.RequestContext{}
		c.Request.SetMethod(tc.method)
		c.Request.URI().SetPath(tc.path)
		c.Request.SetHost(tc.host)

		route.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != tc.expectedResult {
			t.Errorf("Expected %v for path %s, but got %v", tc.expectedResult, tc.path, statusCode)
		}
	}
}

func TestRegexOrder(t *testing.T) {
	route := newRoutes()

	err := route.Add(config.RouteOptions{
		Methods: []string{"GET"},
		Paths:   []string{"~ ^/(api/v1/live/subtitle|api/v2/w33/live/)"},
	}, regexkHandler)
	assert.NoError(t, err)

	err = route.Add(config.RouteOptions{
		Paths: []string{"/api/v1/hello", "/api/v2/world"},
	}, generalkHandler)
	assert.NoError(t, err)

	c := &app.RequestContext{}
	c.Request.SetMethod("GET")
	c.Request.URI().SetPath("api/v1/hello/aaa")

	route.ServeHTTP(context.Background(), c)
	statusCode := c.Response.StatusCode()

	if statusCode != 204 {
		t.Errorf("wrong expected %d got %v", 204, statusCode)
	}
}

func TestDuplicateRoutes(t *testing.T) {
	route := newRoutes()

	err := route.Add(config.RouteOptions{
		Methods: []string{"GET", "POST"},
		Paths:   []string{"/foo"},
	}, exactkHandler)
	assert.NoError(t, err)

	err = route.Add(config.RouteOptions{
		Methods: []string{"GET"},
		Paths:   []string{"/foo"},
	}, exactkHandler)
	assert.ErrorIs(t, err, router.ErrAlreadyExists)

	err = route.Add(config.RouteOptions{
		Methods: []string{},
		Paths:   []string{"/foo"},
	}, exactkHandler)
	assert.ErrorIs(t, err, router.ErrAlreadyExists)

	err = route.Add(config.RouteOptions{
		Methods: []string{"GET"},
		Paths:   []string{"/"},
	}, exactkHandler)
	assert.NoError(t, err)

	err = route.Add(config.RouteOptions{
		Methods: []string{"GET"},
		Paths:   []string{"/"},
	}, exactkHandler)
	assert.ErrorIs(t, err, router.ErrAlreadyExists)
}

var testString = `/hello/world/you/bbb/dddd`

var pattern = `/hello/world/(You|aaa|ccc)`

func BenchmarkCaseSensitive(b *testing.B) {
	re := regexp.MustCompile(pattern)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := re.MatchString(testString)
		assert.False(b, result)
	}
}

func BenchmarkCaseInsensitive(b *testing.B) {
	re := regexp.MustCompile(`(?i)` + pattern)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := re.MatchString(testString)
		assert.True(b, result)
	}
}

func TestGetHost(t *testing.T) {
	ctx := app.NewContext(0)
	ctx.Request.SetHost("example.com")

	getHostFn := getHost(ctx)

	// First call should compute the host
	host1 := getHostFn()
	assert.Equal(t, "example.com", host1)

	// Second call should return cached value
	host2 := getHostFn()
	assert.Equal(t, "example.com", host2)
}

func TestRouteAddErrors(t *testing.T) {
	route := newRoutes()

	t.Run("empty paths", func(t *testing.T) {
		err := route.Add(config.RouteOptions{
			Paths: []string{},
		}, generalkHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "paths cannot be empty")
	})

	t.Run("exact route empty path", func(t *testing.T) {
		err := route.Add(config.RouteOptions{
			Paths: []string{"="},
		}, generalkHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exact route cannot be empty")
	})

	t.Run("prefix route empty path", func(t *testing.T) {
		err := route.Add(config.RouteOptions{
			Paths: []string{"^~"},
		}, generalkHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prefix route cannot be empty")
	})

	t.Run("regex route empty expression", func(t *testing.T) {
		err := route.Add(config.RouteOptions{
			Paths: []string{"~"},
		}, generalkHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "regular expression route cannot be empty")
	})

	t.Run("case insensitive regex route empty expression", func(t *testing.T) {
		err := route.Add(config.RouteOptions{
			Paths: []string{"~*"},
		}, generalkHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "regular expression route cannot be empty")
	})

	t.Run("invalid path format", func(t *testing.T) {
		err := route.Add(config.RouteOptions{
			Paths: []string{"invalid_path"},
		}, generalkHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is invalid path")
	})

	t.Run("invalid http method", func(t *testing.T) {
		err := route.Add(config.RouteOptions{
			Paths:   []string{"/test"},
			Methods: []string{"INVALID"},
		}, generalkHandler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not valid")
	})

	t.Run("checkRegexpRoute with methods", func(t *testing.T) {
		regex, _ := regexp.Compile("/test/(a|b)")
		setting := routeSetting{
			regex:       regex,
			route:       &config.RouteOptions{Methods: []string{"GET"}},
			middlewares: nil,
		}

		// Should return false for non-matching method
		result := checkRegexpRoute(setting, "POST", "/test/a")
		assert.False(t, result)

		// Should return true for matching method and path
		result = checkRegexpRoute(setting, "GET", "/test/a")
		assert.True(t, result)
	})
}
