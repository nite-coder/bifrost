package timecache

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/runtime"
)

type TimeCache struct {
	t        atomic.Value
	interval time.Duration
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
	}

	tc.t.Store(time.Now())
	go tc.refresh()
	return tc
}

// Now returns the current time from the TimeCache instance.
//
// No parameters.
// Returns a time.Time value representing the current time.
func (tc *TimeCache) Now() time.Time {
	return tc.t.Load().(time.Time)
}

func (tc *TimeCache) refresh() {
	defer func() {
		if r := recover(); r != nil {
			var err error
			switch v := r.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("%v", v)
			}
			stackTrace := runtime.StackTrace()
			slog.Error("timecache panic recovered",
				slog.String("error", err.Error()),
				slog.String("stack", stackTrace),
			)
		}
	}()

	ticker := time.NewTicker(tc.interval)
	for range ticker.C {
		tc.t.Store(time.Now())
	}
}
