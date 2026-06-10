package gateway

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/nite-coder/bifrost/pkg/provider"
)

// ResolverDiscovery implements service discovery using the internal resolver.
type ResolverDiscovery struct {
	upstream *Upstream
}

// NewResolverDiscovery creates a new ResolverDiscovery instance.
func NewResolverDiscovery(upstream *Upstream) *ResolverDiscovery {
	return &ResolverDiscovery{
		upstream: upstream,
	}
}

// GetInstances resolves upstream targets using DNS and returns a list of instances.
func (d *ResolverDiscovery) GetInstances(
	_ context.Context,
	_ provider.GetInstanceOptions,
) ([]provider.Instancer, error) {
	instances := make([]provider.Instancer, 0)

	for _, targetOptions := range d.upstream.options.Targets {
		targetHost, targetPort, err := net.SplitHostPort(targetOptions.Target)
		if err != nil {
			targetHost = targetOptions.Target
		}

		ips, err := d.upstream.bifrost.resolver.Lookup(context.Background(), targetHost)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup target '%s', error: %w", targetHost, err)
		}

		for _, ip := range ips {
			if len(targetPort) > 0 {
				ip = net.JoinHostPort(ip, targetPort)
			} else {
				ip = net.JoinHostPort(ip, "0")
			}

			addr, err := net.ResolveTCPAddr("tcp", ip)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve target '%s', error: %w", ip, err)
			}

			instance := provider.NewInstance(addr, targetOptions.Weight)

			if len(targetOptions.Tags) > 0 {
				for key, val := range targetOptions.Tags {
					key = strings.TrimSpace(key)
					val = strings.TrimSpace(val)
					instance.SetTag(key, val)
				}
			}

			instance.SetTag("server_name", targetHost)

			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// Watch is not supported by ResolverDiscovery and returns provider.ErrWatchNotSupported.
func (d *ResolverDiscovery) Watch(
	_ context.Context,
	_ provider.GetInstanceOptions,
) (<-chan []provider.Instancer, error) {
	return nil, provider.ErrWatchNotSupported
}

// Close releases resources used by ResolverDiscovery.
func (d *ResolverDiscovery) Close() error {
	return nil
}

// StaticDiscovery implements service discovery using static targets (e.g. for virtual AI models).
type StaticDiscovery struct {
	upstream *Upstream
}

// NewStaticDiscovery creates a new StaticDiscovery instance.
func NewStaticDiscovery(upstream *Upstream) *StaticDiscovery {
	return &StaticDiscovery{
		upstream: upstream,
	}
}

type dummyAddr struct {
	addr string
}

func (a dummyAddr) Network() string { return "static" }
func (a dummyAddr) String() string  { return a.addr }

// GetInstances returns static targets as instances.
func (d *StaticDiscovery) GetInstances(
	_ context.Context,
	_ provider.GetInstanceOptions,
) ([]provider.Instancer, error) {
	instances := make([]provider.Instancer, 0, len(d.upstream.options.Targets))

	for _, targetOptions := range d.upstream.options.Targets {
		addr := dummyAddr{addr: targetOptions.Target}
		instance := provider.NewInstance(addr, targetOptions.Weight)

		if len(targetOptions.Tags) > 0 {
			for key, val := range targetOptions.Tags {
				instance.SetTag(key, val)
			}
		}
		instance.SetTag("server_name", targetOptions.Target)
		instances = append(instances, instance)
	}

	return instances, nil
}

// Watch is not supported by StaticDiscovery and returns provider.ErrWatchNotSupported.
func (d *StaticDiscovery) Watch(
	_ context.Context,
	_ provider.GetInstanceOptions,
) (<-chan []provider.Instancer, error) {
	return nil, provider.ErrWatchNotSupported
}

// Close releases resources used by StaticDiscovery.
func (d *StaticDiscovery) Close() error {
	return nil
}
