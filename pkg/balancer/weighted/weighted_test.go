package weighted

import (
	"context"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
	"github.com/stretchr/testify/assert"
)

func TestWeighted(t *testing.T) {
	proxyOptions1 := httpproxy.Options{
		Target:      "http://backend1",
		Protocol:    config.ProtocolHTTP,
		Weight:      1,
		FailTimeout: 10 * time.Second,
		MaxFails:    10,
	}
	proxy1, _ := httpproxy.New(proxyOptions1, nil)

	proxyOptions2 := httpproxy.Options{
		Target:      "http://backend2",
		Protocol:    config.ProtocolHTTP,
		Weight:      2,
		FailTimeout: 10 * time.Second,
		MaxFails:    1,
	}
	proxy2, _ := httpproxy.New(proxyOptions2, nil)

	proxyOptions3 := httpproxy.Options{
		Target:      "http://backend3",
		Protocol:    config.ProtocolHTTP,
		Weight:      3,
		FailTimeout: 10 * time.Second,
		MaxFails:    100,
	}
	proxy3, _ := httpproxy.New(proxyOptions3, nil)

	proxies := []proxy.Proxy{
		proxy1,
		proxy2,
		proxy3,
	}

	b, _ := NewBalancer(proxies)

	t.Run("success", func(t *testing.T) {
		hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
		for i := 0; i < 6000; i++ {
			proxy, err := b.Select(context.Background(), nil)
			assert.NoError(t, err)
			assert.NotNil(t, proxy)
			hits[proxy.Target()]++
		}

		// Assert that weight distribution is roughly correct
		assert.InDelta(t, 1000, hits["http://backend1"], 100)
		assert.InDelta(t, 2000, hits["http://backend2"], 100)
		assert.InDelta(t, 3000, hits["http://backend3"], 100)
	})

	t.Run("one proxy failed", func(t *testing.T) {
		_ = proxy1.AddFailedCount(100)

		hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
		for i := 0; i < 6000; i++ {
			proxy, err := b.Select(context.Background(), nil)
			assert.NoError(t, err)
			assert.NotNil(t, proxy)
			hits[proxy.Target()]++
		}

		assert.InDelta(t, 0, hits["http://backend1"], 0)
		assert.InDelta(t, 2400, hits["http://backend2"], 100)
		assert.InDelta(t, 3600, hits["http://backend3"], 100)
	})

	t.Run("no live upstream", func(t *testing.T) {
		_ = proxy1.AddFailedCount(1000)
		_ = proxy2.AddFailedCount(1000)
		_ = proxy3.AddFailedCount(1000)

		for i := 0; i < 6000; i++ {
			proxy, err := b.Select(context.Background(), nil)
			assert.ErrorIs(t, err, balancer.ErrNotAvailable)
			assert.Nil(t, proxy)
		}
	})

}
