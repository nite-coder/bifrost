package roundrobin

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

func TestRoundRobin(t *testing.T) {
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
	time.Sleep(2 * time.Second) // wait and proxy1 should be available

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
			assert.ErrorIs(t, err, balancer.ErrNoAvailable)
			assert.Nil(t, proxy)
		}
	})
}
