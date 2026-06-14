package roundrobin

import (
	"context"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

// Init registers the round-robin balancer with the balancer registry.
func Init() error {
	return balancer.Register(
		[]string{"round_robin"},
		func(endpoints []*target.Endpoint, _ any) (balancer.Balancer, error) {
			b := NewBalancer(endpoints)
			return b, nil
		},
	)
}

// Balancer implements a round-robin load balancer.
type Balancer struct {
	Counter   atomic.Uint64
	endpoints []*target.Endpoint
}

// NewBalancer creates a new round-robin balancer with the given endpoints.
func NewBalancer(endpoints []*target.Endpoint) *Balancer {
	return &Balancer{
		endpoints: endpoints,
	}
}

// Select returns the next available endpoint in round-robin order.
func (b *Balancer) Select(_ context.Context, _ *app.RequestContext) (*target.Endpoint, error) {
	n := len(b.endpoints)
	if n == 0 {
		return nil, balancer.ErrNotAvailable
	}

	count := b.Counter.Add(1)
	startIdx := int((count - 1) % uint64(n)) //nolint:gosec
	for i := range n {
		idx := (startIdx + i) % n
		ep := b.endpoints[idx]
		if ep.State == nil || ep.State.IsAvailable() {
			return ep, nil
		}
	}
	return nil, balancer.ErrNotAvailable
}
