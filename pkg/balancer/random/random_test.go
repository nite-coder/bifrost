package random_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/balancer/random"
	"github.com/nite-coder/bifrost/pkg/target"
)

func createTestEndpoint(addr string, maxFails uint, failTimeout time.Duration) *target.Endpoint {
	return &target.Endpoint{
		Address: addr,
		Weight:  1,
		State:   target.NewState(maxFails, failTimeout),
	}
}

func TestRandom(t *testing.T) {
	_ = random.Init()

	t.Run("success", func(t *testing.T) {
		eps := []*target.Endpoint{
			createTestEndpoint("10.0.1.1:80", 1, 10*time.Second),
			createTestEndpoint("10.0.1.2:80", 1, 10*time.Second),
			createTestEndpoint("10.0.1.3:80", 1, 10*time.Second),
		}
		b := random.NewBalancer(eps)

		hits := map[string]int{}
		for range 10000 {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			hits[ep.Address]++
		}
		assert.InDelta(t, 3333, hits["10.0.1.1:80"], 500)
		assert.InDelta(t, 3333, hits["10.0.1.2:80"], 500)
		assert.InDelta(t, 3333, hits["10.0.1.3:80"], 500)
	})

	t.Run("two endpoints failed", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, 10*time.Second)
		ep2 := createTestEndpoint("10.0.1.2:80", 1, 10*time.Second)
		ep3 := createTestEndpoint("10.0.1.3:80", 1, 10*time.Second)
		ep1.State.RecordFailure()
		ep2.State.RecordFailure()

		b := random.NewBalancer([]*target.Endpoint{ep1, ep2, ep3})

		for range 100 {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			assert.Equal(t, "10.0.1.3:80", ep.Address)
		}
	})

	t.Run("no live endpoint", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, time.Second)
		ep1.State.RecordFailure()
		b := random.NewBalancer([]*target.Endpoint{ep1})
		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("nil endpoints", func(t *testing.T) {
		b := random.NewBalancer(nil)
		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("registration", func(t *testing.T) {
		factory := balancer.Factory("random")
		require.NotNil(t, factory)
		ep := createTestEndpoint("10.0.1.1:80", 0, 0)
		b, err := factory([]*target.Endpoint{ep}, nil)
		require.NoError(t, err)
		require.NotNil(t, b)
		epOut, err := b.Select(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "10.0.1.1:80", epOut.Address)
	})
}
