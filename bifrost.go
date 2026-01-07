package bifrost

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	stdruntime "runtime"
	"runtime/debug"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/runtime"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/initialize"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/urfave/cli/v2"
	_ "go.uber.org/automaxprocs"
)

type options struct {
	version string
	build   string
	flags   []cli.Flag
	init    func(*cli.Context, config.Options) error
}

type Option func(*options)

// WithVersion sets the application version.
func WithVersion(version string) Option {
	return func(o *options) {
		o.version = version
	}
}

// WithFlags adds custom CLI flags.
func WithFlags(flags ...cli.Flag) Option {
	return func(o *options) {
		o.flags = append(o.flags, flags...)
	}
}

// WithInit registers an initialization hook that runs after config loading
// and before the application starts (Master or Worker).
func WithInit(fn func(*cli.Context, config.Options) error) Option {
	return func(o *options) {
		o.init = fn
	}
}

// Run starts the Bifrost application.
// It handles CLI parsing, configuration loading, and process lifecycle management
// (Master/Worker modes, Hot Reload).
func Run(opts ...Option) error {
	opt := &options{
		version: "0.0.0",
		build:   "unknown",
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				opt.build = setting.Value
				break
			}
		}
	}

	for _, o := range opts {
		o(opt)
	}

	cli.VersionPrinter = func(cliCtx *cli.Context) {
		fmt.Printf("version=%s, build=%s, go=%s\n", cliCtx.App.Version, opt.build, stdruntime.Version())
	}

	app := &cli.App{
		Version: opt.version,
		Flags: append([]cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "",
				Usage:   "The path to the configuration file",
			},
			&cli.BoolFlag{
				Name:    "test",
				Aliases: []string{"t"},
				Value:   false,
				Usage:   "Test the gateway conf and then exit",
			},
			&cli.BoolFlag{
				Name:    "testskip",
				Aliases: []string{"ts"},
				Value:   false,
				Usage:   "Test the gateway conf and skip dns check for upstreams",
			},
		}, opt.flags...),
		Action: func(cCtx *cli.Context) error {
			defer func() {
				if r := recover(); r != nil {
					var err error
					switch v := r.(type) {
					case error:
						err = v
					default:
						err = fmt.Errorf("%v", v)
					}

					stackTrace := debug.Stack()
					slog.Error("unknown error",
						slog.String("error", err.Error()),
						slog.String("stack", cast.B2S(stackTrace)),
					)
					os.Exit(1)
				}
			}()

			_ = initialize.Bifrost()

			configPath := cCtx.String("config")

			isTestAndSkip := cCtx.Bool("testskip")
			if isTestAndSkip {
				path, err := config.TestAndSkipResovler(configPath)
				if err != nil {
					slog.Error("failed to load config", "error", err.Error())
					slog.Info("the configuration file test has failed")
					return err
				}

				slog.Info("the config file tested successfully", "path", path)
				return nil
			}

			isTest := cCtx.Bool("test")
			mainOptions, err := config.Load(configPath)
			if err != nil {
				slog.Error("failed to load config", "error", err.Error())
				if isTest {
					slog.Info("the configuration file test has failed")
				}
				return err
			}

			_ = initialize.Logger(mainOptions)

			if isTest {
				slog.Info("the config file tested successfully", "path", mainOptions.ConfigPath())
				return nil
			}

			if runtime.IsWorker() {
				// Execute User Init Hook
				if opt.init != nil {
					if err := opt.init(cCtx, mainOptions); err != nil {
						return err
					}
				}

				runAsWorker(mainOptions)
				return nil
			}

			return runMasterMode(mainOptions)
		},
	}

	return app.Run(os.Args)
}

// runMasterMode starts the Master process which spawns and manages Worker subprocesses.
// This provides PID stability for process managers like Systemd.
// Master runs in foreground mode - process management is handled by Systemd/Docker/K8s.
func runMasterMode(mainOptions config.Options) error {
	slog.Debug("starting in Master-Worker mode", "pid", os.Getpid())

	masterOpts := &runtime.MasterOptions{
		ConfigPath: mainOptions.ConfigPath(),
	}

	master := runtime.NewMaster(masterOpts)
	return master.Run(context.Background())
}

// runAsWorker handles Worker process logic.
// Workers are spawned by Master and handle actual traffic processing.
func runAsWorker(mainOptions config.Options) {
	// Worker inherits Master's stdout/stderr via FD inheritance (zero-copy log aggregation)
	slog.Debug("starting as worker", "pid", os.Getpid())

	_ = initialize.Bifrost()

	// Connect to Master's control plane
	socketPath := runtime.GetControlSocketPath()
	slog.Debug("worker socket path",
		"env", os.Getenv("BIFROST_CONTROL_SOCKET"),
		"socketPath", socketPath,
	)

	if socketPath != "" {
		wcp := runtime.NewWorkerControlPlane(socketPath)
		if err := wcp.Connect(); err != nil {
			slog.Warn("failed to connect to control plane", "error", err)
		} else {
			slog.Debug("worker connected to control plane", "socket", socketPath) // Success log
			defer wcp.Close()

			// Register with Master
			slog.Debug("worker sending register message")
			if err := wcp.Register(); err != nil {
				slog.Warn("failed to register with master", "error", err)
			} else {
				slog.Debug("worker register message sent")
			}

			// Setup FD handler
			fdHandler := runtime.NewWorkerFDHandler(wcp)

			// Start control plane loop
			go safety.Go(context.Background(), func() {
				if err := wcp.Start(context.Background(), fdHandler); err != nil {
					slog.Error("control plane loop exited with error", "error", err)
				}
			})

			// Asynchronously wait for Bifrost to be ready and register listeners
			go safety.Go(context.Background(), func() {
				// Poll until Bifrost instance is available
				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()

				for range ticker.C {
					b := gateway.GetBifrost()
					if b != nil && b.IsActive() {
						// Bifrost is ready, register listeners
						z := b.ZeroDownTime()
						if z != nil {
							listeners := z.GetListeners()
							for _, l := range listeners {
								fdHandler.RegisterListener(l.Listener, l.Key)
							}
							if len(listeners) > 0 {
								slog.Debug("registered listeners with control plane", "count", len(listeners))
							}
						}

						// Notify Master we are ready
						if err := wcp.NotifyReady(); err != nil {
							slog.Warn("failed to notify master ready", "error", err)
						}
						return
					}
				}
			})

		}
	}

	// Run as worker (uses inherited FDs if UPGRADE=1)
	if err := gateway.Run(mainOptions); err != nil {
		slog.Error("worker failed", "error", err)
		os.Exit(1)
	}
}
