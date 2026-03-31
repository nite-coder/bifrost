package balancer

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/proxy"
)

var (
	// ErrNotAvailable is returned when no available upstream is found.
	ErrNotAvailable = errors.New("no available upstream at the moment")
	balancers       = make(map[string]CreateBalancerHandler)
)

// CreateBalancerHandler is a function type that creates a new Balancer.
type CreateBalancerHandler func(proxies []proxy.Proxy, params any) (Balancer, error)

// Balancer defines the interface for load balancing strategies.
type Balancer interface {
	Proxies() []proxy.Proxy
	Select(ctx context.Context, hzCtx *app.RequestContext) (proxy.Proxy, error)
}

// Register registers a new balancer creation handler for the given names.
func Register(names []string, h CreateBalancerHandler) error {
	if len(names) == 0 {
		return errors.New("balancer names cannot be empty")
	}

	for _, name := range names {
		if _, found := balancers[name]; found {
			return fmt.Errorf("balancer '%s' already exists", name)
		}

		balancers[name] = h
	}

	return nil
}

// Factory returns a CreateBalancerHandler for the given balancer name.
func Factory(name string) CreateBalancerHandler {
	if name == "" {
		name = "round_robin"
	}

	return balancers[name]
}
