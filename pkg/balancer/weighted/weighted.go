package weighted

import (
	"context"
	"math"
	"math/rand"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

// Init registers the weighted balancer with the balancer registry.
func Init() error {
	return balancer.Register(
		[]string{"weighted"},
		func(endpoints []*target.Endpoint, _ any) (balancer.Balancer, error) {
			return NewBalancer(endpoints)
		},
	)
}

// Balancer selects an endpoint based on weighted random selection.
type Balancer struct {
	endpoints []*target.Endpoint
}

// NewBalancer creates a new weighted balancer with the given endpoints.
func NewBalancer(endpoints []*target.Endpoint) (*Balancer, error) {
	return &Balancer{endpoints: endpoints}, nil
}

// Select picks an endpoint using weighted random selection, skipping unhealthy endpoints.
func (b *Balancer) Select(_ context.Context, _ *app.RequestContext) (*target.Endpoint, error) {
	if len(b.endpoints) == 0 {
		return nil, balancer.ErrNotAvailable
	}

	var available uint32
	for _, ep := range b.endpoints {
		if ep.State != nil && !ep.State.IsAvailable() {
			continue
		}
		w := ep.Weight
		if w == 0 {
			w = 1
		}
		if w > math.MaxInt32 {
			w = math.MaxInt32
		}
		available += w
	}
	if available == 0 {
		return nil, balancer.ErrNotAvailable
	}

	r := rand.Intn(int(available)) + 1 //nolint:gosec
	for _, ep := range b.endpoints {
		if ep.State != nil && !ep.State.IsAvailable() {
			continue
		}
		w := ep.Weight
		if w == 0 {
			w = 1
		}
		if w > math.MaxInt32 {
			w = math.MaxInt32
		}
		r -= int(w)
		if r <= 0 {
			return ep, nil
		}
	}
	return nil, balancer.ErrNotAvailable
}
