package main

import (
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

const (
	port        = ":8004"
	debugPort   = ":18004"
	actionQuery = "action"
	upstream    = "http://127.0.0.1:8000/orders"
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

func main() {

	reqClient = getFastReqClient()

	orderResp = []byte(order)

	app := fiber.New(
		fiber.Config{
			IdleTimeout:  time.Second * 3,
			ReadTimeout:  time.Second * 3,
			WriteTimeout: time.Second * 3,
		},
	)

	app.Post("/", requestHandler)
	app.Post("/orders", placeOrderHandler)
	app.Post("/upstream", upstreamHandler)

	err := app.Listen(port)
	if err != nil {
		hlog.Error(err)
	}
}

func requestHandler(ctx fiber.Ctx) error {
	ctx.Response().Header.SetContentType("text/plain; charset=utf8")
	return ctx.Send(ctx.Request().Body())
}

func placeOrderHandler(ctx fiber.Ctx) error {
	ctx.Response().Header.SetContentType("application/json; charset=utf8")
	return ctx.Send(ctx.Request().Body())
}

func upstreamHandler(ctx fiber.Ctx) error {
	//stdCtx := ctx.UserContext()

	// 從請求池中分別獲取一個request、response實例
	req, resp := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	// 回收實例到請求池
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()
	// 設定請求方式
	req.Header.SetMethod(ctx.Method())

	// 設定請求地址
	req.SetRequestURI("http://127.0.0.1:8000/orders")
	req.SetBodyRaw(ctx.Body())

	// 發起請求

	if err := reqClient.Do(req, resp); err != nil {
		fmt.Println("fasthttp client err: ", err)
		return err
	}

	ctx.Response().Header.SetContentType("application/json; charset=utf8")
	return ctx.Send(resp.Body())
}

func getFastReqClient() *fasthttp.Client {
	reqClient := &fasthttp.Client{
		// 讀超時時間,不設定read超時,可能會造成連接復用失效
		ReadTimeout: time.Second * 5,
		// 寫超時時間
		WriteTimeout: time.Second * 5,
		// 5秒後，關閉空閒的活動連接
		MaxIdleConnDuration: time.Second * 10,
		// 當true時,從請求中去掉User-Agent標頭
		NoDefaultUserAgentHeader: true,
		// 當true時，header中的key按照原樣傳輸，默認會根據標準化轉化
		DisableHeaderNamesNormalizing: true,
		//當true時,路徑按原樣傳輸，默認會根據標準化轉化
		DisablePathNormalizing: true,
		Dial: (&fasthttp.TCPDialer{
			// 最大並行數，0表示無限制
			Concurrency: 16,
			// 將 DNS 快取時間從默認分鐘增加到一小時
			DNSCacheDuration: time.Hour,
		}).Dial,
	}
	return reqClient
}
