package gateway

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/go-resty/resty/v2"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/stretchr/testify/assert"
)

type TestOrder struct {
	ID    string `json:"id"`
	Price string `json:"price"`
}

func TestBifrost(t *testing.T) {
	// setup upstream
	options := config.NewOptions()
	options.Upstreams["backend"] = config.UpstreamOptions{
		Strategy: config.RoundRobinStrategy,
		Targets: []config.TargetOptions{
			{
				Target: "127.0.0.1",
			},
		},
	}

	// setup service
	options.Services["orders"] = config.ServiceOptions{
		Url: "http://backend:8000",
	}

	// setup route
	options.Routes["get_order"] = config.RouteOptions{
		Paths: []string{
			"/",
		},
		ServiceID: "orders",
	}

	// setup server
	options.Servers["apiv1"] = config.ServerOptions{
		Bind:        "localhost:8080",
		ReusePort:   true,
		TCPQuickAck: true,
		TCPFastOpen: true,
		PPROF:       true,
	}

	options.Servers["apiv1_tls"] = config.ServerOptions{
		Bind: "localhost:8443",
		TLS: config.TLSOptions{
			CertPEM: "../../test/certs/localhost.crt",
			KeyPEM:  "../../test/certs/localhost.key",
		},
	}

	options.Servers["apiv1_http2"] = config.ServerOptions{
		Bind:  "localhost:8442",
		HTTP2: true,
		TLS: config.TLSOptions{
			CertPEM: "../../test/certs/localhost.crt",
			KeyPEM:  "../../test/certs/localhost.key",
		},
	}

	bifrost, err := NewBifrost(options, false)
	assert.NoError(t, err)

	go bifrost.Run()

	// setup backend server (the server need to be started after bifrost due to data race hlog)
	backendServ := server.New(
		server.WithHostPorts("127.0.0.1:8000"),
		server.WithExitWaitTime(1*time.Second),
	)

	backendServ.Any("/api/v1/orders", func(ctx context.Context, c *app.RequestContext) {
		order := &TestOrder{
			ID:    "1",
			Price: "100",
		}

		c.JSON(200, order)
	})

	go backendServ.Spin()

	time.Sleep(2 * time.Second) // wait for server ready

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = bifrost.Shutdown(ctx)

		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = backendServ.Shutdown(ctx)
	}()

	t.Run("get order", func(t *testing.T) {
		client := resty.New()
		client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

		urls := []string{
			"http://localhost:8080/api/v1/orders",
			"https://localhost:8443/api/v1/orders",
		}

		for _, url := range urls {
			resp, err := client.R().
				Get(url)

			assert.NoError(t, err)

			testOrder := &TestOrder{}
			err = json.Unmarshal(resp.Body(), testOrder)
			assert.NoError(t, err)

			assert.Equal(t, 200, resp.StatusCode())
			assert.Equal(t, "1", testOrder.ID)
			assert.Equal(t, "100", testOrder.Price)
		}
	})
}
