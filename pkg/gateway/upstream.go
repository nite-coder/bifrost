package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/provider/dns"
	"github.com/nite-coder/bifrost/pkg/provider/k8s"
	"github.com/nite-coder/bifrost/pkg/provider/nacos"
	"github.com/nite-coder/bifrost/pkg/target"
)

const defaultSubscriberBufferSize = 64

// Upstream manages a set of backend targets and provides a balancer for load balancing.
type Upstream struct {
	mu            sync.RWMutex
	discovery     provider.ServiceDiscovery
	bifrost       *Bifrost
	options       *config.UpstreamOptions
	subscribers   []chan []*target.Endpoint
	targets       map[string]*target.Target
	endpointsHash string
	balancer      atomic.Value
	watchOnce     sync.Once
	cancel        context.CancelFunc
	isExclusive   atomic.Bool
}

// Close shuts down the upstream, cancels watches, and closes subscribers.
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

// Balancer returns the current load balancer for this upstream.
func (u *Upstream) Balancer() balancer.Balancer {
	b := u.balancer.Load()
	if b == nil {
		return nil
	}
	if bal, ok := b.(balancer.Balancer); ok {
		return bal
	}
	return nil
}

func newUpstream(bifrost *Bifrost, upstreamOptions config.UpstreamOptions) (*Upstream, error) {
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
		targets: make(map[string]*target.Target),
	}

	for _, tgtOpt := range upstreamOptions.Targets {
		upstream.targets[tgtOpt.Target] = &target.Target{
			Name:      tgtOpt.Target,
			Weight:    tgtOpt.Weight,
			Tags:      tgtOpt.Tags,
			Endpoints: make(map[string]*target.Endpoint),
		}
	}

	if strings.HasPrefix(upstreamOptions.ID, "ai:") {
		upstream.discovery = NewStaticDiscovery(upstream)
	} else {
		switch strings.ToLower(upstreamOptions.Discovery.Type) {
		case "dns":
			if !bifrost.options.Providers.DNS.Enabled {
				return nil, fmt.Errorf("dns provider is disabled for upstream ID: %s", upstreamOptions.ID)
			}
			discovery, dErr := dns.NewDNSServiceDiscovery(
				bifrost.options.Providers.DNS.Servers,
				bifrost.options.Providers.DNS.Valid,
			)
			if dErr != nil {
				return nil, dErr
			}
			upstream.discovery = discovery
		case "nacos":
			if !bifrost.options.Providers.Nacos.Discovery.Enabled {
				return nil, fmt.Errorf("nacos discovery provider is disabled for upstream ID: %s", upstreamOptions.ID)
			}
			opts := nacos.Options{
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
			discovery, err = nacos.NewNacosServiceDiscovery(opts)
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
			upstream.discovery = NewResolverDiscovery(upstream)
		}
	}

	if err = upstream.refreshEndpoints(nil); err != nil {
		return nil, err
	}
	return upstream, nil
}

// Endpoints returns a flat slice of all endpoints across all targets.
func (u *Upstream) Endpoints() []*target.Endpoint {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.flattenEndpoints()
}

// Subscribe returns a channel that receives the current endpoints immediately,
// followed by ongoing updates.
func (u *Upstream) Subscribe() <-chan []*target.Endpoint {
	u.watch()
	u.mu.Lock()
	defer u.mu.Unlock()
	ch := make(chan []*target.Endpoint, defaultSubscriberBufferSize)
	u.subscribers = append(u.subscribers, ch)
	return ch
}

// Unsubscribe removes a subscriber channel and closes it.
func (u *Upstream) Unsubscribe(ch <-chan []*target.Endpoint) {
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

func (u *Upstream) flattenEndpoints() []*target.Endpoint {
	var total int
	for _, t := range u.targets {
		total += len(t.Endpoints)
	}
	slice := make([]*target.Endpoint, 0, total)
	for _, t := range u.targets {
		for _, ep := range t.Endpoints {
			slice = append(slice, ep)
		}
	}
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Address < slice[j].Address
	})
	return slice
}

