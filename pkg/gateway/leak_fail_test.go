package gateway

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	runtime.GC()
	initialGoroutines := runtime.NumGoroutine()
	t.Logf("[Initial] Goroutines: %d", initialGoroutines)

	// --- 證據 1: 模擬初始化失敗 ---
	t.Log("--- 證據 1: 模擬初始化失敗 ---")
	failOptions := baseOptions
	failOptions.Services["fail_svc"] = config.ServiceOptions{URL: "://invalid"} // 故意出錯

	for i := 1; i <= 2; i++ {
		_, err := NewBifrost(failOptions, false)
		require.Error(t, err)
		runtime.GC()
		t.Logf("嘗試 #%d (失敗) 後 Goroutines: %d", i, runtime.NumGoroutine())
	}

	// --- 證據 2: Direct Proxy (URL) 成功重載 ---
	t.Log("--- 證據 2: 模擬 Direct Proxy 成功重載 ---")
	directOptions := config.NewOptions()
	directOptions.Servers["test_server"] = config.ServerOptions{Bind: "127.0.0.1:0"}
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("direct_%d", i)
		directOptions.Services[id] = config.ServiceOptions{ID: id, URL: "http://localhost/"}
	}

	bifrost, err := NewBifrost(directOptions, false)
	require.NoError(t, err)

	runtime.GC()
	baseline := runtime.NumGoroutine()
	t.Logf("使用 Direct Proxy 啟動後 baseline: %d", baseline)

	for i := 1; i <= 2; i++ {
		newB, err := NewBifrost(directOptions, true)
		require.NoError(t, err)
		_ = bifrost.Close()
		bifrost = newB
		runtime.GC()
		t.Logf("Direct Proxy 重載 #%d 後 Goroutines: %d", i, runtime.NumGoroutine())
	}

	final := runtime.NumGoroutine()
	assert.LessOrEqual(t, final, baseline+5, "CUMULATIVE Leak detected!")
	_ = bifrost.Close()
}
