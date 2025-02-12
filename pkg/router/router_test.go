package router

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
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

func loadStaticRouter() *Router {
	router := NewRouter()
	_ = router.Add("GET", "/", General, exactkHandler)
	_ = router.Add("GET", "/foo", General, exactkHandler)
	_ = router.Add("GET", "/foo/bar/baz/", General, exactkHandler)
	_ = router.Add("GET", "/foo/bar/baz/qux/quux", General, exactkHandler)
	_ = router.Add("GET", "/foo/bar/baz/qux/quux/corge/grault/garply/waldo/fred", General, exactkHandler)
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
