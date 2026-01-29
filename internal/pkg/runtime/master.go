package runtime

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
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/blackbear/pkg/cast"
)

// Allow mocking for tests
var (
	lookupUser  = user.Lookup
	lookupGroup = user.LookupGroup
)

// execCommandContext allows mocking exec.CommandContext in tests.
var execCommandContext = exec.CommandContext

// startCommand allows mocking cmd.Start in tests.
var startCommand = func(cmd *exec.Cmd) error {
	return cmd.Start()
}

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
	// User to run worker as
	User string
	// Group to run worker as
	Group string
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
	workerDoneCh   chan struct{} // Signals when a worker exits
	workerExitCh   chan struct{} // Closed when the current worker process has finished waiting
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
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT, syscall.SIGUSR1)
	defer signal.Stop(sigCh)

	slog.Info("master started",
		"pid", os.Getpid(),
		"workerPID", m.WorkerPID(),
	)

	// Wait for worker to be ready
	// This ensures we digest the initial "ready" signal so it doesn't linger for hot reload.
	select {
	case <-m.readyCh:
		slog.Info("initial worker is ready")
	case <-time.After(30 * time.Second):
		slog.Warn("initial worker ready timeout, proceeding anyway")
	}

	// Notify systemd that service is ready (for Type=notify)
	// This is called after Worker is spawned and control plane is ready
	NotifySystemdReady()

	for {
		select {

		case <-ctx.Done():
			return m.Shutdown(ctx)

		case <-m.stopCh:
			return nil

		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				slog.Log(ctx, log.LevelNotice, "received SIGHUP, triggering hot reload")
				if err := m.handleReload(ctx); err != nil {
					slog.Error("hot reload failed", "error", err)
				}

			case syscall.SIGTERM, syscall.SIGINT:
				slog.Info("received shutdown signal", "signal", sig)
				return m.Shutdown(ctx)

			case syscall.SIGUSR1:
				slog.Info("received SIGUSR1, forwarding to worker for log rotation")
				// Note: Master's own log rotation is handled automatically by pkg/log
				// which also listens for SIGUSR1 in a background goroutine.
				m.mu.RLock()
				worker := m.currentWorker
				m.mu.RUnlock()

				if worker != nil && worker.Process != nil {
					if err := worker.Process.Signal(syscall.SIGUSR1); err != nil {
						slog.Error("failed to forward SIGUSR1 to worker", "error", err, "workerPID", worker.Process.Pid)
					}
				}
			}

		case <-m.workerDoneCh:
			// Worker exited unexpectedly
			if m.State() == MasterStateShuttingDown {
				continue
			}

			// If we are reloading, the old worker is expected to exit (or if it crashed early,
			// the new worker is already on the way). We shouldn't restart the old one.
			if m.State() == MasterStateReloading {
				slog.Info("worker exited during hot reload (expected or ignored)", "pid", m.WorkerPID())
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

				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return m.Shutdown(ctx)
				case <-m.stopCh:
					return nil
				}

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
		// We use workerExitCh to wait since watchWorker owns the Wait() call
		done := m.workerExitCh

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
	m.mu.Lock()
	exitCh := make(chan struct{})
	m.workerExitCh = exitCh
	m.mu.Unlock()

	go safety.Go(ctx, func() {
		m.watchWorker(cmd, exitCh)
	})

	return nil
}

