package main

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
)

func WithDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

const (
	bind        = ":8000"
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

	orderbook = `{
		"current": 1711255163345,
		"update": 1711255163342,
		"asks": [
			[
				"63837.9",
				"0.25997"
			],
			[
				"63839.4",
				"0.14"
			],
			[
				"63839.5",
				"0.18"
			],
			[
				"63844.6",
				"0.00812"
			],
			[
				"63845.7",
				"0.01256"
			],
			[
				"63846.3",
				"0.02726"
			],
			[
				"63847.8",
				"0.12405"
			],
			[
				"63847.9",
				"0.20674"
			],
			[
				"63850.3",
				"0.14139"
			],
			[
				"63850.4",
				"0.23563"
			]
		],
		"bids": [
			[
				"63837.8",
				"0.96189"
			],
			[
				"63837.6",
				"0.00583"
			],
			[
				"63835.5",
				"0.47803"
			],
			[
				"63833.7",
				"0.06774"
			],
			[
				"63833",
				"0.25504"
			],
			[
				"63832.7",
				"0.02192"
			],
			[
				"63829.3",
				"0.05346"
			],
			[
				"63827",
				"0.10805"
			],
			[
				"63826.9",
				"0.05348"
			],
			[
				"63826.6",
				"0.18"
			]
		]
	}`
)

var (
	orderResp     []byte
	orderbookResp []byte
)

func main() {

	orderResp = []byte(order)
	orderbookResp = []byte(orderbook)

	opts := []config.Option{
		server.WithHostPorts(bind),
		server.WithIdleTimeout(time.Second * 60),
		server.WithReadTimeout(time.Second * 3),
		server.WithWriteTimeout(time.Second * 3),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		WithDefaultServerHeader(true),
	}
	h := server.New(opts...)

	// h.Use(func(c context.Context, ctx *app.RequestContext) {
	// 	//fmt.Println("futures/usdt/orders")
	// })
	h.POST("/", echoHandler)
	h.Any("/spot/order", placeOrderHandler)
	h.POST("/spot/orders", placeOrderHandler)
	h.POST("/futures/usdt/orders", placeOrderHandler)
	h.POST("/options/orders", placeOrderHandler)
	h.GET("/order_book", orderBookHandler)
	h.DELETE("cancel_order", cancelOrderHandler)
	h.POST("/long", longHandler)
	h.GET("/dynamic_upstream", findUpstreamHandler)

	h.Spin()
}

func echoHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBody(ctx.Request.Body())
}

func placeOrderHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("application/json; charset=utf8")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBody(orderResp)
}

func orderBookHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("application/json; charset=utf8")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBody(orderbookResp)
}

func cancelOrderHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("application/json; charset=utf8")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBody(orderResp)
}

func longHandler(c context.Context, ctx *app.RequestContext) {
	time.Sleep(10 * time.Second)
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.String(200, "hello")
}

func findUpstreamHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.String(200, "find upstream")
}
