package gateway

import (
	"context"
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/zero"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/cloudwego/netpoll"
)

var (
	bifrost     *Bifrost
	spaceByte   = []byte{byte(' ')}
	httpMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodConnect}
)

var runTask = gopool.CtxGo

func setRunner(runner func(ctx context.Context, f func())) {
	runTask = runner
}

// DisableGopool disables the Go pool.
//
// No parameters.
// Returns an error type.

func DisableGopool() error {
	_ = netpoll.DisableGopool()
	runTask = func(ctx context.Context, f func()) {
		go f()
	}
	return nil
}

func isValidHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodConnect:
		return true
	default:
		return false
	}
}

func getStackTrace() string {
	stackBuf := make([]uintptr, 50)
	length := runtime.Callers(3, stackBuf)
	stack := stackBuf[:length]

	var b strings.Builder
	frames := runtime.CallersFrames(stack)

	for {
		frame, more := frames.Next()

		if !strings.Contains(frame.File, "runtime/") {
			_, _ = b.WriteString(fmt.Sprintf("\n\tFile: %s, Line: %d. Function: %s", frame.File, frame.Line, frame.Function))
		}

		if !more {
			break
		}
	}
	return b.String()
}

// Run starts the bifrost server with the given options.
//
// mainOptions is the configuration for the bifrost server.
// err is the error that occurred during the startup process.
func Run(mainOptions config.Options) (err error) {
	if !mainOptions.Gopool {
		return DisableGopool()
	}

	bifrost, err = NewBifrost(mainOptions, false)
	if err != nil {
		slog.Error("fail to start bifrost", "error", err)
		return err
	}

	ctx := context.Background()
	// shutdown bifrost
	defer func() {
		_ = shutdown(ctx)
	}()

	config.OnChanged = func() error {
		slog.Debug("reloading...")

		_, mainOpts, err := config.LoadDynamic(mainOptions)
		if err != nil {
			slog.Error("fail to load config", "error", err)
			return err
		}

		err = config.ValidateValue(mainOpts)
		if err != nil {
			slog.Error("fail to validate config", "error", err)
			return err
		}

		newBifrost, err := NewBifrost(mainOpts, true)
		if err != nil {
			return err
		}
		defer func() {
			newBifrost.Stop()
		}()

		isReloaded := false

		for id, httpServer := range bifrost.HttpServers {
			newHTTPServer, found := newBifrost.HttpServers[id]
			if found && httpServer.Bind() == newHTTPServer.Bind() {
				httpServer.SetEngine(newHTTPServer.Engine())
				isReloaded = true
			}
		}

		slog.Info("bifrost is reloaded successfully", "isReloaded", isReloaded)

		return nil
	}

	err = config.Watch()
	if err != nil {
		return err
	}

	go func() {
		defer func() {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			_ = shutdown(ctx)
		}()

		bifrost.Run()
	}()

	go func() {
		for _, httpServer := range bifrost.HttpServers {
			for {
				conn, err := net.Dial("tcp", httpServer.Bind())
				if err == nil {
					conn.Close()
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
		}

		slog.Info("bifrost started successfully", "pid", os.Getpid())

		zeroDT := bifrost.ZeroDownTime()

		if zeroDT != nil && zeroDT.IsUpgraded() {
			err := zeroDT.Shutdown(ctx)
			if err != nil {
				return
			}
		}

		if mainOptions.IsDaemon {
			if err := bifrost.ZeroDownTime().WaitForUpgrade(ctx); err != nil {
				slog.Error("failed to upgrade process", "error", err)
				return
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigChan

	slog.Debug("received shutdown signal", "pid", os.Getpid())
	return nil
}

// RunAsDaemon runs the current process as a daemon.
//
// It takes mainOptions of type config.Options which contains the user and group information to run the daemon process.
// Returns an error if the daemon process fails to start.
func RunAsDaemon(mainOptions config.Options) error {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Env = append(os.Environ(), "DAEMONIZED=1")

	if mainOptions.User != "" {
		u, err := user.Lookup(mainOptions.User)
		if err != nil {
			return fmt.Errorf("failed to lookup user %s: %v", mainOptions.User, err)
		}
		uid, _ := strconv.Atoi(u.Uid)
		gid, _ := strconv.Atoi(u.Gid)

		if mainOptions.Group != "" {
			g, err := user.LookupGroup(mainOptions.Group)
			if err != nil {
				return fmt.Errorf("failed to lookup group %s: %v", mainOptions.Group, err)
			}
			gid, _ = strconv.Atoi(g.Gid)
		}

		// it requires root permission
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(uid),
				Gid: uint32(gid),
			},
		}
	}

	err := cmd.Start()
	if err != nil {
		slog.Error("fail to start daemon", "error", err)
		return err
	}

	slog.Debug("daemon process started", "pid", cmd.Process.Pid)
	return nil
}

// StopDaemon stops the daemon process.
//
// It takes mainOptions of type config.Options which contains the upgrade socket and PID file information.
// Returns an error if the daemon process fails to stop.
func StopDaemon(mainOptions config.Options) error {
	zeroOpts := zero.Options{
		UpgradeSock: mainOptions.UpgradeSock,
		PIDFile:     mainOptions.PIDFile,
	}

	zeroDT := zero.New(zeroOpts)

	ctx := context.Background()
	err := zeroDT.Shutdown(ctx)
	if err != nil {
		slog.Error("fail to stop", "error", err)
		return err
	}
	return nil
}

// Upgrade upgrades the daemon process.
//
// It takes mainOptions of type config.Options which contains the upgrade socket and PID file information.
// Returns an error if the upgrade fails.
func Upgrade(mainOptions config.Options) error {
	zeroOpts := zero.Options{
		UpgradeSock: mainOptions.UpgradeSock,
		PIDFile:     mainOptions.PIDFile,
	}

	zeroDT := zero.New(zeroOpts)

	if err := zeroDT.Upgrade(); err != nil {
		slog.Error("fail to upgrade", "error", err)
		return err
	}

	return nil
}

func shutdown(ctx context.Context) error {
	if bifrost != nil {
		if err := bifrost.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, "failed to shutdown server", "error", err)
			return err
		}
	}

	slog.Info("bifrost is shutdown successfully", "pid", os.Getpid())
	return nil
}
