package gateway

import (
	"hash/fnv"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRoundRobin(t *testing.T) {
	proxy1, _ := newReverseProxy("http://backend1", false, 1)
	proxy2, _ := newReverseProxy("http://backend2", false, 1)
	proxy3, _ := newReverseProxy("http://backend3", false, 1)

	upstream := &Upstream{
		proxies: []*Proxy{
			proxy1,
			proxy2,
			proxy3,
		},
		counter: atomic.Uint64{},
	}

	expected := []string{"http://backend1", "http://backend2", "http://backend3"}
	for _, e := range expected {
		proxy := upstream.roundRobin()
		assert.NotNil(t, proxy)
		assert.Equal(t, e, proxy.target)
	}
}

func TestWeighted(t *testing.T) {
	proxy1, _ := newReverseProxy("http://backend1", false, 1)
	proxy2, _ := newReverseProxy("http://backend2", false, 2)
	proxy3, _ := newReverseProxy("http://backend3", false, 3)

	upstream := &Upstream{
		proxies: []*Proxy{
			proxy1,
			proxy2,
			proxy3,
		},
		totalWeight: 6,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
	for i := 0; i < 6000; i++ {
		proxy := upstream.weighted()
		assert.NotNil(t, proxy)
		hits[proxy.target]++
	}

	// Assert that weight distribution is roughly correct
	assert.InDelta(t, 1000, hits["http://backend1"], 100)
	assert.InDelta(t, 2000, hits["http://backend2"], 100)
	assert.InDelta(t, 3000, hits["http://backend3"], 100)
}

func TestRandom(t *testing.T) {
	proxy1, _ := newReverseProxy("http://backend1", false, 1)
	proxy2, _ := newReverseProxy("http://backend2", false, 1)
	proxy3, _ := newReverseProxy("http://backend3", false, 1)

	upstream := &Upstream{
		proxies: []*Proxy{
			proxy1,
			proxy2,
			proxy3,
		},
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	hits := map[string]int{"http://backend1": 0, "http://backend2": 0, "http://backend3": 0}
	for i := 0; i < 10000; i++ {
		proxy := upstream.random()
		assert.NotNil(t, proxy)
		hits[proxy.target]++
	}

	// Assert that each proxy was selected roughly equally
	assert.InDelta(t, 3333, hits["http://backend1"], 500)
	assert.InDelta(t, 3333, hits["http://backend2"], 500)
	assert.InDelta(t, 3333, hits["http://backend3"], 500)
}

func TestHashing(t *testing.T) {
	proxy1, _ := newReverseProxy("http://backend1", false, 1)
	proxy2, _ := newReverseProxy("http://backend2", false, 1)
	proxy3, _ := newReverseProxy("http://backend3", false, 1)

	upstream := &Upstream{
		proxies: []*Proxy{
			proxy1,
			proxy2,
			proxy3,
		},
		hasher: fnv.New32a(),
	}

	keys := []string{"key1", "key2", "key3"}
	expected := map[string]string{
		"key1": "http://backend3",
		"key2": "http://backend1",
		"key3": "http://backend1",
	}

	for _, key := range keys {
		proxy := upstream.hasing(key)
		assert.NotNil(t, proxy)
		assert.Equal(t, expected[key], proxy.target)
	}
}
