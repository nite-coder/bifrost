package dns

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/miekg/dns"
)

var (
	ErrNotFound = errors.New("no records found")
)

type Resolver struct {
	options *Options
	client  *dns.Client
}

type Options struct {
	// dns server for querying
	AddrPort string
	Valid    time.Duration
}

func NewResolver(option Options) (*Resolver, error) {
	r := new(Resolver)
	r.options = &option
	r.client = new(dns.Client)

	if option.AddrPort == "" {
		servers := GetDNSServers()

		if len(servers) == 0 {
			return nil, fmt.Errorf("%w; can't get dns server", ErrNotFound)
		}

		option.AddrPort = servers[0].String()
	}

	return r, nil
}

func (r *Resolver) Lookup(ctx context.Context, host string) ([]string, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(host), dns.TypeA)

	in, _, err := r.client.ExchangeContext(ctx, m, r.options.AddrPort)
	if err != nil {
		return nil, fmt.Errorf("dns: query failed: %w", err)
	}

	var ips []string
	for _, answer := range in.Answer {
		if a, ok := answer.(*dns.A); ok {
			ips = append(ips, a.A.String())
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("%w; can't get ip for %s", ErrNotFound, host)
	}

	return ips, nil
}
