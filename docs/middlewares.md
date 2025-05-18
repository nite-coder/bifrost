# Middlewares

Bifrost supports both built-in and custom middlewares that can be applied within `servers`, `routes`, and `services` configurations.

* `servers`: Middleware in this scope is executed for every HTTP request that passes through this server.
* `routes`: Middleware applied here will execute for requests matching this route.
* `services`: Middleware applied here will execute for requests matching this business service.

You can also develop custom middlewares directly in native Golang.

## Built-In Middlewares

Currently supported middlewares are below.

* [AddPrefix](#addprefix): Add a prefix to the request path.
* [Cors](#cors): A Middleware for Cross-Origin Resource Sharing.
* [IPRestriction](#iprestriction): Control client IP address that can access the service.
* [Mirror](#mirror): Mirror the request to another service.
* [Parallel](#parallel): Execute a group of middlewares concurrently.
* [RateLimit](#ratelimit): To control the Number of Requests going to a service
* [ReplacePath](#replacepath): Replace the request path.
* [ReplacePathRegex](#replacepathregex): Replace the request path with a regular expression.
* [RequestTermination](#requesttermination): Response the content to client and terminate the request.
* [RequestTransformer](#requesttransformer): Apply a request transformation to the request.
* [ResponseTransformer](#responsetransformer): Apply a response transformation to the response.
* [SetVars](#setvars): Set variables in the request context.
* [StripPrefix](#stripprefix): Remove a prefix from the request path.
* [TrafficSplitter](#trafficsplitter): Route requests to different services based on weights.
* [UARestriction](#uarestriction): Control user agent that can access the service.

### AddPrefix

Adds a prefix to the original request path before forwarding upstream.

Original request: `/foo` \
Forwarded path for upstream: `/api/v1/foo`

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: add_prefix
        params:
          prefix: /api/v1
```

### Cors

A Middleware for Cross-Origin Resource Sharing.

Original request: `/foo` \
Forwarded path for upstream: `/api/v1/foo`

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: cors
        params:
          allow_all_origins: true
          allow_methods:
            ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]
          allow_headers: ["Origin", "Content-Length", "Content-Type"]
          max_age: 12m
```

### IPRestriction

Control client IP address that can access the service.  Either one of `allow` or `deny` attribute must be specified. They cannot be used together.

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: ip_restriction
        params:
          allow: ["192.168.1.1"] # allow and deny can't be used at the same time
          deny: ["192.168.1.0/24"]
          rejected_http_status_code: 403
          rejected_http_content_type: application/json
          rejected_http_response_body: "forbidden"
```

### Mirror

Mirrors the request to another service.

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: mirror
        params:
          service_id: service2
```

### Parallel

Execute a group of middlewares concurrently. If errors occur in any of the middlewares, the request will be terminated.

```yaml
routes:
  order:
    paths:
      - /order
    service_id: order_service
    middlewares:
      - type: parallel
        params:
          - type: setvars
            params:
              - key: user_id
                value: $http.request.header.user_id
          - type: setvars
            params:
              - key: is_vip
                value: true
```

### RateLimit

The `RateLimit` middleware ensures that services receive a fair share of requests and allows you to define what "fair" means.
The following example allows a user to send a maximum of 10 requests within 2 seconds.  If redis server is crashed, the requests will be passed. (downgrade)

```yaml
routes:
  order:
    paths:
      - /orders
    middlewares:
      - type: rate_limit
        params:
          strategy: local # local, redis
          limit_by: user_id:$var.user_id  # allow to use directive
          limit: 10
          window_size: 2s
          header_limit: x-ratelimit-limit
          header_remaining: x-ratelimit-remaining
          header_reset: x-ratelimit-reset
          rejected_http_status_code: 429 # when hit the rate limit
          rejected_http_content_type: application/json
          rejected_http_response_body: {"error": "too many requests"}
    service_id: order_service
```

### ReplacePath

Replaces the entire original request path with a different path before forwarding upstream. If the original request includes a query string, it will also be forwarded.

Original request: `/api/v1/user?name=john` \
Forwarded path for upstream: `/hoo/user?name=john`

```yaml
routes:
  user:
    paths:
      - /api/v1/user
    service_id: service1
    middlewares:
      - type: replace_path
        params:
          path: /hoo/user
```

### ReplacePathRegex

Replaces the entire original request path with a different path via regular expression before forwarding upstream. If the original request includes a query string, it will also be forwarded.

Original request: `/api/v1/user?name=john` \
Forwarded path for upstream: `/hoo/user?name=john`

```yaml
routes:
  user:
    paths:
      - /api/v1/user
    service_id: service1
    middlewares:
      - type: replace_path_regex
        params:
          regex: ^/api/v1/(.*)$
          replacement: /hoo/$1
```

### RequestTermination

Returns the content to the client and terminates the request.

```yaml
routes:
  mock:
    paths:
      - /mock/order
    service_id: service1
    middlewares:
      - type: request_termination
        params:
          status_code: 200
          content_type: application/json
          body: "hello bifrost"

```

### RequestTransformer

Apply a request transformation to the request. The `add` option means that if the value doesn't exist, the value will be added. The `set` option means that whether the value exists or not, it will be added or overwritten.

```yaml
routes:
  mock:
    paths:
      - /mock/order
    service_id: service1
    middlewares:
      - type: request_transformer
        params:
          remove:
            headers:
              - x-user-id
            querystring:
              - mode
          add:
            headers:
              x-id: 123
          set:
            headers:
              x-source: "web"
```

### ResponseTransformer

Apply a response transformation to the response.  The `add` option means that if the value doesn't exist, the value will be added. The `set` option means that whether the value exists or not, it will be added or overwritten.

```yaml
routes:
  mock:
    paths:
      - /mock/order
    service_id: service1
    middlewares:
      - type: response_transformer
        params:
          remove:
            headers:
              - x-server
          add:
            headers:
              x-source: web
              x-http-start: $http.start
              x-http-finish: $http.finish
          set:
            headers:
              x-source: "web"
```

### SetVars

Set variables in the request context.

```yaml
routes:
  orders:
    paths:
      - /api/v1/orders
    service_id: service1
    middlewares:
      - type: setvars
        params:
          - key: user_id
            value: "123456" # you can use directive here
            default: "0" # if the variable is not set, the default value will be used
```

### StripPrefix

Removes a part of the original request path before forwarding upstream.

Original request: `/api/v1/payment` \
Forwarded path for upstream: `/payment`

```yaml
routes:
  payment:
    paths:
      - /api/v1/payment
    service_id: service1
    middlewares:
      - type: strip_prefix
        params:
          prefixes:
            - /api/v1
```

### TrafficSplitter

Route requests to different services based on weights.

```yaml
servers:
  apiv1:
    bind: ":8001"
    reuse_port: true
    middlewares:
      - type: traffic_splitter
        params:
          key: $my_order
          destinations:
            - weight: 80
              to: old_service
            - weight: 20
              to: new_service
```

### UARestriction

Control user agent that can access the service.  Either one of `allow` or `deny` attribute must be specified. They cannot be used together.
You can use regex to match user agent.

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: ip_restriction
        params:
          deny: ["bad-agent"] # allow and deny can't be used at the same time
          rejected_http_status_code: 403
          rejected_http_content_type: application/json
          rejected_http_response_body: "forbidden"
```

## Custom Middlewares

Since Bifrost uses the hertz HTTP framework at its core, the middleware interface standards can be found in [Hertz](https://www.cloudwego.io/docs/hertz/overview/)

```go
func (m *AddPrefixMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {

}
```

To develop a custom middleware, follow these steps:

1. `Develop a New Middleware`: Create a new middleware implementing the ServeHTTP(c context.Context, ctx *app.RequestContext) interface.
1. `Register Middleware`: Register the middleware with the gateway.
1. `Configure Middleware`: Apply the middleware to `servers`、`routes`、`services` as needed.

Here’s an example of building a middleware that records the request’s entry and exit time and returns this information in HTTP headers.

### Create A New Middleware

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

### Registering Middleware

```golang
package main

import (
 "github.com/nite-coder/bifrost/pkg/config"
 "github.com/nite-coder/bifrost/pkg/gateway"
 "github.com/nite-coder/bifrost/pkg/middleware"
 "github.com/cloudwego/hertz/pkg/app"
)

func registerMiddlewares() error {
 err := middleware.RegisterMiddleware("timing", func(param any) (app.HandlerFunc, error) {
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

### Configuring Middleware

In this example, we configure it for `servers`, which adds the `x-time-in` and `x-time-out` headers to every HTTP response.

```yaml
servers:
  test-server:
    bind: ":8001"
  middlewares:
    - type: timing
```
