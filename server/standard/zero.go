package main

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

type ZeroDownTime struct {
	mu            sync.Mutex
	options       *ZeroDownTimeOptions
	listeners     []*listenInfo
	stopWaitingCh chan bool
	isShutdownCh  chan bool
	listenerOnce  sync.Once
}

type ZeroDownTimeOptions struct {
	SocketPath string
	PIDFile    string
}

func New(opts ZeroDownTimeOptions) *ZeroDownTime {
	return &ZeroDownTime{
		options:       &opts,
		stopWaitingCh: make(chan bool, 1),
		isShutdownCh:  make(chan bool, 1),
	}
}

func (z *ZeroDownTime) Close(ctx context.Context) error {
	for _, info := range z.listeners {
		_ = info.listener.Close()
	}

	z.stopWaitingCh <- true
	<-z.isShutdownCh
	return nil
}

func (z *ZeroDownTime) Upgrade() error {
	conn, err := net.Dial("unix", z.options.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to upgrade socket: %v", err)
	}
	defer conn.Close()

	slog.Info("Connected to upgrade socket")
	return nil
}

func (z *ZeroDownTime) IsUpgraded() bool {
	return os.Getenv("UPGRADE") != ""
}

func (z *ZeroDownTime) Listener(address string) (net.Listener, error) {
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
			for count > 0 {
				// fd starting from 3
				fd := 2 + count

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

				count--
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

	listener, err := net.Listen("tcp", address)
	if err != nil {
		slog.Error("failed to create tcp listener", "error", err)
		return nil, err
	}

	info := &listenInfo{
		listener: listener,
		Key:      address,
	}

	slog.Info("tcp Listener is created", "addr", address)

	z.mu.Lock()
	z.listeners = append(z.listeners, info)
	z.mu.Unlock()

	return listener, nil
}

func (z *ZeroDownTime) WaitForUpgrade(ctx context.Context) error {
	socket, err := net.Listen("unix", z.options.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to open upgrade socket: %v", err)
	}
	defer func() {
		socket.Close()
		z.isShutdownCh <- true
	}()

	slog.Info("unix socket is created", "path", z.options.SocketPath)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("recover from panic:", r)
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
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
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
	b, err := os.ReadFile(z.options.PIDFile)
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
		_ = os.Remove(z.options.PIDFile)
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

func (z *ZeroDownTime) WritePID(ctx context.Context) error {
	pid := os.Getpid()
	err := os.WriteFile(z.options.PIDFile, []byte(fmt.Sprintf("%d", pid)), 0644)
	if err != nil {
		slog.ErrorContext(ctx, "failed to write PID file", "error", err)
		return err
	}

	return nil
}