func (u *Upstream) refreshEndpoints(results []provider.DiscoveryResult) error {
	var err error
	if len(results) == 0 && u.discovery != nil {
		opts := provider.GetInstanceOptions{
			Namespace: u.options.Discovery.Namespace,
			Name:      u.options.Discovery.Name,
		}
		results, err = u.discovery.GetInstances(context.Background(), opts)
		if err != nil {
			return err
		}
	} else if len(results) == 0 {
		return fmt.Errorf("no instances found for upstream ID: %s", u.options.ID)
	}

	maxFails := u.bifrost.options.Default.Upstream.MaxFails
	if u.options.HealthCheck.Passive.MaxFails != nil {
		maxFails = *u.options.HealthCheck.Passive.MaxFails
	}
	failTimeout := u.options.HealthCheck.Passive.FailTimeout
	if failTimeout <= 0 {
		failTimeout = u.bifrost.options.Default.Upstream.FailTimeout
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	seen := make(map[string]bool, len(results))
	for _, r := range results {
		tgt := u.getOrCreateTarget(r.Target)
		tgt.Weight = r.Weight
		tgt.Tags = r.Tags
		seen[r.Target] = true

		newMap := make(map[string]*target.Endpoint, len(r.Nodes))
		for _, inst := range r.Nodes {
			address := inst.Address().String()
			serverName := address
			if inst.Address().Network() != "static" {
				if h, _, e := net.SplitHostPort(address); e == nil {
					serverName = h
				}
			}
			if existing, found := tgt.Endpoints[address]; found {
				tags := make(map[string]string)
				maps.Copy(tags, inst.Tags())
				if _, ok := tags["server_name"]; !ok {
					tags["server_name"] = serverName
				}
				ep := &target.Endpoint{
					Address: existing.Address,
					Weight:  inst.Weight(),
					Tags:    tags,
					State:   existing.State,
				}
				newMap[address] = ep
			} else {
				state := target.NewState(maxFails, failTimeout)
				tags := make(map[string]string)
				maps.Copy(tags, inst.Tags())
				if _, ok := tags["server_name"]; !ok {
					tags["server_name"] = serverName
				}
				ep := &target.Endpoint{
					Address: address,
					Weight:  inst.Weight(),
					Tags:    tags,
					State:   state,
				}
				newMap[address] = ep
			}
		}
		tgt.Endpoints = newMap
	}

	for name, tgt := range u.targets {
		if !seen[name] {
			tgt.Endpoints = make(map[string]*target.Endpoint)
		}
	}

	flat := u.flattenEndpoints()
	newHash := target.EndpointHash(flat)
	if newHash == u.endpointsHash {
		return nil
	}
	u.endpointsHash = newHash

	u.rebuildBalancer(flat)

	for _, ch := range u.subscribers {
		select {
		case ch <- flat:
		default:
			slog.Warn("upstream subscriber channel full, dropping update", "upstream_id", u.options.ID)
		}
	}
	return nil
}

func (u *Upstream) rebuildBalancer(endpoints []*target.Endpoint) {
	factory := balancer.Factory(u.options.Balancer.Type)
	if factory == nil {
		slog.Error("unsupported balancer type", "type", u.options.Balancer.Type)
		return
	}
	b, err := factory(endpoints, u.options.Balancer.Params)
	if err != nil {
		slog.Error("failed to create balancer", "upstream_id", u.options.ID, "error", err)
		return
	}
	u.balancer.Store(b)
}

func (u *Upstream) getOrCreateTarget(name string) *target.Target {
	if t, ok := u.targets[name]; ok {
		return t
	}
	t := &target.Target{
		Name:      name,
		Endpoints: make(map[string]*target.Endpoint),
	}
	u.targets[name] = t
	return t
}

func (u *Upstream) watch() {
	u.watchOnce.Do(func() {
		if u.discovery == nil {
			return
		}
		opts := provider.GetInstanceOptions{
			Name: u.options.Discovery.Name,
		}
		ctx, cancel := context.WithCancel(context.Background())
		u.cancel = cancel
		watchCh, err := u.discovery.Watch(ctx, opts)
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
			for results := range watchCh {
				if rErr := u.refreshEndpoints(results); rErr != nil {
					slog.Warn("failed to resfresh upstream", "error", rErr.Error(), "upstream_id", u.options.ID)
				}
			}
		})
	})
}
