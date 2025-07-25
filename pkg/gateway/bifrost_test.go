package gateway

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/go-resty/resty/v2"
	"github.com/nite-coder/bifrost/pkg/config"
	_ "github.com/nite-coder/bifrost/pkg/middleware/cors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2"
)

type TestOrder struct {
	ID    string `json:"id"`
	Price string `json:"price"`
}

func TestBifrost(t *testing.T) {
	// setup upstream

	options := config.NewOptions()

	options.Metrics.Prometheus.Enabled = true
	options.Metrics.Prometheus.ServerID = "apiv1"

	options.AccessLogs["main"] = config.AccessLogOptions{
		Output: "",
		Template: `      {"time":"$time",
      "remote_addr":"$network.peer.address",
      "host": "$http.request.host",
      "request":"$http.request",
      "req_body":"$http.request.body",
      "x_forwarded_for":"$http.request.header.X-Forwarded-For",
      "upstream_addr":"$upstream.request.host",
      "upstream.request":"$upstream.request",
      "upstream_duration":$upstream.duration,
      "upstream_status":$upstream.response.status_code,
      "status":$http.response.status_code,
      "grpc_status":"$grpc.status_code",
      "grpc_messaage":"$grpc.message",
      "duration":$http.request.duration}`,
	}

	options.Upstreams["backend"] = config.UpstreamOptions{
		Balancer: "round_robin",
		Targets: []config.TargetOptions{
			{
				Target: "127.0.0.1",
			},
		},
	}

	// setup service
	options.Services["orders"] = config.ServiceOptions{
		URL: "http://backend:8000",
	}

	// setup route
	getOrderRoute := config.RouteOptions{
		ID: "get_order",
		Paths: []string{
			"/",
		},
		ServiceID: "orders",
	}

	options.Routes = append(options.Routes, &getOrderRoute)

	// setup server
	options.Servers["apiv1"] = config.ServerOptions{
		Bind:        "localhost:8080",
		ReusePort:   true,
		TCPQuickAck: true,
		TCPFastOpen: true,
		Backlog:     4096,
		PPROF:       true,
		AccessLogID: "main",
		TrustedCIDRS: []string{
			"127.0.0.1/32",
		},
		Middlewares: []config.MiddlwareOptions{
			{
				Type: "cors",
			},
		},
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

		clientIP := c.ClientIP()
		assert.Equal(t, "127.0.0.1", clientIP)

		c.JSON(200, order)
	})

	go backendServ.Spin()

	time.Sleep(2 * time.Second) // wait for server ready

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = bifrost.ShutdownNow(ctx)

		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = backendServ.Shutdown(ctx)
	}()

	t.Run("get order", func(t *testing.T) {
		client := resty.New()
		client = client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

		urls := []string{
			"http://localhost:8080/api/v1/orders",
			"https://localhost:8443/api/v1/orders",
		}

		for _, url := range urls {

			if url == "https://localhost:8442/api/v1/orders" {
				client.SetTransport(&http2.Transport{
					AllowHTTP: true,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
					DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
						return net.Dial(network, addr)
					},
				})
			}

			resp, err := client.R().
				Get(url)

			assert.NoError(t, err)

			testOrder := &TestOrder{}
			err = sonic.Unmarshal(resp.Body(), testOrder)
			assert.NoError(t, err)

			assert.Equal(t, 200, resp.StatusCode())
			assert.Equal(t, "1", testOrder.ID)
			assert.Equal(t, "100", testOrder.Price)

			t.Log(resp.Request.RawRequest.Proto)
		}
	})

	t.Run("test http2", func(t *testing.T) {
		client := http.Client{
			Transport: &http2.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		resp, err := client.Get("https://localhost:8442/spot/orders")
		assert.NoError(t, err)
		assert.Equal(t, "HTTP/2.0", resp.Proto)
	})

	t.Run("test metric endpoint", func(t *testing.T) {
		client := resty.New()
		client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

		resp, err := client.R().
			Get("http://localhost:8080/metrics")

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode())

		isUnknown := strings.Contains(resp.String(), "unknown")
		assert.False(t, isUnknown, "metric endpoint has unknown labels")
	})
}
