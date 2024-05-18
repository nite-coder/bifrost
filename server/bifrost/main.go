package main

import (
	"context"
	"http-benchmark/gateway"
	"http-benchmark/middleware"
	"os"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/route"
	hertzslog "github.com/hertz-contrib/logger/slog"
)

const (
	port = ":8001"
)

func WithDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

func main() {

	//gopool.SetCap(20000)

	logger := hertzslog.NewLogger(hertzslog.WithOutput(os.Stdout))
	hlog.SetLogger(logger)

	// upstreams
	loadBalancerOpts := gateway.LoadBalancerOptions{
		ID: "default",
		Instances: []gateway.Instance{
			{
				Address: "http://127.0.0.1:8000",
				Weight:  1,
			},
		},
	}
	upstream := gateway.NewLoadBalancer(loadBalancerOpts)

	// routes
	router := gateway.NewRouter()

	_ = router.AddRoute(gateway.Route{
		Match: "/spot/orders",
	}, upstream.ServeHTTP)

	_ = router.AddRoute(gateway.Route{
		Match: "/options*",
	}, upstream.ServeHTTP)

	_ = router.AddRoute(gateway.Route{
		Match: "~ ^/futures/(usdt|btc)/orders$",
	}, upstream.ServeHTTP)

	// middleware
	timeM := middleware.NewTimeMiddleware()

	// bifrost engine
	bengine := gateway.NewEngine()
	bengine.Use(timeM.ServeHTTP)
	bengine.Use(router.ServeHTTP)

	switcher := gateway.NewSwitcher(bengine)

	// hertz server
	opts := []config.Option{
		server.WithHostPorts(port),
		server.WithIdleTimeout(time.Second * 60),
		server.WithReadTimeout(time.Second * 3),
		server.WithWriteTimeout(time.Second * 3),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		WithDefaultServerHeader(true),
	}
	h := server.Default(opts...)

	options := config.NewOptions(opts)
	hengine := route.NewEngine(options)
	hengine.Use(switcher.ServeHTTP)
	h.Engine = hengine
	h.Spin()
}

func echoHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.Response.SetBody(ctx.Request.Body())
}
