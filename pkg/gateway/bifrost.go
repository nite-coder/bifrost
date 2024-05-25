package gateway

import (
	"fmt"
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/middleware"
	"http-benchmark/pkg/provider/file"

	"github.com/cloudwego/hertz/pkg/app"
)

type Bifrost struct {
	httpServers []*HTTPServer
}

func (b *Bifrost) Run() {
	for i := 0; i < len(b.httpServers)-1; i++ {
		go b.httpServers[i].Run()
	}

	b.httpServers[len(b.httpServers)-1].Run()
}

func Load(opts domain.Options) (*Bifrost, error) {

	bifrsot := &Bifrost{}

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

		httpServer, err := NewHTTPServer(entry, opts)
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

func Variable(c *app.RequestContext, name string) (string, bool) {

	switch name {

	case TIME:

		return c.GetString(TIME), true
	case REMOTE_ADDR:
		return c.RemoteAddr().String(), true
	default:
		return "", false
	}

}

func init() {
	_ = RegisterMiddleware("strip_prefix", func(param map[string]any) (app.HandlerFunc, error) {
		prefixes := param["prefixes"].([]string)
		m := middleware.NewStripPrefixMiddleware(prefixes)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("add_prefix", func(param map[string]any) (app.HandlerFunc, error) {
		prefix := param["prefix"].(string)
		m := middleware.NewAddPrefixMiddleware(prefix)
		return m.ServeHTTP, nil
	})
}
