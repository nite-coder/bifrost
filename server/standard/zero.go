package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

type ZeroDownTime struct {
	options       *ZeroDownTimeOptions
	isDaemon      bool
	listener      net.Listener
	stopWaitingCh chan bool
}

type ZeroDownTimeOptions struct {
	Bind       string
	SocketPath string
	PIDFile    string
}

func New(opts ZeroDownTimeOptions) *ZeroDownTime {
	return &ZeroDownTime{
		options:       &opts,
		stopWaitingCh: make(chan bool, 1),
	}
}

func (z *ZeroDownTime) Close(ctx context.Context) error {
	z.stopWaitingCh <- true
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

func (z *ZeroDownTime) Listener() (net.Listener, error) {
	var err error

	if z.IsUpgraded() {
		listenerFile := os.NewFile(3, "")
		z.listener, err = net.FileListener(listenerFile)
		if err != nil {
			log.Fatalf("Failed to create listener from file: %v", err)
			return nil, err
		}
	}

	if z.listener == nil {
		z.listener, err = net.Listen("tcp", z.options.Bind)
		if err != nil {
			return nil, err
		}
	}

	return z.listener, nil
}

func (z *ZeroDownTime) WaitForUpgrade(ctx context.Context) error {
	socket, err := net.Listen("unix", z.options.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to open upgrade socket: %v", err)
	}
	defer func() {
		socket.Close()
	}()

	slog.Info("unix socket is created", "path", z.options.SocketPath)

	go func() {
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

			file, err := z.listener.(*net.TCPListener).File()
			if err != nil {
				slog.ErrorContext(ctx, "failed to get listener file", "error", err)
				continue
			}

			cmd := exec.Command(os.Args[0], os.Args[1:]...)
			cmd.Env = append(os.Environ(), "UPGRADE=1")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.ExtraFiles = []*os.File{file}
			if err := cmd.Start(); err != nil {
				slog.ErrorContext(ctx, "failed to start child process", "error", err)
				continue
			}
		}
	}()

	for range z.stopWaitingCh {
		slog.Info("stop waiting for upgrade signal", "pid", os.Getpid())
		return nil
	}

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
