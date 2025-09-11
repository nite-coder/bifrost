package main

import (
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/initialize"
)

func main() {
	_ = initialize.Bifrost()

	options := config.NewOptions()

	// setup upstream
	options.Upstreams["test_upstream"] = config.UpstreamOptions{
		Balancer: config.BalancerOptions{
			Type: "round_robin",
		},
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
		URL: "http://test_upstream:8000",
	}

	// setup route
	allRoutes := config.RouteOptions{
		ID: "all_routes",
		Paths: []string{
			"/",
		},
		ServiceID: "test_service",
	}
	options.Routes = append(options.Routes, &allRoutes)

	// setup server
	options.Servers["api_server"] = config.ServerOptions{
		Bind: "127.0.0.1:8001",
	}

	err := gateway.Run(options)
	if err != nil {
		panic(err)
	}
}