// spawnWorker starts a new Worker process with inherited file descriptors and keys.
func (m *Master) spawnWorker(ctx context.Context, extraFiles []*os.File, keys []string) (*exec.Cmd, error) {
	args := m.options.Args
	if m.options.ConfigPath != "" {
		args = append([]string{"-c", m.options.ConfigPath}, args...)
	}

	cmd := execCommandContext(ctx, m.options.Binary, args...)

	// Handle User/Group switching
	if m.options.User != "" || m.options.Group != "" {
		sysProcAttr := &syscall.SysProcAttr{}
		if cmd.SysProcAttr != nil {
			sysProcAttr = cmd.SysProcAttr
		}

		uid, err := cast.ToUint32(os.Getuid())
		if err != nil {
			return nil, fmt.Errorf("failed to cast uid: %w", err)
		}
		gid, err := cast.ToUint32(os.Getgid())
		if err != nil {
			return nil, fmt.Errorf("failed to cast gid: %w", err)
		}
		foundUser := false

		if m.options.User != "" {
			u, err := lookupUser(m.options.User)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup user '%s': %w", m.options.User, err)
			}
			u64, err := strconv.ParseUint(u.Uid, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse uid '%s': %w", u.Uid, err)
			}
			uid = uint32(u64)

			// If group is not specified, use user's primary group
			if m.options.Group == "" {
				g64, err := strconv.ParseUint(u.Gid, 10, 32)
				if err != nil {
					return nil, fmt.Errorf("failed to parse gid '%s': %w", u.Gid, err)
				}
				gid = uint32(g64)
			}
			foundUser = true
		}

		if m.options.Group != "" {
			g, err := lookupGroup(m.options.Group)
			if err != nil {
				return nil, fmt.Errorf("failed to lookup group '%s': %w", m.options.Group, err)
			}
			g64, err := strconv.ParseUint(g.Gid, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse gid '%s': %w", g.Gid, err)
			}
			gid = uint32(g64)
		}

		// Only set Credential if something actually changed from current process
		// or if we explicitly found a user (even if UID matches, might want to enforce)
		if foundUser || m.options.Group != "" {
			sysProcAttr.Credential = &syscall.Credential{
				Uid: uid,
				Gid: gid,
			}
			cmd.SysProcAttr = sysProcAttr
			slog.Info("configured worker credentials", "uid", uid, "gid", gid)
		}
	}

	// Set worker role via environment variable
	// Note: Socket path is base64 encoded because Abstract Namespace contains NUL byte
	socketPathEncoded := base64.StdEncoding.EncodeToString([]byte(m.controlPlane.SocketPath()))

	// If Env is nil, use current environment.
	// This ensures we respect any Env set by mock execCommandContext while defaulting to system env.
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}

	cmd.Env = append(cmd.Env,
		EnvBifrostRole+"="+RoleWorker,
		"BIFROST_CONTROL_SOCKET="+socketPathEncoded,
	)

	// Inherit stdout/stderr from Master (for log aggregation)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Pass listener FDs for zero-downtime reload
	if len(extraFiles) > 0 {
		cmd.ExtraFiles = extraFiles
		cmd.Env = append(cmd.Env,
			"UPGRADE=1",
			fmt.Sprintf("BIFROST_FD_COUNT=%d", len(extraFiles)),
		)
		if len(keys) > 0 {
			// Encode keys as a comma-separated string, then base64 encode it
			keysStr := strings.Join(keys, ",")
			encodedKeys := base64.StdEncoding.EncodeToString([]byte(keysStr))
			cmd.Env = append(cmd.Env, "BIFROST_LISTENER_KEYS="+encodedKeys)
		}
	}

	if err := startCommand(cmd); err != nil {
		return nil, fmt.Errorf("failed to start worker: %w", err)
	}

	slog.Debug("spawned new worker",
		"workerPID", cmd.Process.Pid,
		"extraFiles", len(extraFiles),
		"keys", keys,
	)

	return cmd, nil
}

// watchWorker monitors the Worker process and handles unexpected exits.
func (m *Master) watchWorker(cmd *exec.Cmd, exitCh chan struct{}) {
	// Disable signal forwarding to this child
	// We handle signals manually in Master and forward them if needed
	// (Actually exec.Cmd doesn't automatically forward signals unless we utilize specific SysProcAttr)

	err := cmd.Wait()
	close(exitCh)

	m.mu.RLock()
	currentWorker := m.currentWorker
	m.mu.RUnlock()

	// Only notify if this is still the current worker
	if currentWorker == cmd {
		exitCode := -1
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
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

	// Step 2b: Drain any stale ready signals
	select {
	case <-m.readyCh:
		slog.Info("drained stale ready signal")
	default:
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
	// Create new exit channel for the new worker
	m.workerExitCh = make(chan struct{})
	newExitCh := m.workerExitCh
	m.mu.Unlock()

	// Start watching new worker
	go safety.Go(ctx, func() {
		m.watchWorker(newCmd, newExitCh)
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

	slog.Log(ctx, log.LevelNotice, "hot reload completed",
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
		slog.Info("worker ready", "workerPID", msg.WorkerPID)
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
