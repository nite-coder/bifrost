package gateway

import (
	"fmt"
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/provider/file"
	"log/slog"
	"regexp"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/rs/dnscache"
)

var (
	reIsVariable = regexp.MustCompile(`\$\w+(-\w+)*`)
	spaceByte    = []byte{byte(' ')}
	questionByte = []byte{byte('?')}
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
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for range t.C {
			bifrsot.resolver.Refresh(true)
		}
		slog.Info("refresh dns resolver")
	}()

	logger, err := newLogger(opts.Bifrost.Logging)
	if err != nil {
		return nil, err
	}
	slog.SetDefault(logger)

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

		httpServer, err := NewHTTPServer(bifrsot, entry, opts)
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
