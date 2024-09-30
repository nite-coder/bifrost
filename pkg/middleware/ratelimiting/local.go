package ratelimiting

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/nite-coder/blackbear/pkg/cache/v2"
)

type LocalLimiter struct {
	options *Options
	cache   *cache.Cache[string, *atomic.Int64]
	mu      sync.Mutex
}

func NewLocalLimiter(options Options) *LocalLimiter {
	return &LocalLimiter{
		options: &options,
		cache:   cache.NewCache[string, *atomic.Int64](5 * time.Minute),
	}
}

func (l *LocalLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	counter, found := l.cache.Get(key)

	if found {
		current := counter.Load()
		if current >= l.options.Limit {
			return false
		}

		counter.Add(1)
		return true
	}

	counter = &atomic.Int64{}
	counter.Add(1)
	l.cache.PutWithTTL(key, counter, l.options.WindowSize)
	return true
}
