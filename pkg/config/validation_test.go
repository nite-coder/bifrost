package config

import (
	"testing"

	"github.com/nite-coder/bifrost/pkg/dns"
	"github.com/nite-coder/bifrost/pkg/router"
	"github.com/stretchr/testify/assert"
)

func TestUpstreamDNS(t *testing.T) {
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

func TestDpulicateRoutes(t *testing.T) {
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
}

func TestEmptyTargetUpstream(t *testing.T) {
	options := NewOptions()

	options.Upstreams["test"] = UpstreamOptions{}

	err := validateUpstreams(options.Upstreams)
	assert.Error(t, err)
}

func TestValidateService(t *testing.T) {

	t.Run("service not found", func(t *testing.T) {
		options := NewOptions()
		options.Routes["route1"] = RouteOptions{
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}

		err := validateRoutes(options, true)
		assert.Error(t, err)
	})

	t.Run("service url ip", func(t *testing.T) {
		options := NewOptions()
		options.Routes["route1"] = RouteOptions{
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Services["test1"] = ServiceOptions{
			Url: "http://10.1.2.16:8088",
		}

		err := validateRoutes(options, true)
		assert.NoError(t, err)
	})

	t.Run("service url domain", func(t *testing.T) {
		options := NewOptions()
		options.Routes["route1"] = RouteOptions{
			Paths:     []string{"/hello"},
			ServiceID: "test1",
		}
		options.Services["test1"] = ServiceOptions{
			Url: "http://google.com",
		}

		err := validateRoutes(options, true)
		assert.NoError(t, err)
	})

	t.Run("service url no upstream", func(t *testing.T) {
		options := NewOptions()

		options.Services["service1"] = ServiceOptions{
			Url: "http://test1/hello",
		}

		err := validateServices(options, true)
		assert.Error(t, err)
	})
}
