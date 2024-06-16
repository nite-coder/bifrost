package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/log"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/rs/dnscache"
	"github.com/valyala/bytebufferpool"
)

type Service struct {
	bifrost         *Bifrost
	options         config.ServiceOptions
	upstreams       map[string]*Upstream
	proxy           *ReverseProxy
	upstream        *Upstream
	dynamicUpstream string
}

func newService(bifrost *Bifrost, opts *config.ServiceOptions, upstreamOptions map[string]config.UpstreamOptions) (*Service, error) {

	svc := &Service{
		bifrost:   bifrost,
		options:   *opts,
		upstreams: make(map[string]*Upstream),
	}

	addr, err := url.Parse(opts.Url)
	if err != nil {
		return nil, err
	}

	host := addr.Hostname()

	// validate
	if len(host) == 0 {
		return nil, fmt.Errorf("service host can't be empty. service_id: %s", opts.ID)
	}

	if len(opts.Protocol) == 0 {
		opts.Protocol = config.ProtocolHTTP
	}

	// dynamic service
	if host[0] == '$' {
		svc.dynamicUpstream = host
		return svc, nil
	}

	// exist upstream
	upstreamOpts, found := upstreamOptions[host]
	if found {
		upstreamOpts.ID = host

		upstream, err := newUpstream(bifrost, svc.options, upstreamOpts)
		if err != nil {
			return nil, err
		}

		svc.upstream = upstream
		return svc, nil
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

	var dnsResolver dnscache.DNSResolver
	if allowDNS(host) {
		_, err := bifrost.resolver.LookupHost(context.Background(), host)
		if err != nil {
			return nil, fmt.Errorf("lookup service host error: %v", err)
		}
		dnsResolver = bifrost.resolver
	}

	switch strings.ToLower(addr.Scheme) {
	case "http":
		if dnsResolver != nil {
			clientOpts = append(clientOpts, client.WithDialer(newHTTPDialer(dnsResolver)))
		}
	case "https":
		clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: opts.TLSVerify,
		}))
	}

	url := fmt.Sprintf("%s://%s%s", addr.Scheme, host, addr.Path)

	if addr.Port() != "" {
		url = fmt.Sprintf("%s://%s:%s%s", addr.Scheme, host, addr.Port(), addr.Path)
	}

	proxy, err := newSingleHostReverseProxy(url, bifrost.opts.Observability.Tracing.Enabled, 0, clientOpts...)
	if err != nil {
		return nil, err
	}

	svc.proxy = proxy
	return svc, nil
}

func (svc *Service) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	logger := log.FromContext(c)
	defer ctx.Abort()
	done := make(chan bool)

	gopool.CtxGo(c, func() {
		defer func() {
			done <- true
		}()

		if len(svc.dynamicUpstream) > 0 {
			upstreamName := ctx.GetString(svc.dynamicUpstream)

			if len(upstreamName) == 0 {
				logger.Warn("upstream is not found", slog.String("name", upstreamName))
				ctx.Abort()
				return
			}

			var found bool
			svc.upstream, found = svc.upstreams[upstreamName]
			if !found {
				logger.Warn("upstream is not found", slog.String("name", upstreamName))
				ctx.Abort()
				return
			}
		}

		proxy := svc.proxy
		if svc.upstream != nil && proxy == nil {
			ctx.Set(config.UPSTREAM, svc.upstream.opts.ID)
			switch svc.upstream.opts.Strategy {
			case config.RoundRobinStrategy, "":
				proxy = svc.upstream.roundRobin()
			case config.WeightedStrategy:
				proxy = svc.upstream.weighted()
			case config.RandomStrategy:
				proxy = svc.upstream.random()
			}
		}

		if proxy == nil {
			logger.ErrorContext(c, "no proxy found")
			ctx.Abort()
			return
		}

		startTime := time.Now()
		proxy.ServeHTTP(c, ctx)

		dur := time.Since(startTime)
		mic := dur.Microseconds()
		duration := float64(mic) / 1e6
		responseTime := strconv.FormatFloat(duration, 'f', -1, 64)
		ctx.Set(config.UPSTREAM_DURATION, responseTime)

		if ctx.GetBool("target_timeout") {
			ctx.Response.SetStatusCode(504)
		} else {
			ctx.Set(config.UPSTREAM_STATUS, ctx.Response.StatusCode())
		}
	})

	select {
	case <-c.Done():
		time := time.Now()
		ctx.Set(config.CLIENT_CANCELED_AT, time)

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