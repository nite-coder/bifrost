package chash_test

import (
	"context"
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

	ep1 := createTestEndpoint("10.0.1.1:80")
	ep2 := createTestEndpoint("10.0.1.2:80")
	ep3 := createTestEndpoint("10.0.1.3:80")
	eps := []*target.Endpoint{ep1, ep2, ep3}

	t.Run("success", func(t *testing.T) {
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
		factory := balancer.Factory("hashing")
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
}
