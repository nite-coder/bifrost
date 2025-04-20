package gateway

import (
	"hash/fnv"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/proxy"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
	"github.com/nite-coder/bifrost/pkg/resolver"

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

	upstream := &Upstream{
		counter: atomic.Uint64{},
	}

	upstream.proxies.Store(proxies)

	t.Run("success", func(t *testing.T) {
		expected := []string{"http://backend1", "http://backend2", "http://backend3"}
		for _, e := range expected {
			proxy := upstream.roundRobin()
			assert.NotNil(t, proxy)
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
			proxy := upstream.roundRobin()
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

		proxyies := []proxy.Proxy{proxy1, proxy2, proxy3}
		upstream.proxies.Store(proxyies)

		for i := 0; i < 6000; i++ {
			proxy := upstream.roundRobin()
			assert.Nil(t, proxy)
		}
	})
}

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

	upstream := &Upstream{
		totalWeight: 6,
	}

	upstream.proxies.Store(proxies)

	t.Run("success", func(t *testing.T) {
		hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
		for i := 0; i < 6000; i++ {
			proxy := upstream.weighted()
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
			proxy := upstream.weighted()
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
			proxy := upstream.weighted()
			assert.Nil(t, proxy)
		}
	})

}

func TestRandom(t *testing.T) {
	proxyOptions1 := httpproxy.Options{
		Target:      "http://backend1",
		Protocol:    config.ProtocolHTTP,
		Weight:      1,
		FailTimeout: 10 * time.Second,
		MaxFails:    1,
	}
	proxy1, _ := httpproxy.New(proxyOptions1, nil)

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
		MaxFails:    1,
	}
	proxy3, _ := httpproxy.New(proxyOptions3, nil)

	upstream := &Upstream{}

	proxies := []proxy.Proxy{
		proxy1,
		proxy2,
		proxy3,
	}

	upstream.proxies.Store(proxies)

	t.Run("success", func(t *testing.T) {
		hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
		for i := 0; i < 10000; i++ {
			proxy := upstream.random()
			assert.NotNil(t, proxy)
			hits[proxy.Target()]++
		}

		// Assert that each proxy was selected roughly equally
		assert.InDelta(t, 3333, hits["http://backend1"], 500)
		assert.InDelta(t, 3333, hits["http://backend2"], 500)
		assert.InDelta(t, 3333, hits["http://backend3"], 500)
	})

	t.Run("two proxy failed", func(t *testing.T) {
		_ = proxy1.AddFailedCount(100)
		_ = proxy2.AddFailedCount(100)

		hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
		for i := 0; i < 10000; i++ {
			proxy := upstream.random()
			assert.NotNil(t, proxy)
			hits[proxy.Target()]++
		}

		// Assert that each proxy was selected roughly equally
		assert.InDelta(t, 0, hits["http://backend1"], 0)
		assert.InDelta(t, 0, hits["http://backend2"], 0)
		assert.InDelta(t, 10000, hits["http://backend3"], 0)
	})

	t.Run("no live upstream", func(t *testing.T) {
		_ = proxy1.AddFailedCount(100)
		_ = proxy2.AddFailedCount(100)
		_ = proxy3.AddFailedCount(100)

		for i := 0; i < 10000; i++ {
			proxy := upstream.random()
			assert.Nil(t, proxy)
		}
	})

}

func TestHashing(t *testing.T) {
	proxyOptions1 := httpproxy.Options{
		Target:      "http://backend1",
		Protocol:    config.ProtocolHTTP,
		Weight:      1,
		FailTimeout: 10 * time.Minute,
		MaxFails:    1,
	}
	proxy1, _ := httpproxy.New(proxyOptions1, nil)

	proxyOptions2 := httpproxy.Options{
		Target:      "http://backend2",
		Protocol:    config.ProtocolHTTP,
		Weight:      1,
		FailTimeout: 10 * time.Minute,
		MaxFails:    1,
	}
	proxy2, _ := httpproxy.New(proxyOptions2, nil)

	proxyOptions3 := httpproxy.Options{
		Target:      "http://backend3",
		Protocol:    config.ProtocolHTTP,
		Weight:      1,
		FailTimeout: 10 * time.Minute,
		MaxFails:    1,
	}
	proxy3, _ := httpproxy.New(proxyOptions3, nil)

	proxies := []proxy.Proxy{
		proxy1,
		proxy2,
		proxy3,
	}

	upstream := &Upstream{
		hasher: fnv.New32a(),
	}

	upstream.proxies.Store(proxies)

	t.Run("success", func(t *testing.T) {
		keys := []string{"key1", "key2", "key3"}
		expected := map[string]string{
			"key1": "http://backend3",
			"key2": "http://backend1",
			"key3": "http://backend1",
		}

		for _, key := range keys {
			proxy := upstream.hasing(key)
			assert.NotNil(t, proxy)
			assert.Equal(t, expected[key], proxy.Target())
		}
	})

	t.Run("two proxies failed", func(t *testing.T) {
		_ = proxy1.AddFailedCount(100)
		_ = proxy2.AddFailedCount(100)

		keys := []string{"key1", "key2", "key3"}

		for _, key := range keys {
			proxy := upstream.hasing(key)
			assert.NotNil(t, proxy)
			assert.Equal(t, "http://backend3", proxy.Target())
		}
	})

	t.Run("no live upstream", func(t *testing.T) {
		_ = proxy1.AddFailedCount(100)
		_ = proxy2.AddFailedCount(100)
		_ = proxy3.AddFailedCount(100)

		keys := []string{"key1", "key2", "key3"}

		for _, key := range keys {
			proxy := upstream.hasing(key)
			assert.Nil(t, proxy)
		}
	})
}

