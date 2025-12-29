package zero

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/stretchr/testify/assert"
)

func TestGetPIDFile(t *testing.T) {
	tests := []struct {
		name     string
		pidFile  string
		expected string
	}{
		{"Default", "", "./logs/bifrost.pid"},
		{"Custom", "./custom.pid", "./custom.pid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{PIDFile: tt.pidFile}
			if got := opts.GetPIDFile(); got != tt.expected {
				t.Errorf("GetPIDFile() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetUpgradeSock(t *testing.T) {
	tests := []struct {
		name        string
		upgradeSock string
		expected    string
	}{
		{"Default", "", "./logs/bifrost.sock"},
		{"Custom", "./custom.sock", "./custom.sock"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{UpgradeSock: tt.upgradeSock}
			if got := opts.GetUpgradeSock(); got != tt.expected {
				t.Errorf("GetUpgradeSock() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsUpgraded(t *testing.T) {
	t.Run("upgraded", func(t *testing.T) {
		z := New(Options{})
		z.envGetter = func(k string) string {
			if k == "UPGRADE" {
				return "1"
			}
			return ""
		}
		if !z.IsUpgraded() {
			t.Error("Expected upgraded state")
		}
	})

	t.Run("not upgraded", func(t *testing.T) {
		z := New(Options{})
		z.envGetter = func(string) string { return "" }
		if z.IsUpgraded() {
			t.Error("Expected normal state")
		}
	})
}

func TestListener(t *testing.T) {
	t.Run("new listener creation", func(t *testing.T) {
		z := New(Options{})

		listenOptions := &ListenerOptions{
			Network: "tcp",
			Address: "localhost:0",
		}

		l, err := z.Listener(context.Background(), listenOptions)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		if len(z.listeners) != 1 {
			t.Errorf("Expected 1 listener, got %d", len(z.listeners))
		}
	})

	t.Run("listener with proxy protocol", func(t *testing.T) {
		z := New(Options{})

		listenOptions := &ListenerOptions{
			Network:       "tcp",
			Address:       "localhost:0",
			ProxyProtocol: true,
		}

		l, err := z.Listener(context.Background(), listenOptions)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		if len(z.listeners) != 1 {
			t.Errorf("Expected 1 listener, got %d", len(z.listeners))
		}
	})

	t.Run("reuse listener when upgraded", func(t *testing.T) {
		// mock upgraded
		z := New(Options{})
		z.envGetter = func(k string) string {
			if k == "UPGRADE" {
				return "1"
			}
			if k == "LISTENERS" {
				return `[{"Key":"localhost:1234"}]`
			}
			return ""
		}

		// mock file descriptor
		z.fileOpener = func(name string) (*os.File, error) {
			return os.NewFile(uintptr(3), ""), nil
		}

		listenOptions := &ListenerOptions{
			Network: "tcp",
			Address: "localhost:1234",
		}

		l, err := z.Listener(context.Background(), listenOptions)
		if err != nil {
			t.Fatal(err)
		}
		if l == nil {
			t.Error("Expected existing listener")
		}
	})
}

func TestClose(t *testing.T) {
	z := New(Options{})
	ctx := context.Background()

	// Create a listener
	listenOptions := &ListenerOptions{
		Network: "tcp",
		Address: "localhost:0",
	}

	l, err := z.Listener(ctx, listenOptions)
	if err != nil {
		t.Fatalf("Listener() error = %v", err)
	}

	// Close ZeroDownTime
	err = z.Close(ctx)
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Try to use the closed listener
	_, err = l.Accept()
	if err == nil {
		t.Error("Listener should be closed")
	}
}

func TestWaitForUpgrade(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zero_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := Options{
		UpgradeSock: tmpDir + "/test.sock",
		PIDFile:     tmpDir + "/test.pid",
	}
	z := New(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		safety.Go(ctx, func() {
			assert.Eventually(t, func() bool {
				conn, err := net.Dial("unix", opts.GetUpgradeSock())
				if err == nil {
					conn.Close()
					return true
				}
				return false
			}, 2*time.Second, 50*time.Millisecond, "Failed to connect to upgrade socket")
			z.Close(ctx)
		})
	}()

	err = z.WaitForUpgrade(ctx)
	if err != nil {
		t.Fatalf("WaitForUpgrade() error = %v", err)
	}
}

func TestPIDOperations(t *testing.T) {
	tempFile, _ := os.CreateTemp("", "pidfile")
	defer os.Remove(tempFile.Name())

	z := New(Options{PIDFile: tempFile.Name()})

	t.Run("write pid", func(t *testing.T) {
		err := z.writePID()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("read pid", func(t *testing.T) {
		pid, err := z.GetPID()
		if err != nil {
			t.Fatal(err)
		}
		if pid != os.Getpid() {
			t.Errorf("Expected PID %d, got %d", os.Getpid(), pid)
		}
	})
}

type mockProcess struct {
	signals []os.Signal
	pid     int
	killed  bool
	err     error
	wait    bool
}

func (m *mockProcess) Signal(sig os.Signal) error {
	if m.killed {
		return os.ErrProcessDone
	}

	if sig == syscall.SIGTERM && !m.wait {
		m.killed = true
	}

	m.signals = append(m.signals, sig)
	return m.err
}

func (m *mockProcess) Kill() error {
	if !m.wait {
		m.killed = true
	}
	return nil
}

func (m *mockProcess) Wait() (*os.ProcessState, error) {
	return nil, m.err
}

func (m *mockProcess) Release() error {
	return m.err
}

type mockProcessFinder struct {
	proc process
}

func (m *mockProcessFinder) FindProcess(pid int) (process, error) {
	return m.proc, nil
}

func TestQuitProcess(t *testing.T) {
	t.Run("normal quit", func(t *testing.T) {
		mp := &mockProcess{pid: 123}

		z := New(Options{
			QuitTimout: 1 * time.Second,
		})
		z.processFinder = &mockProcessFinder{proc: mp}

		err := z.Quit(context.Background(), 123, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(mp.signals) == 0 || mp.signals[0] != syscall.SIGTERM {
			t.Error("Expected SIGTERM signal")
		}
	})

	t.Run("quit timeout", func(t *testing.T) {
		mp := &mockProcess{pid: 123, wait: true}

		z := New(Options{
			QuitTimout: 1 * time.Second,
		})
		z.processFinder = &mockProcessFinder{proc: mp}

		err := z.Quit(context.Background(), 123, false)
		assert.ErrorIs(t, err, ErrKillTimeout)
	})
}

func TestWritePIDAtomicity(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zero_test_atomic")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidFile := tmpDir + "/test.pid"
	z := New(Options{PIDFile: pidFile})

	t.Run("atomic write creates temp file then renames", func(t *testing.T) {
		err := z.writePID()
		assert.NoError(t, err)

		// Verify PID file exists
		_, err = os.Stat(pidFile)
		assert.NoError(t, err)

		// Verify temp file is cleaned up
		_, err = os.Stat(pidFile + ".tmp")
		assert.True(t, os.IsNotExist(err))

		// Verify content
		pid, err := z.GetPID()
		assert.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)
	})

	t.Run("atomic write overwrites existing file", func(t *testing.T) {
		// Write a different PID
		err := os.WriteFile(pidFile, []byte("99999"), 0600)
		assert.NoError(t, err)

		// Overwrite with current PID
		err = z.writePID()
		assert.NoError(t, err)

		pid, err := z.GetPID()
		assert.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)
	})
}

func TestValidatePIDFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zero_test_validate")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidFile := tmpDir + "/test.pid"

	t.Run("no PID file exists", func(t *testing.T) {
		z := New(Options{PIDFile: pidFile})

		isRunning, pid, err := z.ValidatePIDFile()
		assert.NoError(t, err)
		assert.False(t, isRunning)
		assert.Equal(t, 0, pid)
	})

	t.Run("PID file with running process", func(t *testing.T) {
		z := New(Options{PIDFile: pidFile})
		mp := &mockProcess{pid: os.Getpid()}
		z.processFinder = &mockProcessFinder{proc: mp}

		// Write current PID
		err := z.writePID()
		assert.NoError(t, err)

		isRunning, pid, err := z.ValidatePIDFile()
		assert.NoError(t, err)
		assert.True(t, isRunning)
		assert.Equal(t, os.Getpid(), pid)
	})

	t.Run("PID file with dead process", func(t *testing.T) {
		z := New(Options{PIDFile: pidFile})
		mp := &mockProcess{pid: 99999, killed: true}
		z.processFinder = &mockProcessFinder{proc: mp}

		// Write a fake PID
		err := os.WriteFile(pidFile, []byte("99999"), 0600)
		assert.NoError(t, err)

		isRunning, pid, err := z.ValidatePIDFile()
		assert.NoError(t, err)
		assert.False(t, isRunning)
		assert.Equal(t, 99999, pid)
	})
}

func TestWritePIDWithLock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zero_test_lock")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidFile := tmpDir + "/test.pid"

	t.Run("successful lock and write", func(t *testing.T) {
		z := New(Options{PIDFile: pidFile})

		lockFile, err := z.WritePIDWithLock()
		assert.NoError(t, err)
		assert.NotNil(t, lockFile)

		// Verify PID file was written
		pid, err := z.GetPID()
		assert.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)

		// Release lock
		err = z.ReleasePIDLock(lockFile)
		assert.NoError(t, err)
	})

	t.Run("lock prevents concurrent access", func(t *testing.T) {
		z1 := New(Options{PIDFile: pidFile})
		z2 := New(Options{PIDFile: pidFile})

		// First process acquires lock
		lockFile1, err := z1.WritePIDWithLock()
		assert.NoError(t, err)
		assert.NotNil(t, lockFile1)

		// Second process should fail to acquire lock
		lockFile2, err := z2.WritePIDWithLock()
		assert.Error(t, err)
		assert.Nil(t, lockFile2)
		assert.Contains(t, err.Error(), "failed to acquire lock")

		// Release first lock
		err = z1.ReleasePIDLock(lockFile1)
		assert.NoError(t, err)

		// Now second process should succeed
		lockFile2, err = z2.WritePIDWithLock()
		assert.NoError(t, err)
		assert.NotNil(t, lockFile2)

		err = z2.ReleasePIDLock(lockFile2)
		assert.NoError(t, err)
	})
}

func TestForceWritePID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zero_test_force")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pidFile := tmpDir + "/test.pid"

	t.Run("force write ignores lock", func(t *testing.T) {
		z1 := New(Options{PIDFile: pidFile}) // Lock holder
		z2 := New(Options{PIDFile: pidFile}) // Force writer

		// 1. z1 acquires lock
		lockFile1, err := z1.WritePIDWithLock()
		assert.NoError(t, err)
		assert.NotNil(t, lockFile1)

		// 2. z2 writes with lock (should fail)
		_, err = z2.WritePIDWithLock()
		assert.Error(t, err)

		// 3. z2 force writes (should succeed)
		err = z2.ForceWritePID()
		assert.NoError(t, err)

		// 4. Verify PID file was updated (implicitly check content if we could, but err check is good)
		pid, err := z2.GetPID()
		assert.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)

		// Cleanup
		z1.ReleasePIDLock(lockFile1)
	})
}

func TestReleasePIDLock(t *testing.T) {
	t.Run("release nil file", func(t *testing.T) {
		z := New(Options{})
		err := z.ReleasePIDLock(nil)
		assert.NoError(t, err)
	})
}

func TestQuitWithRemovePIDFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zero_test_quit")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidFile := tmpDir + "/test.pid"

	t.Run("PID file removed after process terminates", func(t *testing.T) {
		mp := &mockProcess{pid: 123}
		z := New(Options{
			PIDFile:    pidFile,
			QuitTimout: 1 * time.Second,
		})
		z.processFinder = &mockProcessFinder{proc: mp}

		// Write PID file
		err := os.WriteFile(pidFile, []byte("123"), 0600)
		assert.NoError(t, err)

		// Verify PID file exists before quit
		_, err = os.Stat(pidFile)
		assert.NoError(t, err)

		// Quit with removePIDFile=true
		err = z.Quit(context.Background(), 123, true)
		assert.NoError(t, err)

		// Verify PID file is removed after quit
		_, err = os.Stat(pidFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("PID file not removed when removePIDFile is false", func(t *testing.T) {
		mp := &mockProcess{pid: 123}
		z := New(Options{
			PIDFile:    pidFile,
			QuitTimout: 1 * time.Second,
		})
		z.processFinder = &mockProcessFinder{proc: mp}

		// Write PID file
		err := os.WriteFile(pidFile, []byte("123"), 0600)
		assert.NoError(t, err)

		// Quit with removePIDFile=false
		err = z.Quit(context.Background(), 123, false)
		assert.NoError(t, err)

		// Verify PID file still exists
		_, err = os.Stat(pidFile)
		assert.NoError(t, err)
	})

	t.Run("no error when PID file already deleted", func(t *testing.T) {
		mp := &mockProcess{pid: 123}
		z := New(Options{
			PIDFile:    pidFile + "_nonexistent",
			QuitTimout: 1 * time.Second,
		})
		z.processFinder = &mockProcessFinder{proc: mp}

		// Quit with removePIDFile=true on non-existent file
		err := z.Quit(context.Background(), 123, true)
		assert.NoError(t, err)
	})
}

func TestUpgrade(t *testing.T) {
	t.Run("upgrade success", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "zero_test_upgrade")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		sockPath := tmpDir + "/test.sock"
		z := New(Options{UpgradeSock: sockPath})

		// Start a mock server to accept connections
		listener, err := net.Listen("unix", sockPath)
		assert.NoError(t, err)
		defer listener.Close()

		go func() {
			conn, _ := listener.Accept()
			if conn != nil {
				conn.Close()
			}
		}()

		// Give time for listener to start
		assert.Eventually(t, func() bool {
			conn, err := net.Dial("unix", sockPath)
			if err == nil {
				conn.Close()
				return true
			}
			return false
		}, 2*time.Second, 10*time.Millisecond, "Mock listener failed to start")

		err = z.Upgrade()
		assert.NoError(t, err)
	})

	t.Run("upgrade failure no socket", func(t *testing.T) {
		z := New(Options{UpgradeSock: "/nonexistent/path.sock"})
		err := z.Upgrade()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to upgrade socket")
	})
}

func TestRemoveUpgradeSock(t *testing.T) {
	t.Run("remove existing socket", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "zero_test_remove")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		sockPath := tmpDir + "/test.sock"
		z := New(Options{UpgradeSock: sockPath})

		// Create a regular file to simulate socket (since closing a unix listener removes the socket)
		err = os.WriteFile(sockPath, []byte{}, 0600)
		assert.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(sockPath)
		assert.NoError(t, err)

		// Remove socket
		err = z.RemoveUpgradeSock()
		assert.NoError(t, err)

		// Verify file is removed
		_, err = os.Stat(sockPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("remove non-existent socket", func(t *testing.T) {
		z := New(Options{UpgradeSock: "/nonexistent/path.sock"})
		err := z.RemoveUpgradeSock()
		assert.NoError(t, err)
	})
}

func TestListenerWithConfig(t *testing.T) {
	t.Run("listener with custom config", func(t *testing.T) {
		z := New(Options{})

		config := &net.ListenConfig{}
		listenOptions := &ListenerOptions{
			Config:  config,
			Network: "tcp",
			Address: "localhost:0",
		}

		l, err := z.Listener(context.Background(), listenOptions)
		assert.NoError(t, err)
		assert.NotNil(t, l)
		defer l.Close()
	})

	t.Run("listener cache hit same key", func(t *testing.T) {
		z := New(Options{})

		// Create first listener with specific address pattern
		listenOptions := &ListenerOptions{
			Network: "tcp",
			Address: "localhost:0",
		}

		l1, err := z.Listener(context.Background(), listenOptions)
		assert.NoError(t, err)
		defer l1.Close()

		// Request listener with same Address key (cache key is Address, not actual bound port)
		// Since localhost:0 is the key, requesting it again should return the cached one
		l2, err := z.Listener(context.Background(), listenOptions)
		assert.NoError(t, err)

		// Should be the same listener (cache hit based on Key)
		assert.Equal(t, l1.Addr().String(), l2.Addr().String())
	})
}

func TestGetPIDErrors(t *testing.T) {
	t.Run("invalid PID content", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "zero_test_pid")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		pidFile := tmpDir + "/test.pid"
		z := New(Options{PIDFile: pidFile})

		// Write invalid PID content
		err = os.WriteFile(pidFile, []byte("not-a-number"), 0600)
		assert.NoError(t, err)

		_, err = z.GetPID()
		assert.Error(t, err)
	})

	t.Run("file does not exist", func(t *testing.T) {
		z := New(Options{PIDFile: "/nonexistent/path.pid"})
		_, err := z.GetPID()
		assert.Error(t, err)
	})
}

func TestValidatePIDFileErrors(t *testing.T) {
	t.Run("FindProcess error", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "zero_test_validate_err")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		pidFile := tmpDir + "/test.pid"
		z := New(Options{PIDFile: pidFile})

		// Mock FindProcess to return an error
		z.processFinder = &mockProcessFinderWithError{}

		// Write a valid PID
		err = os.WriteFile(pidFile, []byte("12345"), 0600)
		assert.NoError(t, err)

		_, _, err = z.ValidatePIDFile()
		assert.Error(t, err)
	})
}

type mockProcessFinderWithError struct{}

func (m *mockProcessFinderWithError) FindProcess(pid int) (process, error) {
	return nil, errors.New("mock FindProcess error")
}

func TestQuitErrors(t *testing.T) {
	t.Run("FindProcess error", func(t *testing.T) {
		z := New(Options{QuitTimout: 1 * time.Second})
		z.processFinder = &mockProcessFinderWithError{}

		err := z.Quit(context.Background(), 123, false)
		assert.Error(t, err)
	})

	t.Run("Signal error", func(t *testing.T) {
		mp := &mockProcess{pid: 123, err: errors.New("signal error")}
		z := New(Options{QuitTimout: 1 * time.Second})
		z.processFinder = &mockProcessFinder{proc: mp}

		err := z.Quit(context.Background(), 123, false)
		assert.Error(t, err)
	})
}

func TestWaitForUpgradeStateError(t *testing.T) {
	t.Run("already in waiting state", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "zero_test_wait")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		z := New(Options{
			UpgradeSock: tmpDir + "/test.sock",
		})

		// Manually set state to waiting
		z.state = waitingState

		err = z.WaitForUpgrade(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state is not default")
	})
}

