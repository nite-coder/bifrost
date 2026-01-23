package timecache

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
)

type TimeCache struct {
	t        atomic.Value
	interval time.Duration
	stopCh   chan struct{}
}

// New returns a new TimeCache instance with the specified interval.
//
// The interval parameter specifies the duration between cache refreshes.
// If interval is 0, it defaults to 1 second; if it's less than 1 millisecond, it defaults to 1 millisecond.
// *TimeCache
func New(interval time.Duration) *TimeCache {
	if interval == 0 {
		interval = time.Second
	} else if interval < time.Millisecond {
		interval = time.Millisecond
	}

	tc := &TimeCache{
		interval: interval,
		stopCh:   make(chan struct{}),
	}

	tc.t.Store(time.Now())
	go safety.Go(context.Background(), tc.refresh)
	return tc
}

// Now returns the current time from the TimeCache instance.
//
// No parameters.
// Returns a time.Time value representing the current time.
func (tc *TimeCache) Now() time.Time {
	return tc.t.Load().(time.Time)
}

func (tc *TimeCache) Close() {
	close(tc.stopCh)
}

func (tc *TimeCache) refresh() {
	ticker := time.NewTicker(tc.interval) 
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tc.t.Store(time.Now())
		case <-tc.stopCh:
			return
		}
	}
}
