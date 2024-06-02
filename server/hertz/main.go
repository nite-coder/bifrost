package main

import (
	"math"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/hertz-contrib/reverseproxy"
)

func withDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

func main() {

	hzOpts := []config.Option{
		server.WithHostPorts(":8200"),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithSenseClientDisconnection(true),
		server.WithSenseClientDisconnection(true),
		withDefaultServerHeader(true),
	}

	h := server.New(hzOpts...)

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

	//proxy, _ := gateway.NewSingleHostReverseProxy("http://localhost:8000", defaultClientOptions...)
	proxy, _ := reverseproxy.NewSingleHostReverseProxy("http://localhost:8000", defaultClientOptions...)
	h.POST("/spot/orders", proxy.ServeHTTP)
	h.Spin()
}
