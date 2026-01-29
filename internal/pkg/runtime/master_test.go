package runtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelperProcess isn't a real test. It's used to mock a child process.
// It's invoked by fakeExecCommandContext.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Behave as a worker
	// Check for instructions in args
	args := os.Args
	for _, arg := range args {
		if arg == "FORCE_ERROR" {
			os.Exit(1)
		}
	}

	// Simulate work
	// Handle signals for graceful shutdown testing
	fmt.Fprintln(os.Stderr, "Helper process started, setting up signal handler")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "Helper process received signal: %v\n", sig)
		os.Exit(0)
	case <-time.After(2 * time.Second):
		fmt.Fprintln(os.Stderr, "Helper process timeout")
		os.Exit(0)
	}
}

func fakeExecCommandContext(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--"}
	cs = append(cs, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", EnvBifrostRole + "=" + RoleWorker}
	return cmd
}

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
	// Swap execCommandContext
	oldExec := execCommandContext
	execCommandContext = fakeExecCommandContext
	defer func() { execCommandContext = oldExec }()

	t.Run("spawn success", func(t *testing.T) {
		m := NewMaster(&MasterOptions{
			Binary: "/bin/echo",
			Args:   []string{"hello"},
		})

		// Initialize control plane (needed for socket path)
		m.controlPlane = NewControlPlane(nil)

		cmd, err := m.spawnWorker(context.Background(), nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.NotNil(t, cmd.Process)

		// Wait for process to exit
		state, err := cmd.Process.Wait()
		assert.NoError(t, err)
		assert.Equal(t, 0, state.ExitCode())
	})

	t.Run("spawn failure", func(t *testing.T) {
		// Mock failure
		oldExecCtx := execCommandContext
		execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.Command("non-existent-binary-xyz-12345")
		}
		defer func() { execCommandContext = oldExecCtx }()

		m := NewMaster(&MasterOptions{
			Binary: "should-fail",
		})
		m.controlPlane = NewControlPlane(nil)

		cmd, err := m.spawnWorker(context.Background(), nil, nil)
		assert.Error(t, err)
		assert.Nil(t, cmd)
	})
}

func TestMaster_SpawnWorker_UserGroup(t *testing.T) {
	// Swap execCommandContext
	oldExec := execCommandContext
	execCommandContext = fakeExecCommandContext
	defer func() { execCommandContext = oldExec }()

	// Swap startCommand
	oldStart := startCommand
	startCommand = func(cmd *exec.Cmd) error {
		if cmd.Process == nil {
			cmd.Process = &os.Process{Pid: 12345}
		}
		return nil
	}
	defer func() { startCommand = oldStart }()

	// Mock User/Group lookup
	oldLookupUser := lookupUser
	oldLookupGroup := lookupGroup
	defer func() {
		lookupUser = oldLookupUser
		lookupGroup = oldLookupGroup
	}()

	mockUsers := map[string]*user.User{
		"testuser":  {Uid: "1001", Gid: "1001"},
		"otheruser": {Uid: "1002", Gid: "1002"},
	}
	mockGroups := map[string]*user.Group{
		"testgroup":  {Gid: "2001"},
		"othergroup": {Gid: "2002"},
	}

	lookupUser = func(username string) (*user.User, error) {
		if u, ok := mockUsers[username]; ok {
			return u, nil
		}
		return nil, user.UnknownUserError(username)
	}

	lookupGroup = func(name string) (*user.Group, error) {
		if g, ok := mockGroups[name]; ok {
			return g, nil
		}
		return nil, user.UnknownGroupError(name)
	}

	t.Run("user only (use primary group)", func(t *testing.T) {
		m := NewMaster(&MasterOptions{
			Binary: "test",
			User:   "testuser",
		})
		m.controlPlane = NewControlPlane(nil)

		cmd, err := m.spawnWorker(context.Background(), nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd.SysProcAttr)
		require.NotNil(t, cmd.SysProcAttr.Credential)
		assert.Equal(t, uint32(1001), cmd.SysProcAttr.Credential.Uid)
		assert.Equal(t, uint32(1001), cmd.SysProcAttr.Credential.Gid)
	})

	t.Run("group only (keep current uid, change gid)", func(t *testing.T) {
		m := NewMaster(&MasterOptions{
			Binary: "test",
			Group:  "testgroup",
		})
		m.controlPlane = NewControlPlane(nil)

		currentUID := os.Getuid()
		cmd, err := m.spawnWorker(context.Background(), nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd.SysProcAttr)
		require.NotNil(t, cmd.SysProcAttr.Credential)
		assert.Equal(t, uint32(currentUID), cmd.SysProcAttr.Credential.Uid)
		assert.Equal(t, uint32(2001), cmd.SysProcAttr.Credential.Gid)
	})

	t.Run("user and group", func(t *testing.T) {
		m := NewMaster(&MasterOptions{
			Binary: "test",
			User:   "testuser",
			Group:  "testgroup",
		})
		m.controlPlane = NewControlPlane(nil)

		cmd, err := m.spawnWorker(context.Background(), nil, nil)
		require.NoError(t, err)
		require.NotNil(t, cmd.SysProcAttr)
		require.NotNil(t, cmd.SysProcAttr.Credential)
		assert.Equal(t, uint32(1001), cmd.SysProcAttr.Credential.Uid)
		assert.Equal(t, uint32(2001), cmd.SysProcAttr.Credential.Gid)
	})

	t.Run("neither (inherit)", func(t *testing.T) {
		m := NewMaster(&MasterOptions{
			Binary: "test",
		})
		m.controlPlane = NewControlPlane(nil)

		cmd, err := m.spawnWorker(context.Background(), nil, nil)
		require.NoError(t, err)
		// Assuming default behavior is nil SysProcAttr or nil Credential if not set
		if cmd.SysProcAttr != nil {
			assert.Nil(t, cmd.SysProcAttr.Credential)
		}
	})

	t.Run("invalid user", func(t *testing.T) {
		m := NewMaster(&MasterOptions{
			Binary: "test",
			User:   "invalid",
		})
		m.controlPlane = NewControlPlane(nil)

		_, err := m.spawnWorker(context.Background(), nil, nil)
		assert.Error(t, err)
	})
}

