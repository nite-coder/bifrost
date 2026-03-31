package random

import (
	"context"
	"math/rand/v2"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/proxy"
)

// Init registers the random balancer.
func Init() error {
	return balancer.Register([]string{"random"}, func(proxies []proxy.Proxy, _ any) (balancer.Balancer, error) {
		b := NewBalancer(proxies)
		return b, nil
	})
}

// Balancer implements a random load balancing strategy.
type Balancer struct {
	proxies []proxy.Proxy
}

// NewBalancer creates a new random Balancer instance.
func NewBalancer(proxies []proxy.Proxy) *Balancer {
	return &Balancer{
		proxies: proxies,
	}
}

// Proxies returns the list of proxies managed by the balancer.
func (b *Balancer) Proxies() []proxy.Proxy {
	return b.proxies
}

// Select picks a random available proxy.
func (b *Balancer) Select(_ context.Context, hzCtx *app.RequestContext) (proxy.Proxy, error) {
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
	//nolint:gosec
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
