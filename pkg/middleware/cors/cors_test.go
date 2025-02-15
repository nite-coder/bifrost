/*
 * The MIT License (MIT)
 *
 * Copyright (c) 2016 Gin-Gonic
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.

 * This file may have been modified by CloudWeGo authors. All CloudWeGo
 * Modifications are Copyright 2022 CloudWeGo Authors.
 */

package cors

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/test/assert"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
)

func newTestRouter(c Config) *route.Engine {
	router := route.NewEngine(config.NewOptions([]config.Option{}))
	router.Use(NewMiddleware(c))
	router.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "get")
	})
	router.POST("/", func(ctx context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "post")
	})
	router.PATCH("/", func(ctx context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "patch")
	})

	return router
}

func performRequest(r *route.Engine, method, origin string, headers ...ut.Header) *ut.ResponseRecorder {
	url := "/"
	for _, h := range headers {
		if h.Key == "Host" {
			url = DefaultSchemas[0] + h.Value + url
		}
	}
	if len(origin) > 0 {
		headers = append(headers, ut.Header{Key: "Origin", Value: origin})
	}

	return ut.PerformRequest(r, method, url, nil, headers...)
}

func TestConfigAddAllow(t *testing.T) {
	config := Config{}
	config.AddAllowMethods("POST")
	config.AddAllowMethods("GET", "PUT")
	config.AddExposeHeaders()

	config.AddAllowHeaders("Some", " cool")
	config.AddAllowHeaders("header")
	config.AddExposeHeaders()

	config.AddExposeHeaders()
	config.AddExposeHeaders("exposed", "header")
	config.AddExposeHeaders("hey")

	assert.DeepEqual(t, config.AllowMethods, []string{"POST", "GET", "PUT"})
	assert.DeepEqual(t, config.AllowHeaders, []string{"Some", " cool", "header"})
	assert.DeepEqual(t, config.ExposeHeaders, []string{"exposed", "header", "hey"})
}

func TestBadConfig(t *testing.T) {
	assert.Panic(t, func() { NewMiddleware(Config{}) })
	assert.Panic(t, func() {
		NewMiddleware(Config{
			AllowAllOrigins: true,
			AllowOrigins:    []string{"http://google.com"},
		})
	})
	assert.Panic(t, func() {
		NewMiddleware(Config{
			AllowAllOrigins: true,
			AllowOriginFunc: func(origin string) bool { return false },
		})
	})
	assert.Panic(t, func() {
		NewMiddleware(Config{
			AllowOrigins: []string{"google.com"},
		})
	})
}

func TestNormalize(t *testing.T) {
	values := normalize([]string{
		"http-Access ", "Post", "POST", " poSt  ",
		"HTTP-Access", "",
	})
	assert.DeepEqual(t, values, []string{"http-access", "post", ""})

	values = normalize(nil)
	assert.Nil(t, values)

	values = normalize([]string{})
	assert.DeepEqual(t, values, []string{})
}

func TestConvert(t *testing.T) {
	methods := []string{"Get", "GET", "get"}

	assert.DeepEqual(t, []string{"GET", "GET", "GET"}, convert(methods, strings.ToUpper))
}

func TestGenerateNormalHeaders_AllowAllOrigins(t *testing.T) {
	header := generateNormalHeaders(Config{
		AllowAllOrigins: false,
	})
	assert.DeepEqual(t, header["Access-Control-Allow-Origin"], "")
	assert.DeepEqual(t, header["Vary"], "Origin")
	assert.DeepEqual(t, 1, len(header))

	header = generateNormalHeaders(Config{
		AllowAllOrigins: true,
	})
	assert.DeepEqual(t, header["Access-Control-Allow-Origin"], "*")
	assert.DeepEqual(t, header["Vary"], "")
	assert.DeepEqual(t, 1, len(header))
}

func TestGenerateNormalHeaders_AllowCredentials(t *testing.T) {
	header := generateNormalHeaders(Config{
		AllowCredentials: true,
	})
	assert.DeepEqual(t, header["Access-Control-Allow-Credentials"], "true")
	assert.DeepEqual(t, header["Vary"], "Origin")
	assert.DeepEqual(t, 2, len(header))
}

