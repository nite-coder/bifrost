package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	config1 "http-benchmark/pkg/config"
	"math"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/rs/dnscache"
)

type Upstream struct {
	opts    config1.UpstreamOptions
	proxies []*ReverseProxy
	index   atomic.Uint64
}

var defaultClientOptions = []config.ClientOption{
	client.WithNoDefaultUserAgentHeader(true),
	client.WithDisableHeaderNamesNormalizing(true),
	client.WithDisablePathNormalizing(true),
	client.WithMaxConnsPerHost(math.MaxInt),
	client.WithDialTimeout(10 * time.Second),
	client.WithClientReadTimeout(10 * time.Second),
	client.WithWriteTimeout(10 * time.Second),
	client.WithKeepAlive(true),
}

func NewUpstream(bifrost *Bifrost, serviceOpts config1.ServiceOptions, opts config1.UpstreamOptions) (*Upstream, error) {

	if len(opts.ID) == 0 {
		return nil, fmt.Errorf("upstream id can't be empty")
	}

	if len(opts.Targets) == 0 {
		return nil, fmt.Errorf("targets can't be empty. upstream id: %s", opts.ID)
	}

	// direct proxy
	clientOpts := defaultClientOptions

	if opts.DailTimeout != nil {
		clientOpts = append(clientOpts, client.WithDialTimeout(*opts.DailTimeout))
	}

	if opts.ReadTimeout != nil {
		clientOpts = append(clientOpts, client.WithClientReadTimeout(*opts.ReadTimeout))
	}

	if opts.WriteTimeout != nil {
		clientOpts = append(clientOpts, client.WithWriteTimeout(*opts.WriteTimeout))
	}

	if opts.MaxConnWaitTimeout != nil {
		clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(*opts.MaxConnWaitTimeout))
	}

	if opts.MaxIdleConnsPerHost != nil {
		clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*opts.MaxIdleConnsPerHost))
	}

	upstream := &Upstream{
		opts:    opts,
		proxies: make([]*ReverseProxy, 0),
	}

	for _, targetOpts := range opts.Targets {

		targetHost, targetPort, err := net.SplitHostPort(targetOpts.Target)
		if err != nil {
			targetHost = targetOpts.Target
		}

		var dnsResolver dnscache.DNSResolver
		if allowDNS(targetHost) {
			_, err := bifrost.resolver.LookupHost(context.Background(), targetHost)
			if err != nil {
				return nil, fmt.Errorf("lookup upstream host error: %v", err)
			}
			dnsResolver = bifrost.resolver
		}

		switch serviceOpts.Protocol {
		case config1.ProtocolHTTP:
			if dnsResolver != nil {
				clientOpts = append(clientOpts, client.WithDialer(newHTTPDialer(dnsResolver)))
			}
		case config1.ProtocolHTTPS:
			clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
				InsecureSkipVerify: serviceOpts.TLSVerify,
			}))
		}

		port := targetPort
		if serviceOpts.Port > 0 {
			port = strconv.FormatInt(int64(serviceOpts.Port), 10)
		}

		url := fmt.Sprintf("%s://%s:%s%s", serviceOpts.Protocol, targetHost, port, serviceOpts.Path)
		proxy, err := NewSingleHostReverseProxy(url, bifrost.opts.Observability.Tracing.Enabled, clientOpts...)

		if err != nil {
			return nil, err
		}
		upstream.proxies = append(upstream.proxies, proxy)
	}

	return upstream, nil
}

func (u *Upstream) pickupByRoundRobin() *ReverseProxy {
	if len(u.proxies) == 1 {
		return u.proxies[0]
	}

	index := u.index.Add(1)
	proxy := u.proxies[(int(index)-1)%len(u.proxies)]
	return proxy
}

func allowDNS(address string) bool {

	ip := net.ParseIP(address)
	if ip != nil {
		return false
	}

	if address == "localhost" || address == "[::1]" {
		return false
	}

	return true
}
