package main

import (
	"context"
	"fmt"
	"time"
	"unsafe"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/route"
)

const (
	port        = ":8001"
	debugPort   = ":18001"
	actionQuery = "action"
	order       = `{
		"id": "1852454420",
		"text": "t-abc123",
		"amend_text": "-",
		"create_time": "1710488334",
		"update_time": "1710488334",
		"create_time_ms": 1710488334073,
		"update_time_ms": 1710488334074,
		"status": "closed",
		"currency_pair": "BTC_USDT",
		"type": "limit",
		"account": "unified",
		"side": "buy",
		"amount": "0.001",
		"price": "65000",
		"time_in_force": "gtc",
		"iceberg": "0",
		"left": "0",
		"filled_amount": "0.001",
		"fill_price": "63.4693",
		"filled_total": "63.4693",
		"avg_deal_price": "63469.3",
		"fee": "0.00000022",
		"fee_currency": "BTC",
		"point_fee": "0",
		"gt_fee": "0",
		"gt_maker_fee": "0",
		"gt_taker_fee": "0",
		"gt_discount": false,
		"rebated_fee": "0",
		"rebated_fee_currency": "USDT",
		"finish_as": "filled"
	  }`
)

var (
	orderResp        []byte
	orderPath        = []byte(`/place_order`)
	futuresOrderPath = []byte(`/futures/orders`)
	myUpstream       = []byte("http://127.0.0.1:8000")
)

func WithDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

func main() {

	//_ = netpoll.SetNumLoops(1)

	orderResp = []byte(order)

	// setup upstreams
	loadBalancerOpts := LoadBalancerOptions{
		ID: "default",
		Instances: []Instance{
			{
				Address: "http://127.0.0.1:8000",
				Weight:  1,
			},
		},
	}
	upstream := NewLoadBalancer(loadBalancerOpts)

	// setup routes
	router := NewRouter()

	router.AddRoute(Route{
		Match:    "/spot/orders",
		Method:   []string{"POST"},
		Upstream: "spot-order",
		Entry:    []string{"web"},
	}, upstream.ServeHTTP)

	router.Regexp("^/futures/(usdt|btc)/orders$", upstream.ServeHTTP)

	// setup bifrost engine
	bengine := NewEngine()
	bengine.Use(router.ServeHTTP)
	bengine.Use(func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("bifrst: not found")
	})
	switcher := NewSwitcher(bengine)

	// create hertz server
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

func B2s(b []byte) string {
	/* #nosec G103 */
	return *(*string)(unsafe.Pointer(&b))
}
