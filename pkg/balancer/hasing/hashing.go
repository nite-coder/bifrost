package hasing

import (
	"context"
	"hash"
	"hash/fnv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/variable"
)

func init() {
	_ = balancer.Register("hashing", func(proxies []proxy.Proxy, option *config.UpstreamOptions) (balancer.Balancer, error) {
		b := NewBalancer(proxies, option.HashOn)
		return b, nil
	})
}

type HashingBalancer struct {
	hasher  hash.Hash32
	hashon  string
	proxies []proxy.Proxy
}

func NewBalancer(proxies []proxy.Proxy, hashon string) *HashingBalancer {
	return &HashingBalancer{
		proxies: proxies,
		hashon:  hashon,
		hasher:  fnv.New32a(),
	}
}

func (b *HashingBalancer) Proxies() []proxy.Proxy {
	return b.proxies
}

func (b *HashingBalancer) Select(ctx context.Context, c *app.RequestContext) (proxy.Proxy, error) {
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

	val := variable.GetString(b.hashon, c)
	b.hasher.Write([]byte(val))
	hashValue := b.hasher.Sum32()
	failedReconds := map[string]bool{}

findLoop:
	var allProxies []proxy.Proxy
	if len(failedReconds) > 0 {
		allProxies = make([]proxy.Proxy, len(b.proxies))
		copy(allProxies, b.proxies)
		for failedProxyID := range failedReconds {
			for idx, proxy := range allProxies {
				if proxy.ID() == failedProxyID {
					allProxies = append(allProxies[:idx], allProxies[idx+1:]...)
					break
				}
			}
		}
	} else {
		allProxies = b.proxies
	}
	if len(allProxies) == 0 {
		return nil, balancer.ErrNoAvailable
	}
	selectedIndex := int(hashValue) % len(allProxies)
	proxy := allProxies[selectedIndex]
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
