package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKeepAlive(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		k := NewKeepAlive(nil)
		assert.Equal(t, 1*time.Second, k.options.InitialBackoff)
		assert.Equal(t, 32*time.Second, k.options.MaxBackoff)
		assert.Equal(t, 5, k.options.MaxRestartsPerMinute)
		assert.Equal(t, 1*time.Second, k.currentBackoff)
	})

	t.Run("custom options", func(t *testing.T) {
		opts := &KeepAliveOptions{
			InitialBackoff:       2 * time.Second,
			MaxBackoff:           16 * time.Second,
			MaxRestartsPerMinute: 3,
		}
		k := NewKeepAlive(opts)
		assert.Equal(t, 2*time.Second, k.options.InitialBackoff)
		assert.Equal(t, 16*time.Second, k.options.MaxBackoff)
		assert.Equal(t, 3, k.options.MaxRestartsPerMinute)
	})

	t.Run("invalid options use defaults", func(t *testing.T) {
		opts := &KeepAliveOptions{
			InitialBackoff:       -1,
			MaxBackoff:           0,
			MaxRestartsPerMinute: -5,
		}
		k := NewKeepAlive(opts)
		assert.Equal(t, 1*time.Second, k.options.InitialBackoff)
		assert.Equal(t, 32*time.Second, k.options.MaxBackoff)
		assert.Equal(t, 5, k.options.MaxRestartsPerMinute)
	})
}

func TestKeepAlive_ShouldRestart(t *testing.T) {
	t.Run("allows restart when under limit", func(t *testing.T) {
		k := NewKeepAlive(&KeepAliveOptions{
			InitialBackoff:       1 * time.Second,
			MaxRestartsPerMinute: 5,
		})

		shouldRestart, backoff, err := k.ShouldRestart()
		assert.True(t, shouldRestart)
		assert.Equal(t, 1*time.Second, backoff)
		assert.NoError(t, err)
	})

	t.Run("returns error when limit exceeded", func(t *testing.T) {
		k := NewKeepAlive(&KeepAliveOptions{
			InitialBackoff:       1 * time.Second,
			MaxRestartsPerMinute: 2,
		})

		// Record 2 restarts
		k.RecordRestart()
		k.RecordRestart()

		shouldRestart, _, err := k.ShouldRestart()
		assert.False(t, shouldRestart)
		assert.ErrorIs(t, err, ErrRestartLimitExceeded)
	})
}

func TestKeepAlive_RecordRestart(t *testing.T) {
	t.Run("doubles backoff on each restart", func(t *testing.T) {
		k := NewKeepAlive(&KeepAliveOptions{
			InitialBackoff: 1 * time.Second,
			MaxBackoff:     32 * time.Second,
		})

		assert.Equal(t, 1*time.Second, k.CurrentBackoff())

		k.RecordRestart()
		assert.Equal(t, 2*time.Second, k.CurrentBackoff())

		k.RecordRestart()
		assert.Equal(t, 4*time.Second, k.CurrentBackoff())

		k.RecordRestart()
		assert.Equal(t, 8*time.Second, k.CurrentBackoff())
	})

	t.Run("backoff capped at max", func(t *testing.T) {
		k := NewKeepAlive(&KeepAliveOptions{
			InitialBackoff: 8 * time.Second,
			MaxBackoff:     16 * time.Second,
		})

		k.RecordRestart() // 16s
		k.RecordRestart() // still 16s (capped)
		k.RecordRestart() // still 16s (capped)

		assert.Equal(t, 16*time.Second, k.CurrentBackoff())
	})
}

func TestKeepAlive_Reset(t *testing.T) {
	k := NewKeepAlive(&KeepAliveOptions{
		InitialBackoff:       1 * time.Second,
		MaxRestartsPerMinute: 10,
	})

	// Record several restarts
	k.RecordRestart()
	k.RecordRestart()
	k.RecordRestart()

	assert.Equal(t, 8*time.Second, k.CurrentBackoff())
	assert.Equal(t, 3, k.RestartsInLastMinute())

	// Reset
	k.Reset()

	assert.Equal(t, 1*time.Second, k.CurrentBackoff())
	assert.Equal(t, 0, k.RestartsInLastMinute())
}

func TestKeepAlive_RestartsInLastMinute(t *testing.T) {
	t.Run("counts recent restarts", func(t *testing.T) {
		k := NewKeepAlive(&KeepAliveOptions{
			MaxRestartsPerMinute: 10,
		})

		assert.Equal(t, 0, k.RestartsInLastMinute())

		k.RecordRestart()
		assert.Equal(t, 1, k.RestartsInLastMinute())

		k.RecordRestart()
		k.RecordRestart()
		assert.Equal(t, 3, k.RestartsInLastMinute())
	})
}

func TestKeepAlive_PruneOldRestarts(t *testing.T) {
	t.Run("prunes entries older than 1 minute", func(t *testing.T) {
		k := NewKeepAlive(&KeepAliveOptions{
			MaxRestartsPerMinute: 10,
		})

		// Manually add old restart times
		k.mu.Lock()
		k.restartTimes = []time.Time{
			time.Now().Add(-2 * time.Minute),  // old - should be pruned
			time.Now().Add(-90 * time.Second), // old - should be pruned
			time.Now().Add(-30 * time.Second), // recent - should stay
			time.Now(),                        // recent - should stay
		}
		k.mu.Unlock()

		// RestartsInLastMinute calls pruneOldRestarts internally
		count := k.RestartsInLastMinute()
		assert.Equal(t, 2, count)
	})
}

func TestKeepAlive_ConcurrencySafety(t *testing.T) {
	k := NewKeepAlive(&KeepAliveOptions{
		InitialBackoff:       1 * time.Millisecond,
		MaxBackoff:           1 * time.Second,
		MaxRestartsPerMinute: 100,
	})

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				k.RecordRestart()
				_, _, _ = k.ShouldRestart()
				_ = k.CurrentBackoff()
				_ = k.RestartsInLastMinute()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Just verify no race condition occurred
	require.NotNil(t, k)
}
