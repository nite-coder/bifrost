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

type CommandRunner interface {
	Command(name string, arg ...string) *exec.Cmd
}

type EnvGetter func(string) string

type process interface {
	Signal(os.Signal) error
	Kill() error
	Wait() (*os.ProcessState, error)
	Release() error
}

type ProcessFinder interface {
	FindProcess(pid int) (process, error)
}

type FileOpener func(name string) (*os.File, error)

type defaultCommandRunner struct{}

func (d *defaultCommandRunner) Command(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

var defaultEnvGetter = os.Getenv

type defaultProcessFinder struct{}

func (d *defaultProcessFinder) FindProcess(pid int) (process, error) {
	return os.FindProcess(pid)
}

var defaultFileOpener = func(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_RDWR|os.O_CREATE|syscall.O_CLOEXEC, 0600)
}

type listenInfo struct {
	listener net.Listener `json:"-"`
	Key      string       `json:"key"`
}

type State int8

const (
	defaultState State = iota
	waitingState
)

type ZeroDownTime struct {
	mu            sync.Mutex
	options       *Options
	listeners     []*listenInfo
	stopWaitingCh chan bool
	isShutdownCh  chan bool
	listenerOnce  sync.Once
	state         State
	QuitTimeout   time.Duration
	command       CommandRunner
	envGetter     EnvGetter
	processFinder ProcessFinder
	fileOpener    FileOpener
}

type ListenerOptions struct {
	Network       string
	Address       string
	Config        *net.ListenConfig
	ProxyProtocol bool
}

type Options struct {
	UpgradeSock string
	PIDFile     string
	QuitTimout  time.Duration
}

func (opts Options) GetPIDFile() string {
	if opts.PIDFile == "" {
		return "./logs/bifrost.pid"
	}
	return opts.PIDFile
}

func (opts Options) GetUpgradeSock() string {
	if opts.UpgradeSock == "" {
		return "./logs/bifrost.sock"
	}
	return opts.UpgradeSock
}

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

func (z *ZeroDownTime) Close(ctx context.Context) error {
	for _, info := range z.listeners {
		_ = info.listener.Close()
	}

	if z.state == waitingState {
		z.stopWaitingCh <- true
		<-z.isShutdownCh
	}

	return nil
}

func (z *ZeroDownTime) Upgrade() error {
	conn, err := net.Dial("unix", z.options.GetUpgradeSock())
	if err != nil {
		return fmt.Errorf("failed to connect to upgrade socket: %w", err)
	}
	defer conn.Close()

	return nil
}

func (z *ZeroDownTime) IsUpgraded() bool {
	return z.envGetter("UPGRADE") != ""
}

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
		listener, err = net.Listen(options.Network, options.Address)
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

	socket, err := net.Listen("unix", z.options.GetUpgradeSock())
	if err != nil {
		return fmt.Errorf("failed to open upgrade socket: %w", err)
	}
	defer func() {
		socket.Close()
		z.isShutdownCh <- true
	}()

	slog.Info("unix socket is created", "path", z.options.GetUpgradeSock())

	go safety.Go(context.Background(), func() {
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

			files := []*os.File{}

			for _, l := range z.listeners {
				proxylistener, ok := l.listener.(*proxyproto.Listener)

				if ok {
					f, err := proxylistener.Listener.(*net.TCPListener).File()
					if err != nil {
						slog.ErrorContext(ctx, "failed to get listener file", "error", err)
						continue
					}
					files = append(files, f)
				} else {
					f, err := l.listener.(*net.TCPListener).File()
					if err != nil {
						slog.ErrorContext(ctx, "failed to get listener file", "error", err)
						continue
					}
					files = append(files, f)
				}
			}

			slog.Info("listeners count", "count", len(files))

			b, err := sonic.Marshal(z.listeners)
			if err != nil {
				slog.Error("failed to marshal listeners", "error", err)
				break
			}

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
	if removePIDFile {
		if err := os.Remove(z.options.GetPIDFile()); err != nil {
			slog.Error("failed to remove PID file", "error", err)
			return err
		}
	}

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
				// Process no longer exists
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
			return nil
		}
	}

	return ErrKillTimeout
}

func (z *ZeroDownTime) WritePID() error {
	pid := os.Getpid()

	dir := filepath.Dir(z.options.GetPIDFile())
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	data := []byte(strconv.Itoa(pid))
	err := os.WriteFile(z.options.GetPIDFile(), data, 0600)
	if err != nil {
		slog.Error("failed to write PID file", "error", err)
		return err
	}

	return nil
}

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
