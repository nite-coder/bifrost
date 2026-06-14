package gateway

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/proxy"
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
	runtime.GC() //nolint:revive // explicit GC for memory leak verification
	initialGoroutines := runtime.NumGoroutine()
	t.Logf("Initial goroutines: %d", initialGoroutines)

	bf, err := NewBifrost(options, ModeNormal)
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

		err = up.refreshEndpoints([]provider.DiscoveryResult{
			{Target: addrStr, Nodes: []provider.Instancer{provider.NewInstance(addr, 1)}},
		})
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			if svc.upstream == nil {
				return false
			}
			bal := svc.upstream.Balancer()
			if bal == nil {
				return false
			}
			var found bool
			svc.proxyByAddress.Range(func(_, v any) bool {
				if p, ok := v.(proxy.Proxy); ok && strings.Contains(p.Target(), addrStr) {
					found = true
					return false
				}
				return true
			})
			return found
		}, time.Second, 5*time.Millisecond)

		runtime.GC() //nolint:revive // explicit GC for memory leak verification
		t.Logf("After Refresh #%d, goroutines: %d", i, runtime.NumGoroutine())
	}

	// 3. Simulate reload (Bifrost.Close)
	err = bf.Close()
	require.NoError(t, err)

	// Give some time for any async cleanup (though we expect none for leaked resources) and assert
	assert.Eventually(t, func() bool {
		runtime.GC() // explicit GC for memory leak verification
		finalCount := runtime.NumGoroutine()
		t.Logf("Checking final goroutines after Bifrost.Close: %d (initial: %d)", finalCount, initialGoroutines)
		return finalCount <= initialGoroutines+3
	}, time.Second, 10*time.Millisecond, "gRPC Proxy Goroutine leak detected!")
}
