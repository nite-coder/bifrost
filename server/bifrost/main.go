package main

import (
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/gateway"
	"http-benchmark/pkg/middleware"
	"log/slog"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

func main() {

	_ = gateway.RegisterMiddleware("time", func(param map[string]any) (app.HandlerFunc, error) {
		m := middleware.NewTimeMiddleware()
		return m.ServeHTTP, nil
	})

	// bifrost, err := gateway.LoadFromConfig("./config.yaml")
	// if err != nil {
	// 	slog.Error("load config error", "error", err)
	// }

	opts := buildOptions()
	bifrost, err := gateway.Load(opts)
	if err != nil {
		slog.Error("load bifrost error", err)
		return
	}

	bifrost.Run()
}

func buildOptions() domain.Options {
	return domain.Options{

		Middlewares: []domain.MiddlwareOptions{
			{
				ID:   "time_trace",
				Kind: "time",
			},
		},

		Transports: []domain.TransportOptions{
			{
				ID:                 "mytransport",
				InsecureSkipVerify: true,
				MaxConnsPerHost:    2000,
				WriteTimeout:       3 * time.Second,
				ReadTimeout:        3 * time.Second,
				DailTimeout:        3 * time.Second,
			},
		},

		Entries: []domain.EntryOptions{
			{
				ID:   "apiv1",
				Bind: ":8001",
				// Middlewares: []gateway.MiddlwareOptions{
				// 	{
				// 		Link: "time_trace",
				// 	},
				// },
				AccessLog: domain.AccessLogOptions{
					Enabled:    false,
					Escape:     "json",
					BufferSize: 64 * gateway.KB,
					TimeFormat: "2006-01-02T15:04:05",
					FilePath:   "./logs/access.log",
					Template: `{"time":"$time",
					"remote_addr":"$remote_addr",
					"request":"$request_method $request_path $request_protocol",
					"status":$status,
					"req_body":"$request_body",
					"upstream_addr":"$upstream_addr",
					"upstream_status":$upstream_status,
					"x_forwarded_for":"$header_X-Forwarded-For",
					"duration":$duration,
					"upstream_response_time":$upstream_response_time}`,
				},
				ReusePort: true,
			},
			{
				ID:   "apiv2",
				Bind: ":8002",
			},
		},

		Routes: []domain.RouteOptions{
			{
				Match: "/api/v1*",
				Middlewares: []domain.MiddlwareOptions{
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
				Middlewares: []domain.MiddlwareOptions{
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

		Upstreams: []domain.UpstreamOptions{
			{
				ID: "default",
				Servers: []domain.BackendServerOptions{
					{
						URL:    "http://127.0.0.1:8000",
						Weight: 1,
					},
				},
			},
		},
	}
}
