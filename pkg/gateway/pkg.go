package gateway

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bytedance/sonic"
	cgopool "github.com/cloudwego/gopkg/concurrency/gopool"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/netpoll"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/connector/redis"
	"github.com/nite-coder/bifrost/pkg/log"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
	"github.com/nite-coder/bifrost/pkg/zero"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/valyala/bytebufferpool"
)

var (
	defaultBifrost      = &atomic.Value{}
	spaceByte           = []byte{byte(' ')}
	defaultTrustedCIDRs = []*net.IPNet{
		{ // 0.0.0.0/0 (IPv4)
			IP:   net.IP{0x0, 0x0, 0x0, 0x0},
			Mask: net.IPMask{0x0, 0x0, 0x0, 0x0},
		},
		{ // ::/0 (IPv6)
			IP:   net.IP{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			Mask: net.IPMask{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
	}
)

// GetBifrost retrieves the current instance of Bifrost.
// It returns a pointer to the Bifrost instance stored in the defaultBifrost atomic value.
func GetBifrost() *Bifrost {
	val, _ := defaultBifrost.Load().(*Bifrost)
	return val
}

// SetBifrost sets the current instance of Bifrost.
// It stores the given Bifrost instance in the defaultBifrost atomic value.
// This function is used to set the Bifrost instance after it has been created.
// It is typically called by the top-level application code to initialize the Bifrost instance.
func SetBifrost(bifrost *Bifrost) {
	defaultBifrost.Store(bifrost)
}

// Run starts the bifrost server with the given options.
//
// mainOptions is the configuration for the bifrost server.
// err is the error that occurred during the startup process.
func Run(mainOptions config.Options) (err error) {
	// validate config file
	err = config.ValidateConfig(mainOptions, true)
	if err != nil {
		return err
	}

	netpollConfig := netpoll.Config{}

	if mainOptions.EventLoops > 0 {
		netpollConfig.PollerNum = mainOptions.EventLoops
	}

	if mainOptions.Gopool {
		cgopool.SetPanicHandler(func(ctx context.Context, r any) {
			if r := recover(); r != nil {
				var err error
				switch v := r.(type) {
				case error:
					err = v
				default:
					err = fmt.Errorf("%v", v)
				}
				stackTrace := cast.B2S(debug.Stack())
				slog.Error("netpoll panic recovered",
					slog.String("error", err.Error()),
					slog.String("stack", stackTrace),
				)
			}
		})
		netpollConfig.Runner = cgopool.CtxGo
		safety.Go = cgopool.CtxGo
	} else {
		netpollConfig.Runner = func(ctx context.Context, f func()) {
			go safety.Go(ctx, f)
		}
	}

	err = netpoll.Configure(netpollConfig)
	if err != nil {
		return err
	}

	err = redis.Initialize(context.Background(), mainOptions.Redis)
	if err != nil {
		return err
	}

	httpproxy.SetChunkedTransfer(mainOptions.Experiment.ChunkedTransfer)

	bifrost, err := NewBifrost(mainOptions, false)
	if err != nil {
		slog.Error("failed to start bifrost", "error", err)
		return err
	}
	SetBifrost(bifrost)

	ctx := context.Background()

	config.OnChanged = func() error {
		slog.Debug("reloading...")

		if mainOptions.ConfigPath() != "" {
			newMainOptions, err := config.Load(mainOptions.ConfigPath())
			if err != nil {
				slog.Error("failed to load config", "error", err)
				return err
			}
			mainOptions = newMainOptions
		}

		b, err := sonic.Marshal(mainOptions)
		if err != nil {
			return err
		}

		oldBifrost := GetBifrost()
		if oldBifrost != nil {
			b1, err := sonic.Marshal(oldBifrost.options)
			if err != nil {
				return err
			}

			sha256sum := sha256.Sum256(b)
			sha256sum1 := sha256.Sum256(b1)
			if sha256sum == sha256sum1 {
				// the content of config is not changed
				slog.Error("bifrost is reloaded successfully", "isReloaded", false)
				return nil
			}
		}

		// validate config file
		err = config.ValidateConfig(mainOptions, true)
		if err != nil {
			return err
		}

		newBifrost, err := NewBifrost(mainOptions, true)
		if err != nil {
			return err
		}

		isReloaded := false

		for id, httpServer := range bifrost.httpServers {
			newHTTPServer, found := newBifrost.httpServers[id]
			if found && httpServer.Bind() == newHTTPServer.Bind() {
				httpServer.SetEngine(newHTTPServer.Engine())
				isReloaded = true
				_ = newHTTPServer.Shutdown(ctx)
			}
		}

		slog.Log(ctx, log.LevelNotice, "bifrost is reloaded successfully", "isReloaded", isReloaded)

		return nil
	}

	if mainOptions.IsWatch() {
		err = config.Watch()
		if err != nil {
			return err
		}
	}

	go safety.Go(context.Background(), func() {
		bifrost.Run()
	})

	go safety.Go(context.Background(), func() {
		// Readiness check: verify all HTTP servers are accepting connections
		readinessTimeout := 30 * time.Second
		readinessStart := time.Now()
		readinessDeadline := readinessStart.Add(readinessTimeout)

		slog.Debug("starting readiness check for HTTP servers", "timeout", readinessTimeout, "serverCount", len(bifrost.httpServers))

		for serverID, httpServer := range bifrost.httpServers {
			serverReady := false
			attempts := 0

			for time.Now().Before(readinessDeadline) {
				attempts++
				dialer := &net.Dialer{
					Timeout: 2 * time.Second,
				}
				conn, err := dialer.DialContext(ctx, "tcp", httpServer.Bind())
				if err == nil {
					conn.Close()
					serverReady = true
					slog.Debug("HTTP server is ready",
						"serverID", serverID,
						"bind", httpServer.Bind(),
						"attempts", attempts,
						"elapsed", time.Since(readinessStart).Round(time.Millisecond),
					)
					break
				}
				time.Sleep(500 * time.Millisecond)
			}

			if !serverReady {
				slog.Error("HTTP server failed readiness check",
					"serverID", serverID,
					"bind", httpServer.Bind(),
					"attempts", attempts,
					"timeout", readinessTimeout,
				)
				return
			}
		}

		readinessElapsed := time.Since(readinessStart).Round(time.Millisecond)
		slog.Log(ctx, log.LevelNotice, "bifrost is started successfully",
			"pid", os.Getpid(),
			"routes", len(bifrost.options.Routes),
			"services", len(bifrost.options.Services),
			"middlewares", len(bifrost.options.Middlewares),
			"upstreams", len(bifrost.options.Upstreams),
			"readinessCheckDuration", readinessElapsed,
		)

		zeroDT := bifrost.ZeroDownTime()

		if zeroDT != nil && zeroDT.IsUpgraded() {
			upgradeStart := time.Now()
			slog.Info("upgrade process started",
				"newPID", os.Getpid(),
			)

			// Validate old process is running before trying to quit it
			slog.Debug("validating PID file for old process")
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
				slog.Debug("old process validated",
					"oldPID", oldPID,
					"isRunning", isRunning,
				)
			}

			// Use WritePIDWithLock for atomic PID file handling
			slog.Debug("acquiring PID file lock and writing new PID")
			lockFile, err := zeroDT.WritePIDWithLock()
			if err != nil {
				slog.Error("failed to write PID with lock", "error", err)
				return
			}
			defer func() {
				if err := zeroDT.ReleasePIDLock(lockFile); err != nil {
					slog.Error("failed to release PID lock", "error", err)
				}
			}()
			slog.Debug("PID file updated successfully",
				"newPID", os.Getpid(),
			)

			// Only quit old process if it was running
			if isRunning && oldPID > 0 {
				slog.Info("sending termination signal to old process",
					"oldPID", oldPID,
				)
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

			slog.Log(ctx, log.LevelNotice, "upgrade completed successfully",
				"oldPID", oldPID,
				"newPID", os.Getpid(),
				"upgradeDuration", time.Since(upgradeStart).Round(time.Millisecond),
			)
		}

		if mainOptions.IsDaemon {
			slog.Debug("initializing daemon mode")

			// Use WritePIDWithLock for daemon mode as well
			lockFile, err := zeroDT.WritePIDWithLock()
			if err != nil {
				slog.Error("failed to write PID with lock", "error", err)
				return
			}
			defer func() {
				if err := zeroDT.ReleasePIDLock(lockFile); err != nil {
					slog.Error("failed to release PID lock", "error", err)
				}
			}()
			slog.Debug("daemon PID file created", "pid", os.Getpid())

			err = zeroDT.RemoveUpgradeSock()
			if err != nil {
				slog.Error("failed to remove upgrade sock file", "error", err)
				return
			}
			slog.Debug("upgrade socket cleaned up")

			slog.Info("daemon mode ready, waiting for upgrade signals",
				"pid", os.Getpid(),
				"upgradeSock", mainOptions.UpgradeSock,
			)
			if err := bifrost.ZeroDownTime().WaitForUpgrade(ctx); err != nil {
				slog.Error("failed to wait for upgrade process", "error", err)
				return
			}
		}
	})

	var sigs os.Signal
	defer func() {
		// shutdown bifrost
		if sigs == syscall.SIGINT {
			_ = shutdown(ctx, true)
			return
		}
		_ = shutdown(ctx, false)
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sigs = <-stopChan

	slog.Debug("received shutdown signal", "signal", sigs, "pid", os.Getpid())
	return nil
}

// RunAsDaemon runs the current process as a daemon.
//
// It takes mainOptions of type config.Options which contains the user and group information to run the daemon process.
// Returns an error if the daemon process fails to start.
func RunAsDaemon(mainOptions config.Options) error {
	if os.Geteuid() != 0 {
		return errors.New("must be run as root to execute in daemon mode")
	}

	cmd := exec.CommandContext(context.TODO(), os.Args[0], os.Args[1:]...)
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
		slog.Error("failed to run as daemon", "error", err)
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

	oldPID, err := zeroDT.GetPID()
	if err != nil {
		return err
	}
	err = zeroDT.Quit(ctx, oldPID, true)
	if err != nil {
		slog.Error("failed to stop", "error", err)
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
		slog.Error("failed to upgrade", "error", err)
		return err
	}

	return nil
}

func shutdown(ctx context.Context, now bool) error {
	bifrost := GetBifrost()
	if defaultBifrost != nil {
		var err error

		if now {
			err = bifrost.ShutdownNow(ctx)
		} else {
			err = bifrost.Shutdown(ctx)
		}

		if err != nil {
			slog.ErrorContext(ctx, "failed to shutdown server", "error", err)
			return err
		}
	}

	slog.Log(ctx, log.LevelNotice, "bifrost is shutdown successfully", "pid", os.Getpid())
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
	go safety.Go(context.Background(), func() {
		defer close(c)
		wg.Wait()
	})
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
