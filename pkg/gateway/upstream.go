package gateway

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	grpcproxy "github.com/nite-coder/bifrost/pkg/proxy/grpc"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"

	"github.com/cloudwego/hertz/pkg/app/client"
	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	httpServiceOpenConnections *prom.GaugeVec
)

func init() {
	httpServiceOpenConnections = prom.NewGaugeVec(
		prom.GaugeOpts{
			Name: "http_service_open_connections",
			Help: "Number of open connections for services",
		},
		[]string{"service_id", "target"},
	)

	prom.MustRegister(httpServiceOpenConnections)
}

type Upstream struct {
	opts        *config.UpstreamOptions
	proxies     atomic.Value
	counter     atomic.Uint64
	totalWeight uint32
	hasher      hash.Hash32
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

func createHTTPUpstream(bifrost *Bifrost, serviceOpts config.ServiceOptions, upstreamOptions config.UpstreamOptions) (*Upstream, error) {
	clientOpts := httpproxy.DefaultClientOptions()

	if serviceOpts.Timeout.Dail > 0 {
		clientOpts = append(clientOpts, client.WithDialTimeout(serviceOpts.Timeout.Dail))
	} else if bifrost.options.Default.Service.Timeout.Dail > 0 {
		clientOpts = append(clientOpts, client.WithDialTimeout(bifrost.options.Default.Service.Timeout.Dail))
	}

	if serviceOpts.Timeout.Read > 0 {
		clientOpts = append(clientOpts, client.WithClientReadTimeout(serviceOpts.Timeout.Read))
	} else if bifrost.options.Default.Service.Timeout.Read > 0 {
		clientOpts = append(clientOpts, client.WithDialTimeout(bifrost.options.Default.Service.Timeout.Read))
	}

	if serviceOpts.Timeout.Write > 0 {
		clientOpts = append(clientOpts, client.WithWriteTimeout(serviceOpts.Timeout.Write))
	} else if bifrost.options.Default.Service.Timeout.Write > 0 {
		clientOpts = append(clientOpts, client.WithDialTimeout(bifrost.options.Default.Service.Timeout.Write))
	}

	if serviceOpts.Timeout.MaxConnWait > 0 {
		clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(serviceOpts.Timeout.MaxConnWait))
	} else if bifrost.options.Default.Service.Timeout.MaxConnWait > 0 {
		clientOpts = append(clientOpts, client.WithDialTimeout(bifrost.options.Default.Service.Timeout.MaxConnWait))
	}

	if serviceOpts.MaxConnsPerHost != nil {
		clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*serviceOpts.MaxConnsPerHost))
	} else if bifrost.options.Default.Service.MaxConnsPerHost != nil {
		clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*bifrost.options.Default.Service.MaxConnsPerHost))
	}

	upstream := &Upstream{
		opts:   &upstreamOptions,
		hasher: fnv.New32a(),
	}

	proxies, err := buildHTTPProxyList(bifrost, upstream, clientOpts, serviceOpts, upstreamOptions)
	if err != nil {
		return nil, err
	}

	upstream.proxies.Store(proxies)

	return upstream, nil
}

func createGRPCUpstream(bifrost *Bifrost, serviceOptions config.ServiceOptions, upstreamOptions config.UpstreamOptions) (*Upstream, error) {
	upstream := &Upstream{
		opts:   &upstreamOptions,
		hasher: fnv.New32a(),
	}

	proxies, err := buildGRPCProxyList(bifrost, upstream, serviceOptions, upstreamOptions)
	if err != nil {
		return nil, err
	}

	upstream.proxies.Store(proxies)

	return upstream, nil
}

func newUpstream(bifrost *Bifrost, serviceOpts config.ServiceOptions, upstreamOptions config.UpstreamOptions) (*Upstream, error) {

	if len(upstreamOptions.ID) == 0 {
		return nil, errors.New("upstream id can't be empty")
	}

	if len(upstreamOptions.Targets) == 0 {
		return nil, fmt.Errorf("targets can't be empty. upstream id: %s", upstreamOptions.ID)
	}

	switch serviceOpts.Protocol {
	case config.ProtocolHTTP, config.ProtocolHTTP2:
		return createHTTPUpstream(bifrost, serviceOpts, upstreamOptions)
	case config.ProtocolGRPC:
		return createGRPCUpstream(bifrost, serviceOpts, upstreamOptions)
	}

	return nil, nil
}

