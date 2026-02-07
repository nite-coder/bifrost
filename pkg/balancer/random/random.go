package random

import (
	"context"
	"math/rand/v2"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/proxy"
)

func Init() error {
	return balancer.Register([]string{"random"}, func(proxies []proxy.Proxy, params any) (balancer.Balancer, error) {
		b := NewBalancer(proxies)
		return b, nil
	})
}

// RandomBalancer implements a random load balancing algorithm.
type RandomBalancer struct {
	proxies []proxy.Proxy
}

// NewBalancer creates a new RandomBalancer instance.
func NewBalancer(proxies []proxy.Proxy) *RandomBalancer {
	return &RandomBalancer{
		proxies: proxies,
	}
}

// Proxies returns the list of proxies managed by the balancer.
func (b *RandomBalancer) Proxies() []proxy.Proxy {
	return b.proxies
}

// Select picks a random available proxy.
func (b *RandomBalancer) Select(ctx context.Context, hzCtx *app.RequestContext) (proxy.Proxy, error) {
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
	selectedIndex := rand.IntN(len(b.proxies))
	p := b.proxies[selectedIndex]
	if p.IsAvailable() {
		return p, nil
	}
	// no live upstream
	if len(failedRecords) == len(b.proxies) {
		return nil, balancer.ErrNotAvailable
	}
	failedRecords[p.ID()] = true
	goto findLoop
}
