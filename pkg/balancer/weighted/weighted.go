package weighted

import (
	"context"
	"math"
	"math/rand/v2"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/proxy"
)

func Init() error {
	return balancer.Register([]string{"weighted"}, func(proxies []proxy.Proxy, param any) (balancer.Balancer, error) {
		return NewBalancer(proxies)
	})
}

// WeightedBalancer implements a weighted random load balancing algorithm.
type WeightedBalancer struct {
	totalWeight uint32
	proxies     []proxy.Proxy
}

// NewBalancer creates a new WeightedBalancer instance.
// It calculates the total weight and ensures it doesn't exceed MaxInt32.
func NewBalancer(proxies []proxy.Proxy) (*WeightedBalancer, error) {
	var totalWeight uint32
	for _, p := range proxies {
		weight := p.Weight()
		if weight == 0 {
			weight = 1
		}
		totalWeight += weight
	}

	// Clamp total weight to MaxInt32 to avoid issues with random number generation.
	if totalWeight > math.MaxInt32 {
		totalWeight = math.MaxInt32
	}

	return &WeightedBalancer{
		proxies:     proxies,
		totalWeight: totalWeight,
	}, nil
}

// Proxies returns the list of proxies managed by the balancer.
func (b *WeightedBalancer) Proxies() []proxy.Proxy {
	return b.proxies
}

// Select picks a proxy based on weights.
func (b *WeightedBalancer) Select(ctx context.Context, hzCtx *app.RequestContext) (proxy.Proxy, error) {
	if len(b.proxies) == 0 {
		return nil, balancer.ErrNotAvailable
	}

	if len(b.proxies) == 1 {
		p := b.proxies[0]
		if p.IsAvailable() {
			return p, nil
		}
		return nil, balancer.ErrNotAvailable
	}

	failedRecords := map[string]bool{}

findLoop:
	// nolint:gosec
	randomWeight := int64(rand.IntN(int(b.totalWeight)))

	for _, p := range b.proxies {
		weight := int64(p.Weight())
		if weight == 0 {
			weight = 1
		}
		randomWeight -= weight
		if randomWeight < 0 {
			if p.IsAvailable() {
				return p, nil
			}
			// No live upstream
			if len(failedRecords) == len(b.proxies) {
				return nil, balancer.ErrNotAvailable
			}
			failedRecords[p.ID()] = true
			goto findLoop
		}
	}
	return nil, balancer.ErrNotAvailable
}
