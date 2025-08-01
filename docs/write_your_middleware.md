# Write Your first Middleware

Since Bifrost uses the hertz HTTP framework at its core, the middleware interface standards can be found in [Hertz](https://www.cloudwego.io/docs/hertz/overview/)

```go
func (m *AddPrefixMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {

}
```

To develop a custom middleware, follow these steps:

1. `Create A New Middleware`: Create a new middleware implementing the ServeHTTP(c context.Context, ctx *app.RequestContext) interface.
1. `Register Middleware`: Register the middleware with the gateway.
1. `Configure Middleware`: Apply the middleware to `servers`、`routes`、`services` as needed.

Here’s an example of building a middleware that records the request’s entry and exit time and returns this information in HTTP headers.

## Create A New Middleware

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

## Registering Middleware

```golang
package main

import (
 "github.com/nite-coder/bifrost/pkg/config"
 "github.com/nite-coder/bifrost/pkg/gateway"
 "github.com/nite-coder/bifrost/pkg/middleware"
 "github.com/cloudwego/hertz/pkg/app"
)

func registerMiddlewares() error {
 err := middleware.RegisterMiddleware([]string{}"timing"}, func(param any) (app.HandlerFunc, error) {
  m := TimingMiddleware{}
  return m.ServeHTTP, nil
 })
 if err != nil {
  return err
 }

 return nil
}

func main() {
 err := registerMiddlewares()
 if err != nil {
  panic(err)
 }

 options, err := config.Load("./config.yaml")
 if err != nil {
  panic(err)
 }

 err = gateway.Run(options)
 if err != nil {
  panic(err)
 }
}
```

## Configuring Middleware

In this example, we configure it for `servers`, which adds the `x-time-in` and `x-time-out` headers to every HTTP response.

```yaml
servers:
  test-server:
    bind: ":8001"
  middlewares:
    - type: timing
```
