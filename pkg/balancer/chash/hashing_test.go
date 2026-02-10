package chash

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashing(t *testing.T) {
	_ = Init()
	proxyOptions1 := httpproxy.Options{
		Target:      "http://backend1",
		Protocol:    config.ProtocolHTTP,
		FailTimeout: 10 * time.Minute,
		MaxFails:    1,
	}
	proxy1, _ := httpproxy.New(proxyOptions1, nil)

	proxyOptions2 := httpproxy.Options{
		Target:      "http://backend2",
		Protocol:    config.ProtocolHTTP,
		FailTimeout: 10 * time.Minute,
		MaxFails:    1,
	}
	proxy2, _ := httpproxy.New(proxyOptions2, nil)

	proxyOptions3 := httpproxy.Options{
		Target:      "http://backend3",
		Protocol:    config.ProtocolHTTP,
		FailTimeout: 10 * time.Minute,
		MaxFails:    1,
	}
	proxy3, _ := httpproxy.New(proxyOptions3, nil)

	proxies := []proxy.Proxy{
		proxy1,
		proxy2,
		proxy3,
	}

	t.Run("success", func(t *testing.T) {
		keys := []string{"key1", "key2", "key3"}

		for _, key := range keys {
			factory := balancer.Factory("hashing")
			params := map[string]any{"hash_on": "$var.uid"}

			b, err := factory(proxies, params)
			assert.NoError(t, err)

			hzctx := app.NewContext(0)
			hzctx.Set("uid", key)

			// Call multiple times and ensure it always returns the same proxy
			p1, err := b.Select(context.Background(), hzctx)
			assert.NoError(t, err)
			assert.NotNil(t, p1)

			p2, err := b.Select(context.Background(), hzctx)
			assert.NoError(t, err)
			assert.Equal(t, p1.Target(), p2.Target())
		}
	})

	t.Run("concurrency", func(t *testing.T) {
		b := NewBalancer(proxies, "$var.uid", defaultReplicas)
		ctx := context.Background()

		// Run 100 goroutines to call Select concurrently
		for i := range 100 {
			t.Run(fmt.Sprintf("worker-%d", i), func(t *testing.T) {
				t.Parallel()
				for j := 0; j < 100; j++ {
					hzctx := app.NewContext(0)
					hzctx.Set("uid", "some-key")
					p, err := b.Select(ctx, hzctx)
					assert.NoError(t, err)
					assert.NotNil(t, p)
				}
			})
		}
	})

	t.Run("two proxies failed", func(t *testing.T) {
		_ = proxy1.AddFailedCount(100)
		_ = proxy2.AddFailedCount(100)

		keys := []string{"key1", "key2", "key3"}

		for _, key := range keys {
			b := NewBalancer(proxies, "$var.uid", defaultReplicas)
			hzctx := app.NewContext(0)
			hzctx.Set("uid", key)
			proxy, err := b.Select(context.Background(), hzctx)
			assert.NoError(t, err)
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
			b := NewBalancer(proxies, "$var.uid", defaultReplicas)
			hzctx := app.NewContext(0)
			hzctx.Set("uid", key)
			proxy, err := b.Select(context.Background(), hzctx)
			assert.ErrorIs(t, err, balancer.ErrNotAvailable)
			assert.Nil(t, proxy)
		}
	})

	t.Run("registration error paths", func(t *testing.T) {
		factory := balancer.Factory("hashing")
		assert.NotNil(t, factory)

		// nil params
		b, err := factory(proxies, nil)
		assert.Error(t, err)
		assert.Nil(t, b)

		// invalid hash_on type
		b, err = factory(proxies, map[string]any{"hash_on": 123})
		assert.Error(t, err)
		assert.Nil(t, b)

		// missing hash_on
		b, err = factory(proxies, map[string]any{"other": "val"})
		assert.Error(t, err)
		assert.Nil(t, b)
	})

	t.Run("proxies getter", func(t *testing.T) {
		b := NewBalancer(proxies, "$var.uid", defaultReplicas)
		assert.Equal(t, 3, len(b.Proxies()))
	})

	t.Run("nil proxies", func(t *testing.T) {
		b := NewBalancer(nil, "$var.uid", defaultReplicas)
		p, err := b.Select(context.Background(), nil)
		assert.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, p)
	})

	t.Run("single proxy failed", func(t *testing.T) {
		p1Options := httpproxy.Options{
			Target:      "http://backend1",
			Protocol:    config.ProtocolHTTP,
			FailTimeout: 10 * time.Minute,
			MaxFails:    1,
		}
		p1, _ := httpproxy.New(p1Options, nil)
		_ = p1.AddFailedCount(1)

		b := NewBalancer([]proxy.Proxy{p1}, "$var.uid", defaultReplicas)
		p, err := b.Select(context.Background(), nil)
		assert.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, p)
	})

	t.Run("consistency check", func(t *testing.T) {
		// 1. Create 3 proxies
		p1, _ := httpproxy.New(httpproxy.Options{Target: "http://h1"}, nil)
		p2, _ := httpproxy.New(httpproxy.Options{Target: "http://h2"}, nil)
		p3, _ := httpproxy.New(httpproxy.Options{Target: "http://h3"}, nil)
		proxies := []proxy.Proxy{p1, p2, p3}

		b := NewBalancer(proxies, "$var.uid", defaultReplicas)

		// 2. Map 1000 keys and record their assignments
		assignments := make(map[int]string)
		for i := 0; i < 1000; i++ {
			hzctx := app.NewContext(0)
			hzctx.Set("uid", i)
			p, _ := b.Select(context.Background(), hzctx)
			assignments[i] = p.Target()
		}

		// 3. Mark p1 as failed (removed from pool)
		_ = p1.AddFailedCount(100)

		// 4. Map the same 1000 keys again and see how many moved
		changedOnAliveNodes := 0
		for i := 0; i < 1000; i++ {
			hzctx := app.NewContext(0)
			hzctx.Set("uid", i)
			p, _ := b.Select(context.Background(), hzctx)

			oldTarget := assignments[i]
			newTarget := p.Target()

			// A key should stay on the SAME node if its previous node is still alive.
			// Only keys that were on the failed node (h1) should move.
			if oldTarget != "http://h1" && oldTarget != newTarget {
				changedOnAliveNodes++
			}
		}

		assert.Equal(t, 0, changedOnAliveNodes)
		t.Logf("Redistribution count for alive nodes: %d / 1000", changedOnAliveNodes)
	})

	t.Run("weight-based distribution", func(t *testing.T) {
		// 1. Create 2 proxies with different weights (2:1)
		p1, _ := httpproxy.New(httpproxy.Options{Target: "http://h1", Weight: 20}, nil) // Weight 2
		p2, _ := httpproxy.New(httpproxy.Options{Target: "http://h2", Weight: 10}, nil) // Weight 1
		proxies := []proxy.Proxy{p1, p2}

		// Use a fixed replicas to make it predictable
		b := NewBalancer(proxies, "$var.uid", 160)

		// 2. Map 100,000 keys and record their assignments
		distribution := make(map[string]int)
		numKeys := 100000
		for i := 0; i < numKeys; i++ {
			hzctx := app.NewContext(0)
			hzctx.Set("uid", i)
			p, _ := b.Select(context.Background(), hzctx)
			distribution[p.Target()]++
		}

		// 3. Verify distribution matches weights (approx 2:1)
		// Expected: h1 (66.6%), h2 (33.3%)
		h1Count := distribution["http://h1"]
		h2Count := distribution["http://h2"]

		t.Logf("Distribution: h1=%d, h2=%d", h1Count, h2Count)

		// Check if h1 receives approximately 2x the traffic of h2
		ratio := float64(h1Count) / float64(h2Count)
		t.Logf("Measured Ratio (h1/h2): %.4f (Expected around 2.0)", ratio)

		// Tolerance: allow some deviation due to hashing distribution, but should be close to 2.0
		assert.Greater(t, ratio, 1.8, "h1 should receive about 2x traffic of h2")
		assert.Less(t, ratio, 2.2, "h1 should receive about 2x traffic of h2")
	})

	t.Run("consistent failover order", func(t *testing.T) {
		p1, _ := httpproxy.New(httpproxy.Options{Target: "http://h1", MaxFails: 1, FailTimeout: 10 * time.Minute}, nil)
		p2, _ := httpproxy.New(httpproxy.Options{Target: "http://h2", MaxFails: 1, FailTimeout: 10 * time.Minute}, nil)
		p3, _ := httpproxy.New(httpproxy.Options{Target: "http://h3", MaxFails: 1, FailTimeout: 10 * time.Minute}, nil)
		proxies := []proxy.Proxy{p1, p2, p3}

		b := NewBalancer(proxies, "$var.uid", 160)
		key := "failover-key"
		hzctx := app.NewContext(0)
		hzctx.Set("uid", key)

		// 1. Get the candidate order from the ring
		candidates, err := b.ring.GetN(key, 3)
		require.NoError(t, err)
		require.Equal(t, 3, len(candidates))

		// 2. Initial selection should be the first candidate
		p, err := b.Select(context.Background(), hzctx)
		require.NoError(t, err)
		assert.Equal(t, candidates[0], p.ID(), "Should pick the first candidate on the ring")

		// 3. Fail the first candidate, should pick the second one from the ring in clockwise order
		_ = b.nodeMap[candidates[0]].AddFailedCount(1)
		p, err = b.Select(context.Background(), hzctx)
		require.NoError(t, err)
		assert.Equal(t, candidates[1], p.ID(), "Failover should pick the second candidate on the ring")

		// 4. Fail the second candidate, should pick the third one from the ring
		_ = b.nodeMap[candidates[1]].AddFailedCount(1)
		p, err = b.Select(context.Background(), hzctx)
		require.NoError(t, err)
		assert.Equal(t, candidates[2], p.ID(), "Failover should pick the third candidate on the ring")
	})
}
