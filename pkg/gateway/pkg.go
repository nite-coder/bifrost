package gateway

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/zero"
	"github.com/valyala/bytebufferpool"
)

var (
	bifrost           *Bifrost
	spaceByte                                            = []byte{byte(' ')}
	middlewareFactory map[string]CreateMiddlewareHandler = make(map[string]CreateMiddlewareHandler)
	httpMethods                                          = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodConnect}
)

var runTask = gopool.CtxGo

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
		_ = DisableGopool()
	}

	bifrost, err = NewBifrost(mainOptions, false)
	if err != nil {
		slog.Error("fail to start bifrost", "error", err)
		return err
	}

	ctx := context.Background()

	config.OnChanged = func() error {
		slog.Debug("reloading...")

		if mainOptions.From() != "" {
			mainOptions, err = config.Load(mainOptions.From())
			if err != nil {
				slog.Error("fail to load config", "error", err)
				return err
			}
		}

		newBifrost, err := NewBifrost(mainOptions, true)
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

	if mainOptions.IsWatch() {
		err = config.Watch()
		if err != nil {
			return err
		}
	}

	go func() {
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

	defer func() {
		// shutdown bifrost
		_ = shutdown(ctx)
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
	// verify permissions to create the PID file and the upgrade socket file
	dir := filepath.Dir(mainOptions.PIDFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	dir = filepath.Dir(mainOptions.UpgradeSock)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Env = append(os.Environ(), "DAEMONIZED=1")

	if mainOptions.User != "" {
		u, err := user.Lookup(mainOptions.User)
		if err != nil {
			return fmt.Errorf("failed to lookup user %s: %w", mainOptions.User, err)
		}
		uid64, err := strconv.ParseUint(u.Uid, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse uid %s: %w", u.Uid, err)
		}
		uid := uint32(uid64)
		gid64, err := strconv.ParseUint(u.Gid, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse gid %s: %w", u.Gid, err)
		}
		gid := uint32(gid64)

		if mainOptions.Group != "" {
			g, err := user.LookupGroup(mainOptions.Group)
			if err != nil {
				return fmt.Errorf("failed to lookup group %s: %w", mainOptions.Group, err)
			}
			gid64, err = strconv.ParseUint(g.Gid, 10, 32)
			if err != nil {
				return fmt.Errorf("failed to parse gid %s: %w", g.Gid, err)
			}
			gid = uint32(gid64)
		}

		setUserAndGroup(cmd, uint32(uid), uint32(gid))
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

func fullURI(req *protocol.Request) string {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_, _ = buf.Write(req.Method())
	_, _ = buf.Write(spaceByte)
	_, _ = buf.Write(req.URI().FullURI())
	return buf.String()
}

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

func getRandomNumber(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}
