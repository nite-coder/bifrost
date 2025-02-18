package config

import (
	"testing"

	"github.com/nite-coder/bifrost/pkg/dns"
	"github.com/nite-coder/bifrost/pkg/router"
	"github.com/stretchr/testify/assert"
)

func TestValidateRoutes(t *testing.T) {
	t.Run("service not found", func(t *testing.T) {
		options := NewOptions()
		options.Routes["route1"] = RouteOptions{
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}

		err := validateRoutes(options, true)
		assert.Error(t, err)
	})

	t.Run("dpulicate routes", func(t *testing.T) {
		options := NewOptions()

		options.Services["aa"] = ServiceOptions{
			Url: "http://test1/hello",
		}

		options.Routes["test1"] = RouteOptions{
			Paths:     []string{"/hello"},
			ServiceID: "aa",
		}

		options.Routes["test2"] = RouteOptions{
			Methods:   []string{"GET"},
			Paths:     []string{"/hello"},
			ServiceID: "aa",
		}

		err := validateRoutes(options, true)
		assert.ErrorIs(t, err, router.ErrAlreadyExists)
	})
}

func TestValidateService(t *testing.T) {

	t.Run("service url with ip", func(t *testing.T) {
		options := NewOptions()
		options.Routes["route1"] = RouteOptions{
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Services["test1"] = ServiceOptions{
			Url: "http://10.1.2.16:8088",
		}

		err := validateServices(options, true)
		assert.NoError(t, err)

		options.Resolver.SkipTest = true
		err = validateServices(options, true)
		assert.NoError(t, err)
	})

	t.Run("service url with domain", func(t *testing.T) {
		options := NewOptions()
		options.Routes["route1"] = RouteOptions{
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Services["test1"] = ServiceOptions{
			Url: "http://google.com",
		}

		err := validateServices(options, true)
		assert.NoError(t, err)

		options.Resolver.SkipTest = true
		err = validateServices(options, true)
		assert.NoError(t, err)
	})

	t.Run("service url localhost", func(t *testing.T) {
		options := NewOptions()
		options.Routes["route1"] = RouteOptions{
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Services["test1"] = ServiceOptions{
			Url: "http://localhost:8888",
		}

		err := validateServices(options, true)
		assert.NoError(t, err)

		options.Resolver.SkipTest = true
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

		options.Resolver.SkipTest = true
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

		options.Resolver.SkipTest = true
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

		options.Resolver.SkipTest = true
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

		options.Resolver.SkipTest = true
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

		options.Resolver.SkipTest = true
		err = validateUpstreams(options, true)
		assert.NoError(t, err)
	})
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
	assert.ErrorIs(t, err, dns.ErrNotFound)
}
