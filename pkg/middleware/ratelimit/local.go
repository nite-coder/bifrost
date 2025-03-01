package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/blackbear/pkg/cache/v2"
)

type LocalLimiter struct {
	options *Options
	cache   *cache.Cache[string, *item]
	mu      sync.Mutex
}

type item struct {
	expiration time.Time
	counter    uint64
}

func NewLocalLimiter(options Options) *LocalLimiter {
	return &LocalLimiter{
		options: &options,
		cache:   cache.NewCache[string, *item](10 * time.Minute),
	}
}

func (l *LocalLimiter) Allow(ctx context.Context, key string) *AllowResult {
	l.mu.Lock()
	defer l.mu.Unlock()

	result := &AllowResult{
		Limit: l.options.Limit,
	}
	itemVal, found := l.cache.Get(key)

	if !found {
		now := timecache.Now()

		val := &item{
			expiration: now.Add(l.options.WindowSize),
			counter:    1,
		}

		l.cache.PutWithTTL(key, val, l.options.WindowSize)
		result.Allow = true
		result.Remaining = l.options.Limit - val.counter
		result.ResetTime = val.expiration
		return result
	}

	current := itemVal.counter
	if current >= l.options.Limit {
		result.Allow = false
		result.Remaining = 0
		result.ResetTime = itemVal.expiration
		return result
	}

	itemVal.counter++
	result.Allow = true
	result.Remaining = l.options.Limit - itemVal.counter
	result.ResetTime = itemVal.expiration
	return result
}
