package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bytedance/sonic"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
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
	Listener net.Listener `json:"-"`
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
	// QuitTimout is the maximum time to wait for the old process to terminate.
	QuitTimout time.Duration
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
		_ = info.Listener.Close()
	}
	z.mu.Lock()
	isWaiting := z.state == waitingState
	z.mu.Unlock()
	if isWaiting {
		z.stopWaitingCh <- true
	}
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
			// Try new BIFROST_LISTENER_KEYS mechanism first
			listeners, err := InheritedListeners()
			if err != nil {
				slog.Error("failed to get inherited listeners", "error", err)
			}

			if len(listeners) > 0 {
				for key, file := range listeners {
					fileListener, err := net.FileListener(file)
					if err != nil {
						slog.Error("failed to create file listener", "error", err, "key", key)
						_ = file.Close()
						continue
					}

					info := &listenInfo{
						Listener: fileListener,
						Key:      key,
					}
					if options.ProxyProtocol {
						info.Listener = &proxyproto.Listener{Listener: fileListener}
					}

					z.mu.Lock()
					z.listeners = append(z.listeners, info)
					z.mu.Unlock()

					slog.Info("file Listener is created from inheritance", "addr", fileListener.Addr(), "key", key)
				}
				return
			}

			// Fallback to legacy LISTENERS env var (if any)
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
				l.Listener = fileListener
				if options.ProxyProtocol {
					l.Listener = &proxyproto.Listener{Listener: fileListener}
				}
				slog.Info("file Listener is created", "addr", fileListener.Addr(), "fd", fd)
			}
		}
	})
	for _, l := range z.listeners {
		if l.Key == options.Address {
			slog.Info("get listener from cache", "addr", options.Address)
			return l.Listener, nil
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
		Listener: listener,
		Key:      options.Address,
	}
	if options.ProxyProtocol {
		info.Listener = &proxyproto.Listener{Listener: listener}
	}
	z.mu.Lock()
	z.listeners = append(z.listeners, info)
	z.mu.Unlock()
	return listener, nil
}

// GetListeners returns the list of active listeners.
func (z *ZeroDownTime) GetListeners() []*listenInfo {
	z.mu.Lock()
	defer z.mu.Unlock()
	// Return a copy to avoid race conditions
	listeners := make([]*listenInfo, len(z.listeners))
	copy(listeners, z.listeners)
	return listeners
}

// WaitForUpgrade blocks until a SIGHUP signal is received.
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

	// Listen for SIGHUP signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)

	slog.Info("waiting for SIGHUP signal to trigger upgrade", "pid", os.Getpid())

	go safety.Go(context.Background(), func() {
		defer func() {
			signal.Stop(sigCh)
			z.isShutdownCh <- true
		}()

		upgradeCount := 0
		for {
			select {
			case <-z.stopWaitingCh:
				slog.Info("stop waiting for upgrade signal", "pid", os.Getpid())
				return
			case sig := <-sigCh:
				upgradeCount++
				slog.Info("upgrade signal received",
					"signal", sig,
					"upgradeCount", upgradeCount,
					"currentPID", os.Getpid(),
				)

				slog.Debug("collecting listener file descriptors", "listenerCount", len(z.listeners))
				files := []*os.File{}
				for _, l := range z.listeners {
					proxylistener, ok := l.Listener.(*proxyproto.Listener)
					if ok {
						f, err := proxylistener.Listener.(*net.TCPListener).File()
						if err != nil {
							slog.ErrorContext(ctx, "failed to get listener file", "error", err, "key", l.Key)
							continue
						}
						files = append(files, f)
						slog.Debug("listener file descriptor collected", "key", l.Key, "fd", f.Fd())
					} else {
						f, err := l.Listener.(*net.TCPListener).File()
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
					continue
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
		}
	})

	// Wait for goroutine to complete (triggered by Close sending to stopWaitingCh)
	<-z.isShutdownCh
	slog.Info("WaitForUpgrade completed", "pid", os.Getpid())
	return nil
}
