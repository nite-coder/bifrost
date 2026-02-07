package roundrobin

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
	"github.com/stretchr/testify/assert"
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
	err := proxy1.AddFailedCount(1)
	assert.ErrorIs(t, err, proxy.ErrMaxFailedCount)
	assert.Eventually(t, func() bool {
		return proxy1.IsAvailable()
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
			proxy, err := b.Select(context.Background(), nil)
			assert.NotNil(t, proxy)
			assert.NoError(t, err)
			assert.Equal(t, e, proxy.Target())
		}
	})

	t.Run("one proxy failed", func(t *testing.T) {
		err = proxy2.AddFailedCount(1) // proxy 2 is failed
		assert.ErrorIs(t, err, proxy.ErrMaxFailedCount)
		err = proxy3.AddFailedCount(1000) // proxy should be available because it turns off max_fail check
		assert.NoError(t, err)

		expected := []string{"http://backend1", "http://backend3"}
		for _, e := range expected {
			proxy, err := b.Select(context.Background(), nil)
			assert.NoError(t, err)
			assert.NotNil(t, proxy)
			assert.Equal(t, e, proxy.Target())
		}
	})

	t.Run("no live upstream", func(t *testing.T) {
		err = proxy1.AddFailedCount(100)
		assert.ErrorIs(t, err, proxy.ErrMaxFailedCount)
		err = proxy2.AddFailedCount(100)
		assert.ErrorIs(t, err, proxy.ErrMaxFailedCount)

		proxyOptions3 := httpproxy.Options{
			Target:      "http://backend3",
			Protocol:    config.ProtocolHTTP,
			Weight:      1,
			FailTimeout: 10 * time.Second,
			MaxFails:    1,
		}
		proxy3, _ := httpproxy.New(proxyOptions3, nil)
		err = proxy3.AddFailedCount(100)
		assert.ErrorIs(t, err, proxy.ErrMaxFailedCount)

		proxies := []proxy.Proxy{proxy1, proxy2, proxy3}
		b.proxies = proxies

		for i := 0; i < 6000; i++ {
			proxy, err := b.Select(context.Background(), nil)
			assert.ErrorIs(t, err, balancer.ErrNotAvailable)
			assert.Nil(t, proxy)
		}
	})

	t.Run("proxies getter", func(t *testing.T) {
		assert.Equal(t, 3, len(b.Proxies()))
	})

	t.Run("nil proxies", func(t *testing.T) {
		b2 := NewBalancer(nil)
		p, err := b2.Select(context.Background(), nil)
		assert.ErrorIs(t, err, balancer.ErrNotAvailable)
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
		_ = p1.AddFailedCount(1)

		bSingle := NewBalancer([]proxy.Proxy{p1})
		p, err := bSingle.Select(context.Background(), nil)
		assert.ErrorIs(t, err, balancer.ErrNotAvailable)
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
		assert.NoError(t, err)
		assert.NotNil(t, rr)

		p, err := rr.Select(context.Background(), nil)
		assert.NoError(t, err)
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

		p, err := b.Select(context.Background(), nil)
		assert.NoError(t, err)
		assert.Equal(t, "http://backend1", p.Target())
		assert.Equal(t, uint64(math.MaxUint64), b.counter.Load())

		// trigger natural wrap-around to 0
		p, err = b.Select(context.Background(), nil)
		assert.NoError(t, err)
		assert.Equal(t, "http://backend2", p.Target()) // index (0-1)%2 = MaxUint64%2 = 1 -> backend2
		assert.Equal(t, uint64(0), b.counter.Load())

		// next increment to 1
		p, err = b.Select(context.Background(), nil)
		assert.NoError(t, err)
		assert.Equal(t, "http://backend1", p.Target()) // index (1-1)%2 = 0 -> backend1
		assert.Equal(t, uint64(1), b.counter.Load())
	})
}
