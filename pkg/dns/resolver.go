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
	resultCache *cache.Cache[string, []string]
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
		resultCache: resultCache,
	}

	r.options = &option
	r.client = new(dns.Client)

	if option.AddrPort == "" {
		servers := GetDNSServers()

		if len(servers) == 0 {
			return nil, fmt.Errorf("%w; can't get dns server", ErrNotFound)
		}

		option.AddrPort = servers[0].String()
	}

	if err := r.loadHostsFile(); err != nil {
		return nil, fmt.Errorf("failed to load hosts file: %w", err)
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
	if ips, ok := r.resultCache.Get(host); ok {
		return ips, nil
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeA)

	in, _, err := r.client.ExchangeContext(ctx, m, r.options.AddrPort)
	if err != nil {
		return nil, fmt.Errorf("fail to query '%s' to dns server '%s', error: %w", host, r.options.AddrPort, err)
	}

	var ips []string
	for _, answer := range in.Answer {
		if a, ok := answer.(*dns.A); ok {
			ips = append(ips, a.A.String())
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("%w; can't resolve host '%s'", ErrNotFound, host)
	}

	r.resultCache.PutWithTTL(host, ips, r.options.Valid)
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