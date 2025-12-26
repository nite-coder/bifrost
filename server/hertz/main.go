package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	grpcproxy "github.com/nite-coder/bifrost/pkg/proxy/grpc"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
	"github.com/nite-coder/bifrost/pkg/zero"

	configBifrost "github.com/nite-coder/bifrost/pkg/config"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	configHTTP2 "github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
	hertzslog "github.com/hertz-contrib/logger/slog"
)

func withDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

var (
	daemon  = flag.Bool("d", false, "Run in daemon mode")
	upgrade = flag.Bool("u", false, "Perform a hot upgrade")
	shudown = flag.Bool("stop", false, "shutdown server")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	zeroOpts := zero.Options{
		UpgradeSock: "./hertz.sock",
		PIDFile:     "./hertz.pid",
	}

	zeroDT := zero.New(zeroOpts)

	if *upgrade {
		if err := zeroDT.Upgrade(); err != nil {
			log.Fatalf("Upgrade failed: %v", err)
		}
		return
	}

	if *shudown {
		oldPID, err := zeroDT.GetPID()
		if err != nil {
			slog.Error("shutdown error", "error", err)
			return
		}
		_ = zeroDT.Quit(ctx, oldPID, true)
		return
	}

	done := make(chan bool, 1)
	if zeroDT.IsUpgraded() {
		go func() {
			<-done

			upgradeStart := time.Now()
			slog.Info("upgrade process started", "newPID", os.Getpid())

			// Validate old process is running before trying to quit it
			isRunning, oldPID, err := zeroDT.ValidatePIDFile()
			if err != nil {
				slog.Error("failed to validate PID file", "error", err)
				return
			}

			if !isRunning {
				slog.Warn("old process is not running, skipping quit",
					"pid", oldPID,
					"reason", "process not found or already terminated",
				)
			} else {
				slog.Debug("old process validated", "oldPID", oldPID, "isRunning", isRunning)

				// Only quit old process if it was running
				slog.Info("sending termination signal to old process", "oldPID", oldPID)
				quitStart := time.Now()
				err = zeroDT.Quit(ctx, oldPID, false)
				if err != nil {
					slog.Error("failed to quit old process",
						"error", err,
						"oldPID", oldPID,
						"elapsed", time.Since(quitStart).Round(time.Millisecond),
					)
					return
				}
				slog.Info("old process terminated successfully",
					"oldPID", oldPID,
					"quitDuration", time.Since(quitStart).Round(time.Millisecond),
				)
			}

			if *daemon {
				// Use WritePIDWithLock for atomic PID file handling
				lockFile, err := zeroDT.WritePIDWithLock()
				if err != nil {
					slog.Error("failed to write PID with lock", "error", err)
					return
				}
				defer func() { _ = zeroDT.ReleasePIDLock(lockFile) }()
				slog.Debug("PID file updated successfully", "newPID", os.Getpid())

				slog.Info("upgrade completed successfully",
					"oldPID", oldPID,
					"newPID", os.Getpid(),
					"upgradeDuration", time.Since(upgradeStart).Round(time.Millisecond),
				)

				slog.Info("daemon mode ready, waiting for upgrade signals", "pid", os.Getpid())
				if err := zeroDT.WaitForUpgrade(ctx); err != nil {
					slog.Error("failed to upgrade process", "error", err)
					return
				}
			}
		}()

		err := startup(ctx, zeroDT, done)
		if err != nil {
			slog.Error("Failed to start server", "error", err)
			return
		}
	} else {
		go func() {
			<-done

			if *daemon {
				// Use WritePIDWithLock for initial daemon startup
				lockFile, err := zeroDT.WritePIDWithLock()
				if err != nil {
					slog.Error("failed to write PID with lock", "error", err)
					return
				}
				defer func() { _ = zeroDT.ReleasePIDLock(lockFile) }()
				slog.Debug("daemon PID file created", "pid", os.Getpid())

				slog.Info("daemon mode ready, waiting for upgrade signals", "pid", os.Getpid())
				if err := zeroDT.WaitForUpgrade(ctx); err != nil {
					slog.Error("failed to upgrade process", "error", err)
					return
				}
			}
		}()

		err := startup(ctx, zeroDT, done)
		if err != nil {
			slog.Error("Failed to start server", "error", err)
			return
		}
	}
}

func startup(ctx context.Context, zeroDT *zero.ZeroDownTime, done chan bool) error {
	if *daemon && os.Getenv("DAEMONIZED") == "" {
		daemonize()
		return nil
	}

	listenOptions := &zero.ListenerOptions{
		Network: "tcp",
		Address: ":8001",
	}

	listener, err := zeroDT.Listener(ctx, listenOptions)
	if err != nil {
		slog.Error("failed to create listener", "error", err)
	}

	logger := hertzslog.NewLogger(hertzslog.WithOutput(io.Discard))
	hlog.SetLevel(hlog.LevelError)
	hlog.SetLogger(logger)
	hlog.SetSilentMode(true)

	hzOpts := []config.Option{
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithSenseClientDisconnection(true),
		server.WithReadTimeout(time.Second * 3),
		server.WithWriteTimeout(time.Second * 3),
		server.WithKeepAlive(true),
		server.WithALPN(true),
		server.WithStreamBody(true),
		server.WithH2C(true),
		server.WithExitWaitTime(1 * time.Second),
		server.WithListener(listener),
		withDefaultServerHeader(true),
	}

	h := server.New(hzOpts...)

	http2opts := []configHTTP2.Option{}
	h.AddProtocol("h2", factory.NewServerFactory(http2opts...))

	opts := httpproxy.Options{
		Target:   "http://localhost:8000",
		Weight:   1,
		Protocol: configBifrost.ProtocolHTTP,
	}
	httpProxy, err := httpproxy.New(opts, nil)
	if err != nil {
		return err
	}

	grpcOption := grpcproxy.Options{
		Target:    "grpc://127.0.0.1:8501",
		TLSVerify: false,
	}

	grpcProxy, err := grpcproxy.New(grpcOption)
	if err != nil {
		return err
	}
	h.POST("/spot/orders", httpProxy.ServeHTTP)
	h.POST("/helloworld.Greeter/SayHello", grpcProxy.ServeHTTP)
	h.GET("/chunk", httpProxy.ServeHTTP)

	go func() {
		h.Spin()
	}()

	go func() {
		for {
			conn, err := net.Dial("tcp", ":8001")
			if err == nil {
				conn.Close()
				slog.Info("starting server", "bind", ":8001", "isDaemon", *daemon)
				done <- true
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigChan

	slog.Info("received shutdown signal", "pid", os.Getpid())

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.Shutdown(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to shutdown server", "error", err)
		return err
	}

	_ = zeroDT.Close(ctx)

	slog.Info("server is shutdown successfully", "pid", os.Getpid())
	return nil

}

func daemonize() {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "DAEMONIZED=1")

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	fmt.Printf("Daemon process started with PID %d\n", cmd.Process.Pid)

	os.Exit(0)
}
