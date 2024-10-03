package ratelimiting

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocalLimiter(t *testing.T) {
	options := Options{
		Limit:      5,
		WindowSize: time.Second,
	}
	limiter := NewLocalLimiter(options)

	t.Run("Basic functionality", func(t *testing.T) {
		key := "test_key"
		now := time.Now()
		ctx := context.Background()

		for i := 1; i < 6; i++ {
			result := limiter.Allow(ctx, key)
			if !result.Allow {
				t.Errorf("Request %d should be allowed", i+1)
			}

			assert.Equal(t, options.Limit, result.Limit)
			assert.Equal(t, uint64(5-i), result.Remaining) // nolint
			assert.LessOrEqual(t, float64(1), result.ResetTime.Sub(now).Seconds())
		}
		result := limiter.Allow(ctx, key)
		if result.Allow {
			t.Error("6th request should be denied")
		}
	})

	t.Run("Different keys", func(t *testing.T) {
		ctx := context.Background()

		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("key_%d", i)
			result := limiter.Allow(ctx, key)
			if !result.Allow {
				t.Errorf("Request for key %s should be allowed", key)
			}
		}
	})

	t.Run("Window reset", func(t *testing.T) {
		ctx := context.Background()

		key := "reset_key"
		for i := 0; i < 5; i++ {
			result := limiter.Allow(ctx, key)
			if !result.Allow {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}

		result := limiter.Allow(ctx, key)
		if result.Allow {
			t.Error("6th request should be denied")
		}

		time.Sleep(options.WindowSize)

		result = limiter.Allow(ctx, key)
		if !result.Allow {
			t.Error("Request after reset should be allowed")
		}
	})

	t.Run("Concurrent requests", func(t *testing.T) {
		ctx := context.Background()

		key := "concurrent_key"
		concurrentRequests := 100
		allowedCount := atomic.Uint64{}
		var wg sync.WaitGroup

		wg.Add(concurrentRequests)
		for i := 0; i < concurrentRequests; i++ {
			go func() {
				defer wg.Done()

				result := limiter.Allow(ctx, key)
				if result.Allow {
					allowedCount.Add(1)
				}
			}()
		}
		wg.Wait()

		if allowedCount.Load() != options.Limit {
			t.Errorf("Expected %d requests to be allowed, but got %d", options.Limit, allowedCount.Load())
		}
	})
}
