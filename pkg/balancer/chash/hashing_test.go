package chash_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/balancer/chash"
	"github.com/nite-coder/bifrost/pkg/target"
)

func createTestEndpoint(addr string) *target.Endpoint {
	return &target.Endpoint{
		Address: addr,
		Weight:  1,
		State:   target.NewState(1, 10*time.Minute),
	}
}

func TestHashing(t *testing.T) {
	_ = chash.Init()

	t.Run("success", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80")
		ep2 := createTestEndpoint("10.0.1.2:80")
		ep3 := createTestEndpoint("10.0.1.3:80")
		eps := []*target.Endpoint{ep1, ep2, ep3}

		keys := []string{"key1", "key2", "key3"}
		for _, key := range keys {
			params := map[string]any{"hash_on": "$var.uid"}
			b, err := chash.NewBalancer(eps, params)
			require.NoError(t, err)

			hzctx := app.NewContext(0)
			hzctx.Set("uid", key)

			epOut1, err := b.Select(context.Background(), hzctx)
			require.NoError(t, err)
			epOut2, err := b.Select(context.Background(), hzctx)
			require.NoError(t, err)
			assert.Equal(t, epOut1.Address, epOut2.Address, "same key same endpoint")
		}
	})

	t.Run("two endpoints failed", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80")
		ep2 := createTestEndpoint("10.0.1.2:80")
		ep3 := createTestEndpoint("10.0.1.3:80")
		eps := []*target.Endpoint{ep1, ep2, ep3}

		ep1.State.RecordFailure()
		ep2.State.RecordFailure()

		params := map[string]any{"hash_on": "$var.uid"}
		for _, key := range []string{"key1", "key2", "key3"} {
			b, err := chash.NewBalancer(eps, params)
			require.NoError(t, err)

			hzctx := app.NewContext(0)
			hzctx.Set("uid", key)
			ep, err := b.Select(context.Background(), hzctx)
			require.NoError(t, err)
			assert.Equal(t, "10.0.1.3:80", ep.Address)
		}
	})

	t.Run("no live endpoint", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80")
		ep2 := createTestEndpoint("10.0.1.2:80")
		ep3 := createTestEndpoint("10.0.1.3:80")
		eps := []*target.Endpoint{ep1, ep2, ep3}

		ep1.State.RecordFailure()
		ep2.State.RecordFailure()
		ep3.State.RecordFailure()

		params := map[string]any{"hash_on": "$var.uid"}
		b, err := chash.NewBalancer(eps, params)
		require.NoError(t, err)

		hzctx := app.NewContext(0)
		hzctx.Set("uid", "test")
		ep, err := b.Select(context.Background(), hzctx)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("registration error paths", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80")
		ep2 := createTestEndpoint("10.0.1.2:80")
		ep3 := createTestEndpoint("10.0.1.3:80")
		eps := []*target.Endpoint{ep1, ep2, ep3}

		factory := balancer.Factory("chash")
		require.NotNil(t, factory)

		b, err := factory(eps, nil)
		require.Error(t, err)
		assert.Nil(t, b)

		b, err = factory(eps, map[string]any{"hash_on": 123})
		require.Error(t, err)
		assert.Nil(t, b)

		b, err = factory(eps, map[string]any{"other": "val"})
		require.Error(t, err)
		assert.Nil(t, b)
	})

	t.Run("nil endpoints", func(t *testing.T) {
		b, err := chash.NewBalancer(nil, map[string]any{"hash_on": "$var.uid"})
		require.NoError(t, err)
		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("single endpoint failed", func(t *testing.T) {
		e := createTestEndpoint("10.0.1.1:80")
		e.State.RecordFailure()
		b, err := chash.NewBalancer([]*target.Endpoint{e}, map[string]any{"hash_on": "$var.uid"})
		require.NoError(t, err)

		hzctx := app.NewContext(0)
		hzctx.Set("uid", "key")
		ep, err := b.Select(context.Background(), hzctx)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("concurrency", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80")
		ep2 := createTestEndpoint("10.0.1.2:80")
		ep3 := createTestEndpoint("10.0.1.3:80")
		eps := []*target.Endpoint{ep1, ep2, ep3}

		params := map[string]any{"hash_on": "$var.uid"}
		b, err := chash.NewBalancer(eps, params)
		require.NoError(t, err)

		var wg sync.WaitGroup
		numGoroutines := 10
		operationsPerGoroutine := 100

		for range numGoroutines {
			wg.Go(func() {
				for range operationsPerGoroutine {
					hzctx := app.NewContext(0)
					hzctx.Set("uid", "some-key")
					ep, err := b.Select(context.Background(), hzctx)
					require.NoError(t, err)
					assert.NotNil(t, ep)
				}
			})
		}
		wg.Wait()
	})

	t.Run("determinism", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80")
		ep2 := createTestEndpoint("10.0.1.2:80")
		ep3 := createTestEndpoint("10.0.1.3:80")

		eps1 := []*target.Endpoint{ep1, ep2, ep3}
		eps2 := []*target.Endpoint{ep3, ep1, ep2} // Different initial order

		params := map[string]any{"hash_on": "$var.uid"}
		b1, err1 := chash.NewBalancer(eps1, params)
		b2, err2 := chash.NewBalancer(eps2, params)
		require.NoError(t, err1)
		require.NoError(t, err2)

		key := "test-determinism"
		hzctx := app.NewContext(0)
		hzctx.Set("uid", key)

		sel1, err1 := b1.Select(context.Background(), hzctx)
		sel2, err2 := b2.Select(context.Background(), hzctx)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, sel1.Address, sel2.Address, "Selection should be identical regardless of endpoint input order")
	})

	t.Run("weight-based distribution", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80")
		ep1.Weight = 20 // Weight 2
		ep2 := createTestEndpoint("10.0.1.2:80")
		ep2.Weight = 10 // Weight 1
		eps := []*target.Endpoint{ep1, ep2}

		params := map[string]any{"hash_on": "$var.uid"}
		b, err := chash.NewBalancer(eps, params)
		require.NoError(t, err)

		distribution := make(map[string]int)
		numKeys := 100000
		for i := range numKeys {
			hzctx := app.NewContext(0)
			hzctx.Set("uid", i)
			ep, err := b.Select(context.Background(), hzctx)
			require.NoError(t, err)
			distribution[ep.Address]++
		}

		h1Count := distribution["10.0.1.1:80"]
		h2Count := distribution["10.0.1.2:80"]

		t.Logf("Distribution: h1=%d, h2=%d", h1Count, h2Count)
		ratio := float64(h1Count) / float64(h2Count)
		assert.Greater(t, ratio, 1.6, "h1 should receive about 2x traffic of h2")
		assert.Less(t, ratio, 2.3, "h1 should receive about 2x traffic of h2")
	})

	t.Run("consistent failover order", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80")
		ep2 := createTestEndpoint("10.0.1.2:80")
		ep3 := createTestEndpoint("10.0.1.3:80")
		eps := []*target.Endpoint{ep1, ep2, ep3}

		params := map[string]any{"hash_on": "$var.uid"}
		b, err := chash.NewBalancer(eps, params)
		require.NoError(t, err)
		key := "failover-key"
		hzctx := app.NewContext(0)
		hzctx.Set("uid", key)

		// 1. Get the first selected endpoint
		firstEp, err := b.Select(context.Background(), hzctx)
		require.NoError(t, err)

		// 2. Fail the first endpoint, it should pick a different one
		firstEp.State.RecordFailure()
		secondEp, err := b.Select(context.Background(), hzctx)
		require.NoError(t, err)
		assert.NotEqual(t, firstEp.Address, secondEp.Address)

		// 3. Fail the second endpoint, it should pick the third one
		secondEp.State.RecordFailure()
		thirdEp, err := b.Select(context.Background(), hzctx)
		require.NoError(t, err)
		assert.NotEqual(t, firstEp.Address, thirdEp.Address)
		assert.NotEqual(t, secondEp.Address, thirdEp.Address)

		// 4. Fail the third endpoint, it should return ErrNotAvailable
		thirdEp.State.RecordFailure()
		_, err = b.Select(context.Background(), hzctx)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
	})
}
