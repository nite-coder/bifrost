package gateway

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bytedance/sonic"
	cgopool "github.com/cloudwego/gopkg/concurrency/gopool"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/netpoll"
	"github.com/nite-coder/bifrost/internal/pkg/runtime"
	"github.com/nite-coder/bifrost/internal/pkg/task"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/connector/redis"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/zero"
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

	if mainOptions.NumLoops > 0 {
		netpollConfig.PollerNum = mainOptions.NumLoops
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
				stackTrace := runtime.StackTrace()
				slog.Error("netpoll panic recovered",
					slog.String("error", err.Error()),
					slog.String("stack", stackTrace),
				)
			}
		})
		netpollConfig.Runner = cgopool.CtxGo
		task.Runner = cgopool.CtxGo
	} else {
		netpollConfig.Runner = func(ctx context.Context, f func()) {
			go task.Runner(ctx, f)
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

	bifrost, err := NewBifrost(mainOptions, false)
	if err != nil {
		slog.Error("fail to start bifrost", "error", err)
		return err
	}
	SetBifrost(bifrost)

	ctx := context.Background()

	config.OnChanged = func() error {
		slog.Debug("reloading...")

		if mainOptions.ConfigPath() != "" {
			newMainOptions, err := config.Load(mainOptions.ConfigPath())
			if err != nil {
				slog.Error("fail to load config", "error", err)
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

	go task.Runner(context.Background(), func() {
		bifrost.Run()
	})

	go task.Runner(context.Background(), func() {
		for _, httpServer := range bifrost.httpServers {
			for {
				conn, err := net.Dial("tcp", httpServer.Bind())
				if err == nil {
					conn.Close()
					break
				}
				time.Sleep(500 * time.Millisecond)
			}
		}

		slog.Log(ctx, log.LevelNotice, "bifrost is started successfully",
			"pid", os.Getpid(),
			"routes", len(bifrost.options.Routes),
			"services", len(bifrost.options.Services),
			"middlewares", len(bifrost.options.Middlewares),
			"upstreams", len(bifrost.options.Upstreams),
		)

		zeroDT := bifrost.ZeroDownTime()

		if zeroDT != nil && zeroDT.IsUpgraded() {
			// shutdown old process
			oldPID, err := zeroDT.GetPID()
			if err != nil {
				return
			}

			err = zeroDT.WritePID()
			if err != nil {
				return
			}

			err = zeroDT.Quit(ctx, oldPID, false)
			if err != nil {
				return
			}
		}

		if mainOptions.IsDaemon {
			err = zeroDT.WritePID()
			if err != nil {
				slog.Error("fail to write pid", "error", err)
				return
			}

			err = zeroDT.RemoveUpgradeSock()
			if err != nil {
				slog.Error("fail to remove upgrade sock file", "error", err)
				return
			}
			if err := bifrost.ZeroDownTime().WaitForUpgrade(ctx); err != nil {
				slog.Error("fail to wait for upgrade process", "error", err)
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
		slog.Error("fail to run as daemon", "error", err)
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
	go task.Runner(context.Background(), func() {
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

func getRandomNumber(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}
