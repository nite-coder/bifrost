package target

import (
	"sync"
	"time"

	"github.com/nite-coder/bifrost/pkg/timecache"
)

// State tracks the health of an endpoint with failure counting and timeout.
type State struct {
	mu           sync.RWMutex
	failedCount  uint
	maxFails     uint
	failTimeout  time.Duration
	failExpireAt time.Time
}

// NewState creates a new State with the given max failures and fail timeout.
func NewState(maxFails uint, failTimeout time.Duration) *State {
	return &State{
		maxFails:    maxFails,
		failTimeout: failTimeout,
	}
}

// IsAvailable returns true if the endpoint is healthy and accepting requests.
func (s *State) IsAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.maxFails == 0 {
		return true
	}
	now := timecache.Now()
	if now.After(s.failExpireAt) {
		return true
	}
	return s.failedCount < s.maxFails
}

// RecordFailure increments the failure count, resetting if past the fail timeout.
func (s *State) RecordFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := timecache.Now()
	if now.After(s.failExpireAt) {
		s.failExpireAt = now.Add(s.failTimeout)
		s.failedCount = 1
	} else if s.failedCount < s.maxFails {
		s.failedCount++
	}
}
