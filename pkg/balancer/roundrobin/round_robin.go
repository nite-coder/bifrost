package roundrobin

import (
	"context"
	"math"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
)

func init() {
	_ = balancer.Register("round_robin", func(proxies []proxy.Proxy, option *config.UpstreamOptions) (balancer.Balancer, error) {
		b := NewBalancer(proxies)
		return b, nil
	})
}

type RoundRobinBalancer struct {
	counter atomic.Uint64
	proxies []proxy.Proxy
}

func NewBalancer(proxies []proxy.Proxy) *RoundRobinBalancer {
	return &RoundRobinBalancer{
		counter: atomic.Uint64{},
		proxies: proxies,
	}
}

func (b *RoundRobinBalancer) Proxies() []proxy.Proxy {
	return b.proxies
}

func (b *RoundRobinBalancer) Select(ctx context.Context, hzCtx *app.RequestContext) (proxy.Proxy, error) {
	if b.proxies == nil {
		return nil, balancer.ErrNoAvailable
	}

	if len(b.proxies) == 1 {
		proxy := b.proxies[0]
		if proxy.IsAvailable() {
			return proxy, nil
		}
		return nil, balancer.ErrNoAvailable
	}

	failedReconds := map[string]bool{}

findLoop:
	b.counter.Add(1)
	if b.counter.Load() >= uint64(math.MaxUint64) {
		b.counter.Store(1)
	}
	// By subtracting 1 from the counter value, the code is effectively making the counter 0-indexed,
	// so that the first element in the u.proxies list is selected when the counter is at 1.
	index := (b.counter.Load() - 1) % uint64(len(b.proxies))
	proxy := b.proxies[index]
	if proxy.IsAvailable() {
		return proxy, nil
	}
	// no live upstream
	if len(failedReconds) == len(b.proxies) {
		return nil, balancer.ErrNoAvailable
	}
	failedReconds[proxy.ID()] = true
	goto findLoop
}
