package target_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/target"
)

func TestState_IsAvailable_NoMaxFails(t *testing.T) {
	s := target.NewState(0, time.Second)
	assert.True(t, s.IsAvailable())
	s.RecordFailure()
	assert.True(t, s.IsAvailable(), "no max fails means always available")
}

func TestState_FailThenRecover(t *testing.T) {
	s := target.NewState(2, 50*time.Millisecond)
	assert.True(t, s.IsAvailable())

	s.RecordFailure()
	assert.True(t, s.IsAvailable(), "one failure < maxFails=2")

	s.RecordFailure()
	assert.False(t, s.IsAvailable(), "two failures >= maxFails=2")

	assert.Eventually(t, func() bool {
		return s.IsAvailable()
	}, 200*time.Millisecond, 10*time.Millisecond, "should recover after failTimeout")
}
