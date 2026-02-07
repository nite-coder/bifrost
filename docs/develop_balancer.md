# Develop a Custom Balancer

Bifrost allows you to create custom load balancers to distribute traffic according to your specific needs. This guide will walk you through the steps to implement and register a custom balancer.

## Balancer Interface

A balancer must implement the `Balancer` interface, defined in `pkg/balancer/pkg.go`:

```go
type Balancer interface {
    // Proxies returns the list of proxies managed by the balancer.
    Proxies() []proxy.Proxy

    // Select picks a suitable proxy for the request.
    Select(ctx context.Context, hzCtx *app.RequestContext) (proxy.Proxy, error)
}
```

## Step-by-Step Implementation

Let's implement a simple **Random Balancer** as an example.

### 1. Create the Balancer Struct

Define a struct that holds your proxies. It should satisfy the `Balancer` interface.

```go
package mybalancer

import (
    "context"
    "math/rand"
    "time"

    "github.com/cloudwego/hertz/pkg/app"
    "github.com/nite-coder/bifrost/pkg/balancer"
    "github.com/nite-coder/bifrost/pkg/proxy"
)

type RandomBalancer struct {
    proxies []proxy.Proxy
}

func NewBalancer(proxies []proxy.Proxy) *RandomBalancer {
    return &RandomBalancer{
        proxies: proxies,
    }
}

// Proxies returns the underlying proxies
func (b *RandomBalancer) Proxies() []proxy.Proxy {
    return b.proxies
}

// Select picks a random available proxy
func (b *RandomBalancer) Select(ctx context.Context, hzCtx *app.RequestContext) (proxy.Proxy, error) {
    if len(b.proxies) == 0 {
        return nil, balancer.ErrNotAvailable
    }

    // Filter available proxies
    var availableProxies []proxy.Proxy
    for _, p := range b.proxies {
        if p.IsAvailable() {
            availableProxies = append(availableProxies, p)
        }
    }

    if len(availableProxies) == 0 {
        return nil, balancer.ErrNotAvailable
    }

    // Pick a random one
    // Note: In production, consider using a better random source or algorithm
    idx := rand.Intn(len(availableProxies))
    return availableProxies[idx], nil
}
```

### 2. Register Your Balancer

You need to register your balancer implementation with Bifrost so it can be used in the configuration. Use `balancer.Register` in your `init()` function or main startup logic.

```go
func Init() error {
    return balancer.Register([]string{"my_random", "custom_random"}, func(proxies []proxy.Proxy, params any) (balancer.Balancer, error) {
        // You can parse `params` here if your balancer needs configuration
        return NewBalancer(proxies), nil
    })
}
```

### 3. Use It in Configuration

Once registered, you can use your custom balancer name in `config.yaml`:

```yaml
upstreams:
  my-upstream-service:
    balancer: my_random  # Use the name you registered
    targets:
      - target: 127.0.0.1:8081
      - target: 127.0.0.1:8082
```

## Handling Parameters

If your balancer requires configuration, the `params` argument in the register function receives the raw configuration (usually `map[string]any`). You can use libraries like `mapstructure` to decode it into a struct.

Example:

```go
type Config struct {
    Seed int64 `mapstructure:"seed"`
}

func Init() error {
	return balancer.Register([]string{"seeded_random"}, func(proxies []proxy.Proxy, params any) (balancer.Balancer, error) {
		var cfg Config
		// ... decode params into cfg ...
		
		return NewSeededBalancer(proxies, cfg.Seed), nil
	})
}
```

And in `config.yaml`:

```yaml
upstreams:
  my-app:
    balancer: seeded_random
    balancer_options:
      seed: 12345
```
