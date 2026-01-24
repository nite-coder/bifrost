package gateway

import (
	"fmt"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGRPCProxyLeak_Evidence(t *testing.T) {
	// This test aims to demonstrate that when Service is gRPC type:
	// 1. Upstream Refresh (instance update) leaks old gRPC connection goroutines.
	// 2. Bifrost reload (Close) also leaks gRPC connection goroutines.

	options := config.NewOptions()
	options.Servers["test_server"] = config.ServerOptions{
		Bind: "127.0.0.1:0",
	}

	// Configure a gRPC service
	options.Services["grpc_svc"] = config.ServiceOptions{
		ID:       "grpc_svc",
		URL:      "http://grpc-backend/",
		Protocol: config.ProtocolGRPC,
	}

	options.Upstreams["grpc-backend"] = config.UpstreamOptions{
		Discovery: config.DiscoveryOptions{
			Type: "static", // Use static to avoid background watch in NewBifrost
		},
	}

	// 1. Base line
	runtime.GC()
	initialGoroutines := runtime.NumGoroutine()
	t.Logf("Initial goroutines: %d", initialGoroutines)

	bf, err := NewBifrost(options, false)
	require.NoError(t, err)

	svc, ok := bf.Service("grpc_svc")
	require.True(t, ok)

	up := svc.Upstream()
	require.NotNil(t, up)

	// 2. Simulate multiple instance updates (Refresh)
	// Each new instance triggers grpc.NewClient
	for i := 1; i <= 3; i++ {
		addrStr := fmt.Sprintf("127.0.0.1:%d", 10000+i)
		addr, _ := net.ResolveTCPAddr("tcp", addrStr)

		// Call refreshProxies directly to simulate update without background goroutine
		err := up.refreshProxies([]provider.Instancer{
			provider.NewInstance(addr, 1),
		})
		require.NoError(t, err)

		runtime.GC()
		t.Logf("After Refresh #%d, goroutines: %d", i, runtime.NumGoroutine())
	}

	// 3. Simulate reload (Bifrost.Close)
	err = bf.Close()
	require.NoError(t, err)

	// Give some time for any async cleanup (though we expect none for leaked resources)
	time.Sleep(200 * time.Millisecond)
	runtime.GC()
	finalCount := runtime.NumGoroutine()
	t.Logf("Final goroutines after Bifrost.Close: %d", finalCount)

	// Assertion: If no leak, finalCount should return close to initialGoroutines.
	// Since gRPC starts goroutines that are never closed currently, this will fail.
	assert.LessOrEqual(t, finalCount, initialGoroutines+3, "gRPC Proxy Goroutine leak detected!")
}
