package random

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/proxy"
)

func init() {
	_ = balancer.Register([]string{"random"}, func(proxies []proxy.Proxy, params any) (balancer.Balancer, error) {
		b := NewBalancer(proxies)
		return b, nil
	})
}

type RandomBalancer struct {
	counter atomic.Uint64
	proxies []proxy.Proxy
}

func NewBalancer(proxies []proxy.Proxy) *RandomBalancer {
	return &RandomBalancer{
		counter: atomic.Uint64{},
		proxies: proxies,
	}
}

func (b *RandomBalancer) Proxies() []proxy.Proxy {
	return b.proxies
}

func (b *RandomBalancer) Select(ctx context.Context, hzCtx *app.RequestContext) (proxy.Proxy, error) {
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
	selectedIndex, _ := getRandomNumber(int64(len(b.proxies)))
	proxy := b.proxies[selectedIndex]
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

func getRandomNumber(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}
