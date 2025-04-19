package config

import (
	"os"
	"testing"

	_ "github.com/nite-coder/bifrost/pkg/middleware/cors"
	"github.com/nite-coder/bifrost/pkg/resolver"
	"github.com/nite-coder/bifrost/pkg/router"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	dnsResolver, _ = resolver.NewResolver(resolver.Options{})
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestValidateProviders(t *testing.T) {

	t.Run("file provider", func(t *testing.T) {
		options := NewOptions()

		options.Providers.File.Enabled = true
		err := validateProviders(options)
		assert.Error(t, err)
	})

	t.Run("nacos config provider", func(t *testing.T) {
		options := NewOptions()

		options.Providers.Nacos.Config.Enabled = true
		err := validateProviders(options)
		assert.Error(t, err)

		options.Providers.Nacos.Config.Endpoints = []string{
			"http://localhost:8848",
		}

		options.Providers.Nacos.Config.Files = []*File{
			{
				DataID: "abc.yaml",
			},
		}

		err = validateProviders(options)
		assert.NoError(t, err)
	})

	t.Run("nacos discovery provider", func(t *testing.T) {
		options := NewOptions()

		options.Providers.Nacos.Discovery.Enabled = true
		err := validateProviders(options)
		assert.Error(t, err)

		options.Providers.Nacos.Discovery.Endpoints = []string{
			"http://localhost:8848",
		}

		err = validateProviders(options)
		assert.NoError(t, err)
	})
}

func TestValidateRoutes(t *testing.T) {

	t.Run("success", func(t *testing.T) {
		options := NewOptions()

		options.Services["aa"] = ServiceOptions{
			Url: "http://test1/hello",
		}

		route1 := RouteOptions{
			ID:        "route1",
			Paths:     []string{"/hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, &route1)

		route2 := RouteOptions{
			ID:        "route2",
			Paths:     []string{"= /hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, &route2)

		route3 := RouteOptions{
			ID:        "route3",
			Methods:   []string{"GET"},
			Paths:     []string{"^~ /hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, &route3)

		route4 := RouteOptions{
			ID:        "route4",
			Paths:     []string{"~ /hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, &route4)

		route5 := RouteOptions{
			ID:        "route5",
			Paths:     []string{"~* /hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, &route5)

		err := validateRoutes(options, true)
		assert.NoError(t, err)
	})

	t.Run("service not found", func(t *testing.T) {
		options := NewOptions()
		route1 := RouteOptions{
			ID:        "route1",
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Routes = append(options.Routes, &route1)

		err := validateRoutes(options, true)
		assert.ErrorContains(t, err, "the service 'test1' can't be found in the route 'route1'")
	})

	t.Run("duplicate routes1", func(t *testing.T) {
		options := NewOptions()

		options.Services["aa"] = ServiceOptions{
			Url: "http://test1/hello",
		}

		options.Servers["apiv1"] = ServerOptions{}

		test1 := &RouteOptions{
			ID:        "test1",
			Paths:     []string{"/hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, test1)

		test2 := &RouteOptions{
			ID:        "test2",
			Methods:   []string{"GET"},
			Paths:     []string{"/hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, test2)

		err := validateRoutes(options, true)
		assert.ErrorIs(t, err, router.ErrAlreadyExists)
	})

	t.Run("duplicate routes2", func(t *testing.T) {
		options := NewOptions()

		options.Servers["apiv1"] = ServerOptions{}

		options.Services["aa"] = ServiceOptions{
			Url: "http://test1/hello",
		}

		test1 := &RouteOptions{
			ID:        "test1",
			Methods:   []string{"GET", "POST"},
			Paths:     []string{"= /hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, test1)

		test2 := &RouteOptions{
			ID:        "test2",
			Methods:   []string{"GET", "POST"},
			Paths:     []string{"= /hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, test2)

		err := validateRoutes(options, true)
		assert.ErrorIs(t, err, router.ErrAlreadyExists)
	})

	t.Run("two servers", func(t *testing.T) {
		options := NewOptions()

		options.Servers["apiv1"] = ServerOptions{}
		options.Servers["apiv2"] = ServerOptions{}

		options.Services["aa"] = ServiceOptions{
			Url: "http://test1/hello",
		}

		route1 := RouteOptions{
			ID:        "route1",
			Servers:   []string{"apiv1"},
			Paths:     []string{"/hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, &route1)

		route2 := RouteOptions{
			ID:        "route2",
			Servers:   []string{"apiv2"},
			Paths:     []string{"^~ /hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, &route2)

		err := validateRoutes(options, true)
		assert.NoError(t, err)

		route3 := RouteOptions{
			ID:        "route3",
			Servers:   []string{"apiv2"},
			Paths:     []string{"^~ /hello"},
			ServiceID: "aa",
		}
		options.Routes = append(options.Routes, &route3)

		err = validateRoutes(options, true)
		assert.ErrorIs(t, err, router.ErrAlreadyExists)
	})

	t.Run("middlewares", func(t *testing.T) {
		options := NewOptions()
		route1 := &RouteOptions{
			ID:        "route1",
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Routes = append(options.Routes, route1)

		service := ServiceOptions{
			ID:  "test1",
			Url: "http://localhost:8888",
		}

		corsMiddleware := MiddlwareOptions{
			ID:   "cors",
			Type: "cors",
		}

		route1.Middlewares = append(route1.Middlewares, corsMiddleware)

		options.Services["test1"] = service

		err := validateRoutes(options, true)
		assert.NoError(t, err)

		noMiddleware := MiddlwareOptions{
			ID:  "no",
			Use: "no",
		}

		route1.Middlewares = append(route1.Middlewares, noMiddleware)

		err = validateRoutes(options, true)
		assert.Error(t, err)
	})
}

func TestValidateMiddlewares(t *testing.T) {

	t.Run("normal", func(t *testing.T) {
		options := NewOptions()

		options.Middlewares["cors_id"] = MiddlwareOptions{
			Type: "cors",
		}
		err := validateMiddlewares(options, true)
		assert.NoError(t, err)
	})

	t.Run("not found middleware", func(t *testing.T) {
		options := NewOptions()

		options.Middlewares["cors_id"] = MiddlwareOptions{
			Type: "cors11",
		}
		err := validateMiddlewares(options, true)
		assert.Error(t, err)
	})

	t.Run("can't run as use mode", func(t *testing.T) {
		options := NewOptions()

		options.Middlewares["cors_id"] = MiddlwareOptions{
			Use: "cors",
		}
		err := validateMiddlewares(options, true)
		assert.Error(t, err)
	})
}

func TestValidateServer(t *testing.T) {

	t.Run("bind", func(t *testing.T) {
		options := NewOptions()

		server := ServerOptions{
			Bind: "",
		}

		options.Servers["apiv1"] = server

		err := validateServers(options, true)
		assert.ErrorContains(t, err, "the bind can't be empty for server")
	})

	t.Run("client ip", func(t *testing.T) {
		options := NewOptions()

		server := ServerOptions{
			Bind: ":8080",
			TrustedCIDRS: []string{
				"0.0.0.0/0",
				"192.168.0.10/32",
			},
		}

		options.Servers["apiv1"] = server

		err := validateServers(options, true)
		assert.NoError(t, err)

		server = ServerOptions{
			Bind: ":8080",
			TrustedCIDRS: []string{
				"192.168.0.1",
			},
		}

		options.Servers["apiv1"] = server

		err = validateServers(options, true)
		assert.Error(t, err)
	})

	t.Run("access log", func(t *testing.T) {
		options := NewOptions()

		options.AccessLogs["mylog"] = AccessLogOptions{
			Output: "stdout",
		}

		server := ServerOptions{
			Bind:        ":8080",
			AccessLogID: "mylog",
		}

		options.Servers["apiv1"] = server

		err := validateServers(options, true)
		assert.NoError(t, err)

		server = ServerOptions{
			Bind:        ":8080",
			AccessLogID: "mylog1",
		}

		options.Servers["apiv1"] = server

		err = validateServers(options, true)
		assert.Error(t, err)
	})

	t.Run("middlewares", func(t *testing.T) {
		options := NewOptions()

		server := ServerOptions{
			Bind: ":8080",
		}

		corsMiddleware := MiddlwareOptions{
			ID:   "cors",
			Type: "cors",
		}

		server.Middlewares = append(server.Middlewares, corsMiddleware)

		options.Servers["apiv1"] = server

		err := validateServers(options, true)
		assert.NoError(t, err)

		noMiddleware := MiddlwareOptions{
			ID:  "aaa",
			Use: "aaa",
		}

		server.Middlewares = append(server.Middlewares, noMiddleware)

		options.Servers["apiv1"] = server

		err = validateServers(options, true)
		assert.Error(t, err)
	})

}

func TestValidateService(t *testing.T) {

	t.Run("service url with ip", func(t *testing.T) {
		options := NewOptions()
		route1 := &RouteOptions{
			ID:        "route1",
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Routes = append(options.Routes, route1)

		options.Services["test1"] = ServiceOptions{
			Url: "http://10.1.2.16:8088",
		}

		err := validateServices(options, true)
		assert.NoError(t, err)

		options.SkipResolver = true
		err = validateServices(options, true)
		assert.NoError(t, err)
	})

	t.Run("service url with domain", func(t *testing.T) {
		options := NewOptions()
		route1 := &RouteOptions{
			ID:        "route1",
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Routes = append(options.Routes, route1)

		options.Services["test1"] = ServiceOptions{
			Url: "http://google.com",
		}

		err := validateServices(options, true)
		assert.NoError(t, err)

		options.SkipResolver = true
		err = validateServices(options, true)
		assert.NoError(t, err)
	})

	t.Run("service url localhost", func(t *testing.T) {
		options := NewOptions()
		route1 := &RouteOptions{
			ID:        "route1",
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Routes = append(options.Routes, route1)

		options.Services["test1"] = ServiceOptions{
			Url: "http://localhost:8888",
		}

		err := validateServices(options, true)
		assert.NoError(t, err)

		options.SkipResolver = true
		err = validateServices(options, true)
		assert.NoError(t, err)
	})

	t.Run("service url no upstream", func(t *testing.T) {
		options := NewOptions()

		options.Services["service1"] = ServiceOptions{
			Url: "http://test1/hello",
		}

		err := validateServices(options, true)
		assert.Error(t, err)

		options.SkipResolver = true
		err = validateServices(options, true)
		assert.Error(t, err)
	})

	t.Run("middlewares", func(t *testing.T) {
		options := NewOptions()
		route1 := &RouteOptions{
			ID:        "route1",
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Routes = append(options.Routes, route1)

		service := ServiceOptions{
			ID:  "test1",
			Url: "http://localhost:8888",
		}

		corsMiddleware := MiddlwareOptions{
			ID:   "cors",
			Type: "cors",
		}

		service.Middlewares = append(service.Middlewares, corsMiddleware)

		options.Services["test1"] = service

		err := validateServices(options, true)
		assert.NoError(t, err)

		noMiddleware := MiddlwareOptions{
			ID:  "no",
			Use: "no",
		}

		service.Middlewares = append(service.Middlewares, noMiddleware)
		options.Services["test1"] = service

		err = validateServices(options, true)
		assert.Error(t, err)
	})
}

func TestValidateUpstream(t *testing.T) {

	t.Run("upstream target with ip", func(t *testing.T) {
		options := NewOptions()
		options.Upstreams["test"] = UpstreamOptions{
			Targets: []TargetOptions{
				{
					Target: "10.1.2.250:8088",
				},
				{
					Target: "10.1.1.1",
				},
			},
		}

		err := validateUpstreams(options, true)
		assert.NoError(t, err)

		options.SkipResolver = true
		err = validateUpstreams(options, true)
		assert.NoError(t, err)
	})

	t.Run("upstream target with domain", func(t *testing.T) {
		options := NewOptions()
		options.Upstreams["test"] = UpstreamOptions{
			Targets: []TargetOptions{
				{
					Target: "google.com",
				},
				{
					Target: "github.com",
				},
			},
		}

		err := validateUpstreams(options, true)
		assert.NoError(t, err)

		options.SkipResolver = true
		err = validateUpstreams(options, true)
		assert.NoError(t, err)
	})

	t.Run("upstream target with localhost", func(t *testing.T) {
		options := NewOptions()
		options.Upstreams["test"] = UpstreamOptions{
			Targets: []TargetOptions{
				{
					Target: "localhost:999",
				},
				{
					Target: "localhost",
				},
				// TODO: support ipv6
				// {
				// 	Target: "[::1]:999",
				// },
			},
		}

		err := validateUpstreams(options, true)
		assert.NoError(t, err)

		options.SkipResolver = true
		err = validateUpstreams(options, true)
		assert.NoError(t, err)
	})

	t.Run("target with local hostname", func(t *testing.T) {
		options := NewOptions()
		options.Upstreams["test"] = UpstreamOptions{
			Targets: []TargetOptions{
				{
					Target: "dev1:999",
				},
				{
					Target: "dev2",
				},
			},
		}

		err := validateUpstreams(options, true)
		assert.Error(t, err)

		options.SkipResolver = true
		err = validateUpstreams(options, true)
		assert.NoError(t, err)
	})
}

func TestValidateMetrics(t *testing.T) {

	t.Run("success", func(t *testing.T) {
		options := NewOptions()

		options.Metrics.Prometheus.Enabled = true
		options.Metrics.Prometheus.ServerID = "test"

		options.Servers["test"] = ServerOptions{
			ID:   "test",
			Bind: ":8080",
		}

		err := validateMetrics(options, true)
		assert.NoError(t, err)
	})

	t.Run("no server id", func(t *testing.T) {
		options := NewOptions()

		options.Metrics.Prometheus.Enabled = true
		options.Metrics.Prometheus.ServerID = "test"

		err := validateMetrics(options, true)
		assert.ErrorContains(t, err, "the server_id 'test' for the prometheus is not found")
	})

}

func TestValidateTracing(t *testing.T) {
	options := NewOptions()

	options.Tracing.Enabled = true
	options.Tracing.ServiceName = "bifrost"
	options.Tracing.Propagators = append(options.Tracing.Propagators, "tracecontext", "baggage")

	err := validateTracing(options.Tracing)
	assert.NoError(t, err)
}

func TestValidateResolver(t *testing.T) {
	options := NewOptions()
	err := validateResolver(options)
	assert.NoError(t, err)

	options.Resolver.Order = []string{"last", "a", "cname"}
	err = validateResolver(options)
	assert.NoError(t, err)

	options.Resolver.Order = []string{"srv"}
	err = validateResolver(options)
	assert.Error(t, err)

	options.Resolver.Hostsfile = "/not/exists"
	err = validateResolver(options)
	assert.Error(t, err)
}

func TestConfigDNS(t *testing.T) {
	options := NewOptions()

	options.Servers["srv"] = ServerOptions{
		Bind: ":80",
	}

	options.Services["hello"] = ServiceOptions{
		Url: "http://www.google.com:8001",
	}

	options.Upstreams["test"] = UpstreamOptions{
		Targets: []TargetOptions{
			{
				Target: "www.google.com:8000",
			},
			{
				Target: "www.google.com",
			},
		},
	}

	err := ValidateConfig(options, true)
	assert.NoError(t, err)

	options.Upstreams["test"] = UpstreamOptions{
		Targets: []TargetOptions{
			{
				Target: "www.google.com123",
			},
		},
	}

	err = ValidateConfig(options, true)
	assert.ErrorIs(t, err, resolver.ErrNotFound)
}
