package main

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hertz-contrib/http2/factory"
	hertzslog "github.com/hertz-contrib/logger/slog"
	"github.com/hertz-contrib/websocket"
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
		server.WithH2C(true),
		WithDefaultServerHeader(true),
	}
	h := server.New(opts...)

	logger := hertzslog.NewLogger(hertzslog.WithOutput(io.Discard))
	hlog.SetLevel(hlog.LevelError)
	hlog.SetLogger(logger)
	hlog.SetSilentMode(true)

	h.AddProtocol("h2", factory.NewServerFactory())

	h.POST("/", echoHandler)
	h.Any("/spot/order", placeOrderHandler)
	h.Any("/spot/orders", placeOrderHandler)
	h.Any("/api/v1/spot/orders", placeOrderHandler)
	h.POST("/futures/usdt/orders", placeOrderHandler)
	h.POST("/options/orders", placeOrderHandler)
	h.GET("/order_book", orderBookHandler)
	h.DELETE("cancel_order", cancelOrderHandler)
	h.POST("/long", longHandler)
	h.GET("/dynamic_upstream", findUpstreamHandler)
	h.GET("/websocket", wssHandler)

	h.GET("/users/:user_id/orders", func(c context.Context, ctx *app.RequestContext) {
		userID := ctx.Param("user_id")
		ctx.String(200, "orders:"+userID)
	})

	h.GET("/users/:name/orders1", func(c context.Context, ctx *app.RequestContext) {
		name := ctx.Param("name")
		ctx.String(200, "order1:"+name)
	})

	h.Spin()
}

func echoHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBody(ctx.Request.Body())
}

var placeOrderCounter atomic.Uint64

func placeOrderHandler(c context.Context, ctx *app.RequestContext) {
	mode := ctx.Query("mode")

	if len(mode) > 0 {
		if (placeOrderCounter.Load() % 99) == 0 {
			time.Sleep(100 * time.Millisecond)
		} else {
			time.Sleep(5 * time.Millisecond)
		}
		placeOrderCounter.Add(1)
	}

	// slog.Info("request proto", "proto", ctx.Request.Header.GetProtocol())

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

var upgrader = websocket.HertzUpgrader{
	CheckOrigin: func(r *app.RequestContext) bool {
		return true
	},
} // use default options

func wssHandler(c context.Context, ctx *app.RequestContext) {
	err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				// slog.ErrorContext(c, "read err:", "error", err)
				break
			}
			// slog.Info("recv", "msg", string(msg))

			err = conn.WriteMessage(websocket.TextMessage, orderResp)
			if err != nil {
				// slog.ErrorContext(c, "write err:", "error", err)
				break
			}
		}
	})

	if err != nil {
		slog.ErrorContext(c, "upgrade err:", "error", err)
		return
	}
}
