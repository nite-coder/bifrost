package gateway

import (
	"context"
	"testing"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestBifrost(t *testing.T) {
	options := config.NewOptions()

	// setup upstream
	options.Upstreams["test_upstream"] = config.UpstreamOptions{
		Strategy: config.RoundRobinStrategy,
		Targets: []config.TargetOptions{
			{
				Target: "127.0.0.1:8000",
			},
		},
	}

	// setup service
	options.Services["test_service"] = config.ServiceOptions{
		Url: "http://test_upstream:8000",
	}

	// setup route
	options.Routes["all_routes"] = config.RouteOptions{
		Paths: []string{
			"/",
		},
		ServiceID: "test_service",
	}

	// setup server
	options.Servers["api_server"] = config.ServerOptions{
		Bind: "localhost:8001",
	}

	bifrost, err := NewBifrost(options, false)
	assert.NoError(t, err)

	err = bifrost.Shutdown(context.Background())
	assert.NoError(t, err)
}
