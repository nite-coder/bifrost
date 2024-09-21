package main

import (
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
)

func main() {

	options := config.NewOptions()

	// setup upstream
	options.Upstreams["test_server"] = config.UpstreamOptions{
		Strategy: config.RoundRobinStrategy,
		Targets: []config.TargetOptions{
			{
				Target: "127.0.0.1:8000",
			},
			{
				Target: "127.0.0.1:80",
			},
		},
	}

	// setup service
	options.Services["test_service"] = config.ServiceOptions{
		Url: "http://test_server:8000",
	}

	// setup route
	options.Routes["test_route"] = config.RouteOptions{
		Paths: []string{
			"/",
		},
		ServiceID: "test_service",
	}

	// setup server
	options.Servers["test_server"] = config.ServerOptions{
		Bind: ":8001",
	}

	err := gateway.Run(options)
	if err != nil {
		panic(err)
	}
}
