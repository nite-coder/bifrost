package gateway

import (
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/tracer/accesslog"
	"http-benchmark/pkg/tracer/prometheus"
	"http-benchmark/pkg/zero"
	"log/slog"
	"time"

	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/rs/dnscache"
)

type Bifrost struct {
	options     *config.Options
	HttpServers map[string]*HTTPServer
	resolver    *dnscache.Resolver
	stopCh      chan bool
	zero        *zero.ZeroDownTime
}

func (b *Bifrost) Run() {
	i := 0
	for _, server := range b.HttpServers {
		if i == len(b.HttpServers)-1 {
			// last server need to blocked
			server.Run()
		}
		go server.Run()
		i++
	}
}

func (b *Bifrost) ZeroDownTime() *zero.ZeroDownTime {
	return b.zero
}

func (b *Bifrost) Stop() {
	b.stopCh <- true
}

func (b *Bifrost) Shutdown() {
	b.Stop()

}

func Load(opts config.Options, isReload bool) (*Bifrost, error) {
	// validate
	err := config.Validate(opts)
	if err != nil {
		return nil, err
	}

	zeroOptions := zero.Options{
		UpgradeSock: opts.UpgradeSock,
		PIDFile:     opts.PIDFile,
	}

	bifrsot := &Bifrost{
		resolver:    &dnscache.Resolver{},
		HttpServers: make(map[string]*HTTPServer),
		options:     &opts,
		stopCh:      make(chan bool),
		zero:        zero.New(zeroOptions),
	}

	go func() {
		t := time.NewTimer(1 * time.Hour)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				bifrsot.resolver.Refresh(true)
				slog.Info("refresh dns cache successfully")
			case <-bifrsot.stopCh:
				return
			}
		}
	}()

	// system logger
	logger, err := log.NewLogger(opts.Logging)
	if err != nil {
		return nil, err
	}
	slog.SetDefault(logger)

	tracers := []tracer.Tracer{}

	// prometheus tracer
	if opts.Metrics.Prometheus.Enabled && !isReload {
		promOpts := []prometheus.Option{}

		if len(opts.Metrics.Prometheus.Buckets) > 0 {
			promOpts = append(promOpts, prometheus.WithHistogramBuckets(opts.Metrics.Prometheus.Buckets))
		}

		promTracer := prometheus.NewTracer(promOpts...)
		tracers = append(tracers, promTracer)
	}

	// access log
	accessLogTracers := map[string]*accesslog.Tracer{}
	if len(opts.AccessLogs) > 0 && !isReload {

		for id, accessLogOptions := range opts.AccessLogs {
			if !accessLogOptions.Enabled {
				continue
			}

			accessLogTracer, err := accesslog.NewTracer(accessLogOptions)
			if err != nil {
				return nil, err
			}

			if accessLogTracer != nil {
				accessLogTracers[id] = accessLogTracer
			}
		}
	}

	for id, server := range opts.Servers {
		if id == "" {
			return nil, fmt.Errorf("http server id can't be empty")
		}

		server.ID = id

		if server.Bind == "" {
			return nil, fmt.Errorf("http server bind can't be empty")
		}

		_, found := bifrsot.HttpServers[id]
		if found {
			return nil, fmt.Errorf("http server '%s' already exists", id)
		}

		if len(server.AccessLogID) > 0 {
			_, found := opts.AccessLogs[server.AccessLogID]
			if !found {
				return nil, fmt.Errorf("access log '%s' was not found in server '%s'", server.AccessLogID, server.ID)
			}

			accessLogTracer, found := accessLogTracers[server.AccessLogID]
			if found {
				tracers = append(tracers, accessLogTracer)
			}
		}

		httpServer, err := newHTTPServer(bifrsot, server, tracers)
		if err != nil {
			return nil, err
		}

		bifrsot.HttpServers[id] = httpServer
	}

	return bifrsot, nil
}
