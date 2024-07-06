package gateway

import (
	"context"
	"http-benchmark/pkg/config"
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

func registerByNodeType(route *Routes, nodeType nodeType) *Routes {
	switch nodeType {
	case nodeTypeExact:
		_ = route.Add(config.RouteOptions{
			Paths: []string{"= /market/btc", "= /"},
		}, exactkHandler)
	case nodeTypePrefix:
		_ = route.Add(config.RouteOptions{
			Paths: []string{"^= /market/btc", "^= /"},
		}, prefixHandler)
	case nodeTypeRegex:
		_ = route.Add(config.RouteOptions{
			Paths: []string{"~ /market/(btc|usdt|eth)$", "~ ^/$"},
		}, regexkHandler)

		_ = route.Add(config.RouteOptions{
			Paths: []string{"~ /market/(btc|usdt|eth)"},
		}, nil) // test two regexs router order
	case nodeTypeGeneral:
		_ = route.Add(config.RouteOptions{
			Paths: []string{"/market/btc", "/"},
		}, generalkHandler)
	}

	return route
}

func TestRoutePriorityAndRoot(t *testing.T) {

	t.Run("exact match", func(t *testing.T) {
		nodeTypes := []nodeType{nodeTypeExact, nodeTypePrefix, nodeTypeGeneral, nodeTypeRegex}
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

	t.Run("prefix match", func(t *testing.T) {
		nodeTypes := []nodeType{nodeTypePrefix, nodeTypeGeneral, nodeTypeRegex}
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

	t.Run("regex match", func(t *testing.T) {
		nodeTypes := []nodeType{nodeTypeGeneral, nodeTypeRegex}
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
	})

	t.Run("general match", func(t *testing.T) {
		nodeTypes := []nodeType{nodeTypeGeneral}
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

}

func TestRootRoute(t *testing.T) {
	route := newRoutes()

	_ = route.Add(config.RouteOptions{
		Paths: []string{"= /"},
		//Methods: []string{"POST"},
	}, exactkHandler)

	_ = route.Add(config.RouteOptions{
		Paths: []string{"^= /"},
		//Methods: []string{"POST"},
	}, prefixHandler)

	_ = route.Add(config.RouteOptions{
		Paths: []string{"/"},
		//Methods: []string{"POST"},
	}, generalkHandler)

	c := app.NewContext(0)
	c.Request.SetMethod("POST")
	c.Request.URI().SetPath("/")

	route.ServeHTTP(context.Background(), c)
	statusCode := c.Response.StatusCode()

	if statusCode != 201 {
		t.Errorf("Expected %v for path %s, but got %v", 201, "/", statusCode)
	}
}

func TestRoutes(t *testing.T) {
	route := newRoutes()

	err := route.Add(config.RouteOptions{
		Paths: []string{"^= /market/btc", "^= /spot"},
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

		route.ServeHTTP(context.Background(), c)
		statusCode := c.Response.StatusCode()

		if statusCode != tc.expectedResult {
			t.Errorf("Expected %v for path %s, but got %v", tc.expectedResult, tc.path, statusCode)
		}
	}
}

func TestDuplicateHTTPMethods(t *testing.T) {
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
	assert.ErrorIs(t, err, ErrAlreadyExists)

	err = route.Add(config.RouteOptions{
		Methods: []string{},
		Paths:   []string{"/foo"},
	}, exactkHandler)
	assert.NoError(t, err)
}
