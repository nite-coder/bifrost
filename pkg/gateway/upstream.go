package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	bifrostConfig "http-benchmark/pkg/config"
	"math"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/rs/dnscache"
)

type Upstream struct {
	opts        bifrostConfig.UpstreamOptions
	proxies     []*ReverseProxy
	counter     atomic.Uint64
	totalWeight int
	rng         *rand.Rand
}

var defaultClientOptions = []config.ClientOption{
	client.WithNoDefaultUserAgentHeader(true),
	client.WithDisableHeaderNamesNormalizing(true),
	client.WithDisablePathNormalizing(true),
	client.WithMaxConnsPerHost(math.MaxInt),
	client.WithDialTimeout(10 * time.Second),
	client.WithClientReadTimeout(10 * time.Second),
	client.WithWriteTimeout(10 * time.Second),
	client.WithMaxIdleConnDuration(120 * time.Second),
	client.WithKeepAlive(true),
}

func newUpstream(bifrost *Bifrost, serviceOpts bifrostConfig.ServiceOptions, opts bifrostConfig.UpstreamOptions) (*Upstream, error) {

	if len(opts.ID) == 0 {
		return nil, fmt.Errorf("upstream id can't be empty")
	}

	if len(opts.Targets) == 0 {
		return nil, fmt.Errorf("targets can't be empty. upstream id: %s", opts.ID)
	}

	// direct proxy
	clientOpts := defaultClientOptions

	if serviceOpts.DailTimeout != nil {
		clientOpts = append(clientOpts, client.WithDialTimeout(*serviceOpts.DailTimeout))
	}

	if serviceOpts.ReadTimeout != nil {
		clientOpts = append(clientOpts, client.WithClientReadTimeout(*serviceOpts.ReadTimeout))
	}

	if serviceOpts.WriteTimeout != nil {
		clientOpts = append(clientOpts, client.WithWriteTimeout(*serviceOpts.WriteTimeout))
	}

	if serviceOpts.MaxConnWaitTimeout != nil {
		clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(*serviceOpts.MaxConnWaitTimeout))
	}

	if serviceOpts.MaxIdleConnsPerHost != nil {
		clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*serviceOpts.MaxIdleConnsPerHost))
	}

	upstream := &Upstream{
		opts:    opts,
		proxies: make([]*ReverseProxy, 0),
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	for _, targetOpts := range opts.Targets {

		if opts.Strategy == bifrostConfig.WeightedStrategy && targetOpts.Weight == 0 {
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
			clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
				InsecureSkipVerify: serviceOpts.TLSVerify,
			}))
		}

		port := targetPort
		if len(addr.Port()) > 0 {
			port = addr.Port()
		}

		url := fmt.Sprintf("%s://%s:%s%s", serviceOpts.Protocol, targetHost, port, addr.Path)
		proxy, err := newSingleHostReverseProxy(url, bifrost.opts.Observability.Tracing.Enabled, targetOpts.Weight, clientOpts...)

		if err != nil {
			return nil, err
		}
		upstream.proxies = append(upstream.proxies, proxy)
	}

	if opts.Strategy == bifrostConfig.RoundRobinStrategy {
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			for range ticker.C {
				upstream.counter.Store(0)
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
	selectedIndex := u.rng.Intn(len(u.proxies))
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
