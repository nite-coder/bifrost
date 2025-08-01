package weighted

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/proxy"
)

func init() {
	_ = balancer.Register([]string{"weighted"}, func(proxies []proxy.Proxy, param any) (balancer.Balancer, error) {
		return NewBalancer(proxies)
	})
}

type WeightedBalancer struct {
	totalWeight uint32
	proxies     []proxy.Proxy
}

func NewBalancer(proxies []proxy.Proxy) (*WeightedBalancer, error) {

	var totalWeight uint32
	for _, proxy := range proxies {
		weight := proxy.Weight()
		if weight == 0 {
			weight = 1
		}

		totalWeight += weight
	}

	return &WeightedBalancer{
		proxies:     proxies,
		totalWeight: totalWeight,
	}, nil
}

func (b *WeightedBalancer) Proxies() []proxy.Proxy {
	return b.proxies
}

func (b *WeightedBalancer) Select(ctx context.Context, hzCtx *app.RequestContext) (proxy.Proxy, error) {
	if b.proxies == nil {
		return nil, balancer.ErrNotAvailable
	}

	if len(b.proxies) == 1 {
		proxy := b.proxies[0]
		if proxy.IsAvailable() {
			return proxy, nil
		}
		return nil, balancer.ErrNotAvailable
	}

	failedReconds := map[string]bool{}

findLoop:
	if b.totalWeight > math.MaxInt32 {
		b.totalWeight = math.MaxInt32
	}
	val := int64(b.totalWeight)
	randomWeight, _ := getRandomNumber(val)
	for _, proxy := range b.proxies {
		randomWeight -= int64(proxy.Weight())
		if randomWeight < 0 {
			if proxy.IsAvailable() {
				return proxy, nil
			}
			// no live upstream
			if len(failedReconds) == len(b.proxies) {
				return nil, balancer.ErrNotAvailable
			}
			failedReconds[proxy.ID()] = true
			goto findLoop
		}
	}
	return nil, balancer.ErrNotAvailable
}

func getRandomNumber(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}
