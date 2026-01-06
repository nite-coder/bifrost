package zero

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
)

// Environment variable used to identify worker processes.
const EnvBifrostRole = "BIFROST_ROLE"

// RoleWorker is the value of BIFROST_ROLE for worker processes.
const RoleWorker = "worker"

// MasterState represents the current state of the Master process.
type MasterState int8

const (
	// MasterStateIdle indicates the master is idle and ready to spawn a worker.
	MasterStateIdle MasterState = iota
	// MasterStateRunning indicates the master has an active worker.
	MasterStateRunning
	// MasterStateReloading indicates a hot reload is in progress.
	MasterStateReloading
	// MasterStateShuttingDown indicates graceful shutdown is in progress.
	MasterStateShuttingDown
)

// String returns the string representation of MasterState.
func (s MasterState) String() string {
	switch s {
	case MasterStateIdle:
		return "idle"
	case MasterStateRunning:
		return "running"
	case MasterStateReloading:
		return "reloading"
	case MasterStateShuttingDown:
		return "shutting_down"
	default:
		return "unknown"
	}
}

// MasterOptions contains configuration for the Master process.
type MasterOptions struct {
	// PIDFile is the path to store the Master's PID.
	PIDFile string
	// ConfigPath is the path to the configuration file passed to Worker.
	ConfigPath string
	// Binary is the path to the executable (defaults to os.Args[0]).
	Binary string
	// Args are additional arguments passed to the Worker process.
	Args []string
	// GracefulTimeout is the maximum time to wait for Worker graceful shutdown.
	GracefulTimeout time.Duration
	// KeepAlive is the KeepAlive strategy configuration.
	KeepAlive *KeepAliveOptions
}

// Master manages the lifecycle of Worker processes.
// It provides PID stability for process managers (Systemd, Supervisor)
// and handles signal forwarding, hot reload, and worker monitoring.
type Master struct {
	options        *MasterOptions
	currentWorker  *exec.Cmd
	controlPlane   *ControlPlane
	keepAlive      *KeepAlive
	state          MasterState
	mu             sync.RWMutex
	stopCh         chan struct{}
	workerDoneCh   chan struct{}
	readyCh        chan struct{} // Signaled when worker is ready
	listenerFDs    []*os.File    // Inherited listener FDs for reload
	listenerKeys   []string      // Listener keys mapped to FDs
	listenerDataCh chan *listenerData
}

// listenerData holds the FDs and their keys (addresses)
type listenerData struct {
	fds  []*os.File
	keys []string
}

// ErrMasterShuttingDown is returned when operations are attempted during shutdown.
var ErrMasterShuttingDown = errors.New("master is shutting down")

// NewMaster creates a new Master instance with the given options.
func NewMaster(opts *MasterOptions) *Master {
	if opts == nil {
		opts = &MasterOptions{}
	}
	if opts.Binary == "" {
		opts.Binary = os.Args[0]
	}
	if opts.GracefulTimeout <= 0 {
		opts.GracefulTimeout = 30 * time.Second
	}

	return &Master{
		options:        opts,
		keepAlive:      NewKeepAlive(opts.KeepAlive),
		controlPlane:   NewControlPlane(nil),
		state:          MasterStateIdle,
		stopCh:         make(chan struct{}),
		workerDoneCh:   make(chan struct{}, 1),
		readyCh:        make(chan struct{}, 1),
		listenerDataCh: make(chan *listenerData, 1),
	}
}

