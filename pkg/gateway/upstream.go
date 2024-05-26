package gateway

import (
	"context"
	"fmt"
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/log"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/common/config"
)

type Upstream struct {
	opts    domain.UpstreamOptions
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

func NewUpstream(opts domain.UpstreamOptions, transportOptions *domain.TransportOptions) (*Upstream, error) {

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

		if !isValidURL(server.URL) {
			return nil, fmt.Errorf("invalid upstream url: %s, must start with http:// or https://", server.URL)
		}

		proxy, err := NewSingleHostReverseProxy(server.URL, clientOpts...)
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

	var proxy *ReverseProxy
	done := make(chan bool)

	gopool.CtxGo(c, func() {
		switch u.opts.Strategy {
		case domain.RoundRobinStrategy:
			proxy = u.pickupByRoundRobin()
		case domain.RandomStrategy:
			u.serveByRandom(c, ctx)
		default:
			proxy = u.pickupByRoundRobin()
		}

		ctx.Set("X-Forwarded-For", ctx.Request.Header.Get("X-Forwarded-For"))

		if proxy != nil {
			// TODO: remove url.Parse here
			addr, _ := url.Parse(proxy.Target)
			ctx.Set(domain.UPSTREAM_ADDR, addr)
			startTime := time.Now()
			proxy.ServeHTTP(c, ctx)

			ctx.Set(domain.UPSTREAM_STATUS, ctx.Response.StatusCode())

			dur := time.Since(startTime)
			mic := dur.Microseconds()
			duration := float64(mic) / 1e6
			responseTime := strconv.FormatFloat(duration, 'f', -1, 64)
			ctx.Set(domain.UPSTREAM_RESPONSE_TIME, responseTime)

		} else {
			logger.ErrorContext(c, "no proxy found")
		}

		select {
		case <-c.Done():
			logger.Info("client request canceled")
		default:
			done <- true
		}
	})

	select {
	case <-c.Done():
		logger.ErrorContext(c, "client request canceled")
	case <-done:
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