func TestGenerateNormalHeaders_ExposedHeaders(t *testing.T) {
	header := generateNormalHeaders(Config{
		ExposeHeaders: []string{"X-user", "xPassword"},
	})
	assert.DeepEqual(t, header["Access-Control-Expose-Headers"], "X-User,Xpassword")
	assert.DeepEqual(t, header["Vary"], "Origin")
	assert.DeepEqual(t, 2, len(header))
}

func TestGeneratePreflightHeaders(t *testing.T) {
	header := generatePreflightHeaders(Config{
		AllowAllOrigins: false,
	})
	assert.DeepEqual(t, header["Access-Control-Allow-Origin"], "")
	assert.DeepEqual(t, header["Vary"], "Origin")
	assert.DeepEqual(t, 1, len(header))

	header = generateNormalHeaders(Config{
		AllowAllOrigins: true,
	})
	assert.DeepEqual(t, header["Access-Control-Allow-Origin"], "*")
	assert.DeepEqual(t, header["Vary"], "")
	assert.DeepEqual(t, 1, len(header))
}

func TestGeneratePreflightHeaders_AllowCredentials(t *testing.T) {
	header := generatePreflightHeaders(Config{
		AllowCredentials: true,
	})
	assert.DeepEqual(t, header["Access-Control-Allow-Credentials"], "true")
	assert.DeepEqual(t, header["Vary"], "Origin")
	assert.DeepEqual(t, 2, len(header))
}

func TestGeneratePreflightHeaders_AllowMethods(t *testing.T) {
	header := generatePreflightHeaders(Config{
		AllowMethods: []string{"GET ", "post", "PUT", " put  "},
	})
	assert.DeepEqual(t, header["Access-Control-Allow-Methods"], "GET,POST,PUT")
	assert.DeepEqual(t, header["Vary"], "Origin")
	assert.DeepEqual(t, 2, len(header))
}

func TestGeneratePreflightHeaders_AllowHeaders(t *testing.T) {
	header := generatePreflightHeaders(Config{
		AllowHeaders: []string{"X-user", "Content-Type"},
	})
	assert.DeepEqual(t, header["Access-Control-Allow-Headers"], "X-User,Content-Type")
	assert.DeepEqual(t, header["Vary"], "Origin")
	assert.DeepEqual(t, 2, len(header))
}

func TestGeneratePreflightHeaders_MaxAge(t *testing.T) {
	header := generatePreflightHeaders(Config{
		MaxAge: 12 * time.Hour,
	})
	assert.DeepEqual(t, header["Access-Control-Max-Age"], "43200") // 12*60*60
	assert.DeepEqual(t, header["Vary"], "Origin")
	assert.DeepEqual(t, 2, len(header))
}

