package random

import (
	"context"
	"math/rand"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

// Init registers the random balancer with the balancer registry.
func Init() error {
	return balancer.Register(
		[]string{"random"},
		func(endpoints []*target.Endpoint, _ any) (balancer.Balancer, error) {
			b := NewBalancer(endpoints)
			return b, nil
		},
	)
}

// Balancer implements a random load balancer.
type Balancer struct {
	endpoints []*target.Endpoint
}

// NewBalancer creates a new random balancer with the given endpoints.
func NewBalancer(endpoints []*target.Endpoint) *Balancer {
	return &Balancer{
		endpoints: endpoints,
	}
}

// Select returns a random available endpoint.
func (b *Balancer) Select(_ context.Context, _ *app.RequestContext) (*target.Endpoint, error) {
	if len(b.endpoints) == 0 {
		return nil, balancer.ErrNotAvailable
	}

	offset := rand.Intn(len(b.endpoints)) //nolint:gosec
	for i := range b.endpoints {
		idx := (offset + i) % len(b.endpoints)
		ep := b.endpoints[idx]
		if ep.State == nil || ep.State.IsAvailable() {
			return ep, nil
		}
	}
	return nil, balancer.ErrNotAvailable
}
