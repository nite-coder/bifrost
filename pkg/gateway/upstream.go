package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"hash"
	"hash/fnv"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/proxy"
	grpcproxy "http-benchmark/pkg/proxy/grpc"
	httpproxy "http-benchmark/pkg/proxy/http"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/rs/dnscache"
)

type Upstream struct {
	opts        *config.UpstreamOptions
	proxies     []proxy.Proxy
	counter     atomic.Uint64
	totalWeight uint
	hasher      hash.Hash32
	rng         *rand.Rand
}

func loadUpstreams(bifrost *Bifrost, serviceOpts config.ServiceOptions) (map[string]*Upstream, error) {
	upstreams := map[string]*Upstream{}

	for id, upstreamOpts := range bifrost.options.Upstreams {
		upstreamOpts.ID = id

		upstream, err := newUpstream(bifrost, serviceOpts, upstreamOpts)
		if err != nil {
			return nil, err
		}

		upstreams[id] = upstream

	}

	return upstreams, nil
}

func createHTTPUpstream(bifrost *Bifrost, serviceOpts config.ServiceOptions, opts config.UpstreamOptions) (*Upstream, error) {
	clientOpts := httpproxy.DefaultClientOptions()

	if serviceOpts.Timeout.Dail > 0 {
		clientOpts = append(clientOpts, client.WithDialTimeout(serviceOpts.Timeout.Dail))
	}

	if serviceOpts.Timeout.Read > 0 {
		clientOpts = append(clientOpts, client.WithClientReadTimeout(serviceOpts.Timeout.Read))
	}

	if serviceOpts.Timeout.Write > 0 {
		clientOpts = append(clientOpts, client.WithWriteTimeout(serviceOpts.Timeout.Write))
	}

	if serviceOpts.Timeout.MaxConnWait > 0 {
		clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(serviceOpts.Timeout.MaxConnWait))
	}

	if serviceOpts.MaxConnsPerHost != nil {
		clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*serviceOpts.MaxConnsPerHost))
	}

	upstream := &Upstream{
		opts:    &opts,
		proxies: make([]proxy.Proxy, 0),
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

		port := targetPort
		if len(addr.Port()) > 0 {
			port = addr.Port()
		}

		switch strings.ToLower(addr.Scheme) {
		case "http":
			if dnsResolver != nil {
				clientOpts = append(clientOpts, client.WithDialer(newHTTPDialer(dnsResolver)))
			}
		case "https":
			if dnsResolver != nil {
				clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
					InsecureSkipVerify: !serviceOpts.TLSVerify,
				}))
				clientOpts = append(clientOpts, client.WithDialer(newHTTPSDialer(dnsResolver)))
			}
		}

		url := fmt.Sprintf("%s://%s%s", addr.Scheme, targetHost, addr.Path)

		if port != "" {
			url = fmt.Sprintf("%s://%s:%s%s", addr.Scheme, targetHost, port, addr.Path)
		}

		clientOptions := httpproxy.ClientOptions{
			IsTracingEnabled: bifrost.options.Tracing.OTLP.Enabled,
			IsHTTP2:          serviceOpts.Protocol == config.ProtocolHTTP2,
			HZOptions:        clientOpts,
		}

		client, err := httpproxy.NewClient(clientOptions)
		if err != nil {
			return nil, err
		}

		proxyOptions := httpproxy.Options{
			Target:      url,
			Protocol:    serviceOpts.Protocol,
			Weight:      targetOpts.Weight,
			MaxFails:    targetOpts.MaxFails,
			FailTimeout: targetOpts.FailTimeout,
		}

		proxy, err := httpproxy.New(proxyOptions, client)

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

