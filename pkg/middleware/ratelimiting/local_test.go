package ratelimiting

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLocalLimiter(t *testing.T) {
	options := Options{
		Limit:      5,
		WindowSize: time.Second,
	}
	limiter := NewLocalLimiter(options)

	t.Run("Basic functionality", func(t *testing.T) {
		key := "test_key"
		for i := 0; i < 5; i++ {
			if !limiter.Allow(key) {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}
		if limiter.Allow(key) {
			t.Error("6th request should be denied")
		}
	})

	t.Run("Different keys", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("key_%d", i)
			if !limiter.Allow(key) {
				t.Errorf("Request for key %s should be allowed", key)
			}
		}
	})

	t.Run("Window reset", func(t *testing.T) {
		key := "reset_key"
		for i := 0; i < 5; i++ {
			if !limiter.Allow(key) {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}
		if limiter.Allow(key) {
			t.Error("6th request should be denied")
		}

		time.Sleep(options.WindowSize)

		if !limiter.Allow(key) {
			t.Error("Request after reset should be allowed")
		}
	})

	t.Run("Concurrent requests", func(t *testing.T) {
		key := "concurrent_key"
		concurrentRequests := 100
		allowedCount := atomic.Int64{}
		var wg sync.WaitGroup

		wg.Add(concurrentRequests)
		for i := 0; i < concurrentRequests; i++ {
			go func() {
				defer wg.Done()
				if limiter.Allow(key) {
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
