package main

import (
	"http-benchmark/gateway"
	"http-benchmark/middleware"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
)

func main() {

	_ = gateway.RegisterMiddleware("time", func(param map[string]any) (app.HandlerFunc, error) {
		m := middleware.NewTimeMiddleware()
		return m.ServeHTTP, nil
	})

	opts := gateway.Options{

		Middlewares: []gateway.MiddlwareOptions{
			{
				ID:   "time_trace",
				Kind: "time",
			},
		},

		Entries: []gateway.EntryOptions{
			{
				ID:   "apiv1",
				Bind: ":8001",
				Middlewares: []gateway.MiddlwareOptions{
					{
						ID: "time_trace",
					},
				},
				AccessLog: gateway.AccessLogOptions{
					Enabled:  false,
					Template: `{"time": "$time", "request": "$request", "upstream": "$upstream"}`,
				},
			},
			{
				ID:   "apiv2",
				Bind: ":8002",
			},
		},

		Routes: []gateway.RouteOptions{
			{
				Match: "/api/v1*",
				Middlewares: []gateway.MiddlwareOptions{
					{
						Kind: "strip_prefix",
						Params: map[string]any{
							"prefixes": []string{"/api/v1"}},
					},
				},
				Upstream: "default",
			},
			{
				Match:    "/spot/orders",
				Upstream: "default",
			},
			{
				Match: "/orders",
				Middlewares: []gateway.MiddlwareOptions{
					{
						Kind: "add_prefix",
						Params: map[string]any{
							"prefix": "/spot"},
					},
				},
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
