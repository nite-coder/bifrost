package dns

import (
	"bufio"
	"context"
	"errors"
	"fmt"
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
	options     *Options
	client      *dns.Client
	hostsCache  map[string][]string
	dnsCache *cache.Cache[string, []string]
}

type Options struct {
	// dns server for querying
	AddrPort string
	Valid    time.Duration
}

func NewResolver(option Options) (*Resolver, error) {
	resultCache := cache.NewCache[string, []string](cache.NoExpiration)

	r := &Resolver{
		hostsCache:  make(map[string][]string),
		dnsCache: resultCache,
	}

	r.options = &option
	r.client = new(dns.Client)

	if option.AddrPort == "" {
		servers := GetDNSServers()

		if len(servers) == 0 {
			return nil, fmt.Errorf("dns: %w; can't get dns server", ErrNotFound)
		}

		option.AddrPort = servers[0].String()
	}

	err := ValidateDNSServer(option.AddrPort)
	if err != nil {
		return nil, err
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

	in, _, err := r.client.ExchangeContext(ctx, m, r.options.AddrPort)
	if err != nil {
		return nil, fmt.Errorf("dns: fail to query '%s' to dns server '%s', error: %w", host, r.options.AddrPort, err)
	}

	var ips []string
	for _, answer := range in.Answer {
		if a, ok := answer.(*dns.A); ok {
			ips = append(ips, a.A.String())
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("dns: %w; can't resolve host '%s'", ErrNotFound, host)
	}

	r.dnsCache.PutWithTTL(host, ips, r.options.Valid)
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

// ValidateDNSServer checks the responsiveness and validity of a DNS server.
// It sends a query to the given DNS server address and verifies if the server
// responds and returns a successful status code. If the server does not respond
// or returns an error code, an error is returned detailing the issue.

func ValidateDNSServer(addr string) error {

	m := new(dns.Msg)
	m.SetQuestion(".", dns.TypeNS)
	m.RecursionDesired = true

	c := new(dns.Client)
	c.Timeout = 5 * time.Second

	resp, _, err := c.Exchange(m, addr)
	if err != nil {
		return fmt.Errorf("DNS server is not responding: %w", err)
	}

	if resp == nil {
		return errors.New("no response from DNS server")
	}

	if resp.Rcode != dns.RcodeSuccess && resp.Rcode != dns.RcodeNameError {
		return fmt.Errorf("DNS server returned error code: %v", dns.RcodeToString[resp.Rcode])
	}

	return nil
}
