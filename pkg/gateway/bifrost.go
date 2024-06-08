package gateway

import (
	"fmt"
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/provider/file"
	"http-benchmark/pkg/tracer/prometheus"
	"log/slog"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/rs/dnscache"
)

type Bifrost struct {
	httpServers []*HTTPServer
	resolver    *dnscache.Resolver
}

func (b *Bifrost) Run() {
	for i := 0; i < len(b.httpServers)-1; i++ {
		go b.httpServers[i].Run()
	}

	b.httpServers[len(b.httpServers)-1].Run()
}

func Load(opts domain.Options) (*Bifrost, error) {

	bifrsot := &Bifrost{
		resolver: &dnscache.Resolver{},
	}

	go func() {
		t := time.NewTicker(60 * time.Minute)
		defer t.Stop()
		for range t.C {
			bifrsot.resolver.Refresh(true)
		}
		slog.Info("refresh dns resolver")
	}()

	logger, err := log.NewLogger(opts.Observability.Logging)
	if err != nil {
		return nil, err
	}
	slog.SetDefault(logger)

	tracers := []tracer.Tracer{}

	// prometheus tracer
	if opts.Observability.Metrics.Prometheus.Enabled {
		promOpts := []prometheus.Option{
			prometheus.WithEnableGoCollector(true),
			prometheus.WithDisableServer(false),
		}

		if len(opts.Observability.Metrics.Prometheus.Buckets) > 0 {
			promOpts = append(promOpts, prometheus.WithHistogramBuckets(opts.Observability.Metrics.Prometheus.Buckets))
		}

		promTracer := prometheus.NewTracer(":9091", "/metrics", promOpts...)
		tracers = append(tracers, promTracer)
	}

	httpServers := map[string]*HTTPServer{}
	for _, entry := range opts.Entries {

		if entry.ID == "" {
			return nil, fmt.Errorf("http server id can't be empty")
		}

		if entry.Bind == "" {
			return nil, fmt.Errorf("http server bind can't be empty")
		}

		_, found := httpServers[entry.ID]
		if found {
			return nil, fmt.Errorf("http server '%s' already exists", entry.ID)
		}

		httpServer, err := NewHTTPServer(bifrsot, entry, opts, tracers)
		if err != nil {
			return nil, err
		}
		bifrsot.httpServers = append(bifrsot.httpServers, httpServer)
	}

	return bifrsot, nil
}

func LoadFromConfig(path string) (*Bifrost, error) {

	fileProvider := file.NewFileProvider()
	opts, err := fileProvider.Open(path)
	if err != nil {
		return nil, err
	}

	bifrsot, err := Load(opts)
	if err != nil {
		return nil, err
	}

	return bifrsot, nil
}

type CreateMiddlewareHandler func(param map[string]any) (app.HandlerFunc, error)

var middlewareFactory map[string]CreateMiddlewareHandler = make(map[string]CreateMiddlewareHandler)

func RegisterMiddleware(kind string, handler CreateMiddlewareHandler) error {

	if _, found := middlewareFactory[kind]; found {
		return fmt.Errorf("middleware handler '%s' already exists", kind)
	}

	middlewareFactory[kind] = handler

	return nil
}
