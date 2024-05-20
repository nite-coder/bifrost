package gateway

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/common/config"
)

type Upstream struct {
	opts    UpstreamOptions
	proxies []*ReverseProxy
	index   uint64
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

func NewUpstream(opts UpstreamOptions, transportOptions *ClientTransportOptions) (*Upstream, error) {

	if len(opts.ID) == 0 {
		return nil, fmt.Errorf("upstream id can't be empty")
	}

	if len(opts.Servers) == 0 {
		return nil, fmt.Errorf("upstream servers can't be empty. upstream id: %s", opts.ID)
	}

	clientOpts := defaultClientOptions

	if len(opts.ClientTransport) > 0 && transportOptions != nil {
		clientOpts = []config.ClientOption{
			client.WithNoDefaultUserAgentHeader(true),
			client.WithDisableHeaderNamesNormalizing(true),
			client.WithDisablePathNormalizing(true),

			client.WithDialTimeout(transportOptions.DailTimeout),
			client.WithClientReadTimeout(transportOptions.ReadTimeout),
			client.WithWriteTimeout(transportOptions.WriteTimeout),
			client.WithMaxConnWaitTimeout(transportOptions.MaxConnWaitTimeout),
			client.WithMaxConnsPerHost(transportOptions.MaxConnsPerHost),

			//client.WithMaxIdleConns(100),
			//client.WithMaxIdleConnDuration(60 * time.Second),
			client.WithKeepAlive(transportOptions.KeepAlive),
			//client.WithMaxConnDuration(120 * time.Second),
		}
	}

	upstream := &Upstream{
		opts:    opts,
		proxies: make([]*ReverseProxy, 0),
	}

	for _, server := range opts.Servers {

		if !isValidURL(server.URL) {
			return nil, fmt.Errorf("invalid upstream url: %s, must start with http:// or https://", server.URL)
		}

		proxy, err := NewSingleHostReverseProxy(server.URL, clientOpts...)
		if err != nil {
			return nil, err
		}
		upstream.proxies = append(upstream.proxies, proxy)
	}

	return upstream, nil
}

func (u *Upstream) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	defer ctx.Abort()

	var proxy *ReverseProxy

	switch u.opts.Strategy {
	case RoundRobinStrategy:
		proxy = u.pickupByRoundRobin()
	case RandomStrategy:
		u.serveByRandom(c, ctx)
	default:
		proxy = u.pickupByRoundRobin()
	}

	if proxy != nil {
		addr, _ := url.Parse(proxy.Target)
		ctx.Set(UPSTREAM_ADDR, addr.Host)
		startTime := time.Now()
		proxy.ServeHTTP(c, ctx)
		//fmt.Println("proxy done")

		ctx.Set(UPSTREAM_STATUS, ctx.Response.StatusCode())
		dur := time.Since(startTime)
		mic := dur.Microseconds()
		duration := float64(mic) / 1e6
		responseTime := strconv.FormatFloat(duration, 'f', -1, 64)
		ctx.Set(UPSTREAM_RESPONSE_TIME, responseTime)

	} else {
		fmt.Println("no upstream found")
	}
}

func (u *Upstream) pickupByRoundRobin() *ReverseProxy {
	index := atomic.AddUint64(&u.index, 1)
	proxy := u.proxies[(int(index)-1)%len(u.proxies)]
	return proxy
}

func (u *Upstream) serveByRandom(c context.Context, ctx *app.RequestContext) {
}

// isValidURL checks if the given URL starts with http:// or https://
func isValidURL(u string) bool {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsedURL.Scheme)
	return scheme == "http" || scheme == "https"
}