func TestMaster_HandleReload_NotRunning(t *testing.T) {
	m := NewMaster(nil)

	ctx := context.Background()
	err := m.handleReload(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot reload in state")
}

func TestMaster_HandleControlMessage(t *testing.T) {
	m := NewMaster(nil)

	t.Run("ready message signals channel", func(t *testing.T) {
		msg := &ControlMessage{
			Type:      MessageTypeReady,
			WorkerPID: 123,
		}
		m.handleControlMessage(nil, msg)

		select {
		case <-m.readyCh:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Error("readyCh did not receive signal")
		}
	})

	t.Run("register message logs only", func(t *testing.T) {
		msg := &ControlMessage{
			Type:      MessageTypeRegister,
			WorkerPID: 123,
		}
		// Should not panic or block
		m.handleControlMessage(nil, msg)
	})
}

func TestMaster_HandleReload(t *testing.T) {
	// Swap execCommandContext
	oldExec := execCommandContext
	execCommandContext = fakeExecCommandContext
	defer func() { execCommandContext = oldExec }()

	t.Run("successful reload", func(t *testing.T) {
		m := NewMaster(nil)
		m.controlPlane = NewControlPlane(nil) // Needed for socket path

		// 1. Start initial worker
		err := m.spawnAndWatch(context.Background())
		require.NoError(t, err)
		assert.Equal(t, MasterStateRunning, m.State())

		// 2. Trigger reload
		// Since we mocked spawnWorker, we don't need real IPC.
		// However, handleReload sends messages to old worker.
		// We need to Mock ControlPlane or ensure SendMessage doesn't block/fail fatally.
		// SendMessage uses m.conns which requires the worker to have connected.
		// In this mocked test, no worker connects to CP. SendMessage will fail.
		// Master logs warning but continues spawn.

		// We need to simulate the NEW worker sending "Ready".
		// Since we can't easily sync with the "subprocess" (TestHelperProcess),
		// we can run a goroutine to signal readyCh after a delay.
		go func() {
			time.Sleep(100 * time.Millisecond)
			m.readyCh <- struct{}{}
		}()

		err = m.handleReload(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, MasterStateRunning, m.State())

		// Trigger shutdown
		_ = m.Shutdown(context.Background())

		assert.Eventually(t, func() bool {
			return m.State() == MasterStateShuttingDown
		}, 2*time.Second, 100*time.Millisecond)
	})
}

func TestMaster_Run(t *testing.T) {
	// Swap execCommandContext
	oldExec := execCommandContext
	execCommandContext = fakeExecCommandContext
	defer func() { execCommandContext = oldExec }()

	t.Run("run and shutdown context", func(t *testing.T) {
		m := NewMaster(nil)
		// Use unique socket path
		socketPath := filepath.Join(t.TempDir(), "test1.sock")
		m.controlPlane = NewControlPlane(&ControlPlaneOptions{SocketPath: socketPath})

		ctx, cancel := context.WithCancel(context.Background())

		// Run in background
		errCh := make(chan error, 1)
		go func() {
			errCh <- m.Run(ctx)
		}()

		// Wait for start
		assert.Eventually(t, func() bool {
			return m.State() == MasterStateRunning
		}, 2*time.Second, 100*time.Millisecond)

		// Simulate worker readiness
		pid := m.WorkerPID()
		conn, err := net.Dial("unix", socketPath)
		require.NoError(t, err)
		json.NewEncoder(conn).Encode(&ControlMessage{
			Type:      MessageTypeReady,
			WorkerPID: pid,
		})
		conn.Close()

		// Wait for Master to process readiness (it will move past m.readyCh consume)
		assert.Eventually(t, func() bool {
			return m.WorkerPID() == pid
		}, 2*time.Second, 100*time.Millisecond)

		// Trigger stop
		_ = m.Shutdown(context.Background())

		// Cancel context just in case
		cancel()

		// Wait for exit
		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Run did not exit")
		}
	})

	t.Run("run and stop channel", func(t *testing.T) {
		m := NewMaster(nil)
		// Use unique socket path
		socketPath := filepath.Join(t.TempDir(), "test2.sock")
		m.controlPlane = NewControlPlane(&ControlPlaneOptions{SocketPath: socketPath})

		// Run in background
		errCh := make(chan error, 1)
		go func() {
			errCh <- m.Run(context.Background())
		}()

		assert.Eventually(t, func() bool {
			return m.State() == MasterStateRunning
		}, 2*time.Second, 100*time.Millisecond)

		// Simulate worker readiness
		pid := m.WorkerPID()
		conn, err := net.Dial("unix", socketPath)
		require.NoError(t, err)
		json.NewEncoder(conn).Encode(&ControlMessage{
			Type:      MessageTypeReady,
			WorkerPID: pid,
		})
		conn.Close()

		// Wait for Master to confirm readiness
		assert.Eventually(t, func() bool {
			return m.WorkerPID() == pid
		}, 2*time.Second, 100*time.Millisecond)

		// Trigger stop via Shutdown
		_ = m.Shutdown(context.Background())

		select {
		case err := <-errCh:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Run did not exit")
		}
	})
}

