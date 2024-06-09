package gateway

import (
	"fmt"
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/provider/file"
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
	if len(opts.Entries) == 0 {
		return nil, fmt.Errorf("no entry found")
	}

	if len(opts.Routes) == 0 {
		return nil, fmt.Errorf("no route found")
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

		httpServer, err := NewHTTPServer(bifrsot, entry, opts, tracers)
		if err != nil {
			return nil, err
		}

		bifrsot.httpServers[id] = httpServer
	}

	return bifrsot, nil
}

func LoadFromConfig(path string) (*Bifrost, error) {
	reloadCh := make(chan bool)

	// main config file
	fileProviderOpts := domain.FileProviderOptions{
		Path: []string{path},
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
	if mainOpts.Providers.File.Enabled && len(mainOpts.Providers.File.Path) > 0 {
		for _, content := range mainOpts.Providers.File.Path {
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

		defer func() {
			fileProvider.OnChanged = func() error {
				reloadCh <- true
				return nil
			}
			_ = fileProvider.Watch()
		}()
	}

	bifrost, err := Load(mainOpts)
	if err != nil {
		return nil, err
	}

	bifrost.fileProvider = fileProvider
	bifrost.configPath = path
	bifrost.onReload = reload
	bifrost.reloadCh = reloadCh
	bifrost.watch()

	return bifrost, nil
}

func (b *Bifrost) watch() {
	go func() {
		for {
			<-b.reloadCh
			err := b.onReload(b)
			if err != nil {
				slog.Error("fail to reload bifrost", "error", err)
			}
		}
	}()
}

func reload(bifrost *Bifrost) error {
	// main config file
	fileProviderOpts := domain.FileProviderOptions{
		Path: []string{bifrost.configPath},
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
	if mainOpts.Providers.File.Enabled && len(mainOpts.Providers.File.Path) > 0 {
		for _, content := range mainOpts.Providers.File.Path {
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
