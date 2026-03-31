package roundrobin

import (
	"context"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/proxy"
)

// Init registers the round-robin balancer.
func Init() error {
	return balancer.Register(
		[]string{"round_robin"},
		func(proxies []proxy.Proxy, _ any) (balancer.Balancer, error) {
			b := NewBalancer(proxies)
			return b, nil
		},
	)
}

// Balancer implements a round-robin load balancing strategy.
type Balancer struct {
	counter atomic.Uint64
	proxies []proxy.Proxy
}

// NewBalancer creates a new round-robin Balancer instance.
func NewBalancer(proxies []proxy.Proxy) *Balancer {
	return &Balancer{
		counter: atomic.Uint64{},
		proxies: proxies,
	}
}

// Proxies returns the list of proxies managed by the balancer.
func (b *Balancer) Proxies() []proxy.Proxy {
	return b.proxies
}

// Select picks the next available proxy in a round-robin fashion.
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
	// Use natural wrap-around of Uint64.
	count := b.counter.Add(1)

	// By subtracting 1 from the counter value, the code is effectively making the counter 0-indexed,
	// so that the first element in the u.proxies list is selected when the counter is at 1.
	index := (count - 1) % uint64(len(b.proxies))
	p := b.proxies[index]

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
