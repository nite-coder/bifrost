package main

import (
	"context"
	"fmt"
	"time"

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
	engine := route.NewEngine(options)

	router1 := NewRouter()

	// match /futures/usdt/orders
	router1.Regexp("^/futures/(usdt|btc)/orders$", upstream.ServeHTTP)

	router1.Use(func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("bifrst: not found")
	})

	// router2 := NewRouter()
	// router2.Use(func(c context.Context, ctx *app.RequestContext) {
	// 	ctx.String(200, "router2")
	// })

	// ticker := time.NewTicker(2 * time.Second)

	// go func() {
	// 	use2 := true
	// 	for {
	// 		<-ticker.C
	// 		if use2 {
	// 			switcher.SetRouter(router2)
	// 			use2 = false
	// 		} else {
	// 			switcher.SetRouter(router1)
	// 			use2 = true
	// 		}
	// 	}
	// }()

	// engine.POST("/", echoHandler)
	// engine.POST("/spot/orders", placeOrderHandler, func(c context.Context, ctx *app.RequestContext) {
	// 	fmt.Println("1.1")
	// })

	// // match /futures/orders
	// engine.Use(func(c context.Context, ctx *app.RequestContext) {
	// 	//fmt.Println("f.1")
	// 	if !bytes.HasPrefix(ctx.Request.Path(), futuresOrderPath) {
	// 		return
	// 	}

	// 	placeOrderHandler(c, ctx)
	// 	ctx.Abort()
	// })

	// // match /futures/usdt/orders
	// re, _ := regexp.Compile(`^/futures/(usdt|btc)/orders$`)
	// engine.Use(func(c context.Context, ctx *app.RequestContext) {

	// 	if !re.MatchString(string(ctx.Request.Path())) {
	// 		return
	// 	}

	// 	placeOrderHandler(c, ctx)
	// 	ctx.Abort()
	// })

	// engine.Use(func(c context.Context, ctx *app.RequestContext) {
	// 	fmt.Println("done")
	// })

	switcher := NewSwitcher(router1)
	engine.Use(switcher.ServeHTTP)
	h.Engine = engine
	h.Spin()
}

func echoHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.Response.SetBody(ctx.Request.Body())
}
