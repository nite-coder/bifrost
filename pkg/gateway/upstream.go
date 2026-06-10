package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/provider/dns"
	"github.com/nite-coder/bifrost/pkg/provider/k8s"
	"github.com/nite-coder/bifrost/pkg/provider/nacos"
	"github.com/nite-coder/bifrost/pkg/proxy"
)

const defaultSubscriberBufferSize = 64

// Upstream represents a collection of backend targets.
type Upstream struct {
	mu          sync.RWMutex
	discovery   provider.ServiceDiscovery
	bifrost     *Bifrost
	options     *config.UpstreamOptions
	subscribers []chan []*proxy.Endpoint
	endpoints   []*proxy.Endpoint
	watchOnce   sync.Once
	cancel      context.CancelFunc
}

// Close stops watching for updates and closes discovery resources.
func (u *Upstream) Close() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.cancel != nil {
		u.cancel()
	}

	for _, sub := range u.subscribers {
		close(sub)
	}
	u.subscribers = nil

	if u.discovery != nil {
		return u.discovery.Close()
	}
	return nil
}

func newUpstream(
	bifrost *Bifrost,
	upstreamOptions config.UpstreamOptions,
) (*Upstream, error) {
	var err error
	if len(upstreamOptions.ID) == 0 {
		return nil, errors.New("upstream ID cannot be empty")
	}
	if upstreamOptions.Discovery.Type == "" && len(upstreamOptions.Targets) == 0 {
		return nil, fmt.Errorf("targets cannot be empty for upstream ID: %s", upstreamOptions.ID)
	}

	upstream := &Upstream{
		bifrost: bifrost,
		options: &upstreamOptions,
	}

	if strings.HasPrefix(upstreamOptions.ID, "ai:") {
		discovery := NewStaticDiscovery(upstream)
		upstream.discovery = discovery
	} else {
		switch strings.ToLower(upstreamOptions.Discovery.Type) {
		case "dns":
			if !bifrost.options.Providers.DNS.Enabled {
				return nil, fmt.Errorf("dns provider is disabled for upstream ID: %s", upstreamOptions.ID)
			}
			var discovery provider.ServiceDiscovery
			discovery, err = dns.NewDNSServiceDiscovery(
				bifrost.options.Providers.DNS.Servers,
				bifrost.options.Providers.DNS.Valid,
			)
			if err != nil {
				return nil, err
			}
			upstream.discovery = discovery
		case "nacos":
			if !bifrost.options.Providers.Nacos.Discovery.Enabled {
				return nil, fmt.Errorf("nacos discovery provider is disabled for upstream ID: %s", upstreamOptions.ID)
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
			var discovery provider.ServiceDiscovery
			discovery, err = nacos.NewNacosServiceDiscovery(options)
			if err != nil {
				return nil, err
			}
			upstream.discovery = discovery
		case "k8s":
			if !bifrost.options.Providers.K8S.Enabled {
				return nil, fmt.Errorf("k8s provider is disabled for upstream ID: %s", upstreamOptions.ID)
			}
			option := k8s.Options{
				APIServer: bifrost.options.Providers.K8S.APIServer,
			}
			var discovery provider.ServiceDiscovery
			discovery, err = k8s.NewK8sDiscovery(option)
			if err != nil {
				return nil, err
			}
			upstream.discovery = discovery
		default:
			discovery := NewResolverDiscovery(upstream)
			upstream.discovery = discovery
		}
	}

	err = upstream.refreshEndpoints(nil)
	if err != nil {
		return nil, err
	}
	return upstream, nil
}

// Subscribe returns a channel that receives the list of endpoints.
func (u *Upstream) Subscribe() <-chan []*proxy.Endpoint {
	u.watch()

	u.mu.Lock()
	defer u.mu.Unlock()

	ch := make(chan []*proxy.Endpoint, defaultSubscriberBufferSize)
	u.subscribers = append(u.subscribers, ch)
	if len(u.endpoints) > 0 {
		ch <- u.endpoints
	}
	return ch
}

// Unsubscribe removes a subscription channel.
func (u *Upstream) Unsubscribe(ch <-chan []*proxy.Endpoint) {
	u.mu.Lock()
	defer u.mu.Unlock()

	for i, sub := range u.subscribers {
		if sub == ch {
			u.subscribers = append(u.subscribers[:i], u.subscribers[i+1:]...)
			close(sub)
			break
		}
	}
}

func (u *Upstream) refreshEndpoints(instances []provider.Instancer) error {
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
		return fmt.Errorf("no instances found for upstream ID: %s", u.options.ID)
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

	newEndpoints := make([]*proxy.Endpoint, 0, len(instances))
	for _, instance := range instances {
		var address string
		var serverName string
		if instance.Address().Network() == "static" {
			address = instance.Address().String()
			serverName = address
		} else {
			var targetHost, targetPort string
			targetHost, targetPort, err = net.SplitHostPort(instance.Address().String())
			if err != nil {
				targetHost = instance.Address().String()
				targetPort = "0"
			}
			address = net.JoinHostPort(targetHost, targetPort)
			serverName = targetHost
		}
		var state *proxy.TargetState
		if u.bifrost.upstreamManager != nil {
			state = u.bifrost.upstreamManager.GetOrCreateTargetState(address, maxFails, failTimeout)
		} else {
			state = proxy.NewTargetState(maxFails, failTimeout)
		}

		tags := instance.Tags()
		if tags == nil {
			tags = make(map[string]string)
		}
		tags["server_name"] = serverName

		ep := &proxy.Endpoint{
			Address:     address,
			Weight:      instance.Weight(),
			Tags:        tags,
			HealthState: state,
		}
		newEndpoints = append(newEndpoints, ep)
	}

	u.mu.Lock()
	u.endpoints = newEndpoints
	subs := make([]chan []*proxy.Endpoint, len(u.subscribers))
	copy(subs, u.subscribers)
	u.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- newEndpoints:
		default:
			slog.Warn("upstream subscriber channel full, dropping update", "upstream_id", u.options.ID)
		}
	}

	return nil
}

func (u *Upstream) watch() {
	u.watchOnce.Do(func() {
		if u.discovery == nil {
			return
		}
		options := provider.GetInstanceOptions{
			Name: u.options.Discovery.Name,
		}
		ctx, cancel := context.WithCancel(context.Background())
		u.cancel = cancel
		var err error
		var watchCh <-chan []provider.Instancer
		watchCh, err = u.discovery.Watch(ctx, options)
		if err != nil {
			if errors.Is(err, provider.ErrWatchNotSupported) {
				return
			}
			slog.Error("failed to watch upstream", "error", err.Error(), "upstream_id", u.options.ID)
			return
		}

		if watchCh == nil {
			return
		}

		go safety.Go(ctx, func() {
			for instances := range watchCh {
				err := u.refreshEndpoints(instances)
				if err != nil {
					slog.Warn("upstream refresh failed", "error", err.Error(), "upstream_id", u.options.ID)
				}
			}
		})
	})
}