func TestCreateUpstreamAndDnsRefresh(t *testing.T) {

	targetOptions := []config.TargetOptions{
		{
			Target: "127.0.0.1:1234",
			Weight: 1,
		},
		{
			Target: "127.0.0.2:1235",
			Weight: 1,
		},
		{
			Target: "127.0.0.3:1236",
			Weight: 1,
		},
	}

	upstreamOptions := config.UpstreamOptions{
		ID:       "test",
		Strategy: config.RoundRobinStrategy,
		Targets:  targetOptions,
	}

	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	assert.NoError(t, err)

	bifrost := &Bifrost{
		options: &config.Options{
			SkipResolver: true,
			Upstreams: map[string]config.UpstreamOptions{
				"test": upstreamOptions,
			},
		},
		resolver: dnsResolver,
	}

	testService := config.ServiceOptions{
		Protocol: config.ProtocolHTTP,
		Url:      "http://test",
	}

	upstreamMap, err := loadUpstreams(bifrost, config.ServiceOptions{})
	assert.NoError(t, err)
	assert.Len(t, upstreamMap, 1)

	upstream, err := newUpstream(
		bifrost,
		testService,
		config.UpstreamOptions{
			ID:       "test",
			Strategy: config.RoundRobinStrategy,
			Targets:  targetOptions,
		},
	)

	assert.NoError(t, err)
	assert.Len(t, upstream.proxies.Load().([]proxy.Proxy), 3)

	upstream, err = newUpstream(
		bifrost,
		config.ServiceOptions{
			Protocol: config.ProtocolGRPC,
			Url:      "http://test",
		},
		config.UpstreamOptions{
			ID:       "test",
			Strategy: config.RoundRobinStrategy,
			Targets:  targetOptions,
		},
	)

	assert.NoError(t, err)
	proxiies := upstream.proxies.Load().([]proxy.Proxy)
	assert.Len(t, proxiies, 3)
}

func TestRefreshProxies(t *testing.T) {
	t.Run("success with initial DNS instances", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{})
		assert.NoError(t, err)

		upstream := &Upstream{
			bifrost: &Bifrost{
				options: &config.Options{
					SkipResolver: true,
				},
				resolver: dnsResolver,
			},
			options: &config.UpstreamOptions{
				ID: "test",
				Discovery: config.DiscoveryOptions{
					Type: "dns",
					ID:   "test.service",
				},
			},
			ServiceOptions: &config.ServiceOptions{
				Protocol: config.ProtocolHTTP,
				Url:      "http://test.service",
			},
		}

		addr1, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
		assert.NoError(t, err)
		ins1 := provider.NewInstance(addr1, 2)

		addr2, err := net.ResolveTCPAddr("tcp", "127.0.0.2:8080")
		assert.NoError(t, err)
		ins2 := provider.NewInstance(addr2, 3)

		instances := []provider.Instancer{ins1, ins2}

		err = upstream.refreshProxies(instances)
		assert.NoError(t, err)

		plist1 := upstream.proxies.Load().([]proxy.Proxy)
		assert.Len(t, plist1, 2)
		assert.Equal(t, uint32(5), upstream.totalWeight)

		// should be no update
		err = upstream.refreshProxies(instances)
		assert.NoError(t, err)

		plist2 := upstream.proxies.Load().([]proxy.Proxy)
		assert.Len(t, plist2, 2)
		assert.Equal(t, uint32(5), upstream.totalWeight)

		assert.Equal(t, plist1[0].ID(), plist2[0].ID())
		assert.Equal(t, plist1[1].ID(), plist2[1].ID())

	})

	t.Run("fail with no instances", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{})
		assert.NoError(t, err)

		upstream := &Upstream{
			bifrost: &Bifrost{
				options: &config.Options{
					SkipResolver: true,
				},
				resolver: dnsResolver,
			},
			options: &config.UpstreamOptions{
				ID: "test",
				Discovery: config.DiscoveryOptions{
					Type: "dns",
					ID:   "test.service",
				},
			},
			ServiceOptions: &config.ServiceOptions{
				Protocol: config.ProtocolHTTP,
				Url:      "http://test.service",
			},
		}

		err = upstream.refreshProxies([]provider.Instancer{})
		assert.Error(t, err)
	})

}
