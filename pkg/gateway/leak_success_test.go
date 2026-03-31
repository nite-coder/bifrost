package gateway

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/config"
)

func TestBifrostLeakInSuccessPath_Evidence(t *testing.T) {
	numServices := 10
	options := config.NewOptions()
	options.Servers["test_server"] = config.ServerOptions{
		Bind: "127.0.0.1:0",
	}

	options.Providers.DNS.Enabled = true
	options.Providers.DNS.Servers = []string{"8.8.8.8"}

	for i := 0; i < numServices; i++ {
		svcID := fmt.Sprintf("direct_%d", i)
		options.Services[svcID] = config.ServiceOptions{
			ID:  svcID,
			URL: "http://localhost/",
		}
	}

	runtime.GC() //nolint:revive //nolint:revive
	// initialGoroutines := runtime.NumGoroutine() // removed unused

	currentBifrost, err := NewBifrost(options, ModeNormal)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)
	runtime.GC() //nolint:revive //nolint:revive
	baseline := runtime.NumGoroutine()
	t.Logf("Baseline goroutines: %d", baseline)

	for i := 1; i <= 3; i++ {
		newB, err := NewBifrost(options, ModeReload)
		require.NoError(t, err)

		_ = currentBifrost.Close()
		currentBifrost = newB

		time.Sleep(1 * time.Second)
		runtime.GC() //nolint:revive //nolint:revive
		t.Logf("Reload #%d - Goroutines: %d", i, runtime.NumGoroutine())
	}

	final := runtime.NumGoroutine()
	if final > baseline+5 {
		t.Log("Leak detected! Outputting goroutine stack trace...")
		buf := make([]byte, 1<<20)
		bufLen := runtime.Stack(buf, true)
		_, _ = os.Stderr.Write(buf[:bufLen])
	}

	assert.LessOrEqual(t, final, baseline+5, "CUMULATIVE Leak detected in SUCCESS path!")

	_ = currentBifrost.Close()
}
