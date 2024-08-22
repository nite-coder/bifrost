package zero

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
)

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
}

type Options struct {
	UpgradeSock string
	PIDFile     string
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
	return &ZeroDownTime{
		options:       &opts,
		stopWaitingCh: make(chan bool, 1),
		isShutdownCh:  make(chan bool, 1),
		state:         defaultState,
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
		return fmt.Errorf("failed to connect to upgrade socket: %v", err)
	}
	defer conn.Close()

	return nil
}

func (z *ZeroDownTime) IsUpgraded() bool {
	return os.Getenv("UPGRADE") != ""
}

func (z *ZeroDownTime) Listener(ctx context.Context, network string, address string, cfg *net.ListenConfig) (net.Listener, error) {
	var err error

	z.listenerOnce.Do(func() {
		if z.IsUpgraded() {
			str := os.Getenv("LISTENERS")

			if len(str) > 0 {
				err := json.Unmarshal([]byte(str), &z.listeners)
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

				f := os.NewFile(uintptr(fd), "")
				if f == nil {
					break
				}

				fileListener, err := net.FileListener(f)
				if err != nil {
					slog.Error("failed to create file listener", "error", err, "fd", fd)
					continue
				}

				l := z.listeners[index]
				l.listener = fileListener

				index++

				slog.Info("file Listener is created", "addr", fileListener.Addr(), "fd", fd)
			}
		}
	})

	for _, l := range z.listeners {
		if l.Key == address {
			slog.Info("get listener from cache", "addr", address)
			return l.listener, nil
		}
	}

	var listener net.Listener

	if cfg != nil {
		listener, err = cfg.Listen(ctx, network, address)
		if err != nil {
			slog.Error("failed to create listener from config", "error", err, "addr", address, "network", network)
			return nil, err
		}
	} else {
		listener, err = net.Listen(network, address)
		if err != nil {
			slog.Error("failed to create listener", "error", err, "addr", address, "network", network)
			return nil, err
		}
	}

	info := &listenInfo{
		listener: listener,
		Key:      address,
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

	_ = z.writePID()
	socket, err := net.Listen("unix", z.options.UpgradeSock)
	if err != nil {
		return fmt.Errorf("failed to open upgrade socket: %v", err)
	}
	defer func() {
		socket.Close()
		z.isShutdownCh <- true
	}()

	slog.Info("unix socket is created", "path", z.options.UpgradeSock)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("zero: recover from panic:", r)
			}
		}()

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
				f, err := l.listener.(*net.TCPListener).File()
				if err != nil {
					slog.ErrorContext(ctx, "failed to get listener file", "error", err)
					continue
				}
				files = append(files, f)
			}

			slog.Info("listeners count", "count", len(files))

			b, err := json.Marshal(z.listeners)
			if err != nil {
				slog.Error("failed to marshal listeners", "error", err)
				break
			}

			cmd := exec.Command(os.Args[0], os.Args[1:]...)
			cmd.Env = append(os.Environ(), "UPGRADE=1", fmt.Sprintf("LISTENERS=%s", string(b)))
			cmd.Stdin = nil
			cmd.Stdout = nil
			cmd.Stderr = nil
			cmd.ExtraFiles = files
			if err := cmd.Start(); err != nil {
				slog.ErrorContext(ctx, "failed to start child process", "error", err)
				continue
			}
		}
	}()

	<-z.stopWaitingCh
	slog.Info("stop waiting for upgrade signal", "pid", os.Getpid())

	return nil
}

func (z *ZeroDownTime) Shutdown(ctx context.Context) error {
	b, err := os.ReadFile(z.options.GetPIDFile())
	if err != nil {
		slog.Error("shutdown error", "error", err)
		return err
	}

	pid, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		slog.Error("pid is invalid", "error", err)
		return err
	}

	defer func() {
		_ = os.Remove(z.options.GetPIDFile())
	}()

	p, err := os.FindProcess(int(pid))
	if err != nil {
		slog.Error("find process error", "error", err)
		return err
	}

	err = p.Signal(syscall.SIGTERM)
	if err != nil {
		slog.Error("send signal error", "error", err)
		return err
	}

	slog.Info("sent SIGTERM to process", "pid", pid)
	return nil
}

func (z *ZeroDownTime) writePID() error {
	pid := os.Getpid()
	err := os.WriteFile(z.options.GetPIDFile(), []byte(fmt.Sprintf("%d", pid)), 0644)
	if err != nil {
		slog.Error("failed to write PID file", "error", err)
		return err
	}

	return nil
}
