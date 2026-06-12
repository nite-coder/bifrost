package roundrobin_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/balancer/roundrobin"
	"github.com/nite-coder/bifrost/pkg/target"
)

func createTestEndpoint(addr string, maxFails uint, failTimeout time.Duration) *target.Endpoint {
	return &target.Endpoint{
		Address: addr,
		Weight:  1,
		State:   target.NewState(maxFails, failTimeout),
	}
}

func TestRoundRobin(t *testing.T) {
	_ = roundrobin.Init()

	t.Run("success", func(t *testing.T) {
		eps := []*target.Endpoint{
			createTestEndpoint("10.0.1.1:80", 1, time.Second),
			createTestEndpoint("10.0.1.2:80", 1, 10*time.Second),
			createTestEndpoint("10.0.1.3:80", 0, 10*time.Second),
		}
		b := roundrobin.NewBalancer(eps)

		expected := []string{"10.0.1.1:80", "10.0.1.2:80", "10.0.1.3:80"}
		for _, e := range expected {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			require.NotNil(t, ep)
			assert.Equal(t, e, ep.Address)
		}
	})

	t.Run("one endpoint failed", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, time.Second)
		ep2 := createTestEndpoint("10.0.1.2:80", 1, 10*time.Second)
		ep3 := createTestEndpoint("10.0.1.3:80", 0, 10*time.Second)

		ep1.State.RecordFailure()
		ep2.State.RecordFailure()

		eps := []*target.Endpoint{ep1, ep2, ep3}
		b := roundrobin.NewBalancer(eps)

		for range 5 {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			assert.Equal(t, "10.0.1.3:80", ep.Address, "only ep3 should be selected")
		}
	})

	t.Run("no live endpoint", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, 10*time.Second)
		ep1.State.RecordFailure()

		b := roundrobin.NewBalancer([]*target.Endpoint{ep1})
		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		require.Nil(t, ep)
	})

	t.Run("nil endpoints", func(t *testing.T) {
		b := roundrobin.NewBalancer(nil)
		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("counter overflow", func(t *testing.T) {
		eps := []*target.Endpoint{
			createTestEndpoint("10.0.1.1:80", 0, 0),
			createTestEndpoint("10.0.1.2:80", 0, 0),
		}
		b := roundrobin.NewBalancer(eps)
		b.Counter.Store(math.MaxUint64 - 1)

		ep, err := b.Select(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "10.0.1.1:80", ep.Address)
	})
}
