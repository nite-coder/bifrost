package zero

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/bytedance/sonic"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/blackbear/pkg/cast"
	proxyproto "github.com/pires/go-proxyproto"
)

// CommandRunner is an interface for creating exec.Cmd instances.
// It allows for dependency injection in testing scenarios.
type CommandRunner interface {
	Command(name string, arg ...string) *exec.Cmd
}

// EnvGetter is a function type for retrieving environment variables.
type EnvGetter func(string) string

// process is an interface representing an operating system process.
// It abstracts os.Process for testing purposes.
type process interface {
	Signal(os.Signal) error
	Kill() error
	Wait() (*os.ProcessState, error)
	Release() error
}

// ProcessFinder is an interface for finding processes by PID.
// It allows for dependency injection in testing scenarios.
type ProcessFinder interface {
	FindProcess(pid int) (process, error)
}

// FileOpener is a function type for opening files.
type FileOpener func(name string) (*os.File, error)
type defaultCommandRunner struct{}

func (d *defaultCommandRunner) Command(name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(context.TODO(), name, arg...)
}

var defaultEnvGetter = os.Getenv

type defaultProcessFinder struct{}

func (d *defaultProcessFinder) FindProcess(pid int) (process, error) {
	return os.FindProcess(pid)
}

var defaultFileOpener = func(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|syscall.O_CLOEXEC, 0600)
}

// listenInfo holds information about a network listener.
type listenInfo struct {
	listener net.Listener `json:"-"`
	Key      string       `json:"key"`
}

// State represents the current state of the ZeroDownTime instance.
type State int8

const (
	// defaultState indicates the instance is in its initial state.
	defaultState State = iota
	// waitingState indicates the instance is waiting for an upgrade signal.
	waitingState
)

// ZeroDownTime provides zero-downtime restart functionality for server processes.
// It manages PID files, upgrade sockets, and listener inheritance to enable
// seamless process upgrades without dropping connections.
type ZeroDownTime struct {
	command       CommandRunner
	processFinder ProcessFinder
	options       *Options
	stopWaitingCh chan bool
	isShutdownCh  chan bool
	envGetter     EnvGetter
	fileOpener    FileOpener
	listeners     []*listenInfo
	QuitTimeout   time.Duration
	listenerOnce  sync.Once
	mu            sync.Mutex
	state         State
}

// ListenerOptions contains configuration for creating a network listener.
type ListenerOptions struct {
	// Config is an optional net.ListenConfig for customizing the listener.
	Config *net.ListenConfig
	// Network specifies the network type (e.g., "tcp", "tcp4", "tcp6").
	Network string
	// Address is the address to listen on (e.g., ":8080", "localhost:3000").
	Address string
	// ProxyProtocol enables PROXY protocol support for the listener.
	ProxyProtocol bool
}

// Options contains configuration for the ZeroDownTime instance.
type Options struct {
	// UpgradeSock is the path to the Unix socket used for upgrade signaling.
	UpgradeSock string
	// PIDFile is the path to the file where the process ID is stored.
	PIDFile string
	// QuitTimout is the maximum time to wait for the old process to terminate.
	QuitTimout time.Duration
}

// GetPIDFile returns the PID file path, using a default value if not configured.
func (opts Options) GetPIDFile() string {
	if opts.PIDFile == "" {
		return "./logs/bifrost.pid"
	}
	return opts.PIDFile
}

// GetUpgradeSock returns the upgrade socket path, using a default value if not configured.
func (opts Options) GetUpgradeSock() string {
	if opts.UpgradeSock == "" {
		return "./logs/bifrost.sock"
	}
	return opts.UpgradeSock
}

// New creates a new ZeroDownTime instance with the given options.
// If QuitTimeout is not specified, it defaults to 10 seconds.
func New(opts Options) *ZeroDownTime {
	quitTimeout := 10 * time.Second
	if opts.QuitTimout > 0 {
		quitTimeout = opts.QuitTimout
	}
	return &ZeroDownTime{
		options:       &opts,
		stopWaitingCh: make(chan bool, 1),
		isShutdownCh:  make(chan bool, 1),
		state:         defaultState,
		command:       &defaultCommandRunner{},
		envGetter:     defaultEnvGetter,
		processFinder: &defaultProcessFinder{},
		fileOpener:    defaultFileOpener,
		QuitTimeout:   quitTimeout,
	}
}