func TestWritePIDWithLockErrors(t *testing.T) {
	t.Run("directory creation", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "zero_test_lock_dir")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Use a nested path that doesn't exist
		pidFile := tmpDir + "/subdir/test.pid"
		z := New(Options{PIDFile: pidFile})

		lockFile, err := z.WritePIDWithLock()
		assert.NoError(t, err)
		assert.NotNil(t, lockFile)

		// Verify directory was created
		_, err = os.Stat(tmpDir + "/subdir")
		assert.NoError(t, err)

		z.ReleasePIDLock(lockFile)
	})
}

func TestCloseWithWaitingState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zero_test_close")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	z := New(Options{
		UpgradeSock: tmpDir + "/test.sock",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start WaitForUpgrade in background
	go func() {
		_ = z.WaitForUpgrade(ctx)
	}()

	// Wait for state to change
	assert.Eventually(t, func() bool {
		return z.IsWaiting()
	}, 1*time.Second, 10*time.Millisecond, "Zero failed to enter waiting state")

	// Close should work even in waiting state
	err = z.Close(ctx)
	assert.NoError(t, err)
}

func TestWritePIDDirectoryCreation(t *testing.T) {
	t.Run("creates directory if not exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "zero_test_writepid")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Use a nested path that doesn't exist
		pidFile := tmpDir + "/subdir/nested/test.pid"
		z := New(Options{PIDFile: pidFile})

		err = z.writePID()
		assert.NoError(t, err)

		// Verify directories were created
		_, err = os.Stat(tmpDir + "/subdir/nested")
		assert.NoError(t, err)

		// Verify PID file was written
		pid, err := z.GetPID()
		assert.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)
	})
}

