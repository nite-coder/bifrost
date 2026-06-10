package weighted

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
)

func createTestProxy(target string, weight uint32, maxFails uint, failTimeout time.Duration) proxy.Proxy {
	if weight == 0 {
		weight = 1
	}
	p, _ := httpproxy.New(httpproxy.Options{
		Target:   target,
		Protocol: config.ProtocolHTTP,
		Endpoint: &proxy.Endpoint{
			Address:     target,
			Weight:      weight,
			HealthState: proxy.NewTargetState(maxFails, failTimeout),
		},
	}, nil)
	return p
}

func TestWeighted(t *testing.T) {
	_ = Init()
	proxy1 := createTestProxy("http://backend1", 1, 10, 10*time.Second)
	proxy2 := createTestProxy("http://backend2", 2, 1, 10*time.Second)
	proxy3 := createTestProxy("http://backend3", 3, 100, 10*time.Second)

	proxies := []proxy.Proxy{
		proxy1,
		proxy2,
		proxy3,
	}

	b, _ := NewBalancer(proxies)

	t.Run("success", func(t *testing.T) {
		hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
		for range 6000 {
			proxy, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			assert.NotNil(t, proxy)
			hits[proxy.Target()]++
		}

		// Assert that weight distribution is roughly correct
		assert.InDelta(t, 1000, hits["http://backend1"], 100)
		assert.InDelta(t, 2000, hits["http://backend2"], 100)
		assert.InDelta(t, 3000, hits["http://backend3"], 100)
	})

	t.Run("one proxy failed", func(t *testing.T) {
		for range 10 {
			proxy1.Endpoint().HealthState.RecordFailure()
		}

		hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
		for range 6000 {
			proxy, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			assert.NotNil(t, proxy)
			hits[proxy.Target()]++
		}

		assert.InDelta(t, 0, hits["http://backend1"], 0)
		assert.InDelta(t, 2400, hits["http://backend2"], 150)
		assert.InDelta(t, 3600, hits["http://backend3"], 150)
	})

	t.Run("no live upstream", func(t *testing.T) {
		for range 10 {
			proxy1.Endpoint().HealthState.RecordFailure()
		}
		for range 5 {
			proxy2.Endpoint().HealthState.RecordFailure()
		}
		for range 105 {
			proxy3.Endpoint().HealthState.RecordFailure()
		}

		for range 6000 {
			proxy, err := b.Select(context.Background(), nil)
			require.ErrorIs(t, err, balancer.ErrNotAvailable)
			assert.Nil(t, proxy)
		}
	})

	t.Run("proxies getter", func(t *testing.T) {
		assert.Len(t, b.Proxies(), 3)
	})

	t.Run("nil proxies", func(t *testing.T) {
		b2, _ := NewBalancer(nil)
		p, err := b2.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		require.Nil(t, p)
	})

	t.Run("single proxy failed", func(t *testing.T) {
		p1 := createTestProxy("http://backend1", 1, 1, 10*time.Second)
		p1.Endpoint().HealthState.RecordFailure()

		bSingle, _ := NewBalancer([]proxy.Proxy{p1})
		p, err := bSingle.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		require.Nil(t, p)
	})

	t.Run("registration", func(t *testing.T) {
		p1 := createTestProxy("http://backend1", 1, 0, 0)

		factory := balancer.Factory("weighted")
		assert.NotNil(t, factory)
		rr, err := factory([]proxy.Proxy{p1}, nil)
		require.NoError(t, err)
		assert.NotNil(t, rr)

		p, err := rr.Select(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "http://backend1", p.Target())
	})

	t.Run("weight clamping and default weight", func(t *testing.T) {
		p1 := createTestProxy("http://backend1", math.MaxInt32+100, 0, 0)
		p2 := createTestProxy("http://backend2", 0, 0, 0) // should become 1 inside newProxy/NewBalancer

		bLarge, err := NewBalancer([]proxy.Proxy{p1, p2})
		require.NoError(t, err)

		p, err := bLarge.Select(context.Background(), nil)
		require.NoError(t, err)
		assert.NotNil(t, p)

		assert.GreaterOrEqual(t, bLarge.totalWeight, uint32(math.MaxInt32))
	})
}