// Run starts the Master's main loop.
// It spawns the initial Worker and waits for signals:
//   - SIGHUP: triggers hot reload (spawn new Worker, gracefully stop old)
//   - SIGTERM/SIGINT: triggers graceful shutdown of Worker and Master
//
// This method blocks until shutdown is complete.
func (m *Master) Run(ctx context.Context) error {
	// Setup control plane
	if err := m.controlPlane.Listen(); err != nil {
		return fmt.Errorf("failed to start control plane: %w", err)
	}
	defer m.controlPlane.Close()

	// Setup message handler
	m.controlPlane.SetMessageHandler(m.handleControlMessage)
	m.controlPlane.SetFDHandler(m.handleFDTransfer)

	// Start accepting control plane connections
	go safety.Go(ctx, func() {
		if err := m.controlPlane.Accept(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("control plane accept loop exited", "error", err)
		}
	})

	// Spawn initial worker
	if err := m.spawnAndWatch(ctx); err != nil {
		return fmt.Errorf("failed to spawn initial worker: %w", err)
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)

	slog.Info("master started",
		"pid", os.Getpid(),
		"workerPID", m.WorkerPID(),
	)

	// Write PID file
	if err := m.writePIDFile(); err != nil {
		slog.Error("failed to write PID file", "error", err)
		return err
	}
	defer m.removePIDFile()

	// Notify parent process that daemon is ready (for Type=forking)
	// This allows the parent to exit and Systemd to consider startup complete
	if err := NotifyDaemonReady(); err != nil {
		slog.Warn("failed to notify daemon ready", "error", err)
	}

	for {
		select {

		case <-ctx.Done():
			return m.Shutdown(ctx)

		case <-m.stopCh:
			return nil

		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				slog.Info("received SIGHUP, triggering hot reload")
				if err := m.handleReload(ctx); err != nil {
					slog.Error("hot reload failed", "error", err)
				}

			case syscall.SIGTERM, syscall.SIGINT:
				slog.Info("received shutdown signal", "signal", sig)
				return m.Shutdown(ctx)
			}

		case <-m.workerDoneCh:
			// Worker exited unexpectedly
			if m.State() == MasterStateShuttingDown {
				continue
			}

			shouldRestart, backoff, err := m.keepAlive.ShouldRestart()
			if err != nil {
				slog.Error("restart limit exceeded, master exiting", "error", err)
				return err
			}

			if shouldRestart {
				slog.Warn("worker exited unexpectedly, restarting",
					"backoff", backoff,
				)
				m.keepAlive.RecordRestart()
				time.Sleep(backoff)

				if err := m.spawnAndWatch(ctx); err != nil {
					slog.Error("failed to restart worker", "error", err)
				}
			}
		}
	}
}

// Shutdown gracefully stops the Master and its Worker.
func (m *Master) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	if m.state == MasterStateShuttingDown {
		m.mu.Unlock()
		return nil
	}
	m.state = MasterStateShuttingDown
	m.mu.Unlock()

	slog.Info("master shutting down")

	// Send SIGTERM to worker
	if m.currentWorker != nil && m.currentWorker.Process != nil {
		if err := m.currentWorker.Process.Signal(syscall.SIGTERM); err != nil {
			if !errors.Is(err, os.ErrProcessDone) {
				slog.Error("failed to send SIGTERM to worker", "error", err)
			}
		}

		// Wait for worker to exit with timeout
		done := make(chan struct{})
		go safety.Go(ctx, func() {
			_ = m.currentWorker.Wait()
			close(done)
		})

		select {
		case <-done:
			slog.Info("worker exited gracefully")
		case <-time.After(m.options.GracefulTimeout):
			slog.Warn("worker graceful timeout exceeded, sending SIGKILL")
			_ = m.currentWorker.Process.Kill()
		}
	}

	close(m.stopCh)
	return nil
}

// writePIDFile writes the current process ID to the configured PID file.
func (m *Master) writePIDFile() error {
	pidFile := m.options.PIDFile
	if pidFile == "" {
		pidFile = "./logs/bifrost.pid"
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		return fmt.Errorf("failed to create PID file directory: %w", err)
	}

	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0600); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	slog.Info("PID file written", "path", pidFile, "pid", pid)
	m.options.PIDFile = pidFile // Update option with actual path
	return nil
}

// removePIDFile removes the PID file.
func (m *Master) removePIDFile() {
	pidFile := m.options.PIDFile
	if pidFile == "" {
		return
	}
	_ = os.Remove(pidFile)
	slog.Info("PID file removed", "path", pidFile)
}

