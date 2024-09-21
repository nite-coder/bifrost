package http

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol"
)

// Reverse proxy tests.
const (
	fakeHopHeader       = "X-Fake-Hop-Header-For-Test"
	fakeConnectionToken = "X-Fake-Connection-Token"
)

func init() {
	hopHeaders = append(hopHeaders, fakeHopHeader)
	hopHeaders = append(hopHeaders, fakeConnectionToken)
}

func TestReverseProxy(t *testing.T) {
	const backendResponse = "I am the backend"
	const backendStatus = 404
	serv := server.New(
		server.WithHostPorts("127.0.0.1:9990"),
		server.WithExitWaitTime(1*time.Second),
	)

	// client request: /backend
	// updatream: /proxy/backend

	serv.GET("/proxy/backend", func(cc context.Context, ctx *app.RequestContext) {
		if ctx.Query("mode") == "hangup" {
			ctx.GetConn().Close()
			return
		}
		if ctx.Request.Header.Get("X-Forwarded-For") == "" {
			t.Errorf("didn't get X-Forwarded-For header")
		}
		if c := ctx.Request.Header.Get("Connection"); c != "" {
			t.Errorf("handler got Connection header value %q", c)
		}

		if c := ctx.Request.Header.Get("Upgrade"); c != "" {
			t.Errorf("handler got Upgrade header value %q", c)
		}
		if c := ctx.Request.Header.Get("Proxy-Connection"); c != "" {
			t.Errorf("handler got Proxy-Connection header value %q", c)
		}

		ctx.Response.Header.Set("Trailers", "not a special header field name")
		ctx.Response.Header.Set("Trailer", "X-Trailer")
		ctx.Response.Header.Set("X-Foo", "bar")
		ctx.Response.Header.Set("Upgrade", "foo")
		ctx.Response.Header.Set(fakeHopHeader, "foo")
		ctx.Response.Header.Add("X-Multi-Value", "foo")
		ctx.Response.Header.Add("X-Multi-Value", "bar")
		c := protocol.AcquireCookie()
		c.SetKey("flavor")
		c.SetValue("chocolateChip")
		ctx.Response.Header.SetCookie(c)
		protocol.ReleaseCookie(c)
		ctx.Response.Header.Set("X-Trailer", "trailer_value")
		ctx.Response.Header.Set(http.TrailerPrefix+"X-Unannounced-Trailer", "unannounced_trailer_value")
		ctx.Data(backendStatus, "application/json", []byte(backendResponse))
	})

	proxyOptions := Options{
		Target:   "http://127.0.0.1:9990/proxy",
		Protocol: config.ProtocolHTTP,
		Weight:   1,
	}

	proxy, err := New(proxyOptions, nil)
	if err != nil {
		t.Errorf("proxy error: %v", err)
	}

	serv.GET("/backend", func(c context.Context, ctx *app.RequestContext) {
		proxy.ServeHTTP(c, ctx)
	})
	go serv.Spin()
	time.Sleep(time.Second)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = serv.Shutdown(ctx)
	}()

	cli, _ := client.NewClient()
	req := protocol.AcquireRequest()
	res := protocol.AcquireResponse()
	req.Header.Set("Connection", "close, TE")
	req.Header.Add("Te", "foo")
	req.Header.Add("Te", "bar, trailers")
	req.Header.Set("Proxy-Connection", "should be deleted")
	req.Header.Set("Upgrade", "foo")
	req.SetConnectionClose()
	req.SetHost("some-name")
	req.SetRequestURI("http://localhost:9990/backend")
	_ = cli.Do(context.Background(), req, res)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if g, e := res.StatusCode(), backendStatus; g != e {
		t.Errorf("got res.StatusCode %d; expected %d", g, e)
	}
	if g, e := res.Header.Get("X-Foo"), "bar"; g != e {
		t.Errorf("got X-Foo %q; expected %q", g, e)
	}
	if c := res.Header.Get(fakeHopHeader); c != "" {
		t.Errorf("got %s header value %q", fakeHopHeader, c)
	}
	if g, e := res.Header.Get("Trailers"), "not a special header field name"; g != e {
		t.Errorf("header Trailers = %q; want %q", g, e)
	}
	length := 0
	res.Header.VisitAll(func(key, value []byte) {
		if string(key) == "X-Multi-Value" {
			length++
		}
	})
	if length != 2 {
		t.Errorf("got %d X-Multi-Value header values; expected %d", 2, length)
	}
	length = 0
	res.Header.VisitAll(func(key, value []byte) {
		if string(key) == "Set-Cookie" {
			length++
		}
	})
	if length != 1 {
		t.Fatalf("got %d SetCookies, want %d", 1, 0)
	}

	cookie := protocol.AcquireCookie()
	cookie.SetKey("flavor")
	if has := res.Header.Cookie(cookie); !has {
		t.Errorf("unexpected cookie %q", cookie)
	}
	if g, e := string(res.Body()), backendResponse; g != e {
		t.Errorf("got body %q; expected %q", g, e)
	}

	// Test that a backend failing to be reached or one which doesn't return
	// a response results in a StatusBadGateway.
	req.SetRequestURI("http://localhost:9990/backend?mode=hangup")
	_ = cli.Do(context.Background(), req, res)
	if res.StatusCode() != http.StatusBadGateway {
		t.Errorf("request to bad proxy = %v; want 502 StatusBadGateway", res.StatusCode())
	}
}

