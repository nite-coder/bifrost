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

func TestRoundRobin(t *testing.T) {
	_ = Init()
	proxy1 := createTestProxy("http://backend1", 1, time.Second)
	proxy1.Endpoint().HealthState.RecordFailure()
	assert.Eventually(t, func() bool {
		return proxy1.Endpoint().HealthState.IsAvailable()
	}, 2*time.Second, 100*time.Millisecond, "proxy1 should be available after fail timeout")

	proxy2 := createTestProxy("http://backend2", 1, 10*time.Second)
	proxy3 := createTestProxy("http://backend3", 0, 10*time.Second)

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

		proxy3 := createTestProxy("http://backend3", 1, 10*time.Second)
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
		p1 := createTestProxy("http://backend1", 1, 10*time.Second)
		p1.Endpoint().HealthState.RecordFailure()

		bSingle := NewBalancer([]proxy.Proxy{p1})
		p, e := bSingle.Select(context.Background(), nil)
		require.ErrorIs(t, e, balancer.ErrNotAvailable)
		assert.Nil(t, p)
	})

	t.Run("registration", func(t *testing.T) {
		p1 := createTestProxy("http://backend1", 0, 0)

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
		p1 := createTestProxy("http://backend1", 0, 0)
		p2 := createTestProxy("http://backend2", 0, 0)

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
