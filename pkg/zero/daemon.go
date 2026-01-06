package zero

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
)

// DaemonOptions contains configuration for daemonizing the process.
type DaemonOptions struct {
	// PIDFile is the path to store the daemon's PID.
	PIDFile string
	// ReadyTimeout is the maximum time to wait for child ready signal.
	ReadyTimeout time.Duration
	// LogOutput is the path to redirect stdout/stderr (optional).
	LogOutput string
}

// ErrDaemonTimeout is returned when the child doesn't signal ready in time.
var ErrDaemonTimeout = errors.New("daemon child did not signal ready within timeout")

// Daemonize forks the current process as a daemon with Pipe-based synchronization.
// This ensures the parent only exits after the child has:
// 1. Started successfully
// 2. Written the PID file
// 3. Sent a "ready" signal via pipe
//
// This approach guarantees that when Systemd sees ExecStart exit (with Type=forking),
// the PID file is already written and valid.
//
// Returns:
//   - shouldExit: true if parent should exit (daemon started successfully)
//   - error: non-nil if daemonization failed
func Daemonize(opts *DaemonOptions) (bool, error) {
	// Check if we're already the child (daemon)
	if os.Getenv("BIFROST_DAEMONIZED") == "1" {
		// We are the child process.
		// Do not notify yet; wait until Master is fully initialized.
		// FD 3 (pipe) will be used by NotifyDaemonReady later.
		return false, nil
	}

	// We are the parent, spawn child and wait for ready signal
	return spawnDaemonChild(opts)
}

// spawnDaemonChild creates a pipe, forks the child, and waits for ready signal.
func spawnDaemonChild(opts *DaemonOptions) (bool, error) {
	if opts == nil {
		opts = &DaemonOptions{}
	}
	if opts.ReadyTimeout <= 0 {
		opts.ReadyTimeout = 30 * time.Second
	}

	// Create pipe for child->parent communication
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		return false, fmt.Errorf("failed to create pipe: %w", err)
	}
	defer readPipe.Close()

	// Prepare child command with same args
	cmd := exec.CommandContext(context.Background(), os.Args[0], os.Args[1:]...)
	cmd.Env = append(os.Environ(), "BIFROST_DAEMONIZED=1")

	// Pass write end of pipe as FD 3
	cmd.ExtraFiles = []*os.File{writePipe}

	// Detach from terminal
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if opts.LogOutput != "" {
		// Redirect stdout/stderr to file for debugging/logging
		logFile, err := os.OpenFile(opts.LogOutput, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return false, fmt.Errorf("failed to open daemon log file: %w", err)
		}
		// We don't close logFile in parent; child will inherit it.
		// (Actually cmd.Start duplicates it, so we can close it in parent after Start)
		defer logFile.Close()

		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	// Start child
	if err := cmd.Start(); err != nil {
		writePipe.Close()
		return false, fmt.Errorf("failed to start daemon child: %w", err)
	}

	// Close write end in parent (child has it)
	writePipe.Close()

	slog.Debug("spawned daemon child, waiting for ready signal",
		"childPID", cmd.Process.Pid,
		"timeout", opts.ReadyTimeout,
	)

	// Wait for ready signal from child with timeout
	readyCh := make(chan error, 1)
	go safety.Go(context.Background(), func() {
		buf := make([]byte, 5)
		n, err := readPipe.Read(buf)
		if err != nil {
			if err == io.EOF {
				readyCh <- errors.New("child closed pipe without sending ready")
			} else {
				readyCh <- fmt.Errorf("failed to read from pipe: %w", err)
			}
			return
		}
		if string(buf[:n]) == "ready" {
			readyCh <- nil
		} else {
			readyCh <- fmt.Errorf("unexpected message from child: %s", string(buf[:n]))
		}
	})

	select {
	case err := <-readyCh:
		if err != nil {
			// Kill child if it failed to signal ready
			_ = cmd.Process.Kill()
			return false, err
		}
		slog.Info("daemon started successfully", "pid", cmd.Process.Pid)
		return true, nil

	case <-time.After(opts.ReadyTimeout):
		_ = cmd.Process.Kill()
		return false, ErrDaemonTimeout
	}
}

// NotifyDaemonReady should be called by the daemon after it's fully initialized.
// This sends the ready signal to the parent process if running in daemon mode.
func NotifyDaemonReady() error {
	if os.Getenv("BIFROST_DAEMONIZED") != "1" {
		return nil // Not in daemon mode
	}

	pipe := os.NewFile(3, "ready-pipe")
	if pipe == nil {
		return nil // No pipe available
	}
	defer pipe.Close()

	_, err := pipe.WriteString("ready")
	if err != nil {
		return fmt.Errorf("failed to notify parent: %w", err)
	}

	slog.Debug("daemon ready signal sent", "pid", os.Getpid())
	return nil
}

// IsDaemonChild returns true if this process is a daemon child.
func IsDaemonChild() bool {
	return os.Getenv("BIFROST_DAEMONIZED") == "1"
}
