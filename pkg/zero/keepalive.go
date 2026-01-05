package zero

import (
	"errors"
	"sync"
	"time"
)

// ErrRestartLimitExceeded is returned when the maximum restart rate is exceeded.
var ErrRestartLimitExceeded = errors.New("restart limit exceeded: too many restarts in the last minute")

// KeepAliveOptions contains configuration for the KeepAlive strategy.
type KeepAliveOptions struct {
	// InitialBackoff is the initial backoff duration (default: 1s).
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration (default: 32s).
	MaxBackoff time.Duration
	// MaxRestartsPerMinute is the maximum restarts allowed per minute (default: 5).
	MaxRestartsPerMinute int
}

// DefaultKeepAliveOptions returns the default KeepAlive configuration.
func DefaultKeepAliveOptions() *KeepAliveOptions {
	return &KeepAliveOptions{
		InitialBackoff:       1 * time.Second,
		MaxBackoff:           32 * time.Second,
		MaxRestartsPerMinute: 5,
	}
}

// KeepAlive implements exponential backoff restart strategy for Worker processes.
// It prevents restart storms when a Worker has a fatal bug.
type KeepAlive struct {
	options        *KeepAliveOptions
	currentBackoff time.Duration
	restartTimes   []time.Time
	mu             sync.Mutex
}

// NewKeepAlive creates a new KeepAlive instance.
func NewKeepAlive(opts *KeepAliveOptions) *KeepAlive {
	if opts == nil {
		opts = DefaultKeepAliveOptions()
	}
	if opts.InitialBackoff <= 0 {
		opts.InitialBackoff = 1 * time.Second
	}
	if opts.MaxBackoff <= 0 {
		opts.MaxBackoff = 32 * time.Second
	}
	if opts.MaxRestartsPerMinute <= 0 {
		opts.MaxRestartsPerMinute = 5
	}

	return &KeepAlive{
		options:        opts,
		currentBackoff: opts.InitialBackoff,
		restartTimes:   make([]time.Time, 0, opts.MaxRestartsPerMinute+1),
	}
}

// ShouldRestart determines if a Worker should be restarted.
// Returns:
//   - shouldRestart: true if restart is allowed
//   - backoffDuration: how long to wait before restarting
//   - error: non-nil if restart limit exceeded (Master should exit)
func (k *KeepAlive) ShouldRestart() (shouldRestart bool, backoffDuration time.Duration, err error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Prune old entries (older than 1 minute)
	k.pruneOldRestarts()

	// Check if restarts in last minute exceed limit
	if len(k.restartTimes) >= k.options.MaxRestartsPerMinute {
		return false, 0, ErrRestartLimitExceeded
	}

	return true, k.currentBackoff, nil
}

// RecordRestart records a restart event and updates backoff.
func (k *KeepAlive) RecordRestart() {
	k.mu.Lock()
	defer k.mu.Unlock()

	// Add current time to restart times
	k.restartTimes = append(k.restartTimes, time.Now())

	// Double the backoff (up to MaxBackoff)
	k.currentBackoff *= 2
	if k.currentBackoff > k.options.MaxBackoff {
		k.currentBackoff = k.options.MaxBackoff
	}
}

// Reset resets the backoff to initial value.
// Called when Worker has been running stably for a period.
func (k *KeepAlive) Reset() {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.currentBackoff = k.options.InitialBackoff
	k.restartTimes = k.restartTimes[:0]
}

// CurrentBackoff returns the current backoff duration.
func (k *KeepAlive) CurrentBackoff() time.Duration {
	k.mu.Lock()
	defer k.mu.Unlock()

	return k.currentBackoff
}

// RestartsInLastMinute returns the number of restarts in the last minute.
func (k *KeepAlive) RestartsInLastMinute() int {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.pruneOldRestarts()
	return len(k.restartTimes)
}

// pruneOldRestarts removes entries older than 1 minute.
// Must be called with lock held.
func (k *KeepAlive) pruneOldRestarts() {
	cutoff := time.Now().Add(-1 * time.Minute)
	newStart := 0
	for i, t := range k.restartTimes {
		if t.After(cutoff) {
			newStart = i
			break
		}
		newStart = i + 1
	}
	if newStart > 0 && newStart <= len(k.restartTimes) {
		k.restartTimes = k.restartTimes[newStart:]
	}
}
