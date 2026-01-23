package gateway

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
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
				_ = newHTTPServer.Shutdown(ctx)
				isReloaded = true
			}
		}

		if oldBifrost != nil {
			_ = oldBifrost.Close()

			// Update oldBifrost with new resources to prevent leak in next reload
			// and to ensure GetBifrost() returns correct data
			oldBifrost.services = newBifrost.services
			oldBifrost.resolver = newBifrost.resolver
			oldBifrost.middlewares = newBifrost.middlewares
			oldBifrost.options = newBifrost.options
			// Note: tracer/metrics providers are not updated here as they are not re-created on reload
			// or are handled differently (tracer created but maybe not easily swappable without restart)
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

		// In Master-Worker mode, the Master handles hot reload via SIGHUP
		// Worker just needs to run until it receives SIGTERM
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