// State returns the current state of the Master.
func (m *Master) State() MasterState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// WorkerPID returns the PID of the current Worker process.
// Returns 0 if no Worker is running.
func (m *Master) WorkerPID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.currentWorker == nil || m.currentWorker.Process == nil {
		return 0
	}
	return m.currentWorker.Process.Pid
}

// spawnAndWatch spawns a new worker and starts watching it.
func (m *Master) spawnAndWatch(ctx context.Context) error {
	cmd, err := m.spawnWorker(ctx, m.listenerFDs, m.listenerKeys) // Pass stored keys
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.currentWorker = cmd
	m.state = MasterStateRunning
	m.mu.Unlock()

	// Start watching the worker
	go safety.Go(ctx, func() {
		m.watchWorker(cmd)
	})

	return nil
}

// spawnWorker starts a new Worker process with inherited file descriptors and keys.
func (m *Master) spawnWorker(ctx context.Context, extraFiles []*os.File, keys []string) (*exec.Cmd, error) {
	args := m.options.Args
	if m.options.ConfigPath != "" {
		args = append([]string{"-c", m.options.ConfigPath}, args...)
	}

	cmd := exec.CommandContext(ctx, m.options.Binary, args...)

	// Set worker role via environment variable
	// Note: Socket path is base64 encoded because Abstract Namespace contains NUL byte
	socketPathEncoded := base64.StdEncoding.EncodeToString([]byte(m.controlPlane.SocketPath()))
	cmd.Env = append(os.Environ(),
		EnvBifrostRole+"="+RoleWorker,
		"BIFROST_CONTROL_SOCKET="+socketPathEncoded,
	)

	// Inherit stdout/stderr from Master (for log aggregation)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Pass listener FDs for zero-downtime reload
	if len(extraFiles) > 0 {
		cmd.ExtraFiles = extraFiles
		cmd.Env = append(cmd.Env, "UPGRADE=1")
		if len(keys) > 0 {
			// Encode keys as a comma-separated string, then base64 encode it
			keysStr := strings.Join(keys, ",")
			encodedKeys := base64.StdEncoding.EncodeToString([]byte(keysStr))
			cmd.Env = append(cmd.Env, "BIFROST_LISTENER_KEYS="+encodedKeys)
		}
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	slog.Info("spawned new worker",
		"workerPID", cmd.Process.Pid,
		"extraFiles", len(extraFiles),
		"keys", keys,
	)

	return cmd, nil
}

// watchWorker monitors the Worker process and handles unexpected exits.
func (m *Master) watchWorker(cmd *exec.Cmd) {
	state, err := cmd.Process.Wait()

	m.mu.RLock()
	currentWorker := m.currentWorker
	m.mu.RUnlock()

	// Only notify if this is still the current worker
	if currentWorker == cmd {
		exitCode := -1
		if state != nil {
			exitCode = state.ExitCode()
		}

		slog.Info("worker exited",
			"pid", cmd.Process.Pid,
			"exitCode", exitCode,
			"error", err,
		)

		select {
		case m.workerDoneCh <- struct{}{}:
		default:
		}
	}
}

// handleReload performs the hot reload sequence:
//  1. Request FDs from current Worker via ControlPlane
//  2. Spawn new Worker with FDs
//  3. Wait for new Worker ready signal
//  4. Send SIGTERM to old Worker
func (m *Master) handleReload(ctx context.Context) error {
	m.mu.Lock()
	if m.state != MasterStateRunning {
		m.mu.Unlock()
		return fmt.Errorf("cannot reload in state: %s", m.state)
	}
	m.state = MasterStateReloading
	oldWorker := m.currentWorker
	oldWorkerPID := 0
	if oldWorker != nil && oldWorker.Process != nil {
		oldWorkerPID = oldWorker.Process.Pid
	}
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		if m.state == MasterStateReloading {
			m.state = MasterStateRunning
		}
		m.mu.Unlock()
	}()

	slog.Info("starting hot reload", "oldWorkerPID", oldWorkerPID)

	// Step 1: Request FDs from old worker
	var fds []*os.File
	var keys []string
	if oldWorkerPID > 0 {
		// Clear any stale data
		select {
		case <-m.listenerDataCh:
		default:
		}

		if err := m.controlPlane.SendMessage(oldWorkerPID, &ControlMessage{
			Type: MessageTypeFDRequest,
		}); err != nil {
			slog.Warn("failed to request FDs from old worker", "error", err)
			// Continue without FDs - new worker will create new listeners
		} else {
			// Wait for FDs via channel
			select {
			case data := <-m.listenerDataCh:
				fds = data.fds
				keys = data.keys
			case <-time.After(5 * time.Second):
				slog.Warn("timeout waiting for FDs from old worker")
			}
		}
	}

	// Step 2: Spawn new worker with FDs
	m.listenerFDs = fds
	m.listenerKeys = keys
	newCmd, err := m.spawnWorker(ctx, fds, keys)
	if err != nil {
		return fmt.Errorf("failed to spawn new worker: %w", err)
	}

	// Step 3: Wait for new worker ready signal (with timeout)
	readyTimeout := 30 * time.Second
	select {
	case <-m.readyCh:
		slog.Info("new worker is ready", "newWorkerPID", newCmd.Process.Pid)
	case <-time.After(readyTimeout):
		slog.Warn("new worker ready timeout, proceeding anyway")
	}

	// Step 4: Update current worker and stop old one
	m.mu.Lock()
	m.currentWorker = newCmd
	m.mu.Unlock()

	// Start watching new worker
	go safety.Go(ctx, func() {
		m.watchWorker(newCmd)
	})

	// Gracefully stop old worker
	if oldWorker != nil && oldWorker.Process != nil {
		slog.Info("stopping old worker", "oldWorkerPID", oldWorkerPID)
		if err := oldWorker.Process.Signal(syscall.SIGTERM); err != nil {
			if !errors.Is(err, os.ErrProcessDone) {
				slog.Error("failed to send SIGTERM to old worker", "error", err)
			}
		}
	}

	// Reset keepalive on successful reload
	m.keepAlive.Reset()

	slog.Info("hot reload completed",
		"oldWorkerPID", oldWorkerPID,
		"newWorkerPID", newCmd.Process.Pid,
	)

	return nil
}

