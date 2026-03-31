package gateway

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/config"
)

func TestBifrostCumulativeLeak_FinalEvidence(t *testing.T) {
	numServices := 20
	baseOptions := config.NewOptions()
	baseOptions.Servers["test_server"] = config.ServerOptions{Bind: "127.0.0.1:0"}
	baseOptions.Providers.DNS.Enabled = true
	baseOptions.Providers.DNS.Servers = []string{"8.8.8.8"}
	baseOptions.Upstreams["good.upstream"] = config.UpstreamOptions{
		Discovery: config.DiscoveryOptions{Type: "dns", Name: "localhost"},
	}

	for i := 0; i < numServices; i++ {
		svcID := fmt.Sprintf("service_%d", i)
		baseOptions.Services[svcID] = config.ServiceOptions{
			ID:  svcID,
			URL: "http://good.upstream/",
		}
	}

	runtime.GC() //nolint:revive
	initialGoroutines := runtime.NumGoroutine()
	t.Logf("[Initial] Goroutines: %d", initialGoroutines)

	// --- Evidence 1: Simulate initialization failure ---
	t.Log("--- Evidence 1: Simulate initialization failure ---")
	failOptions := baseOptions
	failOptions.Services["fail_svc"] = config.ServiceOptions{URL: "://invalid"} // Deliberate error

	for i := 1; i <= 2; i++ {
		_, err := NewBifrost(failOptions, ModeNormal)
		require.Error(t, err)
		runtime.GC() //nolint:revive
		t.Logf("Try #%d (Fail) after Goroutines: %d", i, runtime.NumGoroutine())
	}

	// --- Evidence 2: Direct Proxy (URL) Reload Success ---
	t.Log("--- Evidence 2: Simulate Direct Proxy Reload Success ---")
	directOptions := config.NewOptions()
	directOptions.Servers["test_server"] = config.ServerOptions{Bind: "127.0.0.1:0"}
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("direct_%d", i)
		directOptions.Services[id] = config.ServiceOptions{ID: id, URL: "http://localhost/"}
	}

	bifrost, err := NewBifrost(directOptions, ModeNormal)
	require.NoError(t, err)

	runtime.GC() //nolint:revive
	baseline := runtime.NumGoroutine()
	t.Logf("Goroutines baseline after Direct Proxy startup: %d", baseline)

	for i := 1; i <= 2; i++ {
		newB, err := NewBifrost(directOptions, ModeReload)
		require.NoError(t, err)
		_ = bifrost.Close()
		bifrost = newB
		runtime.GC() //nolint:revive
		t.Logf("Direct Proxy Reload #%d after Goroutines: %d", i, runtime.NumGoroutine())
	}

	final := runtime.NumGoroutine()
	assert.LessOrEqual(t, final, baseline+5, "CUMULATIVE Leak detected!")
	_ = bifrost.Close()
}
