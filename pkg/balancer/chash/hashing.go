package chash

import (
	"context"
	"errors"
	"slices"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/internal/pkg/consistent"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const (
	defaultReplicas = 160
)

// Init registers the hashing balancers.
func Init() error {
	return balancer.Register(
		[]string{"hashing", "chash"},
		func(proxies []proxy.Proxy, params any) (balancer.Balancer, error) {
			if params == nil {
				return nil, errors.New("params cannot be empty")
			}

			var hashon string
			replicas := defaultReplicas

			if val, ok := params.(map[string]any); ok {
				hashon, ok = val["hash_on"].(string)
				if !ok {
					return nil, errors.New("hash_on is required and must be a string")
				}
			}

			b := NewBalancer(proxies, hashon, replicas)
			return b, nil
		},
	)
}

// Balancer implements a consistent hashing load balancing strategy.
type Balancer struct {
	hashon  string
	proxies []proxy.Proxy
	ring    *consistent.Consistent
	nodeMap map[string]proxy.Proxy // Maps node ID to proxy
}

// NewBalancer creates a new consistent hashing Balancer instance.
// It uses the consistent hashing package for better distribution and performance.
// Each proxy's virtual nodes are scaled by its Weight() to achieve weight-based distribution.
func NewBalancer(proxies []proxy.Proxy, hashon string, replicas int) *Balancer {
	b := &Balancer{
		proxies: proxies,
		hashon:  hashon,
		ring:    consistent.New().SetReplicas(replicas),
		nodeMap: make(map[string]proxy.Proxy),
	}

	// Add all proxies to the consistent hash ring with weight-based replicas
	// Sort proxies by target to ensure deterministic ring construction
	slices.SortFunc(proxies, func(a, b proxy.Proxy) int {
		if a.Target() < b.Target() {
			return -1
		}
		if a.Target() > b.Target() {
			return 1
		}
		return 0
	})

	for _, p := range proxies {
		weight := int(p.Weight())
		if weight <= 0 {
			weight = 1 // Default to 1 if weight is not set or invalid
		}

		// Use AddWithReplicas to scale virtual nodes based on weight
		_ = b.ring.AddWithReplicas(p.Target(), replicas*weight)
		b.nodeMap[p.Target()] = p
	}

	return b
}

// Proxies returns the list of proxies managed by the balancer.
func (b *Balancer) Proxies() []proxy.Proxy {
	return b.proxies
}

// Select picks a proxy from the hash ring based on a hashed value from the request.
// If the selected proxy is unavailable, it tries the next nodes on the ring.
func (b *Balancer) Select(_ context.Context, c *app.RequestContext) (proxy.Proxy, error) {
	if len(b.proxies) == 0 {
		return nil, balancer.ErrNotAvailable
	}

	val := variable.GetString(b.hashon, c)

	// Get all possible candidates from the consistent hash ring in clockwise order
	candidates, err := b.ring.GetN(val, len(b.proxies))
	if err != nil {
		// If GetN returns an error, it means no nodes are available in the ring.
		return nil, balancer.ErrNotAvailable
	}

	for _, nodeID := range candidates {
		p, ok := b.nodeMap[nodeID]
		if ok && p.IsAvailable() {
			return p, nil // Found an available proxy
		}
	}

	// No available proxy found among all candidates
	return nil, balancer.ErrNotAvailable
}
