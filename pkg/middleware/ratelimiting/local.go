package ratelimiting

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/nite-coder/blackbear/pkg/cache/v2"
)

type LocalLimiter struct {
	options *Options
	cache   *cache.Cache[string, *atomic.Uint64]
	mu      sync.Mutex
}

func NewLocalLimiter(options Options) *LocalLimiter {
	return &LocalLimiter{
		options: &options,
		cache:   cache.NewCache[string, *atomic.Uint64](5 * time.Minute),
	}
}

func (l *LocalLimiter) Allow(namespace string) AllowResult {
	l.mu.Lock()
	defer l.mu.Unlock()

	result := AllowResult{
		Limit: l.options.Limit,
	}
	counter, found := l.cache.Get(namespace)

	if !found {
		counter = &atomic.Uint64{}
		counter.Add(1)
		l.cache.PutWithTTL(namespace, counter, l.options.WindowSize)
		result.Allow = true
		result.Remaining = l.options.Limit - 1
		// TODO: Add reset time
		return result
	}

	current := counter.Load()
	if current >= l.options.Limit {
		result.Allow = false
		result.Remaining = 0
		return result
	}

	counter.Add(1)
	result.Allow = true
	result.Remaining = l.options.Limit - counter.Load()
	return result
}
