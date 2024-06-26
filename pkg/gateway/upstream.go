package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"hash"
	"hash/fnv"
	"http-benchmark/pkg/config"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/rs/dnscache"
)

type Upstream struct {
	opts        *config.UpstreamOptions
	proxies     []*ReverseProxy
	counter     atomic.Uint64
	totalWeight int
	hasher      hash.Hash32
	rng         *rand.Rand
}

func newDefaultClientOptions() []hzconfig.ClientOption {
	return []hzconfig.ClientOption{
		client.WithNoDefaultUserAgentHeader(true),
		client.WithDisableHeaderNamesNormalizing(true),
		client.WithDisablePathNormalizing(true),
		client.WithDialTimeout(10 * time.Second),
		client.WithClientReadTimeout(60 * time.Second),
		client.WithWriteTimeout(60 * time.Second),
		client.WithMaxIdleConnDuration(120 * time.Second),
		client.WithKeepAlive(true),
	}
}

func newUpstream(bifrost *Bifrost, serviceOpts config.ServiceOptions, opts config.UpstreamOptions) (*Upstream, error) {

	if len(opts.ID) == 0 {
		return nil, fmt.Errorf("upstream id can't be empty")
	}

	if len(opts.Targets) == 0 {
		return nil, fmt.Errorf("targets can't be empty. upstream id: %s", opts.ID)
	}

	// direct proxy
	clientOpts := newDefaultClientOptions()

	if serviceOpts.Timeout.DailTimeout > 0 {
		clientOpts = append(clientOpts, client.WithDialTimeout(serviceOpts.Timeout.DailTimeout))
	}

	if serviceOpts.Timeout.ReadTimeout > 0 {
		clientOpts = append(clientOpts, client.WithClientReadTimeout(serviceOpts.Timeout.ReadTimeout))
	}

	if serviceOpts.Timeout.WriteTimeout > 0 {
		clientOpts = append(clientOpts, client.WithWriteTimeout(serviceOpts.Timeout.WriteTimeout))
	}

	if serviceOpts.Timeout.MaxConnWaitTimeout > 0 {
		clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(serviceOpts.Timeout.MaxConnWaitTimeout))
	}

	if serviceOpts.MaxIdleConnsPerHost != nil {
		clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*serviceOpts.MaxIdleConnsPerHost))
	}

	upstream := &Upstream{
		opts:    &opts,
		proxies: make([]*ReverseProxy, 0),
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
		hasher:  fnv.New32a(),
	}

	for _, targetOpts := range opts.Targets {

		if opts.Strategy == config.WeightedStrategy && targetOpts.Weight == 0 {
			return nil, fmt.Errorf("weight can't be 0. upstream id: %s, target: %s", opts.ID, targetOpts.Target)
		}

		upstream.totalWeight += targetOpts.Weight

		targetHost, targetPort, err := net.SplitHostPort(targetOpts.Target)
		if err != nil {
			targetHost = targetOpts.Target
		}

		var dnsResolver dnscache.DNSResolver
		if allowDNS(targetHost) {
			_, err := bifrost.resolver.LookupHost(context.Background(), targetHost)
			if err != nil {
				return nil, fmt.Errorf("lookup upstream host error: %v", err)
			}
			dnsResolver = bifrost.resolver
		}

		addr, err := url.Parse(serviceOpts.Url)
		if err != nil {
			return nil, err
		}

		switch strings.ToLower(addr.Scheme) {
		case "http":
			if dnsResolver != nil {
				clientOpts = append(clientOpts, client.WithDialer(newHTTPDialer(dnsResolver)))
			}
		case "https":
			if dnsResolver != nil {
				clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
					InsecureSkipVerify: serviceOpts.TLSVerify,
				}))
				clientOpts = append(clientOpts, client.WithDialer(newHTTPSDialer(dnsResolver)))
			}
		}

		port := targetPort
		if len(addr.Port()) > 0 {
			port = addr.Port()
		}

		url := fmt.Sprintf("%s://%s:%s%s", serviceOpts.Protocol, targetHost, port, addr.Path)
		proxy, err := newSingleHostReverseProxy(url, bifrost.opts.Tracing.Enabled, targetOpts.Weight, clientOpts...)

		if err != nil {
			return nil, err
		}
		upstream.proxies = append(upstream.proxies, proxy)
	}

	if opts.Strategy == config.RoundRobinStrategy {
		go func() {
			t := time.NewTimer(5 * time.Minute)
			defer t.Stop()

			for {
				select {
				case <-bifrost.stopCh:
					return
				case <-t.C:
					upstream.counter.Store(0)
				}
			}
		}()
	}

	return upstream, nil
}

func (u *Upstream) roundRobin() *ReverseProxy {
	if len(u.proxies) == 1 {
		return u.proxies[0]
	}

	index := u.counter.Add(1)
	proxy := u.proxies[(int(index)-1)%len(u.proxies)]
	return proxy
}

func (u *Upstream) weighted() *ReverseProxy {
	if len(u.proxies) == 1 {
		return u.proxies[0]
	}

	randomWeight := u.rng.Intn(u.totalWeight)

	for _, proxy := range u.proxies {
		randomWeight -= proxy.weight
		if randomWeight < 0 {
			return proxy
		}
	}

	return nil
}

func (u *Upstream) random() *ReverseProxy {
	if len(u.proxies) == 1 {
		return u.proxies[0]
	}

	selectedIndex := u.rng.Intn(len(u.proxies))
	return u.proxies[selectedIndex]
}

func (u *Upstream) hasing(key string) *ReverseProxy {
	if len(u.proxies) == 1 {
		return u.proxies[0]
	}

	u.hasher.Write([]byte(key))
	hashValue := u.hasher.Sum32()

	selectedIndex := int(hashValue) % len(u.proxies)
	return u.proxies[selectedIndex]
}

func allowDNS(address string) bool {

	ip := net.ParseIP(address)
	if ip != nil {
		return false
	}

	if address == "localhost" || address == "[::1]" {
		return false
	}

	return true
}