// IsWaiting returns true if the instance is in the waitingState.
func (z *ZeroDownTime) IsWaiting() bool {
	z.mu.Lock()
	defer z.mu.Unlock()
	return z.state == waitingState
}

// Close shuts down the ZeroDownTime instance by closing all listeners
// and stopping the upgrade waiting goroutine if active.
func (z *ZeroDownTime) Close(ctx context.Context) error {
	for _, info := range z.listeners {
		_ = info.listener.Close()
	}
	z.mu.Lock()
	isWaiting := z.state == waitingState
	z.mu.Unlock()
	if isWaiting {
		z.stopWaitingCh <- true
		<-z.isShutdownCh
	}
	return nil
}

// Upgrade triggers an upgrade by connecting to the upgrade socket.
// This signals the running process to spawn a new process and transfer listeners.
func (z *ZeroDownTime) Upgrade() error {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(context.Background(), "unix", z.options.GetUpgradeSock())
	if err != nil {
		return fmt.Errorf("failed to connect to upgrade socket: %w", err)
	}
	defer conn.Close()
	return nil
}

// IsUpgraded returns true if this process was spawned as part of an upgrade.
// It checks for the presence of the UPGRADE environment variable.
func (z *ZeroDownTime) IsUpgraded() bool {
	return z.envGetter("UPGRADE") != ""
}

// Listener returns a network listener for the given options.
// If this is an upgraded process, it attempts to inherit the listener from the parent process.
// Otherwise, it creates a new listener. The listener is cached for reuse.
func (z *ZeroDownTime) Listener(ctx context.Context, options *ListenerOptions) (net.Listener, error) {
	var err error
	z.listenerOnce.Do(func() {
		if z.IsUpgraded() {
			str := os.Getenv("LISTENERS")
			if len(str) > 0 {
				err := sonic.Unmarshal([]byte(str), &z.listeners)
				if err != nil {
					slog.Error("failed to unmarshal LISTENERS", "error", err)
					return
				}
			}
			z.mu.Lock()
			defer z.mu.Unlock()
			index := 0
			count := len(z.listeners)
			for index < count {
				// fd starting from 3
				fd := 3 + index
				l := z.listeners[index]
				index++
				f := os.NewFile(uintptr(fd), "")
				if f == nil {
					break
				}
				fileListener, err := net.FileListener(f)
				if err != nil {
					slog.Error("failed to create file listener", "error", err, "fd", fd)
					continue
				}
				l.listener = fileListener
				if options.ProxyProtocol {
					l.listener = &proxyproto.Listener{Listener: fileListener}
				}
				slog.Info("file Listener is created", "addr", fileListener.Addr(), "fd", fd)
			}
		}
	})
	for _, l := range z.listeners {
		if l.Key == options.Address {
			slog.Info("get listener from cache", "addr", options.Address)
			return l.listener, nil
		}
	}
	var listener net.Listener
	if options.Config != nil {
		listener, err = options.Config.Listen(ctx, options.Network, options.Address)
		if err != nil {
			slog.Error("failed to create listener from config", "error", err, "addr", options.Address, "network", options.Network)
			return nil, err
		}
	} else {
		config := &net.ListenConfig{}
		listener, err = config.Listen(ctx, options.Network, options.Address)
		if err != nil {
			slog.Error("failed to create listener", "error", err, "addr", options.Address, "network", options.Network)
			return nil, err
		}
	}
	info := &listenInfo{
		listener: listener,
		Key:      options.Address,
	}
	if options.ProxyProtocol {
		info.listener = &proxyproto.Listener{Listener: listener}
	}
	z.mu.Lock()
	z.listeners = append(z.listeners, info)
	z.mu.Unlock()
	return listener, nil
}

