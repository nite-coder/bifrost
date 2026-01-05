package zero

import (
	"context"
	"encoding/base64"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMaster(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		m := NewMaster(nil)
		assert.NotNil(t, m)
		assert.Equal(t, os.Args[0], m.options.Binary)
		assert.Equal(t, 30*time.Second, m.options.GracefulTimeout)
		assert.Equal(t, MasterStateIdle, m.State())
	})

	t.Run("custom options", func(t *testing.T) {
		opts := &MasterOptions{
			Binary:          "/usr/bin/test",
			ConfigPath:      "/etc/test.yaml",
			GracefulTimeout: 60 * time.Second,
		}
		m := NewMaster(opts)
		assert.Equal(t, "/usr/bin/test", m.options.Binary)
		assert.Equal(t, "/etc/test.yaml", m.options.ConfigPath)
		assert.Equal(t, 60*time.Second, m.options.GracefulTimeout)
	})
}

func TestMasterState(t *testing.T) {
	t.Run("state string representation", func(t *testing.T) {
		assert.Equal(t, "idle", MasterStateIdle.String())
		assert.Equal(t, "running", MasterStateRunning.String())
		assert.Equal(t, "reloading", MasterStateReloading.String())
		assert.Equal(t, "shutting_down", MasterStateShuttingDown.String())
		assert.Equal(t, "unknown", MasterState(99).String())
	})
}

func TestMaster_WorkerPID(t *testing.T) {
	t.Run("no worker running", func(t *testing.T) {
		m := NewMaster(nil)
		assert.Equal(t, 0, m.WorkerPID())
	})
}

func TestMaster_Shutdown(t *testing.T) {
	t.Run("shutdown from idle state", func(t *testing.T) {
		m := NewMaster(nil)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := m.Shutdown(ctx)
		assert.NoError(t, err)
		assert.Equal(t, MasterStateShuttingDown, m.State())
	})

	t.Run("shutdown is idempotent", func(t *testing.T) {
		m := NewMaster(nil)

		ctx := context.Background()
		err := m.Shutdown(ctx)
		assert.NoError(t, err)

		// Second shutdown should be no-op
		err = m.Shutdown(ctx)
		assert.NoError(t, err)
	})
}

func TestIsWorker(t *testing.T) {
	t.Run("not a worker", func(t *testing.T) {
		os.Unsetenv(EnvBifrostRole)
		assert.False(t, IsWorker())
	})

	t.Run("is a worker", func(t *testing.T) {
		os.Setenv(EnvBifrostRole, RoleWorker)
		defer os.Unsetenv(EnvBifrostRole)
		assert.True(t, IsWorker())
	})
}

func TestGetControlSocketPath(t *testing.T) {
	t.Run("not set", func(t *testing.T) {
		os.Unsetenv("BIFROST_CONTROL_SOCKET")
		assert.Empty(t, GetControlSocketPath())
	})

	t.Run("is set", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("/tmp/test.sock"))
		os.Setenv("BIFROST_CONTROL_SOCKET", encoded)
		defer os.Unsetenv("BIFROST_CONTROL_SOCKET")
		assert.Equal(t, "/tmp/test.sock", GetControlSocketPath())
	})

	t.Run("invalid base64", func(t *testing.T) {
		os.Setenv("BIFROST_CONTROL_SOCKET", "not-valid-base64!!!")
		defer os.Unsetenv("BIFROST_CONTROL_SOCKET")
		assert.Empty(t, GetControlSocketPath())
	})
}

func TestMaster_SpawnWorker(t *testing.T) {
	t.Run("spawn with echo command", func(t *testing.T) {
		m := NewMaster(&MasterOptions{
			Binary: "/bin/echo",
			Args:   []string{"hello"},
		})

		// Initialize control plane
		err := m.controlPlane.Listen()
		require.NoError(t, err)
		defer m.controlPlane.Close()

		cmd, err := m.spawnWorker(context.Background(), nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.NotNil(t, cmd.Process)

		// Wait for process to exit
		state, err := cmd.Process.Wait()
		assert.NoError(t, err)
		assert.Equal(t, 0, state.ExitCode())
	})
}

func TestMaster_HandleReload_NotRunning(t *testing.T) {
	m := NewMaster(nil)

	ctx := context.Background()
	err := m.handleReload(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot reload in state")
}

func TestMaster_IntegrationWithSignal(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping integration test that requires root")
	}

	t.Run("master receives SIGTERM and shuts down", func(t *testing.T) {
		m := NewMaster(&MasterOptions{
			Binary: "/bin/sleep",
			Args:   []string{"60"},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Run master in background
		done := make(chan error, 1)
		go func() {
			done <- m.Run(ctx)
		}()

		// Wait for master to start
		time.Sleep(500 * time.Millisecond)

		// Send SIGTERM to self (master)
		syscall.Kill(os.Getpid(), syscall.SIGTERM)

		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(3 * time.Second):
			t.Fatal("master did not shut down in time")
		}
	})
}
