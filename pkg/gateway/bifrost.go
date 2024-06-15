package gateway

import (
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/provider/file"
	"http-benchmark/pkg/tracer/accesslog"
	"http-benchmark/pkg/tracer/prometheus"
	"log/slog"
	"time"

	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/rs/dnscache"
)

type reloadFunc func(bifrost *Bifrost) error

type Bifrost struct {
	configPath   string
	opts         config.Options
	fileProvider *file.FileProvider
	httpServers  map[string]*HTTPServer
	resolver     *dnscache.Resolver
	reloadCh     chan bool
	onReload     reloadFunc
}

func (b *Bifrost) Run() {
	i := 0
	for _, server := range b.httpServers {
		if i == len(b.httpServers)-1 {
			// last server need to blocked
			server.Run()
		}
		go server.Run()
		i++
	}
}

func LoadFromConfig(path string) (*Bifrost, error) {
	return loadFromConfig(path, false)
}

func loadFromConfig(path string, isReload bool) (*Bifrost, error) {
	if !fileExist(path) {
		return nil, fmt.Errorf("config file not found, path: %s", path)
	}

	// main config file
	fileProviderOpts := config.FileProviderOptions{
		Paths: []string{path},
	}

	fileProvider := file.NewProvider(fileProviderOpts)

	cInfo, err := fileProvider.Open()
	if err != nil {
		return nil, err
	}

	mainOpts, err := parseContent(cInfo[0].Content)
	if err != nil {
		return nil, err
	}
	fileProvider.Reset()

	// file provider
	if mainOpts.Providers.File.Enabled && len(mainOpts.Providers.File.Paths) > 0 {
		for _, content := range mainOpts.Providers.File.Paths {
			fileProvider.Add(content)
		}

		cInfo, err = fileProvider.Open()
		if err != nil {
			return nil, err
		}

		for _, c := range cInfo {
			mainOpts, err = mergeOptions(mainOpts, c.Content)
			if err != nil {
				errMsg := fmt.Sprintf("path: %s, error: %s", c.Path, err.Error())
				return nil, fmt.Errorf(errMsg)
			}
		}
	}

	bifrost, err := load(mainOpts, isReload)
	if err != nil {
		return nil, err
	}

	if !isReload {
		reloadCh := make(chan bool)
		bifrost.fileProvider = fileProvider
		bifrost.configPath = path
		bifrost.onReload = reload
		bifrost.reloadCh = reloadCh

		if mainOpts.Providers.File.Watch {
			fileProvider.Add(path)
			fileProvider.OnChanged = func() error {
				reloadCh <- true
				return nil
			}
			_ = fileProvider.Watch()
			bifrost.watch()
		}
	}

	return bifrost, nil
}

func Load(opts config.Options) (*Bifrost, error) {
	return load(opts, false)
}

func load(opts config.Options, isReload bool) (*Bifrost, error) {
	// validate
	err := validateOptions(opts)
	if err != nil {
		return nil, err
	}

	bifrsot := &Bifrost{
		resolver:    &dnscache.Resolver{},
		httpServers: make(map[string]*HTTPServer),
		opts:        opts,
	}

	go func() {
		t := time.NewTicker(60 * time.Minute)
		defer t.Stop()
		for range t.C {
			bifrsot.resolver.Refresh(true)
		}
		slog.Info("refresh dns resolver")
	}()

	// system logger
	logger, err := log.NewLogger(opts.Observability.Logging)
	if err != nil {
		return nil, err
	}
	slog.SetDefault(logger)

	tracers := []tracer.Tracer{}

	// prometheus tracer
	if opts.Observability.Metrics.Prometheus.Enabled && !isReload {
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

	// access log
	accessLogTracers := map[string]*accesslog.Tracer{}
	if len(opts.Observability.AccessLogs) > 0 && !isReload {

		for id, accessLogOptions := range opts.Observability.AccessLogs {
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

	for id, entry := range opts.Entries {
		if id == "" {
			return nil, fmt.Errorf("http server id can't be empty")
		}

		entry.ID = id

		if entry.Bind == "" {
			return nil, fmt.Errorf("http server bind can't be empty")
		}

		_, found := bifrsot.httpServers[id]
		if found {
			return nil, fmt.Errorf("http server '%s' already exists", id)
		}

		if len(entry.AccessLogID) > 0 {
			_, found := opts.Observability.AccessLogs[entry.AccessLogID]
			if !found {
				return nil, fmt.Errorf("access log '%s' was not found in entry '%s'", entry.AccessLogID, entry.ID)
			}

			accessLogTracer, found := accessLogTracers[entry.AccessLogID]
			if found {
				tracers = append(tracers, accessLogTracer)
			}
		}

		httpServer, err := newHTTPServer(bifrsot, entry, opts, tracers)
		if err != nil {
			return nil, err
		}

		bifrsot.httpServers[id] = httpServer
	}

	return bifrsot, nil
}

func (b *Bifrost) watch() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("bifrost: unknown error when reload config", "error", err)
			}
			b.watch()
		}()

		for {
			<-b.reloadCh
			err := b.onReload(b)
			if err != nil {
				slog.Error("bifrost: fail to reload config", "error", err)
			}
		}
	}()
}

func reload(bifrost *Bifrost) error {
	slog.Info("bifrost: reloading...")

	newBifrost, err := loadFromConfig(bifrost.configPath, true)
	if err != nil {
		return err
	}

	isReloaded := false

	for id, server := range bifrost.httpServers {
		newServer, found := newBifrost.httpServers[id]
		if found && server.entryOpts.Bind == newServer.entryOpts.Bind {
			server.switcher.SetEngine(newServer.switcher.Engine())
			isReloaded = true
		}
	}

	slog.Info("bifrost is reloaded successfully", "isReloaded", isReloaded)

	return nil
}
