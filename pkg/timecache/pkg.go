package timecache

import (
	"sync/atomic"
	"time"
)

var cache atomic.Value

// Set sets the global time cache instance.
func Set(timeCache *TimeCache) {
	val, ok := cache.Load().(*TimeCache)
	if ok && val != nil {
		val.Close()
	}
	cache.Store(timeCache)
}

// Now returns the cached current time.
func Now() time.Time {
	val, ok := cache.Load().(*TimeCache)
	if !ok {
		return time.Now()
	}

	return val.Now()
}
