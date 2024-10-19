package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/tracer/accesslog"
	"github.com/nite-coder/bifrost/pkg/tracer/opentelemetry"
	"github.com/nite-coder/bifrost/pkg/tracer/prometheus"
	"github.com/nite-coder/bifrost/pkg/zero"

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
	return b.shutdown(ctx, false)
}

func (b *Bifrost) ShutdownNow(ctx context.Context) error {
	return b.shutdown(ctx, true)
}

func (b *Bifrost) shutdown(ctx context.Context, now bool) error {
	b.Stop()

	wg := &sync.WaitGroup{}

	maxTimeout := 10 * time.Second

	for _, server := range b.HttpServers {
		wg.Add(1)

		if server.options.Timeout.Graceful > 0 && server.options.Timeout.Graceful > maxTimeout {
			maxTimeout = server.options.Timeout.Graceful
		}

		go func(srv *HTTPServer) {
			defer wg.Done()

			if srv.options.Timeout.Graceful > 0 {
				ctx, cancel := context.WithTimeout(ctx, server.options.Timeout.Graceful)
				defer cancel()
				_ = srv.Shutdown(ctx)
			} else {
				_ = srv.Shutdown(ctx)
			}

			slog.Debug("http server shutdown", "id", srv.options.ID)
		}(server)
	}

	if now {
		maxTimeout = 500 * time.Millisecond
	}
	waitTimeout(wg, maxTimeout)
	return b.zero.Close(ctx)
}

// NewBifrost creates a new instance of Bifrost.
//
// It takes in two parameters: mainOptions of type config.Options and isReload of type bool.
// The mainOptions parameter contains the main configuration options for the Bifrost instance.
// The isReload parameter indicates whether the function is being called during a reload operation.
// The function returns a pointer to Bifrost and an error.
func NewBifrost(mainOptions config.Options, isReload bool) (*Bifrost, error) {
	// validate
	err := config.ValidateConfig(mainOptions)
	if err != nil {
		return nil, err
	}

	err = config.ValidateMapping(mainOptions)
	if err != nil {
		return nil, err
	}

	tCache := timecache.New(mainOptions.TimerResolution)
	timecache.Set(tCache)

	// system logger
	logger, err := log.NewLogger(mainOptions.Logging)
	if err != nil {
		return nil, err
	}
	slog.SetDefault(logger)

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
			return nil, errors.New("http server id can't be empty")
		}

		server.ID = id

		if server.Bind == "" {
			return nil, errors.New("http server bind can't be empty")
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