func TestReverseProxyStripHeadersPresentInConnection(t *testing.T) {
	hopHeaders = append(hopHeaders, fakeHopHeader)
	const backendResponse = "I am the backend"

	// someConnHeader is some arbitrary header to be declared as a hop-by-hop header
	// in the Request's Connection header.
	const someConnHeader = "X-Some-Conn-Header"
	r := server.New(
		server.WithHostPorts("127.0.0.1:9991"),
		server.WithExitWaitTime(1*time.Second),
	)

	r.GET("/proxy/backend", func(cc context.Context, ctx *app.RequestContext) {
		if c := ctx.Request.Header.Get("Connection"); c != "" {
			t.Errorf("handler got header %q = %q; want empty", "Connection", c)
		}
		if c := ctx.Request.Header.Get(fakeConnectionToken); c != "" {
			t.Errorf("handler got header %q = %q; want empty", fakeConnectionToken, c)
		}
		if c := ctx.Request.Header.Get(someConnHeader); c != "" {
			t.Errorf("handler got header %q = %q; want empty", someConnHeader, c)
		}
		ctx.Response.Header.Add("Connection", "Upgrade, "+fakeConnectionToken)
		ctx.Response.Header.Add("Connection", someConnHeader)
		ctx.Response.Header.Set(someConnHeader, "should be deleted")
		ctx.Response.Header.Set(fakeConnectionToken, "should be deleted")
		ctx.Data(200, "application/json", []byte(backendResponse))
	})

	proxyOptions := Options{
		Target:   "http://127.0.0.1:9991/proxy",
		Protocol: config.ProtocolHTTP,
		Weight:   1,
	}
	proxy, err := New(proxyOptions, nil)

	if err != nil {
		t.Errorf("proxy error: %v", err)
	}

	r.GET("/backend", func(cc context.Context, ctx *app.RequestContext) {
		proxy.ServeHTTP(cc, ctx)
	})
	go r.Spin()
	time.Sleep(time.Second)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = r.Shutdown(ctx)
	}()

	cli, _ := client.NewClient()
	req := protocol.AcquireRequest()
	resp := protocol.AcquireResponse()
	req.Header.Set(someConnHeader, "should be deleted")
	req.Header.Add("Connection", "Upgrade, "+fakeConnectionToken)
	req.Header.Add("Connection", someConnHeader)
	req.Header.Set(fakeConnectionToken, "should be deleted")
	req.SetRequestURI("http://localhost:9991/backend")
	_ = cli.Do(context.Background(), req, resp)

	if got, want := string(resp.Body()), backendResponse; got != want {
		t.Errorf("got body %q; want %q", got, want)
	}
	if c := resp.Header.Get("Connection"); c != "" {
		t.Errorf("handler got header %q = %q; want empty", "Connection", c)
	}
	if c := resp.Header.Get(someConnHeader); c != "" {
		t.Errorf("handler got header %q = %q; want empty", someConnHeader, c)
	}
	if c := resp.Header.Get(fakeConnectionToken); c != "" {
		t.Errorf("handler got header %q = %q; want empty", fakeConnectionToken, c)
	}
}