func (u *Upstream) roundRobin() proxy.Proxy {
	proxies := u.proxies.Load().([]proxy.Proxy)
	if len(proxies) == 0 {
		return nil
	}

	if len(proxies) == 1 {
		proxy := proxies[0]
		if proxy.IsAvailable() {
			return proxy
		}
		return nil
	}

	failedReconds := map[string]bool{}

findLoop:
	u.counter.Add(1)

	if u.counter.Load() >= uint64(math.MaxUint64) {
		u.counter.Store(1)
	}

	// By subtracting 1 from the counter value, the code is effectively making the counter 0-indexed,
	// so that the first element in the u.proxies list is selected when the counter is at 1.
	index := (u.counter.Load() - 1) % uint64(len(proxies))
	proxy := proxies[index]

	if proxy.IsAvailable() {
		return proxy
	}

	// no live upstream
	if len(failedReconds) == len(proxies) {
		return nil
	}

	failedReconds[proxy.ID()] = true
	goto findLoop
}

func (u *Upstream) weighted() proxy.Proxy {
	proxies := u.proxies.Load().([]proxy.Proxy)
	if len(proxies) == 0 {
		return nil
	}

	if len(proxies) == 1 {
		proxy := proxies[0]
		if proxy.IsAvailable() {
			return proxy
		}
		return nil
	}

	failedReconds := map[string]bool{}

findLoop:

	if u.totalWeight > math.MaxInt32 {
		u.totalWeight = math.MaxInt32
	}
	val := int64(u.totalWeight)

	randomWeight, _ := getRandomNumber(val)

	for _, proxy := range proxies {
		randomWeight -= int64(proxy.Weight())
		if randomWeight < 0 {

			if proxy.IsAvailable() {
				return proxy
			}

			// no live upstream
			if len(failedReconds) == len(proxies) {
				return nil
			}

			failedReconds[proxy.ID()] = true
			goto findLoop
		}
	}

	return nil
}

func (u *Upstream) random() proxy.Proxy {
	proxies := u.proxies.Load().([]proxy.Proxy)
	if len(proxies) == 0 {
		return nil
	}

	if len(proxies) == 1 {
		proxy := proxies[0]
		if proxy.IsAvailable() {
			return proxy
		}
		return nil
	}

	failedReconds := map[string]bool{}

findLoop:
	selectedIndex, _ := getRandomNumber(int64(len(proxies)))
	proxy := proxies[selectedIndex]

	if proxy.IsAvailable() {
		return proxy
	}

	// no live upstream
	if len(failedReconds) == len(proxies) {
		return nil
	}

	failedReconds[proxy.ID()] = true
	goto findLoop
}

func (u *Upstream) hasing(key string) proxy.Proxy {
	proxies := u.proxies.Load().([]proxy.Proxy)
	if len(proxies) == 0 {
		return nil
	}

	if len(proxies) == 1 {
		proxy := proxies[0]
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
		allProxies = make([]proxy.Proxy, len(proxies))
		copy(allProxies, proxies)

		for failedProxyID := range failedReconds {
			for idx, proxy := range allProxies {
				if proxy.ID() == failedProxyID {
					allProxies = append(allProxies[:idx], allProxies[idx+1:]...)
					break
				}
			}
		}
	} else {
		allProxies = proxies
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
	if len(failedReconds) == len(proxies) {
		return nil
	}

	failedReconds[proxy.ID()] = true
	goto findLoop
}

func buildHTTPProxyList(bifrost *Bifrost, upstream *Upstream, clientOpts []hzconfig.ClientOption, serviceOptions config.ServiceOptions, updateOptions config.UpstreamOptions) ([]proxy.Proxy, error) {
	proxies := make([]proxy.Proxy, 0)

	for _, targetOpts := range updateOptions.Targets {

		if updateOptions.Strategy == config.WeightedStrategy && targetOpts.Weight == 0 {
			return nil, fmt.Errorf("weight can't be 0. upstream id: %s, target: %s", updateOptions.ID, targetOpts.Target)
		}

		upstream.totalWeight += targetOpts.Weight

		targetHost, targetPort, err := net.SplitHostPort(targetOpts.Target)
		if err != nil {
			targetHost = targetOpts.Target
		}

		ips, err := bifrost.resolver.Lookup(context.Background(), targetHost)
		if err != nil {
			return nil, fmt.Errorf("lookup upstream host '%s' error: %w", targetHost, err)
		}

		for _, ip := range ips {
			addr, err := url.Parse(serviceOptions.Url)
			if err != nil {
				return nil, err
			}

			port := targetPort
			if len(addr.Port()) > 0 {
				port = addr.Port()
			}

			if strings.EqualFold(addr.Scheme, "https") {
				clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
					// when client uses ip address to connect to server, client need to set the ServerName to the domain name you want to use
					ServerName:         targetHost,
					InsecureSkipVerify: !serviceOptions.TLSVerify, //nolint:gosec
				}))
			}

			if bifrost.options.Metrics.Prometheus.Enabled {
				clientOpts = append(clientOpts, client.WithConnStateObserve(func(hcs hzconfig.HostClientState) {
					labels := make(prom.Labels)
					labels["service_id"] = serviceOptions.ID
					labels["target"] = hcs.ConnPoolState().Addr

					httpServiceOpenConnections.With(labels).Set(float64(hcs.ConnPoolState().TotalConnNum))
				}))
			}

			url := fmt.Sprintf("%s://%s%s", addr.Scheme, ip, addr.Path)

			if port != "" {
				url = fmt.Sprintf("%s://%s:%s%s", addr.Scheme, ip, port, addr.Path)
			}

			clientOptions := httpproxy.ClientOptions{
				IsHTTP2:   serviceOptions.Protocol == config.ProtocolHTTP2,
				HZOptions: clientOpts,
			}

			client, err := httpproxy.NewClient(clientOptions)
			if err != nil {
				return nil, err
			}

			var maxFails uint
			if updateOptions.HealthCheck.Passive.MaxFails == nil {
				maxFails = bifrost.options.Default.Upstream.MaxFails
			} else {
				maxFails = *updateOptions.HealthCheck.Passive.MaxFails
			}

			var failTimeout time.Duration
			if updateOptions.HealthCheck.Passive.FailTimeout > 0 {
				failTimeout = updateOptions.HealthCheck.Passive.FailTimeout
			} else if bifrost.options.Default.Upstream.FailTimeout > 0 {
				failTimeout = bifrost.options.Default.Upstream.FailTimeout
			}

			proxyOptions := httpproxy.Options{
				Target:           url,
				Protocol:         serviceOptions.Protocol,
				Weight:           targetOpts.Weight,
				MaxFails:         maxFails,
				FailTimeout:      failTimeout,
				HeaderHost:       targetHost,
				IsTracingEnabled: bifrost.options.Tracing.Enabled,
				ServiceID:        serviceOptions.ID,
			}

			proxy, err := httpproxy.New(proxyOptions, client)

			if err != nil {
				return nil, err
			}
			proxies = append(proxies, proxy)
		}
	}

	return proxies, nil
}

