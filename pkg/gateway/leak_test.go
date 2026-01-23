package gateway

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBifrostLeak_Reload(t *testing.T) {
	// Setup options with a DNS upstream which spawns a background goroutine (ticker)
	options := config.NewOptions()
	options.Servers["leak_test"] = config.ServerOptions{
		Bind: "127.0.0.1:0", // Let OS choose port
	}
	options.Services["leak_backend"] = config.ServiceOptions{
		URL: "http://example.com", // Valid domain to trigger DNS lookup
	}
	// Enable DNS provider
	options.Providers.DNS.Enabled = true
	options.Providers.DNS.Servers = []string{"8.8.8.8"}
	options.Providers.DNS.Valid = 1 * time.Second

	options.Upstreams["example.com"] = config.UpstreamOptions{
		Discovery: config.DiscoveryOptions{
			Type: "dns",
			Name: "example.com",
		},
		Targets: []config.TargetOptions{},
	}

	// Create initial Bifrost
	bifrost, err := NewBifrost(options, false)
	require.NoError(t, err)

	go bifrost.Run()
	time.Sleep(500 * time.Millisecond) // Wait for startup

	// Capture initial goroutine count
	runtime.GC()
	initialGoroutines := runtime.NumGoroutine()
	t.Logf("Initial goroutines: %d", initialGoroutines)

	currentBifrost := bifrost
	for i := 0; i < 5; i++ {
		// New Bifrost instance
		newBifrost, err := NewBifrost(options, true) // isReload = true
		require.NoError(t, err)

		// Simulate the "swap" and discard
		// Call Close which is the fix we implemented (simulating pkg.go logic)
		currentBifrost.Close()
		currentBifrost = newBifrost

		// Wait a bit for things to settle
		time.Sleep(100 * time.Millisecond)
	}

	// Force GC again
	runtime.GC()

	// Assert that we have NOT leaked goroutines (with fix)
	// We utilize assert.Eventually to wait for goroutines to settle down
	assert.Eventually(t, func() bool {
		runtime.GC()
		finalGoroutines := runtime.NumGoroutine()
		t.Logf("Current goroutines: %d", finalGoroutines)
		return finalGoroutines <= initialGoroutines+2
	}, 5*time.Second, 500*time.Millisecond, "Goroutines leaked during reload!")

	// Cleanup last instance
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	currentBifrost.ShutdownNow(ctx)
}