func TestReverseProxyStripEmptyConnection(t *testing.T) {
	const backendResponse = "I am the backend"

	// someConnHeader is some arbitrary header to be declared as a hop-by-hop header
	// in the Request's Connection header.
	const someConnHeader = "X-Some-Conn-Header"
	r := server.New(
		server.WithHostPorts("127.0.0.1:9992"),
		server.WithExitWaitTime(1*time.Second),
	)

	r.GET("/proxy/backend", func(cc context.Context, ctx *app.RequestContext) {
		if c := ctx.Request.Header.Get("Connection"); c != "" {
			t.Errorf("handler got header %q = %v; want empty", "Connection", c)
		}
		if c := ctx.Request.Header.Get(someConnHeader); c != "" {
			t.Errorf("handler got header %q = %q; want empty", someConnHeader, c)
		}
		ctx.Response.Header.Add("Connection", "")
		ctx.Response.Header.Add("Connection", someConnHeader)
		ctx.Response.Header.Set(someConnHeader, "should be deleted")
		ctx.Data(200, "application/json", []byte(backendResponse))
	})

	proxyOptions := Options{
		Target:   "http://127.0.0.1:9992/proxy",
		Protocol: config.ProtocolHTTP,
		Weight:   1,
	}
	proxy, err := New(proxyOptions, nil)

	if err != nil {
		t.Errorf("proxy error: %v", err)
	}
	r.GET("/backend", func(cc context.Context, ctx *app.RequestContext) {
		proxy.ServeHTTP(cc, ctx)
	})
	go r.Spin()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = r.Shutdown(ctx)
	}()

	time.Sleep(time.Second)
	cli, _ := client.NewClient()
	req := protocol.AcquireRequest()
	resp := protocol.AcquireResponse()

	req.Header.Add("Connection", "")
	req.Header.Add("Connection", someConnHeader)
	req.Header.Set(someConnHeader, "should be deleted")
	req.SetRequestURI("http://localhost:9992/backend")
	_ = cli.Do(context.Background(), req, resp)

	if got, want := string(resp.Body()), backendResponse; got != want {
		t.Errorf("got body %q; want %q", got, want)
	}
	if c := resp.Header.Get("Connection"); c != "" {
		t.Errorf("handler got header %q = %q; want empty", "Connection", c)
	}
	if c := resp.Header.Get(someConnHeader); c != "" {
		t.Errorf("handler got header %q = %q; want empty", someConnHeader, c)
	}
}

func TestXForwardedFor(t *testing.T) {
	const prevForwardedFor = "client ip"
	const backendResponse = "I am the backend"
	const backendStatus = 404
	r := server.New(
		server.WithHostPorts("127.0.0.1:9993"),
		server.WithExitWaitTime(1*time.Second),
	)

	r.GET("/proxy/backend", func(cc context.Context, ctx *app.RequestContext) {
		if ctx.Request.Header.Get("X-Forwarded-For") == "" {
			t.Errorf("didn't get X-Forwarded-For header")
		}
		if !strings.Contains(ctx.Request.Header.Get("X-Forwarded-For"), prevForwardedFor) {
			t.Errorf("X-Forwarded-For didn't contain prior data")
		}
		ctx.Data(backendStatus, "application/json", []byte(backendResponse))
	})

	proxyOptions := Options{
		Target:   "http://127.0.0.1:9993/proxy",
		Protocol: config.ProtocolHTTP,
		Weight:   1,
	}
	proxy, err := New(proxyOptions, nil)

	if err != nil {
		t.Errorf("proxy error: %v", err)
	}
	r.GET("/backend", proxy.ServeHTTP)
	go r.Spin()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = r.Shutdown(ctx)
	}()

	time.Sleep(time.Second)
	cli, _ := client.NewClient()
	req := protocol.AcquireRequest()
	resp := protocol.AcquireResponse()

	req.Header.Set("Connection", "close")
	req.Header.Set("X-Forwarded-For", prevForwardedFor)
	req.SetConnectionClose()
	req.SetRequestURI("http://localhost:9993/backend")
	_ = cli.Do(context.Background(), req, resp)
	if g, e := resp.StatusCode(), backendStatus; g != e {
		t.Errorf("got res.StatusCode %d; expected %d", g, e)
	}
	if g, e := string(resp.Body()), backendResponse; g != e {
		t.Errorf("got body %q; expected %q", g, e)
	}
}

