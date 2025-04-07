package dns

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/nite-coder/blackbear/pkg/cache/v2"
)

var (
	ErrNotFound = errors.New("no records found")
)

type Resolver struct {
	options    *Options
	client     *dns.Client
	hostsCache map[string][]string
	dnsCache   *cache.Cache[string, []string]
}

type Options struct {
	// dns server for querying
	Servers  []string
	SkipTest bool
}

func NewResolver(option Options) (*Resolver, error) {
	resultCache := cache.NewCache[string, []string](cache.NoExpiration)

	r := &Resolver{
		hostsCache: make(map[string][]string),
		dnsCache:   resultCache,
	}

	r.options = &option
	r.client = new(dns.Client)

	if len(option.Servers) == 0 {
		servers := GetDNSServers()

		if len(servers) == 0 {
			return nil, fmt.Errorf("dns: %w; can't get dns server", ErrNotFound)
		}

		for _, server := range servers {
			option.Servers = append(option.Servers, server.String())
		}
	}

	if !option.SkipTest {
		validServers, err := ValidateDNSServer(option.Servers)
		if err != nil {
			return nil, err
		}
		option.Servers = validServers
	}

	if err := r.loadHostsFile(); err != nil {
		return nil, fmt.Errorf("dns: failed to load hosts file: %w", err)
	}

	return r, nil
}

func (r *Resolver) Lookup(ctx context.Context, host string) ([]string, error) {
	if host == "localhost" || host == "[::1]" {
		return []string{"127.0.0.1"}, nil
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return []string{ip.String()}, nil
	}

	// First, check the hosts cache
	if len(r.hostsCache) > 0 {
		if ips, ok := r.hostsCache[host]; ok && len(ips) > 0 {
			return ips, nil
		}
	}

	// Second, check the result cache
	if ips, ok := r.dnsCache.Get(host); ok {
		return ips, nil
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeA)

	var ips []string
	var minTTL uint32 = 0

	for _, server := range r.options.Servers {
		in, _, err := r.client.ExchangeContext(ctx, m, server)
		if err != nil {
			slog.Debug("dns: failed to resolve host", "host", host, "server", server, "error", err)
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
		r.dnsCache.PutWithTTL(host, ips, ttlDuration)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("dns: %w; can't resolve host '%s'", ErrNotFound, host)
	}

	return ips, nil
}

func (r *Resolver) loadHostsFile() error {
	if _, err := os.Stat("/etc/hosts"); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	file, err := os.Open("/etc/hosts")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		ip := net.ParseIP(fields[0])
		if ip == nil {
			continue
		}
		for _, host := range fields[1:] {
			r.hostsCache[host] = append(r.hostsCache[host], ip.String())
		}
	}

	return scanner.Err()
}

// ValidateDNSServer validates a list of DNS servers by sending a query to each of them
// and checking if they respond with a valid answer. It returns a list of valid servers
// and an error if no valid server is found.

func ValidateDNSServer(servers []string) ([]string, error) {
	m := new(dns.Msg)
	m.SetQuestion(".", dns.TypeNS)
	m.RecursionDesired = true

	c := new(dns.Client)
	c.Timeout = 5 * time.Second

	result := make([]string, 0)
	for _, server := range servers {
		resp, _, err := c.Exchange(m, server)
		if err != nil {
			slog.Debug("DNS server is not responding", "server", server, "error", err)
			continue
		}

		if resp == nil {
			slog.Debug("no response from DNS server", "server", server)
			continue
		}

		if resp.Rcode != dns.RcodeSuccess && resp.Rcode != dns.RcodeNameError {
			slog.Debug("DNS server returned error code", "server", server, "code", dns.RcodeToString[resp.Rcode])
			continue
		}

		result = append(result, server)
	}

	if len(result) == 0 {
		return nil, errors.New("no valid DNS server found")
	}

	return result, nil
}