func TestValidateOrigin(t *testing.T) {
	cors := newCors(Config{
		AllowAllOrigins: true,
	})
	assert.True(t, cors.validateOrigin("http://google.com"))
	assert.True(t, cors.validateOrigin("https://google.com"))
	assert.True(t, cors.validateOrigin("example.com"))
	assert.True(t, cors.validateOrigin("chrome-extension://random-extension-id"))

	cors = newCors(Config{
		AllowOrigins: []string{"https://google.com", "https://github.com"},
		AllowOriginFunc: func(origin string) bool {
			return (origin == "http://abcdefghijklmnopqrstuvwxyz")
		},
		AllowBrowserExtensions: true,
	})
	assert.False(t, cors.validateOrigin("http://google.com"))
	assert.True(t, cors.validateOrigin("https://google.com"))
	assert.True(t, cors.validateOrigin("https://github.com"))
	assert.True(t, cors.validateOrigin("http://abcdefghijklmnopqrstuvwxyz"))
	assert.False(t, cors.validateOrigin("http://example.com"))
	assert.False(t, cors.validateOrigin("google.com"))
	assert.False(t, cors.validateOrigin("chrome-extension://random-extension-id"))

	cors = newCors(Config{
		AllowOrigins: []string{"https://google.com", "https://github.com"},
	})
	assert.False(t, cors.validateOrigin("chrome-extension://random-extension-id"))
	assert.False(t, cors.validateOrigin("file://some-dangerous-file.js"))
	assert.False(t, cors.validateOrigin("wss://socket-connection"))

	cors = newCors(Config{
		AllowOrigins:           []string{"chrome-extension://*", "safari-extension://my-extension-*-app", "*.some-domain.com"},
		AllowBrowserExtensions: true,
		AllowWildcard:          true,
	})
	assert.True(t, cors.validateOrigin("chrome-extension://random-extension-id"))
	assert.True(t, cors.validateOrigin("chrome-extension://another-one"))
	assert.True(t, cors.validateOrigin("safari-extension://my-extension-one-app"))
	assert.True(t, cors.validateOrigin("safari-extension://my-extension-two-app"))
	assert.False(t, cors.validateOrigin("moz-extension://ext-id-we-not-allow"))
	assert.True(t, cors.validateOrigin("http://api.some-domain.com"))
	assert.False(t, cors.validateOrigin("http://api.another-domain.com"))

	cors = newCors(Config{
		AllowOrigins:    []string{"file://safe-file.js", "wss://some-sessions-layer-connection"},
		AllowFiles:      true,
		AllowWebSockets: true,
	})
	assert.True(t, cors.validateOrigin("file://safe-file.js"))
	assert.False(t, cors.validateOrigin("file://some-dangerous-file.js"))
	assert.True(t, cors.validateOrigin("wss://some-sessions-layer-connection"))
	assert.False(t, cors.validateOrigin("ws://not-what-we-expected"))

	cors = newCors(Config{
		AllowOrigins: []string{"*"},
	})
	assert.True(t, cors.validateOrigin("http://google.com"))
	assert.True(t, cors.validateOrigin("https://google.com"))
	assert.True(t, cors.validateOrigin("example.com"))
	assert.True(t, cors.validateOrigin("chrome-extension://random-extension-id"))
}

