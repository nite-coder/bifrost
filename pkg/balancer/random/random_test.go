package random

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
)

func createTestProxy(target string, maxFails uint, failTimeout time.Duration) proxy.Proxy {
	p, _ := httpproxy.New(httpproxy.Options{
		Target:   target,
		Protocol: config.ProtocolHTTP,
		Endpoint: &proxy.Endpoint{
			Address:     target,
			Weight:      1,
			HealthState: proxy.NewTargetState(maxFails, failTimeout),
		},
	}, nil)
	return p
}

func TestRandom(t *testing.T) {
	_ = Init()
	proxy1 := createTestProxy("http://backend1", 1, 10*time.Second)
	proxy2 := createTestProxy("http://backend2", 1, 10*time.Second)
	proxy3 := createTestProxy("http://backend3", 1, 10*time.Second)

	proxies := []proxy.Proxy{
		proxy1,
		proxy2,
		proxy3,
	}

	b := NewBalancer(proxies)

	t.Run("success", func(t *testing.T) {
		hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
		for range 10000 {
			proxy, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			assert.NotNil(t, proxy)
			hits[proxy.Target()]++
		}

		// Assert that each proxy was selected roughly equally
		assert.InDelta(t, 3333, hits["http://backend1"], 500)
		assert.InDelta(t, 3333, hits["http://backend2"], 500)
		assert.InDelta(t, 3333, hits["http://backend3"], 500)
	})

	t.Run("two proxy failed", func(t *testing.T) {
		proxy1.Endpoint().HealthState.RecordFailure()
		proxy2.Endpoint().HealthState.RecordFailure()

		hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
		for range 10000 {
			proxy, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			assert.NotNil(t, proxy)
			hits[proxy.Target()]++
		}

		// Assert that each proxy was selected roughly equally
		assert.InDelta(t, 0, hits["http://backend1"], 0)
		assert.InDelta(t, 0, hits["http://backend2"], 0)
		assert.InDelta(t, 10000, hits["http://backend3"], 0)
	})

	t.Run("no live upstream", func(t *testing.T) {
		proxy1.Endpoint().HealthState.RecordFailure()
		proxy2.Endpoint().HealthState.RecordFailure()
		proxy3.Endpoint().HealthState.RecordFailure()

		for range 10000 {
			proxy, err := b.Select(context.Background(), nil)
			require.ErrorIs(t, err, balancer.ErrNotAvailable)
			assert.Nil(t, proxy)
		}
	})

	t.Run("proxies getter", func(t *testing.T) {
		assert.Len(t, b.Proxies(), 3)
	})

	t.Run("nil proxies", func(t *testing.T) {
		b2 := NewBalancer(nil)
		p, err := b2.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		require.Nil(t, p)
	})

	t.Run("single proxy failed", func(t *testing.T) {
		p1 := createTestProxy("http://backend1", 1, 10*time.Second)
		p1.Endpoint().HealthState.RecordFailure()

		bSingle := NewBalancer([]proxy.Proxy{p1})
		p, err := bSingle.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, p)
	})

	t.Run("registration", func(t *testing.T) {
		p1 := createTestProxy("http://backend1", 0, 0)

		factory := balancer.Factory("random")
		assert.NotNil(t, factory)
		rr, err := factory([]proxy.Proxy{p1}, nil)
		require.NoError(t, err)
		assert.NotNil(t, rr)

		p, err := rr.Select(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "http://backend1", p.Target())
	})
}
