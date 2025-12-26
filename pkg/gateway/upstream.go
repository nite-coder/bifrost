package gateway

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/provider/dns"
	"github.com/nite-coder/bifrost/pkg/provider/k8s"
	"github.com/nite-coder/bifrost/pkg/provider/nacos"
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
	balancer       atomic.Value
	discovery      provider.ServiceDiscovery
	bifrost        *Bifrost
	options        *config.UpstreamOptions
	serviceOptions *config.ServiceOptions
	watchOnce      sync.Once
}

func newUpstream(bifrost *Bifrost, serviceOptions config.ServiceOptions, upstreamOptions config.UpstreamOptions) (*Upstream, error) {
	if len(upstreamOptions.ID) == 0 {
		return nil, errors.New("upstream id can't be empty")
	}
	if upstreamOptions.Discovery.Type == "" && len(upstreamOptions.Targets) == 0 {
		return nil, fmt.Errorf("targets can't be empty. upstream id: %s", upstreamOptions.ID)
	}

	upstream := &Upstream{
		bifrost:        bifrost,
		options:        &upstreamOptions,
		serviceOptions: &serviceOptions,
	}
	switch strings.ToLower(upstreamOptions.Discovery.Type) {
	case "dns":
		if !bifrost.options.Providers.DNS.Enabled {
			return nil, fmt.Errorf("dns provider is disabled. upstream id: %s", upstreamOptions.ID)
		}
		discovery, err := dns.NewDNSServiceDiscovery(bifrost.options.Providers.DNS.Servers, bifrost.options.Providers.DNS.Valid)
		if err != nil {
			return nil, err
		}
		upstream.discovery = discovery
	case "nacos":
		if !bifrost.options.Providers.Nacos.Discovery.Enabled {
			return nil, fmt.Errorf("nacos discovery provider is disabled. upstream id: %s", upstreamOptions.ID)
		}
		options := nacos.Options{
			Username:    bifrost.options.Providers.Nacos.Discovery.Username,
			Password:    bifrost.options.Providers.Nacos.Discovery.Password,
			NamespaceID: bifrost.options.Providers.Nacos.Discovery.NamespaceID,
			Prefix:      bifrost.options.Providers.Nacos.Discovery.Prefix,
			CacheDir:    bifrost.options.Providers.Nacos.Discovery.CacheDir,
			Endpoints:   bifrost.options.Providers.Nacos.Discovery.Endpoints,
			LogDir:      bifrost.options.Providers.Nacos.Discovery.LogDir,
			LogLevel:    bifrost.options.Providers.Nacos.Discovery.LogLevel,
		}
		discovery, err := nacos.NewNacosServiceDiscovery(options)
		if err != nil {
			return nil, err
		}
		upstream.discovery = discovery
	case "k8s":
		if !bifrost.options.Providers.K8S.Enabled {
			return nil, fmt.Errorf("k8s provider is disabled. upstream id: %s", upstreamOptions.ID)
		}
		option := k8s.Options{
			APIServer: bifrost.options.Providers.K8S.APIServer,
		}
		discovery, err := k8s.NewK8sDiscovery(option)
		if err != nil {
			return nil, err
		}
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

func (u *Upstream) Balancer() balancer.Balancer {
	val := u.balancer.Load()
	if val == nil {
		return nil
	}
	b, ok := val.(balancer.Balancer)
	if !ok {
		return nil
	}
	return b
}

// generateProxyHash generates a unique hash for a proxy based on its target and tags.
func generateProxyHash(p proxy.Proxy) string {
	tags := p.Tags()
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	// Sort keys for consistent hash generation
	sort.Strings(keys)

	var builder strings.Builder
	builder.WriteString(p.Target())
	for _, k := range keys {
		builder.WriteString(";")
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(tags[k])
	}

	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

func (u *Upstream) refreshProxies(instances []provider.Instancer) error {
	var err error
	if len(instances) == 0 && u.discovery != nil {
		options := provider.GetInstanceOptions{
			Namespace: u.options.Discovery.Namespace,
			Name:      u.options.Discovery.Name,
		}
		instances, err = u.discovery.GetInstances(context.Background(), options)
		if err != nil {
			return err
		}
	} else if len(instances) == 0 {
		return fmt.Errorf("no instances found, upstream id: %s", u.options.ID)
	}
	newProxies := make([]proxy.Proxy, 0)

	for _, instance := range instances {

		targetHost, targetPort, err := net.SplitHostPort(instance.Address().String())
		if err != nil {
			fmt.Println(instance.Address().String())
			targetHost = instance.Address().String()
		}
		addr, err := url.Parse(u.serviceOptions.URL)
		if err != nil {
			return fmt.Errorf("failed to parse service URL '%s': %w", u.serviceOptions.URL, err)
		}
		port := ""
		if len(addr.Port()) > 0 {
			port = addr.Port()
		} else if targetPort != "" && targetPort != "0" {
			port = targetPort
		}
		serverName, _ := instance.Tag("server_name")
		clientOpts := httpproxy.DefaultClientOptions()
		if u.serviceOptions.Timeout.Dail > 0 {
			clientOpts = append(clientOpts, client.WithDialTimeout(u.serviceOptions.Timeout.Dail))
		} else if u.bifrost.options.Default.Service.Timeout.Dail > 0 {
			clientOpts = append(clientOpts, client.WithDialTimeout(u.bifrost.options.Default.Service.Timeout.Dail))
		}
		if u.serviceOptions.Timeout.Read > 0 {
			clientOpts = append(clientOpts, client.WithClientReadTimeout(u.serviceOptions.Timeout.Read))
		} else if u.bifrost.options.Default.Service.Timeout.Read > 0 {
			clientOpts = append(clientOpts, client.WithClientReadTimeout(u.bifrost.options.Default.Service.Timeout.Read))
		}
		if u.serviceOptions.Timeout.Write > 0 {
			clientOpts = append(clientOpts, client.WithWriteTimeout(u.serviceOptions.Timeout.Write))
		} else if u.bifrost.options.Default.Service.Timeout.Write > 0 {
			clientOpts = append(clientOpts, client.WithWriteTimeout(u.bifrost.options.Default.Service.Timeout.Write))
		}
		if u.serviceOptions.Timeout.MaxConnWait > 0 {
			clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(u.serviceOptions.Timeout.MaxConnWait))
		} else if u.bifrost.options.Default.Service.Timeout.MaxConnWait > 0 {
			clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(u.bifrost.options.Default.Service.Timeout.MaxConnWait))
		}
		if u.serviceOptions.MaxConnsPerHost != nil {
			clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*u.serviceOptions.MaxConnsPerHost))
		} else if u.bifrost.options.Default.Service.MaxConnsPerHost != nil {
			clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*u.bifrost.options.Default.Service.MaxConnsPerHost))
		}
		if strings.EqualFold(addr.Scheme, "https") {
			clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{ // nolint
				// when client uses ip address to connect to server, client need to set the ServerName to the domain name you want to use
				ServerName:         serverName,
				InsecureSkipVerify: !u.serviceOptions.TLSVerify, //nolint:gosec
			}))
		}
		if u.bifrost.options.Metrics.Prometheus.Enabled {
			clientOpts = append(clientOpts, client.WithConnStateObserve(func(hcs hzconfig.HostClientState) {
				labels := make(prom.Labels)
				labels["service_id"] = u.serviceOptions.ID
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
		switch u.serviceOptions.Protocol {
		case "", config.ProtocolHTTP, config.ProtocolHTTP2:
			url = fmt.Sprintf("%s://%s%s", addr.Scheme, targetHost, addr.Path)
			if port != "" {
				url = fmt.Sprintf("%s://%s:%s%s", addr.Scheme, targetHost, port, addr.Path)
			}
			clientOptions := httpproxy.ClientOptions{
				IsHTTP2:   u.serviceOptions.Protocol == config.ProtocolHTTP2,
				HZOptions: clientOpts,
			}
			client, err := httpproxy.NewClient(clientOptions)
			if err != nil {
				return err
			}
			proxyOptions := httpproxy.Options{
				Target:           url,
				Protocol:         u.serviceOptions.Protocol,
				Weight:           instance.Weight(),
				MaxFails:         maxFails,
				FailTimeout:      failTimeout,
				IsTracingEnabled: u.bifrost.options.Tracing.Enabled,
				ServiceID:        u.serviceOptions.ID,
				TargetHostHeader: serverName,
				PassHostHeader:   u.serviceOptions.IsPassHostHeader(),
				Tags:             instance.Tags(),
			}
			proxy, err := httpproxy.New(proxyOptions, client)
			if err != nil {
				return err
			}
			newProxies = append(newProxies, proxy)
		case config.ProtocolGRPC:
			url = fmt.Sprintf("grpc://%s%s", targetHost, addr.Path)
			if port != "" {
				url = fmt.Sprintf("grpc://%s:%s%s", targetHost, port, addr.Path)
			}
			grpcOptions := grpcproxy.Options{
				Target:           url,
				TLSVerify:        u.serviceOptions.TLSVerify,
				Weight:           instance.Weight(),
				MaxFails:         maxFails,
				FailTimeout:      failTimeout,
				IsTracingEnabled: u.bifrost.options.Tracing.Enabled,
				Timeout:          u.serviceOptions.Timeout.GRPC,
				Tags:             instance.Tags(),
			}
			grpcProxy, err := grpcproxy.New(grpcOptions)
			if err != nil {
				return err
			}
			newProxies = append(newProxies, grpcProxy)
		}
	}
	var updatedProxies []proxy.Proxy
	// remove old proxy if not exist in new proxies list
	oldProxies := make([]proxy.Proxy, 0)

	if u.Balancer() != nil && u.Balancer().Proxies() != nil {
		proxies := u.Balancer().Proxies()
		if proxies != nil {
			oldProxies = proxies
		}
	}

	newProxyHashes := make(map[string]proxy.Proxy)
	for _, newProxy := range newProxies {
		hash := generateProxyHash(newProxy)
		newProxyHashes[hash] = newProxy
	}

	oldProxyHashes := make(map[string]proxy.Proxy)
	for _, oldProxy := range oldProxies {
		hash := generateProxyHash(oldProxy)
		oldProxyHashes[hash] = oldProxy
	}

	// find the unchanged proxies (exist in both old and new)
	for hash, oldProxy := range oldProxyHashes {
		if _, found := newProxyHashes[hash]; found {
			updatedProxies = append(updatedProxies, oldProxy)
		}
	}

	// find the new proxies (only exist in new)
	for hash, newProxy := range newProxyHashes {
		if _, found := oldProxyHashes[hash]; !found {
			updatedProxies = append(updatedProxies, newProxy)
		}
	}

	if len(updatedProxies) > 0 {
		slog.Debug("upstream refresh success", "upstream_id", u.options.ID, "proxy_id", updatedProxies[0].ID(), "len", len(updatedProxies))
	}

	factory := balancer.Factory(string(u.options.Balancer.Type))
	balancer, err := factory(updatedProxies, u.options.Balancer.Params)
	if err != nil {
		return err
	}

	u.balancer.Store(balancer)
	return nil
}
func (u *Upstream) watch() {
	u.watchOnce.Do(func() {
		options := provider.GetInstanceOptions{
			Name: u.options.Discovery.Name,
		}
		watchCh, err := u.discovery.Watch(context.Background(), options)
		if err != nil {
			slog.Error("failed to watch upstream", "error", err.Error(), "upstream_id", u.options.ID)
			return
		}
		go safety.Go(context.Background(), func() {
			for instances := range watchCh {
				err := u.refreshProxies(instances)
				if err != nil {
					slog.Warn("upstream refresh failed", "error", err.Error(), "upstream_id", u.options.ID)
				}
			}
		})
	})
}
func loadUpstreams(bifrost *Bifrost, serviceOpts config.ServiceOptions) (map[string]*Upstream, error) {
	upstreams := make(map[string]*Upstream)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errCh := make(chan error, len(bifrost.options.Upstreams))

	for id, upstreamOptions := range bifrost.options.Upstreams {
		wg.Add(1)
		upstreamOptions.ID = id
		currentUpstreamOpts := upstreamOptions // Create a new variable for the goroutine
		currentID := id                        // Create a new variable for the goroutine

		go safety.Go(context.Background(), func() {
			defer wg.Done()

			upstream, err := newUpstream(bifrost, serviceOpts, currentUpstreamOpts)
			if err != nil {
				errCh <- err
				return
			}

			mu.Lock()
			upstreams[currentID] = upstream
			mu.Unlock()
		})
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return nil, err // Return the first error encountered
		}
	}

	return upstreams, nil
}
