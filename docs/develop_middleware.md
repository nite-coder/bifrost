# Develop Your first Middleware

Since Bifrost uses the hertz HTTP framework at its core, the middleware interface standards can be found in [Hertz](https://www.cloudwego.io/docs/hertz/overview/)

```go
func (m *AddPrefixMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {

}
```

To develop a custom middleware, follow these steps:

1. `Create A New Middleware`: Create a new middleware implementing the ServeHTTP(c context.Context, ctx *app.RequestContext) interface.
1. `Register Middleware`: Register the middleware with the gateway.
1. `Configure Middleware`: Apply the middleware to `servers`、`routes`、`services` as needed.

Here’s an example of building a middleware that adds a prefix to the request path. This demonstrates how to handle configuration parameters using `RegisterTyped`.

## Create A New Middleware

First, define your configuration struct and the middleware logic.

```go
package main

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// Config defines the configuration for your middleware.
// Bifrost uses mapstructure to decode the configuration into this struct.
type Config struct {
	Prefix string `mapstructure:"prefix"`
}

type AddPrefixMiddleware struct {
	prefix string
}

func NewMiddleware(prefix string) *AddPrefixMiddleware {
	return &AddPrefixMiddleware{
		prefix: prefix,
	}
}

func (m *AddPrefixMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	// Your middleware logic here
	if m.prefix != "" {
		// Example: Add prefix to something
	}
	ctx.Next(c)
}
```

## Registering Middleware

Use `middleware.RegisterTyped[T]` to register your middleware. This helper function automatically handles the decoding of the configuration map into your struct `T`.

```go
package main

import (
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func registerMiddlewares() error {
	// RegisterTyped handles the complexity of mapstructure decoding for you.
	// 1. Define the config struct type as the generic parameter.
	// 2. The callback function receives the decoded config struct.
	err := middleware.RegisterTyped([]string{"add_prefix"}, func(cfg Config) (app.HandlerFunc, error) {
		// You can perform validation on the decoded config here
		if cfg.Prefix == "" {
			// Return error if validation fails
			// return nil, errors.New("prefix is required")
		}

		m := NewMiddleware(cfg.Prefix)
		return m.ServeHTTP, nil
	})

	return err
}

func main() {
	if err := registerMiddlewares(); err != nil {
		panic(err)
	}

	options, err := config.Load("./config.yaml")
	if err != nil {
		panic(err)
	}
	
	// Initialize logger
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := gateway.Run(options); err != nil {
		panic(err)
	}
}
```

### No Configuration Middleware

If your middleware doesn't require any configuration, you can use `struct{}` as the configuration type.

```go
_ = middleware.RegisterTyped([]string{"simple"}, func(_ struct{}) (app.HandlerFunc, error) {
    return NewMiddleware().ServeHTTP, nil
})
```

## Configuring Middleware

In your `config.yaml`, you can now configure the middleware. The keys in the YAML `options` block will correspond to the `mapstructure` tags in your Go struct.

```yaml
servers:
  test-server:
    bind: ":8001"
    middlewares:
      - type: add_prefix
        options:
          prefix: "/api"
```
