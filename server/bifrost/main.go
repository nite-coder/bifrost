package main

import (
	"http-benchmark/gateway"
	"log/slog"
)

func main() {

	opts := gateway.Options{

		Entries: []gateway.EntryOptions{
			{
				ID:   "apiv1",
				Bind: ":8001",
			},
			{
				ID:   "apiv2",
				Bind: ":8002",
			},
		},

		Routes: []gateway.RouteOptions{
			{
				Match:    "/spot/orders",
				Upstream: "default",
			},
			{
				Match:    "/options*",
				Upstream: "default",
				Entries:  []string{"apiv2"},
			},
			{
				Match:    "~ ^/futures/(usdt|btc)/orders$",
				Upstream: "default",
			},
		},

		Upstreams: []gateway.UpstreamOptions{
			{
				ID: "default",
				Servers: []gateway.BackendServerOptions{
					{
						URL:    "http://127.0.0.1:8000",
						Weight: 1,
					},
					{
						URL:    "http://127.0.0.1:8000",
						Weight: 1,
					},
				},
			},
		},
	}

	bifrost, err := gateway.Load(opts)
	if err != nil {
		slog.Error("load bifrost error", err)
		return
	}

	bifrost.Run()
}
