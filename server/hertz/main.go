package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/netpoll"
	"github.com/hertz-contrib/reverseproxy"
	"github.com/valyala/fasthttp"
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
	orderResp []byte
	reqClient *fasthttp.Client
)

func WithDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

func main() {

	reqClient = getFastReqClient()
	orderResp = []byte(order)

	_ = netpoll.SetNumLoops(2)

	opts := []config.Option{
		server.WithHostPorts(port),
		server.WithIdleTimeout(time.Second * 3),
		server.WithReadTimeout(time.Second * 3),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		WithDefaultServerHeader(true),
	}
	//h := server.New(opts...)
	h := server.Default(opts...)

	h.POST("/", echoHandler)
	h.POST("/orders", placeOrderHandler)

	clientOpts := []config.ClientOption{
		client.WithClientReadTimeout(time.Second * 3),
		client.WithWriteTimeout(time.Second * 3),
		client.WithMaxIdleConnDuration(60 * time.Second),
		client.WithMaxConnsPerHost(256),
		client.WithNoDefaultUserAgentHeader(true),
		client.WithDisableHeaderNamesNormalizing(true),
		client.WithDisablePathNormalizing(true),
		client.WithDialTimeout(5 * time.Second),
	}

	rp, _ := reverseproxy.NewSingleHostReverseProxy("http://127.0.0.1:8000", clientOpts...)

	h.Any("/place_order", rp.ServeHTTP)
	h.Any("/upstream", upstreamOrderHandler)

	h.Spin()
}

func echoHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.Response.SetBody(ctx.Request.Body())
}

func placeOrderHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("application/json; charset=utf8")
	ctx.Response.SetBody(orderResp)
}

func upstreamOrderHandler(c context.Context, ctx *app.RequestContext) {
	nowUTC := time.Now().UTC()
	ctx.Set("start", nowUTC)

	bodyLength := len(ctx.Request.Body())
	ctx.Set("body_length", bodyLength)

	// 從請求池中分別獲取一個request、response實例
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	// 回收實例到請求池
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()
	// 設定請求方式
	req.Header.SetMethodBytes(ctx.Method())

	// 設定請求地址
	req.SetRequestURI("http://127.0.0.1:8000/orders")
	req.SetBodyRaw(ctx.Request.Body())

	// 發起請求
	if err := reqClient.Do(req, resp); err != nil {
		fmt.Println("fasthttp client err: ", err)
		ctx.AbortWithError(500, err)
		return
	}

	ctx.Set("upstream_addr", "127.0.0.1:8080")
	ctx.Set("upstream_status_code", ctx.Response.StatusCode())

	ctx.SetContentType("application/json; charset=utf8")
	ctx.SetStatusCode(ctx.Response.StatusCode())
	ctx.Response.SetBody(resp.Body())

	endUC := time.Now()
	ctx.Set("end", endUC)
}

func getFastReqClient() *fasthttp.Client {

	reqClient := &fasthttp.Client{
		// 讀超時時間,不設定read超時,可能會造成連接復用失效
		ReadTimeout: time.Second * 3,
		// 寫超時時間
		WriteTimeout: time.Second * 3,
		// 60秒後，關閉空閒的活動連接
		MaxIdleConnDuration: time.Minute * 1,
		// 當true時,從請求中去掉User-Agent標頭
		NoDefaultUserAgentHeader: true,
		// 當true時，header中的key按照原樣傳輸，默認會根據標準化轉化
		DisableHeaderNamesNormalizing: true,
		//當true時,路徑按原樣傳輸，默認會根據標準化轉化
		DisablePathNormalizing: true,
		MaxConnsPerHost:        256,
		// Dial: (&fasthttp.TCPDialer{
		// 	// 最大並行數，0表示無限制
		// 	Concurrency: 16,
		// 	// 將 DNS 快取時間從默認分鐘增加到一小時
		// 	DNSCacheDuration: time.Hour,
		// }).Dial,
	}
	return reqClient
}
