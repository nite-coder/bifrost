package dns

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/nite-coder/bifrost/pkg/provider"
)

var (
	ErrNotFound = errors.New("no records found")
)

type DNSServiceDiscovery struct {
	client  *dns.Client
	servers []string
	valid   time.Duration
	ticker  *time.Ticker
}

func NewDNSServiceDiscovery(servers []string, valid time.Duration) (*DNSServiceDiscovery, error) {
	newServers := make([]string, 0)

	for _, server := range servers {
		server = strings.TrimSpace(server)

		if len(server) == 0 {
			continue
		}

		targetHost, targetPort, err := net.SplitHostPort(server)
		if err != nil {
			targetHost = server
		}

		if len(targetPort) == 0 {
			targetPort = "53"
		}

		targetHost = net.JoinHostPort(targetHost, targetPort)
		newServers = append(newServers, targetHost)
	}

	if len(newServers) == 0 {
		return nil, errors.New("no valid dns server found for dns provider")
	}

	if valid.Seconds() <= 0 {
		valid = 30 * time.Second
	}

	client := new(dns.Client)

	d := &DNSServiceDiscovery{
		client:  client,
		servers: newServers,
		valid:   valid,
		ticker:  time.NewTicker(valid),
	}

	return d, nil
}

func (d *DNSServiceDiscovery) GetInstances(ctx context.Context, serviceName string) ([]provider.Instancer, error) {
	instances := make([]provider.Instancer, 0)

	targetHost, targetPort, err := net.SplitHostPort(serviceName)
	if err != nil {
		targetHost = serviceName
	}

	ips, err := d.Lookup(ctx, targetHost)
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

		instance := provider.NewInstance(addr, 1)
		instance.SetTag("server_name", targetHost)

		instances = append(instances, instance)
	}

	return instances, nil
}

func (d *DNSServiceDiscovery) Watch(ctx context.Context, serviceName string) (<-chan []provider.Instancer, error) {
	ch := make(chan []provider.Instancer, 1)

	go func() {
		if d.ticker != nil {
			for range d.ticker.C {
				ch <- nil
			}
		}
	}()

	return ch, nil
}

func (d *DNSServiceDiscovery) Lookup(ctx context.Context, host string) ([]string, error) {
	if host == "localhost" || host == "[::1]" {
		return []string{"127.0.0.1"}, nil
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return []string{ip.String()}, nil
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeA)

	var ips []string
	var minTTL uint32 = 0

	for _, server := range d.servers {
		in, _, err := d.client.ExchangeContext(ctx, m, server)
		if err != nil {
			slog.Debug("dns: failed to resolve host", "host", host, "server", server, "error", err.Error())
			continue
		}

		for _, answer := range in.Answer {
			if a, ok := answer.(*dns.A); ok {
				ips = append(ips, a.A.String())

				if minTTL == 0 || a.Hdr.Ttl < minTTL {
					minTTL = a.Hdr.Ttl
				}
			}
		}

		if len(ips) == 0 {
			continue
		}

		ttlDuration := time.Duration(minTTL) * time.Second
		if ttlDuration.Seconds() < d.valid.Seconds() {
			ttlDuration = d.valid
		}

		d.ticker.Reset(ttlDuration)
		slog.Debug("ttl updated", "host", host, "ips", ips, "ttl", ttlDuration)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("dns: %w; can't resolve host '%s'", ErrNotFound, host)
	}

	ips = slices.Compact(ips)

	return ips, nil
}
