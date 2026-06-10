package proxy

import (
	"sync"
	"time"

	"github.com/nite-coder/bifrost/pkg/timecache"
)

// TargetState holds the shared health state for a physical IP:Port.
type TargetState struct {
	mu           sync.RWMutex
	failedCount  uint
	maxFails     uint
	failTimeout  time.Duration
	failExpireAt time.Time
}

// NewTargetState creates a new TargetState instance.
func NewTargetState(maxFails uint, failTimeout time.Duration) *TargetState {
	return &TargetState{
		maxFails:    maxFails,
		failTimeout: failTimeout,
	}
}

// IsAvailable checks if the target is currently available based on its failure history.
func (ts *TargetState) IsAvailable() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if ts.maxFails == 0 {
		return true
	}
	now := timecache.Now()
	if now.After(ts.failExpireAt) {
		return true
	}
	return ts.failedCount < ts.maxFails
}

// RecordFailure records a request failure for the target.
// Note: As a design decision, this implementation starts/renews the failExpireAt window on the first
// failure after expiry (setting failedCount to 1), rather than waiting until the maxFails threshold is crossed.
// This simplifies the logic while achieving the same functional behavior of blocking targets under persistent failure.
func (ts *TargetState) RecordFailure() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	now := timecache.Now()
	if now.After(ts.failExpireAt) {
		ts.failExpireAt = now.Add(ts.failTimeout)
		ts.failedCount = 1
	} else {
		ts.failedCount++
	}
}

// Endpoint represents a discovered backend target.
type Endpoint struct {
	Address     string
	Weight      uint32
	Tags        map[string]string
	HealthState *TargetState
}
