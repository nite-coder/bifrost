package config

import (
	"testing"

	"github.com/nite-coder/bifrost/pkg/dns"
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
