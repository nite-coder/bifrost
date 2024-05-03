package main

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unsafe"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/hertz-contrib/reverseproxy"
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
	reqClient        *client.Client
	orderPath        = []byte(`/place_order`)
	futuresOrderPath = []byte(`/futures/orders`)
	myUpstream       = []byte("http://127.0.0.1:8000")
)

func WithDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func main() {

	clientOpts := []config.ClientOption{
		client.WithClientReadTimeout(time.Second * 3),
		client.WithWriteTimeout(time.Second * 3),
		client.WithMaxIdleConnDuration(60 * time.Second),
		client.WithMaxConnsPerHost(2000),
		client.WithNoDefaultUserAgentHeader(true),
		client.WithDisableHeaderNamesNormalizing(true),
		client.WithDisablePathNormalizing(true),
		client.WithDialTimeout(5 * time.Second),
	}

	reqClient, _ = client.NewClient(clientOpts...)

	orderResp = []byte(order)

	//_ = netpoll.SetNumLoops(1)

	opts := []config.Option{
		server.WithHostPorts(port),
		server.WithIdleTimeout(time.Second * 60),
		server.WithReadTimeout(time.Second * 3),
		server.WithWriteTimeout(time.Second * 3),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		WithDefaultServerHeader(true),
	}
	//h := server.New(opts...)
	h := server.Default(opts...)

	h.POST("/", echoHandler)
	h.POST("/spot/orders", placeOrderHandler)

	// match /futures/orders
	h.Use(func(c context.Context, ctx *app.RequestContext) {
		if !bytes.HasPrefix(ctx.Request.Path(), futuresOrderPath) {
			return
		}

		placeOrderHandler(c, ctx)
	})

	// match /futures/usdt/orders
	re, _ := regexp.Compile(`^/futures/(usdt|btc)/orders$`)
	h.Use(func(c context.Context, ctx *app.RequestContext) {

		if !re.MatchString(string(ctx.Request.Path())) {
			return
		}

		placeOrderHandler(c, ctx)
	})

	rp, _ := reverseproxy.NewSingleHostReverseProxy("http://127.0.0.1:8000", clientOpts...)
	h.Any("/place_order", rp.ServeHTTP)

	h.Use(func(c context.Context, ctx *app.RequestContext) {
		fmt.Println("done")
	})

	h.Spin()
}

func echoHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.Response.SetBody(ctx.Request.Body())
}

func placeOrderHandler(c context.Context, ctx *app.RequestContext) {
	req, resp := protocol.AcquireRequest(), protocol.AcquireResponse()
	// 回收實例到請求池
	defer func() {
		protocol.ReleaseRequest(req)
		protocol.ReleaseResponse(resp)
	}()
	// 設定請求方式
	req.Header.SetMethodBytes(ctx.Method())

	// 設定請求地址
	var strBuilder strings.Builder
	strBuilder.Write(myUpstream)
	//strBuilder.Write(ctx.Request.Path())
	strBuilder.Write(orderPath)

	req.SetRequestURI(strBuilder.String())
	req.SetBodyRaw(ctx.Request.Body())

	// 發起請求

	if err := reqClient.Do(c, req, resp); err != nil {
		fmt.Println("fasthttp client err: ", err)

	}

	ctx.Response.Header.SetContentType("application/json; charset=utf8")
	ctx.Response.SetStatusCode(resp.StatusCode())
	ctx.Response.SetBody(resp.Body())
	ctx.Abort()
}
