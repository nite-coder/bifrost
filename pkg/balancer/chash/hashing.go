package chash

import (
	"context"
	"errors"
	"slices"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/internal/pkg/consistent"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const defaultReplicas = 160

// Init registers the consistent hashing balancer with the balancer registry.
func Init() error {
	return balancer.Register(
		[]string{"chash"},
		func(endpoints []*target.Endpoint, params any) (balancer.Balancer, error) {
			if params == nil {
				return nil, errors.New("params cannot be empty")
			}
			parsed, ok := params.(map[string]any)
			if !ok {
				return nil, errors.New("params must be a map")
			}
			_, ok = parsed["hash_on"].(string)
			if !ok {
				return nil, errors.New("hash_on is required and must be a string")
			}
			return NewBalancer(endpoints, parsed)
		},
	)
}

// Balancer implements a consistent hashing load balancer.
type Balancer struct {
	hashon  string
	ring    *consistent.Consistent
	nodeMap map[string]*target.Endpoint
}

// NewBalancer creates a new consistent hashing balancer with the given endpoints and params.
func NewBalancer(endpoints []*target.Endpoint, params map[string]any) (*Balancer, error) {
	hashon, ok := params["hash_on"].(string)
	if !ok {
		return nil, errors.New("hash_on is required and must be a string")
	}
	replicas := defaultReplicas
	b := &Balancer{
		hashon:  hashon,
		ring:    consistent.New().SetReplicas(replicas),
		nodeMap: make(map[string]*target.Endpoint),
	}
	sorted := make([]*target.Endpoint, len(endpoints))
	copy(sorted, endpoints)
	slices.SortFunc(sorted, func(a, b *target.Endpoint) int {
		if a.Address < b.Address {
			return -1
		}
		if a.Address > b.Address {
			return 1
		}
		return 0
	})
	for _, ep := range sorted {
		weight := 1
		if ep.Weight > 0 {
			weight = int(ep.Weight)
		}
		_ = b.ring.AddWithReplicas(ep.Address, replicas*weight)
		b.nodeMap[ep.Address] = ep
	}
	return b, nil
}

// Select picks an endpoint based on the configured hash key from the request context.
func (b *Balancer) Select(_ context.Context, c *app.RequestContext) (*target.Endpoint, error) {
	if len(b.nodeMap) == 0 {
		return nil, balancer.ErrNotAvailable
	}
	val := variable.GetString(b.hashon, c)
	candidates, err := b.ring.GetN(val, len(b.nodeMap))
	if err != nil {
		return nil, balancer.ErrNotAvailable
	}
	for _, nodeID := range candidates {
		ep, ok := b.nodeMap[nodeID]
		if ok && (ep.State == nil || ep.State.IsAvailable()) {
			return ep, nil
		}
	}
	return nil, balancer.ErrNotAvailable
}
