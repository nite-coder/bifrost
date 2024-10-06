# 中間件 (Middlewares)

目前 bifrost 是支持一些內建的中間件與自定義中間件，在 `servers`、`routes`、`services` 這三個配置內可以使用。

* `servers`: 這代表每一個 HTTP 請求通過這個 server 進來的，將會執行這個中間件
* `routes`: 這代表符合這個路由的請求，將會執行這個中間件
* `services`: 這代表符合這個業務服務的請求，將會執行這個中間件

另外也可以直接使用原生 Golang 來開發自定義中間件

## 內建中間件

目前支持的中間件有下面

## 自定義中間件

因為 bifrost 底層使用 `hertz` HTTP 框架，所以中間件標準介面可以參考 [Hertz](https://www.cloudwego.io/docs/hertz/overview/)

```go
func (m *AddPrefixMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {

}
```

開發自定義中間件，分下面幾個步驟

1. `開發新的中間件`: 新建立一個 middleware，需要實現 ServeHTTP(c context.Context, ctx *app.RequestContext) 介面
1. `註冊中間件`: 註冊 middleware 到 gateway
1. `配置中間件`: 依照需求，看要配置 middleware 到 `servers`、`routes`、`services` 哪邊

這邊我們將示範如何建立一個中間件，用來紀錄 HTTP 請求進入與返回的時間，並透過返回的 HTTP Header 告知用戶

### 開發新的中間件

```golang
package main

import (
 "context"
 "strconv"
 "time"

 "github.com/cloudwego/hertz/pkg/app"
)

type TimingMiddleware struct {
}

func NewMiddleware() *TimingMiddleware {
 return &TimingMiddleware{}
}

func (t *TimingMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
 startTime := time.Now().UTC().UnixMicro()

 ctx.Next(c)

 endTime := time.Now().UTC().UnixMicro()

 ctx.Response.Header.Add("x-time-in", strconv.FormatInt(startTime, 10))
 ctx.Response.Header.Add("x-time-out", strconv.FormatInt(endTime, 10))
}
```

### 註冊中間件

```golang
func main() {
 options, err := config.Load("./config.yaml")
 if err != nil {
  panic(err)
 }

 err = registerMiddlewares() // 註冊 middlewares
 if err != nil {
  panic(err)
 }

 err = gateway.Run(options)
 if err != nil {
  panic(err)
 }
}

func registerMiddlewares() error {
    // timing 是向 gateway 說我們這個 middleware 的 type 名稱, 必須要是唯一值
 err := gateway.RegisterMiddleware("timing", func(param map[string]any) (app.HandlerFunc, error) {
  m := TimingMiddleware{}
  return m.ServeHTTP, nil
 })
 if err != nil {
  return err
 }

 return nil
}
```

### 配置中間件

這邊我們配置到 `servers`, 這樣每一個 HTTP 請求的返回 `x-time-in`、`x-time-out` 這兩個 header

```yaml
servers:
  test-server:
    bind: ":8001"
  middlewares:
    - type: timing
```
