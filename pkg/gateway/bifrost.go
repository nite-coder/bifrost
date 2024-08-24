package gateway

import (
	"context"
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/tracer/accesslog"
	"http-benchmark/pkg/tracer/opentelemetry"
	"http-benchmark/pkg/tracer/prometheus"
	"http-benchmark/pkg/zero"
	"log/slog"
	"sync"
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
			return
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

func (b *Bifrost) Shutdown(ctx context.Context) error {
	b.Stop()

	var wg sync.WaitGroup

	for _, server := range b.HttpServers {
		wg.Add(1)
		go func(srv *HTTPServer) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()

			_ = srv.Shutdown(ctx)
			slog.Debug("http server shutdown", "id", srv.options.ID)
		}(server)
	}

	wg.Wait()

	slog.Debug("http server is closed")
	return b.zero.Close(ctx)
}

func NewBifrost(mainOptions config.Options, isReload bool) (*Bifrost, error) {
	// validate
	err := config.ValidateValue(mainOptions)
	if err != nil {
		return nil, err
	}

	err = config.ValidateMapping(mainOptions)
	if err != nil {
		return nil, err
	}

	zeroOptions := zero.Options{
		UpgradeSock: mainOptions.UpgradeSock,
		PIDFile:     mainOptions.PIDFile,
	}

	bifrsot := &Bifrost{
		resolver:    &dnscache.Resolver{},
		HttpServers: make(map[string]*HTTPServer),
		options:     &mainOptions,
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
	logger, err := log.NewLogger(mainOptions.Logging)
	if err != nil {
		return nil, err
	}
	slog.SetDefault(logger)

	tracers := []tracer.Tracer{}

	// prometheus
	if mainOptions.Metrics.Prometheus.Enabled && !isReload {
		promOpts := []prometheus.Option{}

		if len(mainOptions.Metrics.Prometheus.Buckets) > 0 {
			promOpts = append(promOpts, prometheus.WithHistogramBuckets(mainOptions.Metrics.Prometheus.Buckets))
		}

		promTracer := prometheus.NewTracer(promOpts...)
		tracers = append(tracers, promTracer)
	}

	// access log
	accessLogTracers := map[string]*accesslog.Tracer{}
	if len(mainOptions.AccessLogs) > 0 && !isReload {

		for id, accessLogOptions := range mainOptions.AccessLogs {
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

	if mainOptions.Tracing.OTLP.Enabled {
		// otel tracing
		t, err := opentelemetry.NewTracer(mainOptions.Tracing)
		if err != nil {
			return nil, err
		}
		tracers = append(tracers, t)
	}

	for id, server := range mainOptions.Servers {
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
			_, found := mainOptions.AccessLogs[server.AccessLogID]
			if !found {
				return nil, fmt.Errorf("access log '%s' was not found in server '%s'", server.AccessLogID, server.ID)
			}

			accessLogTracer, found := accessLogTracers[server.AccessLogID]
			if found {
				tracers = append(tracers, accessLogTracer)
			}
		}

		httpServer, err := newHTTPServer(bifrsot, server, tracers, isReload)
		if err != nil {
			return nil, err
		}

		bifrsot.HttpServers[id] = httpServer
	}

	return bifrsot, nil
}
