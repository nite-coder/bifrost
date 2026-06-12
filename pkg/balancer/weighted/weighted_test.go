package weighted_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/balancer/weighted"
	"github.com/nite-coder/bifrost/pkg/target"
)

func createTestEndpoint(addr string, weight uint32, maxFails uint, failTimeout time.Duration) *target.Endpoint {
	return &target.Endpoint{
		Address: addr,
		Weight:  weight,
		State:   target.NewState(maxFails, failTimeout),
	}
}

func TestWeighted(t *testing.T) {
	_ = weighted.Init()

	t.Run("success", func(t *testing.T) {
		eps := []*target.Endpoint{
			createTestEndpoint("10.0.1.1:80", 1, 10, 10*time.Second),
			createTestEndpoint("10.0.1.2:80", 2, 1, 10*time.Second),
			createTestEndpoint("10.0.1.3:80", 3, 100, 10*time.Second),
		}
		b, err := weighted.NewBalancer(eps)
		require.NoError(t, err)

		hits := map[string]int{}
		for range 6000 {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			hits[ep.Address]++
		}
		assert.InDelta(t, 1000, hits["10.0.1.1:80"], 200)
		assert.InDelta(t, 2000, hits["10.0.1.2:80"], 200)
		assert.InDelta(t, 3000, hits["10.0.1.3:80"], 200)
	})

	t.Run("one endpoint failed", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, 10, 10*time.Second)
		ep2 := createTestEndpoint("10.0.1.2:80", 2, 1, 10*time.Second)
		ep3 := createTestEndpoint("10.0.1.3:80", 3, 100, 10*time.Second)
		for range 10 {
			ep1.State.RecordFailure()
		}
		eps := []*target.Endpoint{ep1, ep2, ep3}

		b, err := weighted.NewBalancer(eps)
		require.NoError(t, err)

		hits := map[string]int{}
		for range 6000 {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			hits[ep.Address]++
		}
		assert.Equal(t, 0, hits["10.0.1.1:80"])
		assert.InDelta(t, 2400, hits["10.0.1.2:80"], 200)
		assert.InDelta(t, 3600, hits["10.0.1.3:80"], 200)
	})

	t.Run("no live endpoint", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, 1, time.Second)
		ep1.State.RecordFailure()
		b, err := weighted.NewBalancer([]*target.Endpoint{ep1})
		require.NoError(t, err)

		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("nil endpoints", func(t *testing.T) {
		b, err := weighted.NewBalancer(nil)
		require.NoError(t, err)

		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("registration", func(t *testing.T) {
		factory := balancer.Factory("weighted")
		require.NotNil(t, factory)
		ep := createTestEndpoint("10.0.1.1:80", 1, 0, 0)
		b, err := factory([]*target.Endpoint{ep}, nil)
		require.NoError(t, err)
		require.NotNil(t, b)

		epOut, err := b.Select(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "10.0.1.1:80", epOut.Address)
	})
}
