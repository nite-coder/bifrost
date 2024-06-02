package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/log"
	"log/slog"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/rs/dnscache"
	"github.com/valyala/bytebufferpool"
)

type Upstream struct {
	opts    domain.UpstreamOptions
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

func NewUpstream(bifrost *Bifrost, opts domain.UpstreamOptions, transportOptions *domain.TransportOptions) (*Upstream, error) {

	if len(opts.ID) == 0 {
		return nil, fmt.Errorf("upstream id can't be empty")
	}

	if len(opts.Servers) == 0 {
		return nil, fmt.Errorf("upstream servers can't be empty. upstream id: %s", opts.ID)
	}

	clientOpts := defaultClientOptions

	if len(opts.ClientTransport) > 0 && transportOptions != nil {
		if transportOptions.DailTimeout != nil {
			clientOpts = append(clientOpts, client.WithDialTimeout(*transportOptions.DailTimeout))
		}

		if transportOptions.ReadTimeout != nil {
			clientOpts = append(clientOpts, client.WithClientReadTimeout(*transportOptions.ReadTimeout))
		}

		if transportOptions.WriteTimeout != nil {
			clientOpts = append(clientOpts, client.WithWriteTimeout(*transportOptions.WriteTimeout))
		}

		if transportOptions.MaxConnWaitTimeout != nil {
			clientOpts = append(clientOpts, client.WithMaxConnWaitTimeout(*transportOptions.MaxConnWaitTimeout))
		}

		if transportOptions.MaxIdleConnsPerHost != nil {
			clientOpts = append(clientOpts, client.WithMaxConnsPerHost(*transportOptions.MaxIdleConnsPerHost))
		}

		if transportOptions.KeepAlive != nil {
			clientOpts = append(clientOpts, client.WithKeepAlive(*transportOptions.KeepAlive))
		}

	}

	upstream := &Upstream{
		opts:    opts,
		proxies: make([]*ReverseProxy, 0),
	}

	for _, server := range opts.Servers {
		parsedURL, err := url.Parse(server.URL)
		if err != nil {
			return nil, err
		}

		scheme := strings.ToLower(parsedURL.Scheme)
		if !(scheme == "http" || scheme == "https") {
			return nil, fmt.Errorf("unsupported scheme: %s", scheme)
		}

		host, _, err := net.SplitHostPort(parsedURL.Host)
		if err != nil {
			host = parsedURL.Host
		}
		var dnsResolver dnscache.DNSResolver
		if !isIP(host) {
			_, err := bifrost.resolver.LookupHost(context.Background(), host)
			if err != nil {
				return nil, fmt.Errorf("lookup upstream host error: %v", err)
			}
			dnsResolver = bifrost.resolver
		}

		var proxyOpts []config.ClientOption
		if strings.HasPrefix(server.URL, "https") {
			skip := true

			if transportOptions != nil && transportOptions.InsecureSkipVerify != nil {
				skip = *transportOptions.InsecureSkipVerify
			}

			proxyOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
				InsecureSkipVerify: skip,
			}))
		} else {
			proxyOpts = append(clientOpts, client.WithDialer(newHTTPDialer(dnsResolver)))
		}

		proxy, err := NewSingleHostReverseProxy(server.URL, proxyOpts...)
		//proxy.SetSaveOriginResHeader(true)
		if err != nil {
			return nil, err
		}
		upstream.proxies = append(upstream.proxies, proxy)
	}

	return upstream, nil
}

func (u *Upstream) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	logger := log.FromContext(c)
	defer ctx.Abort()
	done := make(chan bool)

	gopool.CtxGo(c, func() {
		var proxy *ReverseProxy
		switch u.opts.Strategy {
		case domain.RoundRobinStrategy:
			proxy = u.pickupByRoundRobin()
		case domain.RandomStrategy:
			u.serveByRandom(c, ctx)
		default:
			proxy = u.pickupByRoundRobin()
		}

		if proxy != nil {
			startTime := time.Now()
			proxy.ServeHTTP(c, ctx)

			dur := time.Since(startTime)
			mic := dur.Microseconds()
			duration := float64(mic) / 1e6
			responseTime := strconv.FormatFloat(duration, 'f', -1, 64)
			ctx.Set(domain.UPSTREAM_DURATION, responseTime)

			if ctx.GetBool("upstream_timeout") {
				ctx.Response.SetStatusCode(504)
			} else {
				ctx.Set(domain.UPSTREAM_STATUS, ctx.Response.StatusCode())
			}
		} else {
			logger.ErrorContext(c, "no proxy found")
		}

		done <- true
	})

	select {
	case <-c.Done():
		time := time.Now()
		ctx.Set(domain.CLIENT_CANCELED_AT, time)

		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		buf.Write(ctx.Request.Method())
		buf.Write(spaceByte)
		buf.Write(ctx.Request.URI().FullURI())
		fullURI := buf.String()
		logger.WarnContext(c, "client cancel the request",
			slog.String("full_uri", fullURI),
		)
		<-done
		ctx.Response.SetStatusCode(499)
	case <-done:
	}
}

func (u *Upstream) pickupByRoundRobin() *ReverseProxy {
	index := u.index.Add(1)
	proxy := u.proxies[(int(index)-1)%len(u.proxies)]
	return proxy
}

func (u *Upstream) serveByRandom(c context.Context, ctx *app.RequestContext) {
}

func isIP(str string) bool {
	return net.ParseIP(str) != nil
}
