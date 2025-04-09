package gateway

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"log/slog"
	"math"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/provider/dns"
	"github.com/nite-coder/bifrost/pkg/proxy"
	grpcproxy "github.com/nite-coder/bifrost/pkg/proxy/grpc"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
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
	bifrost        *Bifrost
	options        *config.UpstreamOptions
	ServiceOptions *config.ServiceOptions
	discovery      provider.ServiceDiscovery
	proxies        atomic.Value
	counter        atomic.Uint64
	totalWeight    uint32
	hasher         hash.Hash32
	watchOnce      sync.Once
}

func newUpstream(bifrost *Bifrost, serviceOptions config.ServiceOptions, upstreamOptions config.UpstreamOptions) (*Upstream, error) {
	if len(upstreamOptions.ID) == 0 {
		return nil, errors.New("upstream id can't be empty")
	}

	if len(upstreamOptions.Targets) == 0 {
		return nil, fmt.Errorf("targets can't be empty. upstream id: %s", upstreamOptions.ID)
	}

	upstream := &Upstream{
		bifrost:        bifrost,
		options:        &upstreamOptions,
		ServiceOptions: &serviceOptions,
		hasher:         fnv.New32a(),
	}

	switch strings.ToLower(upstreamOptions.Discovery.Type) {
	case "dns":
		discovery := dns.NewDNSServiceDiscovery(bifrost.options.Providers.DNS.Servers, bifrost.options.Providers.DNS.Valid)
		upstream.discovery = discovery
	default:
		discovery := NewResolverDiscovery(upstream)
		upstream.discovery = discovery
	}

	err := upstream.refreshProxies(nil)
	if err != nil {
		return nil, err
	}

	return upstream, nil
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

func (u *Upstream) refreshProxies(instances []provider.Instancer) error {
	var err error
	if len(instances) == 0 {
		instances, err = u.discovery.GetInstances(context.Background(), u.options.Discovery.ServiceName)
		if err != nil {
			return err
		}
	}

	proxies := make([]proxy.Proxy, 0)

	for _, instance := range instances {

		if u.options.Strategy == config.WeightedStrategy && instance.Weight() == 0 {
			return fmt.Errorf("weight can't be 0. upstream id: %s, target: %s", u.options.ID, instance.Address())
		}

		u.totalWeight += instance.Weight()

		targetHost, targetPort, err := net.SplitHostPort(instance.Address().String())
		if err != nil {
			targetHost = instance.Address().String()
		}

		addr, err := url.Parse(u.ServiceOptions.Url)
		if err != nil {
			return err
		}

		port := ""
		if len(addr.Port()) > 0 {
			port = addr.Port()
		} else if targetPort != "" && targetPort != "0" {
			port = targetPort
		}

		serverName, _ := instance.Tag("server_name")

		clientOpts := httpproxy.DefaultClientOptions()

		if u.ServiceOptions.Timeout.Dail > 0 {
			clientOpts = append(clientOpts, client.WithDialTimeout(u.ServiceOptions.Timeout.Dail))
		} else if u.bifrost.options.Default.Service.Timeout.Dail > 0 {
			clientOpts = append(clientOpts, client.WithDialTimeout(u.bifrost.options.Default.Service.Timeout.Dail))
		}

		if u.ServiceOptions.Timeout.Read > 0 {
			clientOpts = append(clientOpts, client.WithClientReadTimeout(u.ServiceOptions.Timeout.Read))
		} else if u.bifrost.options.Default.Service.Timeout.Read > 0 {
			clientOpts = append(clientOpts, client.WithDialTimeout(u.bifrost.options.Default.Service.Timeout.Read))
		}

		if u.ServiceOptions.Timeout.Write > 0 {
			clientOpts = append(clientOpts, client.WithWriteTimeout(u.ServiceOptions.Timeout.Write))
		} else if u.bifrost.options.Default.Service.Timeout.Write > 0 {
			clientOpts = append(clientOpts, client.WithDialTimeout(u.bifrost.options.Default.Service.Timeout.Write))
		}

		if u.ServiceOptions.Timeout.MaxConnWait > 0 {
			clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(u.ServiceOptions.Timeout.MaxConnWait))
		} else if u.bifrost.options.Default.Service.Timeout.MaxConnWait > 0 {
			clientOpts = append(clientOpts, client.WithDialTimeout(u.bifrost.options.Default.Service.Timeout.MaxConnWait))
		}

		if u.ServiceOptions.MaxConnsPerHost != nil {
			clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*u.ServiceOptions.MaxConnsPerHost))
		} else if u.bifrost.options.Default.Service.MaxConnsPerHost != nil {
			clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*u.bifrost.options.Default.Service.MaxConnsPerHost))
		}

		if strings.EqualFold(addr.Scheme, "https") {
			clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{ // nolint
				// when client uses ip address to connect to server, client need to set the ServerName to the domain name you want to use
				ServerName:         serverName,
				InsecureSkipVerify: !u.ServiceOptions.TLSVerify, //nolint:gosec
			}))
		}

		if u.bifrost.options.Metrics.Prometheus.Enabled {
			clientOpts = append(clientOpts, client.WithConnStateObserve(func(hcs hzconfig.HostClientState) {
				labels := make(prom.Labels)
				labels["service_id"] = u.ServiceOptions.ID
				labels["target"] = hcs.ConnPoolState().Addr

				httpServiceOpenConnections.With(labels).Set(float64(hcs.ConnPoolState().TotalConnNum))
			}))
		}

		var maxFails uint
		if u.options.HealthCheck.Passive.MaxFails == nil {
			maxFails = u.bifrost.options.Default.Upstream.MaxFails
		} else {
			maxFails = *u.options.HealthCheck.Passive.MaxFails
		}

		var failTimeout time.Duration
		if u.options.HealthCheck.Passive.FailTimeout > 0 {
			failTimeout = u.options.HealthCheck.Passive.FailTimeout
		} else if u.bifrost.options.Default.Upstream.FailTimeout > 0 {
			failTimeout = u.bifrost.options.Default.Upstream.FailTimeout
		}

		url := ""
		switch u.ServiceOptions.Protocol {
		case config.ProtocolHTTP, config.ProtocolHTTP2:
			url = fmt.Sprintf("%s://%s%s", addr.Scheme, targetHost, addr.Path)

			if port != "" {
				url = fmt.Sprintf("%s://%s:%s%s", addr.Scheme, targetHost, port, addr.Path)
			}

			clientOptions := httpproxy.ClientOptions{
				IsHTTP2:   u.ServiceOptions.Protocol == config.ProtocolHTTP2,
				HZOptions: clientOpts,
			}

			client, err := httpproxy.NewClient(clientOptions)
			if err != nil {
				return err
			}

			proxyOptions := httpproxy.Options{
				Target:           url,
				Protocol:         u.ServiceOptions.Protocol,
				Weight:           instance.Weight(),
				MaxFails:         maxFails,
				FailTimeout:      failTimeout,
				HeaderHost:       serverName, // when client uses ip address to connect to server, client need to set the host to the domain name you want to use
				IsTracingEnabled: u.bifrost.options.Tracing.Enabled,
				ServiceID:        u.ServiceOptions.ID,
			}

			proxy, err := httpproxy.New(proxyOptions, client)
			if err != nil {
				return err
			}
			proxies = append(proxies, proxy)
		case config.ProtocolGRPC:
			url = fmt.Sprintf("grpc://%s%s", targetHost, addr.Path)

			if port != "" {
				url = fmt.Sprintf("grpc://%s:%s%s", targetHost, port, addr.Path)
			}

			grpcOptions := grpcproxy.Options{
				Target:      url,
				TLSVerify:   u.ServiceOptions.TLSVerify,
				Weight:      1,
				MaxFails:    maxFails,
				FailTimeout: failTimeout,
				Timeout:     u.ServiceOptions.Timeout.GRPC,
			}

			grpcProxy, err := grpcproxy.New(grpcOptions)
			if err != nil {
				return err
			}
			proxies = append(proxies, grpcProxy)
		}
	}

	u.proxies.Store(proxies)
	return nil
}

func (u *Upstream) watch() {
	u.watchOnce.Do(func() {
		watchCh, err := u.discovery.Watch(context.Background(), u.options.Discovery.ServiceName)
		if err != nil {
			slog.Error("fail to watch upstream", "error", err.Error(), "upstream_id", u.options.ID)
		}

		go func() {
			for instances := range watchCh {
				_ = u.refreshProxies(instances)
			}
		}()
	})
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
