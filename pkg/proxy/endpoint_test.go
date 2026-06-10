package proxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTargetState_Health(t *testing.T) {
	ts := NewTargetState(2, 1*time.Second)
	assert.True(t, ts.IsAvailable())

	ts.RecordFailure()
	assert.True(t, ts.IsAvailable(), "Should still be available after 1 failure")

	ts.RecordFailure()
	assert.False(t, ts.IsAvailable(), "Should be unavailable after 2 failures")

	// Wait for FailTimeout to pass
	assert.Eventually(t, func() bool {
		return ts.IsAvailable()
	}, 2*time.Second, 10*time.Millisecond, "Should recover after FailTimeout")
}
