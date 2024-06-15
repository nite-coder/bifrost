package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/log"
	"log/slog"
	"strconv"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/rs/dnscache"
	"github.com/valyala/bytebufferpool"
)

type Service struct {
	bifrost           *Bifrost
	options           config.ServiceOptions
	upstreams         map[string]*Upstream
	proxy             *ReverseProxy
	upstream          *Upstream
	isDynamicUpstream bool
}

func newService(bifrost *Bifrost, opts *config.ServiceOptions, upstreamOptions map[string]config.UpstreamOptions) (*Service, error) {

	svc := &Service{
		bifrost:   bifrost,
		options:   *opts,
		upstreams: make(map[string]*Upstream),
	}

	// validate
	if len(opts.Host) == 0 {
		return nil, fmt.Errorf("service host can't be empty")
	}

	if len(opts.Protocol) == 0 {
		opts.Protocol = config.ProtocolHTTP
	}

	// dynamic service
	if opts.Host[0] == '$' {
		svc.isDynamicUpstream = true
		return svc, nil
	}

	// exist upstream
	upstreamOpts, found := upstreamOptions[opts.Host]
	if found {
		upstreamOpts.ID = opts.Host

		upstream, err := NewUpstream(bifrost, svc.options, upstreamOpts)
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
	if allowDNS(opts.Host) {
		_, err := bifrost.resolver.LookupHost(context.Background(), opts.Host)
		if err != nil {
			return nil, fmt.Errorf("lookup service host error: %v", err)
		}
		dnsResolver = bifrost.resolver
	}

	switch opts.Protocol {
	case config.ProtocolHTTP:
		if dnsResolver != nil {
			clientOpts = append(clientOpts, client.WithDialer(newHTTPDialer(dnsResolver)))
		}
	case config.ProtocolHTTPS:
		clientOpts = append(clientOpts, client.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: opts.TLSVerify,
		}))
	}

	url := fmt.Sprintf("%s://%s:%d%s", opts.Protocol, opts.Host, opts.Port, opts.Path)
	proxy, err := NewSingleHostReverseProxy(url, bifrost.opts.Observability.Tracing.Enabled, clientOpts...)
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

		if svc.isDynamicUpstream {
			upstreamName := ctx.GetString(svc.options.Host)

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
				proxy = svc.upstream.pickupByRoundRobin()
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
