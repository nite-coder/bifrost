package zero

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsDaemonChild(t *testing.T) {
	t.Run("not daemon child", func(t *testing.T) {
		os.Unsetenv("BIFROST_DAEMONIZED")
		assert.False(t, IsDaemonChild())
	})

	t.Run("is daemon child", func(t *testing.T) {
		os.Setenv("BIFROST_DAEMONIZED", "1")
		defer os.Unsetenv("BIFROST_DAEMONIZED")
		assert.True(t, IsDaemonChild())
	})
}

func TestNotifyDaemonReady_NotDaemon(t *testing.T) {
	os.Unsetenv("BIFROST_DAEMONIZED")
	err := NotifyDaemonReady()
	assert.NoError(t, err)
}

func TestDaemonOptions_Defaults(t *testing.T) {
	// Test that nil options get defaults
	opts := &DaemonOptions{}
	assert.Equal(t, time.Duration(0), opts.ReadyTimeout)
}

func TestSpawnDaemonChild_Timeout(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping test that may require special permissions")
	}

	// This test would spawn a child that never signals ready
	// In practice, we'd need a test binary for this
}
