package balancer

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/target"
)

var (
	// ErrNotAvailable is returned when no upstream endpoint is available.
	ErrNotAvailable = errors.New("no available upstream at the moment")
	mu              sync.RWMutex
	balancers       = make(map[string]CreateBalancerHandler)
)

// CreateBalancerHandler is a function that creates a Balancer from endpoints and params.
type CreateBalancerHandler func(endpoints []*target.Endpoint, params any) (Balancer, error)

// Balancer selects an endpoint from available upstream targets.
type Balancer interface {
	Select(ctx context.Context, hzCtx *app.RequestContext) (*target.Endpoint, error)
}

// Register registers a balancer handler under the given names.
func Register(names []string, h CreateBalancerHandler) error {
	if len(names) == 0 {
		return errors.New("balancer names cannot be empty")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, name := range names {
		if _, found := balancers[name]; found {
			return fmt.Errorf("balancer '%s' already exists", name)
		}
		balancers[name] = h
	}
	return nil
}

// Factory returns the registered balancer handler for the given name.
func Factory(name string) CreateBalancerHandler {
	if name == "" {
		name = "round_robin"
	}
	mu.RLock()
	defer mu.RUnlock()
	return balancers[name]
}
