package timecache

import (
	"sync/atomic"
	"time"
)

var (
	cache atomic.Value
)

func Set(timeCache *TimeCache) {
	cache.Store(timeCache)
}

func Now() time.Time {
	val, ok := cache.Load().(*TimeCache)
	if !ok {
		return time.Now()
	}

	return val.Now()
}
