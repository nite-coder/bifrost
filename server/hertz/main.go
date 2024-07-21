package main

import (
	"context"
	"crypto/tls"
	"math"
	"net"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/cloudwego/hertz/pkg/common/config"
	configHTTP2 "github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
	"github.com/hertz-contrib/reverseproxy"
	"golang.org/x/net/http2"
)

func withDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

func main() {

	hzOpts := []config.Option{
		server.WithHostPorts(":8001"),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithSenseClientDisconnection(true),
		server.WithReadTimeout(time.Second * 60),
		server.WithKeepAlive(true),
		server.WithALPN(true),
		server.WithStreamBody(true),
		server.WithH2C(true),
		withDefaultServerHeader(true),
	}

	h := server.New(hzOpts...)

	http2opts := []configHTTP2.Option{}
	h.AddProtocol("h2", factory.NewServerFactory(http2opts...))

	defaultClientOptions := []config.ClientOption{
		client.WithNoDefaultUserAgentHeader(true),
		client.WithDisableHeaderNamesNormalizing(true),
		client.WithDisablePathNormalizing(true),
		client.WithMaxConnsPerHost(math.MaxInt),
		client.WithDialTimeout(10 * time.Second),
		client.WithClientReadTimeout(10 * time.Second),
		client.WithWriteTimeout(10 * time.Second),
		client.WithKeepAlive(true),
	}

	addr, _ := url.Parse("http://localhost:8003")
	stdProxy := httputil.NewSingleHostReverseProxy(addr)
	stdProxy.Transport = &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	h.Use(func(c context.Context, ctx *app.RequestContext) {
		stdReq, _ := adaptor.GetCompatRequest(&ctx.Request)
		stdRW := adaptor.GetCompatResponseWriter(&ctx.Response)

		stdProxy.ServeHTTP(stdRW, stdReq)
	})

	proxy, _ := reverseproxy.NewSingleHostReverseProxy("http://localhost:8000", defaultClientOptions...)
	h.POST("/spot/orders", proxy.ServeHTTP)

	h.Spin()
}
