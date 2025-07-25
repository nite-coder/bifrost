package gateway

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/nite-coder/bifrost/pkg/provider"
)

type ResolverDiscovery struct {
	upstream *Upstream
}

func NewResolverDiscovery(upstream *Upstream) *ResolverDiscovery {
	return &ResolverDiscovery{
		upstream: upstream,
	}
}

func (d *ResolverDiscovery) GetInstances(ctx context.Context, options provider.GetInstanceOptions) ([]provider.Instancer, error) {

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

func (d *ResolverDiscovery) Watch(ctx context.Context, options provider.GetInstanceOptions) (<-chan []provider.Instancer, error) {
	return nil, nil
}
