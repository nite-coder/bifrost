package roundrobin

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

func TestRoundRobin(t *testing.T) {
	_ = Init()
	proxyOptions1 := httpproxy.Options{
		Target:      "http://backend1",
		Protocol:    config.ProtocolHTTP,
		Weight:      1,
		FailTimeout: time.Second,
		MaxFails:    1,
	}
	proxy1, _ := httpproxy.New(proxyOptions1, nil)
	proxy1.Endpoint().HealthState.RecordFailure()
	assert.Eventually(t, func() bool {
		return proxy1.Endpoint().HealthState.IsAvailable()
	}, 2*time.Second, 100*time.Millisecond, "proxy1 should be available after fail timeout")

	proxyOptions2 := httpproxy.Options{
		Target:      "http://backend2",
		Protocol:    config.ProtocolHTTP,
		Weight:      1,
		FailTimeout: 10 * time.Second,
		MaxFails:    1,
	}
	proxy2, _ := httpproxy.New(proxyOptions2, nil)

	proxyOptions3 := httpproxy.Options{
		Target:      "http://backend3",
		Protocol:    config.ProtocolHTTP,
		Weight:      1,
		FailTimeout: 10 * time.Second,
		MaxFails:    0,
	}
	proxy3, _ := httpproxy.New(proxyOptions3, nil)

	proxies := []proxy.Proxy{
		proxy1,
		proxy2,
		proxy3,
	}

	b := NewBalancer(proxies)

	t.Run("success", func(t *testing.T) {
		expected := []string{"http://backend1", "http://backend2", "http://backend3"}

		for _, e := range expected {
			p, e1 := b.Select(context.Background(), nil)
			require.NotNil(t, p)
			require.NoError(t, e1)
			assert.Equal(t, e, p.Target())
		}
	})

	t.Run("one proxy failed", func(t *testing.T) {
		proxy2.Endpoint().HealthState.RecordFailure() // proxy 2 is failed
		proxy3.Endpoint().HealthState.RecordFailure() // proxy should be available because it turns off max_fail check

		expected := []string{"http://backend1", "http://backend3"}
		for _, e := range expected {
			p, e1 := b.Select(context.Background(), nil)
			require.NoError(t, e1)
			require.NotNil(t, p)
			assert.Equal(t, e, p.Target())
		}
	})

	t.Run("no live upstream", func(t *testing.T) {
		proxy1.Endpoint().HealthState.RecordFailure()
		proxy2.Endpoint().HealthState.RecordFailure()

		proxyOptions3 := httpproxy.Options{
			Target:      "http://backend3",
			Protocol:    config.ProtocolHTTP,
			Weight:      1,
			FailTimeout: 10 * time.Second,
			MaxFails:    1,
		}
		proxy3, _ := httpproxy.New(proxyOptions3, nil)
		proxy3.Endpoint().HealthState.RecordFailure()

		proxies := []proxy.Proxy{proxy1, proxy2, proxy3}
		b.proxies = proxies

		for range 6000 {
			p, e := b.Select(context.Background(), nil)
			require.ErrorIs(t, e, balancer.ErrNotAvailable)
			require.Nil(t, p)
		}
	})

	t.Run("proxies getter", func(t *testing.T) {
		assert.Len(t, b.Proxies(), 3)
	})

	t.Run("nil proxies", func(t *testing.T) {
		b2 := NewBalancer(nil)
		p, e := b2.Select(context.Background(), nil)
		require.ErrorIs(t, e, balancer.ErrNotAvailable)
		assert.Nil(t, p)
	})

	t.Run("single proxy failed", func(t *testing.T) {
		p1Options := httpproxy.Options{
			Target:      "http://backend1",
			Protocol:    config.ProtocolHTTP,
			Weight:      1,
			FailTimeout: 10 * time.Second,
			MaxFails:    1,
		}
		p1, _ := httpproxy.New(p1Options, nil)
		p1.Endpoint().HealthState.RecordFailure()

		bSingle := NewBalancer([]proxy.Proxy{p1})
		p, e := bSingle.Select(context.Background(), nil)
		require.ErrorIs(t, e, balancer.ErrNotAvailable)
		assert.Nil(t, p)
	})

	t.Run("registration", func(t *testing.T) {
		p1Options := httpproxy.Options{
			Target: "http://backend1",
		}
		p1, _ := httpproxy.New(p1Options, nil)

		factory := balancer.Factory("round_robin")
		assert.NotNil(t, factory)
		rr, err := factory([]proxy.Proxy{p1}, nil)
		require.NoError(t, err)
		assert.NotNil(t, rr)

		p, e := rr.Select(context.Background(), nil)
		require.NoError(t, e)
		assert.Equal(t, "http://backend1", p.Target())
	})

	t.Run("counter overflow", func(t *testing.T) {
		p1Options := httpproxy.Options{
			Target: "http://backend1",
		}
		p1, _ := httpproxy.New(p1Options, nil)

		p2Options := httpproxy.Options{
			Target: "http://backend2",
		}
		p2, _ := httpproxy.New(p2Options, nil)

		b := NewBalancer([]proxy.Proxy{p1, p2})

		// force counter to near max
		b.counter.Store(math.MaxUint64 - 1)

		p, e := b.Select(context.Background(), nil)
		require.NoError(t, e)
		assert.Equal(t, "http://backend1", p.Target())
		assert.Equal(t, uint64(math.MaxUint64), b.counter.Load())

		// trigger natural wrap-around to 0
		p, e = b.Select(context.Background(), nil)
		require.NoError(t, e)
		assert.Equal(t, "http://backend2", p.Target()) // index (0-1)%2 = MaxUint64%2 = 1 -> backend2
		assert.Equal(t, uint64(0), b.counter.Load())

		// next increment to 1
		p, e = b.Select(context.Background(), nil)
		require.NoError(t, e)
		assert.Equal(t, "http://backend1", p.Target()) // index (1-1)%2 = 0 -> backend1
		assert.Equal(t, uint64(1), b.counter.Load())
	})
}