func TestPassesAllowOrigins(t *testing.T) {
	router := newTestRouter(Config{
		AllowOrigins:     []string{"http://google.com"},
		AllowMethods:     []string{" GeT ", "get", "post", "PUT  ", "Head", "POST"},
		AllowHeaders:     []string{"Content-type", "timeStamp "},
		ExposeHeaders:    []string{"Data", "x-User"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://github.com"
		},
	})

	// no CORS request, origin == ""
	w := performRequest(router, "GET", "")
	assert.DeepEqual(t, "get", w.Body.String())
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Expose-Headers"))

	// no CORS request, origin == host
	var h []ut.Header
	h = append(h, ut.Header{Key: "Host", Value: "facebook.com"})
	w = performRequest(router, "GET", "http://facebook.com", h...)
	assert.DeepEqual(t, "get", w.Body.String())
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Expose-Headers"))

	// no CORS request, origin schema != host schema
	w = performRequest(router, "GET", "https://facebook.com", h...)
	assert.DeepEqual(t, "get", w.Body.String())
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Expose-Headers"))

	// allowed CORS request from func
	w = performRequest(router, "GET", "http://github.com", h...)
	assert.DeepEqual(t, "get", w.Body.String())
	assert.DeepEqual(t, "http://github.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "Data,X-User", w.Header().Get("Access-Control-Expose-Headers"))

	// allowed CORS request from allowOrigins
	w = performRequest(router, "GET", "http://google.com", h...)
	assert.DeepEqual(t, "get", w.Body.String())
	assert.DeepEqual(t, "http://google.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "Data,X-User", w.Header().Get("Access-Control-Expose-Headers"))

	// deny CORS request
	w = performRequest(router, "GET", "https://dummy.com", h...)
	assert.DeepEqual(t, consts.StatusForbidden, w.Code)
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Expose-Headers"))

	// allowed CORS prefligh request
	w = performRequest(router, "OPTIONS", "http://github.com", h...)
	assert.DeepEqual(t, consts.StatusNoContent, w.Code)
	assert.DeepEqual(t, "http://github.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "GET,POST,PUT,HEAD", w.Header().Get("Access-Control-Allow-Methods"))
	assert.DeepEqual(t, "Content-Type,Timestamp", w.Header().Get("Access-Control-Allow-Headers"))
	assert.DeepEqual(t, "43200", w.Header().Get("Access-Control-Max-Age"))

	// deny CORS prefligh request
	w = performRequest(router, "OPTIONS", "http://example.com", h...)
	assert.DeepEqual(t, consts.StatusForbidden, w.Code)
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Methods"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Headers"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Max-Age"))
}

func TestPassesAllowAllOrigins(t *testing.T) {
	router := newTestRouter(Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{" Patch ", "get", "post", "POST"},
		AllowHeaders:     []string{"Content-type", "  testheader "},
		ExposeHeaders:    []string{"Data2", "x-User2"},
		AllowCredentials: false,
		MaxAge:           10 * time.Hour,
	})

	// no CORS request, origin == ""
	w := performRequest(router, "GET", "")
	assert.DeepEqual(t, "get", w.Body.String())
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Expose-Headers"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))

	// allowed CORS request
	w = performRequest(router, "POST", "example.com")
	assert.DeepEqual(t, "post", w.Body.String())
	assert.DeepEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "Data2,X-User2", w.Header().Get("Access-Control-Expose-Headers"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// allowed CORS prefligh request
	w = performRequest(router, "OPTIONS", "https://facebook.com")
	assert.DeepEqual(t, consts.StatusNoContent, w.Code)
	assert.DeepEqual(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "PATCH,GET,POST", w.Header().Get("Access-Control-Allow-Methods"))
	assert.DeepEqual(t, "Content-Type,Testheader", w.Header().Get("Access-Control-Allow-Headers"))
	assert.DeepEqual(t, "36000", w.Header().Get("Access-Control-Max-Age"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestWildcard(t *testing.T) {
	router := newTestRouter(Config{
		AllowOrigins:  []string{"https://*.github.com", "https://api.*", "http://*", "https://facebook.com", "*.golang.org"},
		AllowMethods:  []string{"GET"},
		AllowWildcard: true,
	})

	w := performRequest(router, "GET", "https://gist.github.com")
	assert.DeepEqual(t, 200, w.Code)

	w = performRequest(router, "GET", "https://api.github.com/v1/users")
	assert.DeepEqual(t, 200, w.Code)

	w = performRequest(router, "GET", "https://giphy.com/")
	assert.DeepEqual(t, 403, w.Code)

	w = performRequest(router, "GET", "http://hard-to-find-http-example.com")
	assert.DeepEqual(t, 200, w.Code)

	w = performRequest(router, "GET", "https://facebook.com")
	assert.DeepEqual(t, 200, w.Code)

	w = performRequest(router, "GET", "https://something.golang.org")
	assert.DeepEqual(t, 200, w.Code)

	w = performRequest(router, "GET", "https://something.go.org")
	assert.DeepEqual(t, 403, w.Code)

	router = newTestRouter(Config{
		AllowOrigins: []string{"https://github.com", "https://facebook.com"},
		AllowMethods: []string{"GET"},
	})

	w = performRequest(router, "GET", "https://gist.github.com")
	assert.DeepEqual(t, 403, w.Code)

	w = performRequest(router, "GET", "https://github.com")
	assert.DeepEqual(t, 200, w.Code)
}

func TestPassesAllowOrigins2(t *testing.T) {
	router := newTestRouter(Config{
		AllowOrigins:     []string{"http://google.com"},
		AllowMethods:     []string{" GeT ", "get", "post", "PUT  ", "Head", "POST"},
		AllowHeaders:     []string{"Content-type", "timeStamp "},
		ExposeHeaders:    []string{"Data", "x-User"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://github.com"
		},
	})

	var h []ut.Header
	h = append(h, ut.Header{Key: "Host", Value: "facebook.com"})

	// deny CORS request
	w := performRequest(router, "GET", "https://google.com", h...)
	assert.DeepEqual(t, consts.StatusForbidden, w.Code)
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Origin"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.DeepEqual(t, "", w.Header().Get("Access-Control-Expose-Headers"))
}