var proxyQueryTests = []struct {
	baseSuffix string // suffix to add to backend URL
	reqSuffix  string // suffix to add to frontend's request URL
	want       string // what backend should see for final request URL (without ?)
}{
	{"?sta=tic", "", "sta=tic"},
}

func TestReverseProxyQuery(t *testing.T) {
	r := server.New(
		server.WithHostPorts("127.0.0.1:9995"),
		server.WithExitWaitTime(1*time.Second),
	)

	r.GET("/proxy/backend", func(cc context.Context, ctx *app.RequestContext) {
		ctx.Response.Header.Set("X-Got-Query", string(ctx.Request.QueryString()))
		ctx.Data(200, "application/json", []byte("hi"))
	})

	for i, tt := range proxyQueryTests {
		proxyOptions := Options{
			Target:   "http://127.0.0.1:9995/proxy" + tt.baseSuffix,
			Protocol: config.ProtocolHTTP,
			Weight:   1,
		}
		proxy, _ := New(proxyOptions, nil)

		r.GET("/backend", proxy.ServeHTTP)
		go r.Spin()
		defer func() {
			_ = r.Shutdown(context.TODO())
		}()
		time.Sleep(time.Second)
		cli, _ := client.NewClient()
		req := protocol.AcquireRequest()
		resp := protocol.AcquireResponse()
		req.SetRequestURI("http://localhost:9995/backend" + tt.reqSuffix)
		_ = cli.Do(context.Background(), req, resp)
		if g, e := resp.Header.Get("X-Got-Query"), tt.want; g != e {
			t.Errorf("%d. got query %q; expected %q", i, g, e)
		}
	}
}

func TestReverseProxy_Post(t *testing.T) {
	const backendResponse = "I am the backend"
	const backendStatus = 200
	requestBody := bytes.Repeat([]byte("a"), 1<<20)

	r := server.New(
		server.WithHostPorts("127.0.0.1:9996"),
		server.WithExitWaitTime(1*time.Second),
	)

	r.POST("/proxy/backend", func(cc context.Context, ctx *app.RequestContext) {
		sluproxy := ctx.Request.Body()
		if len(sluproxy) != len(requestBody) {
			t.Errorf("Backend read %d request body bytes; want %d", len(sluproxy), len(requestBody))
		}
		if !bytes.Equal(sluproxy, requestBody) {
			t.Error("Backend read wrong request body.") // 1MB; omitting details
		}
		ctx.Data(backendStatus, "application/json", []byte(backendResponse))
	})

	proxyOptions := Options{
		Target:   "http://127.0.0.1:9996/proxy",
		Protocol: config.ProtocolHTTP,
		Weight:   1,
	}
	proxy, _ := New(proxyOptions, nil)

	r.POST("/backend", proxy.ServeHTTP)
	go r.Spin()
	time.Sleep(time.Second)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = r.Shutdown(ctx)
	}()
	cli, _ := client.NewClient()
	req := protocol.AcquireRequest()
	req.SetMethod("POST")
	req.SetBodyRaw(requestBody)
	resp := protocol.AcquireResponse()
	req.SetConnectionClose()
	req.SetRequestURI("http://localhost:9996/backend")
	_ = cli.Do(context.Background(), req, resp)
	if g, e := resp.StatusCode(), backendStatus; g != e {
		t.Errorf("got res.StatusCode %d; expected %d", g, e)
	}
	if g, e := string(resp.Body()), backendResponse; g != e {
		t.Errorf("got body %q; expected %q", g, e)
	}
}