func buildGRPCProxyList(bifrost *Bifrost, upstream *Upstream, serviceOptions config.ServiceOptions, upstreamOptions config.UpstreamOptions) ([]proxy.Proxy, error) {
	proxies := make([]proxy.Proxy, 0)

	for _, targetOpts := range upstreamOptions.Targets {

		if upstreamOptions.Strategy == config.WeightedStrategy && targetOpts.Weight == 0 {
			return nil, fmt.Errorf("weight can't be 0. upstream id: %s, target: %s", upstreamOptions.ID, targetOpts.Target)
		}

		upstream.totalWeight += targetOpts.Weight

		targetHost, targetPort, err := net.SplitHostPort(targetOpts.Target)
		if err != nil {
			targetHost = targetOpts.Target
		}

		ips, err := bifrost.resolver.Lookup(context.Background(), targetHost)
		if err != nil {
			return nil, fmt.Errorf("lookup upstream host error: %w", err)
		}

		for _, ip := range ips {
			addr, err := url.Parse(serviceOptions.Url)
			if err != nil {
				return nil, err
			}

			port := targetPort
			if len(addr.Port()) > 0 {
				port = addr.Port()
			}

			url := fmt.Sprintf("grpc://%s%s", ip, addr.Path)
			if port != "" {
				url = fmt.Sprintf("grpc://%s:%s%s", ip, port, addr.Path)
			}

			var maxFails uint
			if upstreamOptions.HealthCheck.Passive.MaxFails == nil {
				maxFails = bifrost.options.Default.Upstream.MaxFails
			} else {
				maxFails = *upstreamOptions.HealthCheck.Passive.MaxFails
			}

			var failTimeout time.Duration
			if upstreamOptions.HealthCheck.Passive.FailTimeout > 0 {
				failTimeout = upstreamOptions.HealthCheck.Passive.FailTimeout
			} else if bifrost.options.Default.Upstream.FailTimeout > 0 {
				failTimeout = bifrost.options.Default.Upstream.FailTimeout
			}

			grpcOptions := grpcproxy.Options{
				Target:      url,
				TLSVerify:   serviceOptions.TLSVerify,
				Weight:      1,
				MaxFails:    maxFails,
				FailTimeout: failTimeout,
				Timeout:     serviceOptions.Timeout.GRPC,
			}

			grpcProxy, err := grpcproxy.New(grpcOptions)
			if err != nil {
				return nil, err
			}
			proxies = append(proxies, grpcProxy)
		}
	}

	return proxies, nil
}
