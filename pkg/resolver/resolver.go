package resolver

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"slices"
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
	Servers   []string
	Hostsfile string
	Order     []string
	Timeout   time.Duration
	SkipTest  bool
}

func NewResolver(option Options) (*Resolver, error) {
	if len(option.Order) == 0 {
		option.Order = []string{"last", "a", "cname"}
	}

	if len(option.Servers) == 0 {
		servers := GetDNSServers()

		if len(servers) == 0 {
			return nil, fmt.Errorf("dns: %w; can't get dns server", ErrNotFound)
		}

		for _, server := range servers {
			option.Servers = append(option.Servers, server.String())
		}
	}

	newServers := make([]string, 0)
	for _, server := range option.Servers {
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
	option.Servers = newServers

	if !option.SkipTest {
		validServers, err := ValidateDNSServer(option.Servers)
		if err != nil {
			return nil, err
		}
		option.Servers = validServers
	}

	client := &dns.Client{
		Timeout: option.Timeout,
	}

	r := &Resolver{
		options:    &option,
		client:     client,
		hostsCache: make(map[string][]string),
		dnsCache:   cache.NewCache[string, []string](5 * time.Minute),
	}

	if err := r.loadHostsFile(); err != nil {
		return nil, fmt.Errorf("dns: failed to load hosts file: %w", err)
	}

	return r, nil
}

func (r *Resolver) Lookup(ctx context.Context, host string, queryOrder ...[]string) ([]string, error) {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "[::1]" {
		return []string{"127.0.0.1"}, nil
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return []string{ip.String()}, nil
	}

	// remove last dot
	host = strings.TrimSuffix(host, ".")

	// First, check the hosts cache
	if len(r.hostsCache) > 0 {
		if ips, ok := r.hostsCache[host]; ok && len(ips) > 0 {
			return ips, nil
		}
	}

	if len(queryOrder) == 0 {
		queryOrder = [][]string{r.options.Order}
	}

	for _, order := range queryOrder[0] {
		order = strings.TrimSpace(order)
		switch strings.ToLower(order) {
		case "last":
			if ips, ok := r.dnsCache.Get(host); ok {
				return ips, nil
			}
		case "a":
			// A record
			var ips []string
			var minTTL uint32 = 0

			m := new(dns.Msg)
			m.SetQuestion(dns.Fqdn(host), dns.TypeA)

			for _, server := range r.options.Servers {
				in, _, err := r.client.ExchangeContext(ctx, m, server)
				if err != nil {
					slog.Debug("dns: failed to resolve A record", "host", host, "server", server, "error", err)
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
				break
			}

			if len(ips) > 0 {
				ips = slices.Compact(ips)
				return ips, nil
			}
		case "cname":
			// CNAME record
			var ips []string
			var minTTL uint32 = 0

			m := new(dns.Msg)
			m.SetQuestion(dns.Fqdn(host), dns.TypeCNAME)

			for _, server := range r.options.Servers {
				in, _, err := r.client.ExchangeContext(ctx, m, server)
				if err != nil {
					slog.Debug("dns: failed to resolve CNAME record", "host", host, "server", server, "error", err)
					continue
				}

				for _, answer := range in.Answer {
					if cname, ok := answer.(*dns.CNAME); ok {
						resolvedIPs, err := r.Lookup(ctx, cname.Target, []string{"a"})
						if err != nil {
							slog.Debug("dns: failed to resolve CNAME record", "host", cname.String(), "server", server, "error", err)
							continue
						}

						ips = append(ips, resolvedIPs...)

						if minTTL == 0 || cname.Hdr.Ttl < minTTL {
							minTTL = cname.Hdr.Ttl
						}
					}
				}

				if len(ips) == 0 {
					continue
				}

				ttlDuration := time.Duration(minTTL) * time.Second
				r.dnsCache.PutWithTTL(host, ips, ttlDuration)
			}

			if len(ips) > 0 {
				ips = slices.Compact(ips)
				return ips, nil
			}
		default:
			return nil, fmt.Errorf("dns: unknown order '%s'", order)
		}

	}

	return nil, fmt.Errorf("dns: %w; can't resolve '%s'", ErrNotFound, host)
}

func (r *Resolver) loadHostsFile() error {
	if len(r.options.Hostsfile) == 0 {
		if _, err := os.Stat("/etc/hosts"); errors.Is(err, os.ErrNotExist) {
			return nil
		}

		r.options.Hostsfile = "/etc/hosts"
	}

	if _, err := os.Stat(r.options.Hostsfile); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("dns: hosts file '%s' not found", r.options.Hostsfile)
	}

	file, err := os.Open(r.options.Hostsfile)
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