// handleControlMessage handles messages from workers.
func (m *Master) handleControlMessage(conn net.Conn, msg *ControlMessage) {
	switch msg.Type {
	case MessageTypeRegister:
		slog.Debug("worker registered", "workerPID", msg.WorkerPID)

	case MessageTypeReady:
		slog.Debug("worker ready", "workerPID", msg.WorkerPID)
		select {
		case m.readyCh <- struct{}{}:
		default:
		}

	case MessageTypeFDRequest, MessageTypeShutdown:
		// Not handled by master
	default:
		slog.Debug("master ignored message", "type", msg.Type)
	}
}

// handleFDTransfer handles transferred FDs from a worker.
func (m *Master) handleFDTransfer(fds []*os.File, keys []string) {
	select {
	case m.listenerDataCh <- &listenerData{fds: fds, keys: keys}:
	default:
		slog.Warn("received FDs but no reload in progress or channel full", "count", len(fds))
		for _, f := range fds {
			_ = f.Close()
		}
	}
}

// ControlPlane returns the control plane instance.
func (m *Master) ControlPlane() *ControlPlane {
	return m.controlPlane
}

// IsWorker returns true if the current process is a worker.
func IsWorker() bool {
	return os.Getenv(EnvBifrostRole) == RoleWorker
}

// GetControlSocketPath returns the control socket path from environment.
// The path is base64 encoded to handle Abstract Namespace NUL bytes.
func GetControlSocketPath() string {
	encoded := os.Getenv("BIFROST_CONTROL_SOCKET")
	if encoded == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	return string(decoded)
}
