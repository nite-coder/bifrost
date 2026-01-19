package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogRotation_InodeVerification verifies that stdout and stderr correctly point
// to the new log file (Inode change) after rotation.
func TestLogRotation_InodeVerification(t *testing.T) {
	if os.Getenv("GO_WANT_LOG_HELPER") == "1" {
		runLogRotationTest(t)
		return
	}

	// Run the test in a separate process to avoid global state interference
	// (like os.Stdout redirection and signal handlers) with other tests.
	cmd := exec.Command(os.Args[0], "-test.run=TestLogRotation_InodeVerification")
	cmd.Env = append(os.Environ(), "GO_WANT_LOG_HELPER=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Test failed in helper process: %v\nOutput: %s", err, string(out))
	}
}

func runLogRotationTest(t *testing.T) {
	logDir := t.TempDir()
	logFile := filepath.Join(logDir, "bifrost.log")

	// 1. Initialize Logger and redirect Stdout/Stderr
	opts := config.LoggingOtions{
		Output:                   logFile,
		Level:                    "info",
		DisableRedirectStdStream: false,
	}
	logger, err := log.NewLogger(opts)
	require.NoError(t, err)
	_ = logger

	// Get initial Inode
	fi, err := os.Stat(logFile)
	require.NoError(t, err)
	initialInode := fi.Sys().(*syscall.Stat_t).Ino

	// 2. Simulate external action: rename current log file (like logrotate does)
	rotatedFile := logFile + ".1"
	err = os.Rename(logFile, rotatedFile)
	require.NoError(t, err)

	// 3. Send SIGUSR1 to trigger rotation
	process, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	err = process.Signal(syscall.SIGUSR1)
	require.NoError(t, err)

	// 4. Verify new log file is created and Inode has changed
	assert.Eventually(t, func() bool {
		fi, err := os.Stat(logFile)
		if err != nil {
			return false
		}
		return fi.Sys().(*syscall.Stat_t).Ino != initialInode
	}, 2*time.Second, 100*time.Millisecond, "Log file should be recreated with new Inode")

	// 5. Verify Stdout/Stderr output is indeed written to the NEW log file
	assert.Eventually(t, func() bool {
		fmt.Fprintln(os.Stdout, "TEST_STDOUT_PAYLOAD")
		fmt.Fprintln(os.Stderr, "TEST_STDERR_PAYLOAD")

		content, err := os.ReadFile(logFile)
		if err != nil {
			return false
		}
		s := string(content)
		return assert.Contains(t, s, "TEST_STDOUT_PAYLOAD") &&
			assert.Contains(t, s, "TEST_STDERR_PAYLOAD")
	}, 5*time.Second, 200*time.Millisecond, "New log file should contain stdout/stderr payloads")
}

func TestMaster_SignalForwarding(t *testing.T) {
	// Swap execCommandContext for mocking
	oldExec := execCommandContext
	execCommandContext = fakeExecCommandContext
	defer func() { execCommandContext = oldExec }()

	m := NewMaster(&MasterOptions{
		GracefulTimeout: 100 * time.Millisecond,
	})
	// Use unique socket path
	socketPath := filepath.Join(t.TempDir(), "signal_test.sock")
	m.controlPlane = NewControlPlane(&ControlPlaneOptions{SocketPath: socketPath})

	// Run Master in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- m.Run(ctx)
	}()

	// Wait for Master to start and be running
	assert.Eventually(t, func() bool {
		return m.State() == MasterStateRunning
	}, 2*time.Second, 100*time.Millisecond, "Master should be in running state")

	// Get Worker PID
	workerPID := m.WorkerPID()
	require.NotZero(t, workerPID, "Worker PID should not be zero")

	// Simulate worker readiness
	conn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	json.NewEncoder(conn).Encode(&ControlMessage{
		Type:      MessageTypeReady,
		WorkerPID: workerPID,
	})
	conn.Close()

	// Wait for Master to be truly ready-ready
	assert.Eventually(t, func() bool {
		return m.WorkerPID() == workerPID
	}, 2*time.Second, 100*time.Millisecond)

	// Send SIGUSR1 to Master
	process, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	err = process.Signal(syscall.SIGUSR1)
	require.NoError(t, err)

	// Verify Master handles the signal without crashing
	assert.Eventually(t, func() bool {
		return m.State() == MasterStateRunning
	}, 500*time.Millisecond, 50*time.Millisecond, "Master should remain running after SIGUSR1")

	// Cleanup: Stop Master cleanly
	_ = m.Shutdown(context.Background())
	cancel()
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Error("Master.Run did not exit")
	}
}

// TestLogHelperProcess is a mock worker that reports its actions
func TestLogHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_LOG_HELPER") != "1" {
		return
	}
	fmt.Fprintln(os.Stdout, "LOG_HELPER_READY")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGTERM)

	for {
		sig := <-sigCh
		if sig == syscall.SIGTERM {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "RECEIVED_SIGNAL_%v\n", sig)
	}
}

func fakeLogHelperExec(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestLogHelperProcess", "--"}
	cs = append(cs, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	cmd.Env = append(os.Environ(), "GO_WANT_LOG_HELPER=1")
	return cmd
}