func TestQuitProcessTerminatesAfterSIGKILL(t *testing.T) {
	t.Run("process terminates after SIGKILL", func(t *testing.T) {
		// Create a mock process that:
		// 1. Accepts SIGTERM but doesn't die
		// 2. Responds to Signal(0) indicating still alive for a few checks
		// 3. Then Kill() is called
		// 4. After Kill(), Signal(0) returns ErrProcessDone
		mp := &mockProcessWithKillBehavior{
			killCount:      0,
			signalCount:    0,
			dieAfterKill:   true,
			surviveSignals: 2, // survive 2 Signal(0) checks before timeout
		}

		z := New(Options{
			QuitTimout: 2 * time.Second, // Short timeout to speed up test
		})
		z.processFinder = &mockProcessFinder{proc: mp}

		err := z.Quit(context.Background(), 123, false)
		assert.NoError(t, err)
		assert.True(t, mp.killCalled)
	})
}

type mockProcessWithKillBehavior struct {
	killCount      int
	signalCount    int
	dieAfterKill   bool
	surviveSignals int
	killCalled     bool
}

func (m *mockProcessWithKillBehavior) Signal(sig os.Signal) error {
	if sig == syscall.Signal(0) {
		m.signalCount++
		if m.killCalled {
			return os.ErrProcessDone
		}
		if m.signalCount > m.surviveSignals {
			// Still alive until killed
			return nil
		}
		return nil
	}
	return nil
}

func (m *mockProcessWithKillBehavior) Kill() error {
	m.killCalled = true
	return nil
}

func (m *mockProcessWithKillBehavior) Wait() (*os.ProcessState, error) {
	return nil, nil
}

func (m *mockProcessWithKillBehavior) Release() error {
	return nil
}

func TestListenerCreateError(t *testing.T) {
	t.Run("listener creation error with invalid address", func(t *testing.T) {
		z := New(Options{})

		listenOptions := &ListenerOptions{
			Network: "tcp",
			Address: "invalid:address:format:99999",
		}

		_, err := z.Listener(context.Background(), listenOptions)
		assert.Error(t, err)
	})
}
