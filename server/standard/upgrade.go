package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

type Upgrader struct {
	Options *UpgraderOption

	isDaemon      bool
	listener      net.Listener
	server        *http.Server
	stopUpgradeCh chan bool
}

type UpgraderOption struct {
	Bind       string
	SocketPath string
	PIDFile    string
}

func NewUpgrader(opts *UpgraderOption) *Upgrader {
	return &Upgrader{
		Options:       opts,
		stopUpgradeCh: make(chan bool, 1),
	}
}

func (u *Upgrader) Close(ctx context.Context) error {
	u.stopUpgradeCh <- true

	if u.server != nil {
		if err := u.server.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "failed to shutdown server", "error", err)
			return err
		}
	}

	<-ctx.Done()

	return nil
}

func (u *Upgrader) Upgrade() error {
	conn, err := net.Dial("unix", u.Options.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to upgrade socket: %v", err)
	}
	conn.Close()
	slog.Info("Connected to upgrade socket")
	return nil
}

func (u *Upgrader) WaitForUpgrade(ctx context.Context) error {
	socket, err := net.Listen("unix", u.Options.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to open upgrade socket: %v", err)
	}
	defer func() {
		socket.Close()
	}()

	slog.Info("unix socket is created", "path", u.Options.SocketPath)

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

			file, err := u.listener.(*net.TCPListener).File()
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

	for range u.stopUpgradeCh {
		slog.Info("stop waiting for upgrade signal", "pid", os.Getpid())
		return nil
	}

	return nil
}

func (u *Upgrader) Shutdown(ctx context.Context) error {
	b, err := os.ReadFile(u.Options.PIDFile)
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
		_ = os.Remove(u.Options.PIDFile)
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
