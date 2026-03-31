package gateway

import (
	"context"
	"errors"
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

// Watch is not supported by ResolverDiscovery and returns an error.
func (d *ResolverDiscovery) Watch(
	_ context.Context,
	_ provider.GetInstanceOptions,
) (<-chan []provider.Instancer, error) {
	return nil, errors.New("watch is not supported by resolver discovery")
}

// Close releases resources used by ResolverDiscovery.
func (d *ResolverDiscovery) Close() error {
	return nil
}