// WaitForUpgrade blocks until an upgrade signal is received on the upgrade socket.
// When an upgrade is triggered, it spawns a new process with inherited file descriptors
// and waits for the Close method to be called before returning.
func (z *ZeroDownTime) WaitForUpgrade(ctx context.Context) error {
	z.mu.Lock()
	if z.state != defaultState {
		z.mu.Unlock()
		return fmt.Errorf("state is not default and cannot be upgraded, state=%d", z.state)
	}
	z.state = waitingState
	z.mu.Unlock()
	dir := filepath.Dir(z.options.GetUpgradeSock())
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	config := &net.ListenConfig{}
	socket, err := config.Listen(ctx, "unix", z.options.GetUpgradeSock())
	if err != nil {
		return fmt.Errorf("failed to open upgrade socket: %w", err)
	}
	defer func() {
		socket.Close()
		z.isShutdownCh <- true
	}()
	slog.Info("unix socket is created", "path", z.options.GetUpgradeSock())
	go safety.Go(context.Background(), func() {
		upgradeCount := 0
		for {
			conn, err := socket.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					slog.Info("unix socket is closed", "pid", os.Getpid())
					break
				}
				slog.Info("failed to accept upgrade connection", "error", err)
				continue
			}
			conn.Close()

			upgradeCount++
			slog.Info("upgrade signal received",
				"upgradeCount", upgradeCount,
				"currentPID", os.Getpid(),
			)

			slog.Debug("collecting listener file descriptors", "listenerCount", len(z.listeners))
			files := []*os.File{}
			for _, l := range z.listeners {
				proxylistener, ok := l.listener.(*proxyproto.Listener)
				if ok {
					f, err := proxylistener.Listener.(*net.TCPListener).File()
					if err != nil {
						slog.ErrorContext(ctx, "failed to get listener file", "error", err, "key", l.Key)
						continue
					}
					files = append(files, f)
					slog.Debug("listener file descriptor collected", "key", l.Key, "fd", f.Fd())
				} else {
					f, err := l.listener.(*net.TCPListener).File()
					if err != nil {
						slog.ErrorContext(ctx, "failed to get listener file", "error", err, "key", l.Key)
						continue
					}
					files = append(files, f)
					slog.Debug("listener file descriptor collected", "key", l.Key, "fd", f.Fd())
				}
			}
			slog.Info("listener file descriptors ready for transfer", "count", len(files))

			b, err := sonic.Marshal(z.listeners)
			if err != nil {
				slog.Error("failed to marshal listeners", "error", err)
				break
			}

			slog.Debug("spawning new process",
				"executable", os.Args[0],
				"args", os.Args[1:],
				"fdCount", len(files),
			)

			cmd := z.command.Command(os.Args[0], os.Args[1:]...)
			cmd.Env = append(os.Environ(), "UPGRADE=1", "LISTENERS="+string(b))
			cmd.Stdin = nil
			cmd.Stdout = nil
			cmd.Stderr = nil
			cmd.ExtraFiles = files
			if err := cmd.Start(); err != nil {
				slog.ErrorContext(ctx, "failed to start a new process", "error", err)
				continue
			}

			slog.Info("new process spawned successfully",
				"newPID", cmd.Process.Pid,
				"currentPID", os.Getpid(),
				"fdCount", len(files),
			)
		}
	})
	<-z.stopWaitingCh
	slog.Info("stop waiting for upgrade signal", "pid", os.Getpid())
	return nil
}

var (
	ErrKillTimeout = errors.New("process did not terminate within the timeout period")
)

// Quit sends a signal to the process and waits for it to exit. If the process
// does not exit within the timeout period, it will be sent a SIGKILL signal.
// If removePIDFile is true, the PID file will be removed if it exists.
//
// The signal sent is SIGHUP, which is usually used to restart a process.
// If the process has already exited, this method will return nil immediately.
//
// If the process does not exit within the timeout period, ErrKillTimeout will
// be returned.
func (z *ZeroDownTime) Quit(ctx context.Context, pid int, removePIDFile bool) error {
	process, err := z.processFinder.FindProcess(pid)
	if err != nil {
		slog.Error("find process error", "error", err)
		return err
	}
	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		slog.Error("send signal error", "error", err)
		return err
	}
	// Create a ticker to check process status
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	// Set timeout deadline
	deadline := time.Now().Add(z.QuitTimeout)
	for time.Now().Before(deadline) {
		<-ticker.C
		// Try to send a null signal to check if the process exists
		err := process.Signal(syscall.Signal(0))
		if err != nil {
			if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
				// Process no longer exists, now remove PID file if requested
				if removePIDFile {
					if err := os.Remove(z.options.GetPIDFile()); err != nil {
						if !os.IsNotExist(err) {
							slog.Error("failed to remove PID file", "error", err)
							return err
						}
					}
				}
				return nil
			}
			// Other errors, possibly permission issues
			return fmt.Errorf("check process error: %w", err)
		}
	}
	// If we reach here, it means timeout occurred
	// Optionally send SIGKILL
	err = process.Kill()
	if err != nil {
		return fmt.Errorf("failed to kill process after timeout: %w", err)
	}
	// Check again if the process has terminated
	time.Sleep(100 * time.Millisecond)
	if err := process.Signal(syscall.Signal(0)); err != nil {
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			// Process terminated after SIGKILL, remove PID file if requested
			if removePIDFile {
				if err := os.Remove(z.options.GetPIDFile()); err != nil {
					if !os.IsNotExist(err) {
						slog.Error("failed to remove PID file", "error", err)
						return err
					}
				}
			}
			return nil
		}
	}
	return ErrKillTimeout
}

