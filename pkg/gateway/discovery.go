package gateway

import (
	"context"
	"fmt"
	"net"

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

func (d *ResolverDiscovery) GetInstances(ctx context.Context, serviceName string) ([]provider.Instancer, error) {

	instances := make([]provider.Instancer, 0)

	for _, targetOptions := range d.upstream.options.Targets {

		targetHost, targetPort, err := net.SplitHostPort(targetOptions.Target)
		if err != nil {
			targetHost = targetOptions.Target
		}

		ips, err := d.upstream.bifrost.resolver.Lookup(context.Background(), targetHost)
		if err != nil {
			return nil, fmt.Errorf("fail to lookup target '%s', error: %w", targetHost, err)
		}

		for _, ip := range ips {
			if len(targetPort) > 0 {
				ip = net.JoinHostPort(ip, targetPort)
			} else {
				ip = net.JoinHostPort(ip, "0")
			}

			addr, err := net.ResolveTCPAddr("tcp", ip)
			if err != nil {
				return nil, fmt.Errorf("fail to resolve target '%s', error: %w", ip, err)
			}

			instance := provider.NewInstance(addr, targetOptions.Weight)
			instance.SetTag("server_name", targetHost)

			instances = append(instances, instance)
		}
	}

	return instances, nil
}

func (d *ResolverDiscovery) Watch(ctx context.Context, serviceName string) (<-chan []provider.Instancer, error) {
	return nil, nil
}