func TestMaster_FDTransfer(t *testing.T) {
	m := NewMaster(nil)
	socketPath := filepath.Join(t.TempDir(), "fd_transfer.sock")
	m.controlPlane = NewControlPlane(&ControlPlaneOptions{SocketPath: socketPath})

	m.controlPlane.SetFDHandler(func(fds []*os.File, keys []string) {
		m.handleFDTransfer(fds, keys)
	})

	require.NoError(t, m.controlPlane.Listen())
	defer m.controlPlane.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = m.controlPlane.Accept(ctx)
	}()

	// Connect worker
	wcp := NewWorkerControlPlane(socketPath)
	require.NoError(t, wcp.Connect())
	defer wcp.Close()

	// Send FD
	f, err := os.CreateTemp(t.TempDir(), "test_fd")
	require.NoError(t, err)
	defer f.Close()

	err = wcp.SendFDs([]*os.File{f}, []string{"test_key"})
	require.NoError(t, err)

	select {
	case listenerInfo := <-m.listenerDataCh:
		assert.NotNil(t, listenerInfo)
		assert.Equal(t, "test_key", listenerInfo.keys[0])
		listenerInfo.fds[0].Close()
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for FD transfer")
	}

	// Test case: Channel full/No reload pending
	// Fill the channel
	m.listenerDataCh <- &listenerData{}

	f2, err := os.CreateTemp(t.TempDir(), "test_fd_dropped")
	require.NoError(t, err)
	// handleFDTransfer should close f2
	// We can't easily verify Close() was called on f2 without mocking OS file or using finalizers
	// But we can verify it doesn't block.

	done := make(chan struct{})
	go func() {
		m.handleFDTransfer([]*os.File{f2}, []string{"key"})
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("handleFDTransfer blocked when channel full")
	}
}