// writePID writes the current process ID to the PID file atomically.
// It uses a temporary file and rename to ensure the PID file is never partially written.
// This prevents corruption if the process is killed during the write operation.
func (z *ZeroDownTime) writePID() error {
	pid := os.Getpid()
	dir := filepath.Dir(z.options.GetPIDFile())
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Use temp file + rename for atomic write
	tmpFile := z.options.GetPIDFile() + ".tmp"
	data := []byte(strconv.Itoa(pid))
	err := os.WriteFile(tmpFile, data, 0600)
	if err != nil {
		slog.Error("failed to write temp PID file", "error", err)
		return fmt.Errorf("failed to write temp PID file: %w", err)
	}

	err = os.Rename(tmpFile, z.options.GetPIDFile())
	if err != nil {
		os.Remove(tmpFile)
		slog.Error("failed to rename PID file", "error", err)
		return fmt.Errorf("failed to rename PID file: %w", err)
	}
	return nil
}

// ForceWritePID writes the PID file without acquiring a lock.
// This is used during the upgrade handoff phase where the new process
// needs to update the PID file before the old process exits.
// WARNING: Only use this in upgrade scenarios where you are the "inheritor".
func (z *ZeroDownTime) ForceWritePID() error {
	return z.writePID()
}

// GetPID reads and returns the process ID from the PID file.
// Returns an error if the file cannot be read or the content is not a valid integer.
func (z *ZeroDownTime) GetPID() (int, error) {
	b, err := os.ReadFile(z.options.GetPIDFile())
	if err != nil {
		slog.Error("shutdown error", "error", err)
		return 0, err
	}
	pid, err := cast.ToInt(string(b))
	if err != nil {
		slog.Error("pid is invalid", "error", err)
		return 0, err
	}
	return pid, nil
}

// RemoveUpgradeSock removes the upgrade socket file if it exists.
// Returns nil if the file does not exist.
func (z *ZeroDownTime) RemoveUpgradeSock() error {
	_, err := os.Stat(z.options.GetUpgradeSock())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.Remove(z.options.GetUpgradeSock())
}

// ValidatePIDFile checks if the process specified in the PID file is still running.
// It returns:
//   - isRunning: true if the process is running, false otherwise
//   - pid: the process ID read from the file (0 if file doesn't exist)
//   - error: any error that occurred during validation
//
// If the PID file does not exist, it returns (false, 0, nil).
// This is useful for verifying that an old process is still alive before attempting to stop it.
func (z *ZeroDownTime) ValidatePIDFile() (bool, int, error) {
	pid, err := z.GetPID()
	if err != nil {
		if os.IsNotExist(err) {
			return false, 0, nil
		}
		return false, 0, err
	}

	process, err := z.processFinder.FindProcess(pid)
	if err != nil {
		return false, pid, err
	}

	// Send null signal to check if the process exists
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			return false, pid, nil
		}
		return false, pid, err
	}

	return true, pid, nil
}

// WritePIDWithLock writes the PID file while holding an exclusive file lock.
// This prevents race conditions when multiple processes attempt to write the PID file
// simultaneously, such as during rapid successive upgrades.
//
// The returned *os.File must be kept open to maintain the lock. Call ReleasePIDLock
// when the lock is no longer needed (typically when the process exits or the upgrade completes).
//
// Returns an error if another process already holds the lock.
func (z *ZeroDownTime) WritePIDWithLock() (*os.File, error) {
	dir := filepath.Dir(z.options.GetPIDFile())
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	lockFile := z.options.GetPIDFile() + ".lock"
	f, err := z.fileOpener(lockFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to acquire lock, another process may be running: %w", err)
	}

	// Atomic write PID file
	if err := z.writePID(); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		return nil, err
	}

	return f, nil
}

// ReleasePIDLock releases the file lock acquired by WritePIDWithLock and closes the file.
// It is safe to call with a nil file pointer.
func (z *ZeroDownTime) ReleasePIDLock(f *os.File) error {
	if f == nil {
		return nil
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return f.Close()
}
