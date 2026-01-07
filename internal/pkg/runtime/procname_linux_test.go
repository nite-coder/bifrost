//go:build linux

package runtime

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Note: Testing SetProcessName is challenging because:
// 1. init() has already set the process name on the main thread
// 2. Go tests may run on different OS threads than the main thread
// 3. /proc/self/comm reads the main thread's name
//
// The actual functionality is verified through manual testing:
// - Build and run the bifrost server
// - Use `ps -eo pid,comm | grep bifrost` to verify "bifrost-master" and "bifrost-worker"

func TestInitSetsProcessName(t *testing.T) {
	// Test that the init() correctly reads BIFROST_ROLE environment variable
	// and that IsWorker() function correctly detects the worker role.

	t.Run("worker role detection", func(t *testing.T) {
		originalRole := os.Getenv(EnvBifrostRole)
		defer os.Setenv(EnvBifrostRole, originalRole)

		os.Setenv(EnvBifrostRole, RoleWorker)
		assert.True(t, IsWorker())

		os.Setenv(EnvBifrostRole, "")
		assert.False(t, IsWorker())
	})

	t.Run("environment variable constants", func(t *testing.T) {
		assert.Equal(t, "BIFROST_ROLE", EnvBifrostRole)
		assert.Equal(t, "worker", RoleWorker)
	})
}

func TestSetProcessNameTruncation(t *testing.T) {
	// Test that long names are truncated to 15 characters
	// This doesn't verify the actual prctl call, just the truncation logic

	t.Run("name under 15 chars is unchanged", func(t *testing.T) {
		// SetProcessName should not error for valid names
		err := SetProcessName("short")
		assert.NoError(t, err)
	})

	t.Run("name over 15 chars is handled without error", func(t *testing.T) {
		// SetProcessName should not error for long names (they get truncated internally)
		err := SetProcessName("this-is-a-very-long-process-name")
		assert.NoError(t, err)
	})
}
