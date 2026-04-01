package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/config"
)

// setupHTTP2Server starts a test server that supports ONLY HTTP/2 (via TLS).
func setupHTTP2Server(_ *testing.T, handler http.HandlerFunc) (*httptest.Server, string) {
	ts := httptest.NewUnstartedServer(handler)
	ts.EnableHTTP2 = true
	ts.StartTLS()
	return ts, ts.URL
}

func TestProxy_HTTP2_Basic(t *testing.T) {
	const backendResponse = "I am the HTTP/2 backend"

	ts, url := setupHTTP2Server(t, func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Errorf("upstream got proto %d.%d; want 2.0", r.ProtoMajor, r.ProtoMinor)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(backendResponse))
	})
	defer ts.Close()

	lst, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lst.Addr().String()

	r := server.New(server.WithListener(lst))

	proxyOptions := Options{
		Target:   url,
		Protocol: config.ProtocolHTTP2,
		Weight:   1,
	}

	// Create a client with native TLS config
	clientOpts := ClientOptions{
		IsHTTP2: true,
		HZOptions: append(DefaultClientOptions(),
			client.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}), /* #nosec G402 */
		),
	}
	hClient, err := NewClient(clientOpts)
	require.NoError(t, err)

	proxy, err := New(proxyOptions, hClient)
	require.NoError(t, err)

	r.GET("/backend", proxy.ServeHTTP)

	go r.Spin()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = r.Shutdown(ctx)
	}()

	c, _ := client.NewClient()
	req := protocol.AcquireRequest()
	resp := protocol.AcquireResponse()
	req.SetMethod("GET")
	req.SetRequestURI(fmt.Sprintf("http://%s/backend", addr))

	err = c.Do(context.Background(), req, resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, backendResponse, string(resp.Body()))
}

func TestProxy_HTTP2_GRPC_Trailers(t *testing.T) {
	ts, url := setupHTTP2Server(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/grpc")
		w.Header().Set("Trailer", "Grpc-Status, Grpc-Message")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte("grpc body"))

		w.Header().Set("Grpc-Status", "0")
		w.Header().Set("Grpc-Message", "ok")
	})
	defer ts.Close()

	lst, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := lst.Addr().String()

	r := server.New(server.WithListener(lst))

	proxyOptions := Options{
		Target:   url,
		Protocol: config.ProtocolHTTP2,
		Weight:   1,
	}

	// Create a client with native TLS config
	clientOpts := ClientOptions{
		IsHTTP2: true,
		HZOptions: append(DefaultClientOptions(),
			client.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}), /* #nosec G402 */
		),
	}
	hClient, err := NewClient(clientOpts)
	require.NoError(t, err)

	proxy, err := New(proxyOptions, hClient)
	require.NoError(t, err)

	r.POST("/grpc", proxy.ServeHTTP)
	go r.Spin()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = r.Shutdown(ctx)
	}()

	// Wait for server to start
	assert.Eventually(t, func() bool {
		conn, e := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if e == nil {
			_ = conn.Close()
			return true
		}
		return false
	}, 2*time.Second, 100*time.Millisecond)

	c, _ := client.NewClient()
	req := protocol.AcquireRequest()
	resp := protocol.AcquireResponse()
	req.SetMethod("POST")
	req.SetRequestURI(fmt.Sprintf("http://%s/grpc", addr))

	err = c.Do(context.Background(), req, resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, "application/grpc", resp.Header.Get("Content-Type"))

	// Verify Trailers
	status := resp.Header.Get("Grpc-Status")
	message := resp.Header.Get("Grpc-Message")

	assert.Equal(t, "0", status, "Grpc-Status should be 0")
	assert.Equal(t, "ok", message, "Grpc-Message should be ok")
}

// TestH2UpstreamBaseline verifies that the stdlib http.Client can talk to the test server as H2.
func TestH2UpstreamBaseline(t *testing.T) {
	ts, url := setupHTTP2Server(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Proto", r.Proto)
		w.Header().Set("Trailer", "X-Trailer")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		w.Header().Set("X-Trailer", "trailers-ready")
	})
	defer ts.Close()

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, /* #nosec G402 */
			ForceAttemptHTTP2: true,
		},
	}

	resp, err := httpClient.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "HTTP/2.0", resp.Header.Get("X-Proto"))

	_, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "trailers-ready", resp.Trailer.Get("X-Trailer"))
}

func TestProxy_HTTP2_Timeout(t *testing.T) {
	ts, url := setupHTTP2Server(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond) // Longer than client timeout
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	clientOpts := ClientOptions{
		IsHTTP2: true,
		HZOptions: append(DefaultClientOptions(),
			client.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}), /* #nosec G402 */
		),
	}
	hClient, err := NewClient(clientOpts)
	require.NoError(t, err)

	req := protocol.AcquireRequest()
	resp := protocol.AcquireResponse()
	req.SetRequestURI(url)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = hClient.Do(ctx, req, resp)
	require.Error(t, err)
}
