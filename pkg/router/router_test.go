package router

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func exactHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetStatusCode(201)
}

func prefixHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetStatusCode(202)
}

func generalkHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetStatusCode(204)
}

func loadStaticRouter() *Router {
	router := NewRouter()
	_ = router.Add("GET", "/", Prefix, exactHandler)
	_ = router.Add("GET", "/foo", Prefix, exactHandler)
	_ = router.Add("GET", "/foo/bar/baz/", Prefix, exactHandler)
	_ = router.Add("GET", "/foo/bar/baz/qux/quux", Prefix, exactHandler)
	_ = router.Add("GET", "/foo/bar/baz/qux/quux/corge/grault/garply/waldo/fred", Prefix, exactHandler)
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

type RouteOptions struct {
	Methods []string `yaml:"methods" json:"methods"`
	Paths   []string `yaml:"paths" json:"paths"`
}

func BenchmarkCode(b *testing.B) {
	method := "GET"
	// path1 := "/foo"
	// path5 := "/foo/bar/baz/qux/quux"
	path10 := "/foo/bar/baz/qux/quux/corge/grault/garply/waldo/fred"
	// prefix := "/foo/bar/baz/qux/quux"

	routeSetting := RouteOptions{
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
				return exactHandler
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
		_, _ = router.Find(method, path)
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

func TestRouter(t *testing.T) {
	router := NewRouter()

	router.Add(http.MethodGet, "/", Prefix, generalkHandler)
	router.Add(http.MethodPost, "/orders/123", PreferentialPrefix, prefixHandler)
	router.Add(http.MethodPut, "/foo", Exact, exactHandler)

	middlewares, isDefered := router.Find(http.MethodGet, "/")
	assert.True(t, isDefered)
	assert.Len(t, middlewares, 1)

	middlewares, isDefered = router.Find(http.MethodPut, "/foo")
	assert.False(t, isDefered)
	assert.Len(t, middlewares, 1)

	middlewares, isDefered = router.Find(http.MethodPost, "/orders/123")
	assert.False(t, isDefered)
	assert.Len(t, middlewares, 1)
}

func TestDuplicatedRoutes(t *testing.T) {
	router := NewRouter()

	router.Add(http.MethodGet, "/foo", Prefix, generalkHandler)
	router.Add(http.MethodGet, "/foo", Exact, exactHandler)

	middlewares, isDefered := router.Find(http.MethodGet, "/foo")
	assert.False(t, isDefered)
	assert.Len(t, middlewares, 1)
	assert.True(t, reflect.ValueOf(middlewares[0]).Pointer() == reflect.ValueOf(exactHandler).Pointer())
}

func TestIsValidHTTPMethod(t *testing.T) {
	assert.True(t, IsValidHTTPMethod(http.MethodGet))
	assert.True(t, IsValidHTTPMethod(http.MethodPost))
	assert.True(t, IsValidHTTPMethod(http.MethodPut))
	assert.True(t, IsValidHTTPMethod(http.MethodDelete))
	assert.True(t, IsValidHTTPMethod(http.MethodPatch))
	assert.True(t, IsValidHTTPMethod(http.MethodHead))
	assert.True(t, IsValidHTTPMethod(http.MethodOptions))
	assert.True(t, IsValidHTTPMethod(http.MethodTrace))
	assert.True(t, IsValidHTTPMethod(http.MethodConnect))

	assert.False(t, IsValidHTTPMethod("foo"))
}
