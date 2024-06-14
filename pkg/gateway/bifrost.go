package gateway

import (
	"fmt"
	"http-benchmark/pkg/domain"
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

func Load(opts domain.Options) (*Bifrost, error) {
	// validate
	err := validateOptions(opts)
	if err != nil {
		return nil, err
	}

	bifrsot := &Bifrost{
		resolver:    &dnscache.Resolver{},
		httpServers: make(map[string]*HTTPServer),
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

	// access log
	accessLogTracers := map[string]*accesslog.Tracer{}
	if len(opts.Observability.AccessLogs) > 0 {

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

func LoadFromConfig(path string) (*Bifrost, error) {

	if !fileExist(path) {
		return nil, fmt.Errorf("config file not found, path: %s", path)
	}

	// main config file
	fileProviderOpts := domain.FileProviderOptions{
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
			mainOpts, err = mergedOptions(mainOpts, c.Content)
			if err != nil {
				errMsg := fmt.Sprintf("path: %s, error: %s", c.Path, err.Error())
				return nil, fmt.Errorf(errMsg)
			}
		}
	}

	bifrost, err := Load(mainOpts)
	if err != nil {
		return nil, err
	}

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

	return bifrost, nil
}

func (b *Bifrost) watch() {
	go func() {
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

	// main config file
	fileProviderOpts := domain.FileProviderOptions{
		Paths: []string{bifrost.configPath},
	}

	fileProvider := file.NewProvider(fileProviderOpts)

	cInfo, err := fileProvider.Open()
	if err != nil {
		return err
	}

	mainOpts, err := parseContent(cInfo[0].Content)
	if err != nil {
		return err
	}
	fileProvider.Reset()

	// file provider
	if mainOpts.Providers.File.Enabled && len(mainOpts.Providers.File.Paths) > 0 {
		for _, content := range mainOpts.Providers.File.Paths {
			fileProvider.Add(content)
		}

		cInfo, err = fileProvider.Open()
		if err != nil {
			return err
		}

		for _, c := range cInfo {
			mainOpts, err = mergedOptions(mainOpts, c.Content)
			if err != nil {
				errMsg := fmt.Sprintf("path: %s, error: %s", c.Path, err.Error())
				return fmt.Errorf(errMsg)
			}
		}
	}

	isReload := false
	for key, entryOpts := range mainOpts.Entries {
		entryOpts.ID = key

		engine, err := NewEngine(bifrost, entryOpts, mainOpts)
		if err != nil {
			return err
		}

		server, found := bifrost.httpServers[key]
		if found && server.entryOpts.Bind == entryOpts.Bind {
			server.switcher.SetEngine(engine)
			isReload = true
		}
	}

	if isReload {
		slog.Info("bifrost is reload")
	}

	return nil
}
