package dns

import (
	"context"
	"time"

	"github.com/miekg/dns"
	"github.com/nite-coder/bifrost/pkg/provider"
)

type DNSServiceDiscovery struct {
	client  *dns.Client
	servers []string
}

func NewDNSServiceDiscovery(servers []string, valid time.Duration) *DNSServiceDiscovery {
	return &DNSServiceDiscovery{
		client:  new(dns.Client),
		servers: servers,
	}
}

func (d *DNSServiceDiscovery) GetInstances(ctx context.Context, serviceName string) ([]provider.Instancer, error) {

	return nil, nil
}

func (d *DNSServiceDiscovery) Watch(ctx context.Context, serviceName string) (<-chan []provider.Instancer, error) {
	return nil, nil
}