func createGRPCUpstream(serviceOpts config.ServiceOptions, opts config.UpstreamOptions) (*Upstream, error) {
	upstream := &Upstream{
		opts:    &opts,
		proxies: make([]proxy.Proxy, 0),
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

		addr, err := url.Parse(serviceOpts.Url)
		if err != nil {
			return nil, err
		}

		port := targetPort
		if len(addr.Port()) > 0 {
			port = addr.Port()
		}

		url := fmt.Sprintf("grpc://%s%s", targetHost, addr.Path)
		if port != "" {
			url = fmt.Sprintf("grpc://%s:%s%s", targetHost, port, addr.Path)
		}

		grpcOptions := grpcproxy.Options{
			Target:      url,
			TLSVerify:   serviceOpts.TLSVerify,
			Weight:      1,
			MaxFails:    targetOpts.MaxFails,
			FailTimeout: targetOpts.FailTimeout,
		}

		grpcProxy, err := grpcproxy.New(grpcOptions)
		if err != nil {
			return nil, err
		}
		upstream.proxies = append(upstream.proxies, grpcProxy)
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

func newUpstream(bifrost *Bifrost, serviceOpts config.ServiceOptions, opts config.UpstreamOptions) (*Upstream, error) {

	if len(opts.ID) == 0 {
		return nil, fmt.Errorf("upstream id can't be empty")
	}

	if len(opts.Targets) == 0 {
		return nil, fmt.Errorf("targets can't be empty. upstream id: %s", opts.ID)
	}

	// direct proxy

	switch serviceOpts.Protocol {
	case config.ProtocolHTTP, config.ProtocolHTTP2:
		return createHTTPUpstream(bifrost, serviceOpts, opts)
	case config.ProtocolGRPC:
		return createGRPCUpstream(serviceOpts, opts)
	}

	return nil, nil
}

func (u *Upstream) roundRobin() proxy.Proxy {
	if len(u.proxies) == 1 {
		proxy := u.proxies[0]
		if proxy.IsAvailable() {
			return proxy
		}
		return nil
	}

	failedReconds := map[string]bool{}

findLoop:
	index := u.counter.Add(1)
	proxy := u.proxies[(int(index)-1)%len(u.proxies)]

	if proxy.IsAvailable() {
		return proxy
	}

	// no live upstream
	if len(failedReconds) == len(u.proxies) {
		return nil
	}

	failedReconds[proxy.ID()] = true
	goto findLoop
}

func (u *Upstream) weighted() proxy.Proxy {
	if len(u.proxies) == 1 {
		proxy := u.proxies[0]
		if proxy.IsAvailable() {
			return proxy
		}
		return nil
	}

	failedReconds := map[string]bool{}

findLoop:
	randomWeight := u.rng.Intn(int(u.totalWeight))

	for _, proxy := range u.proxies {
		randomWeight -= int(proxy.Weight())
		if randomWeight < 0 {

			if proxy.IsAvailable() {
				return proxy
			}

			// no live upstream
			if len(failedReconds) == len(u.proxies) {
				return nil
			}

			failedReconds[proxy.ID()] = true
			goto findLoop
		}
	}

	return nil
}

func (u *Upstream) random() proxy.Proxy {
	if len(u.proxies) == 1 {
		proxy := u.proxies[0]
		if proxy.IsAvailable() {
			return proxy
		}
		return nil
	}

	failedReconds := map[string]bool{}

findLoop:
	selectedIndex := u.rng.Intn(len(u.proxies))
	proxy := u.proxies[selectedIndex]

	if proxy.IsAvailable() {
		return proxy
	}

	// no live upstream
	if len(failedReconds) == len(u.proxies) {
		return nil
	}

	failedReconds[proxy.ID()] = true
	goto findLoop
}

func (u *Upstream) hasing(key string) proxy.Proxy {
	if len(u.proxies) == 1 {
		proxy := u.proxies[0]
		if proxy.IsAvailable() {
			return proxy
		}
		return nil
	}

	u.hasher.Write([]byte(key))
	hashValue := u.hasher.Sum32()

	failedReconds := map[string]bool{}

findLoop:
	var allProxies []proxy.Proxy

	if len(failedReconds) > 0 {
		allProxies = make([]proxy.Proxy, len(u.proxies))
		copy(allProxies, u.proxies)

		for failedProxyID := range failedReconds {
			for idx, proxy := range allProxies {
				if proxy.ID() == failedProxyID {
					allProxies = append(allProxies[:idx], allProxies[idx+1:]...)
					break
				}
			}
		}
	} else {
		allProxies = u.proxies
	}

	if len(allProxies) == 0 {
		return nil
	}

	selectedIndex := int(hashValue) % len(allProxies)
	proxy := allProxies[selectedIndex]

	if proxy.IsAvailable() {
		return proxy
	}

	// no live upstream
	if len(failedReconds) == len(u.proxies) {
		return nil
	}

	failedReconds[proxy.ID()] = true
	goto findLoop
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
