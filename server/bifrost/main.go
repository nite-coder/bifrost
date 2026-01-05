package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/initialize"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/zero"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/urfave/cli/v2"
	_ "go.uber.org/automaxprocs"
)

var (
	Build = "commit_id"
)

func main() {
	// Role dispatching: Worker processes are spawned by Master with BIFROST_ROLE=worker
	if zero.IsWorker() {
		runAsWorker()
		return
	}

	// Default: run as Master (or legacy mode if not using Master-Worker pattern)
	runAsMaster()
}

// runAsMaster handles Master process logic and CLI commands.
// In Master-Worker mode, this spawns and manages Worker processes.
// It also handles control commands like -u (upgrade) and -s (stop).
func runAsMaster() {
	cli.VersionPrinter = func(cliCtx *cli.Context) {
		fmt.Printf("version=%s, build=%s\n", cliCtx.App.Version, Build)
	}

	app := &cli.App{
		Version: "0.1.0",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "",
				Usage:   "The path to the configuration file",
			},
			&cli.BoolFlag{
				Name:    "daemon",
				Aliases: []string{"d"},
				Value:   false,
				Usage:   "Daemonize the server (Master-Worker mode)",
			},
			&cli.BoolFlag{
				Name:    "upgrade",
				Aliases: []string{"u"},
				Value:   false,
				Usage:   "Send SIGHUP to running Master to trigger hot reload",
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
			&cli.BoolFlag{
				Name:    "stop",
				Aliases: []string{"s"},
				Value:   false,
				Usage:   "Send SIGTERM to running Master to trigger graceful shutdown",
			},
		},
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
				}
			}()

			var err error

			_ = initialize.Bifrost()

			err = registerMiddlewares()
			if err != nil {
				return err
			}

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

			isStop := cCtx.Bool("stop")
			if isStop {
				return gateway.StopDaemon(mainOptions)
			}

			isUpgrade := cCtx.Bool("upgrade")
			if isUpgrade {
				return gateway.Upgrade(mainOptions)
			}

			if zero.IsWorker() {
				runAsWorker()
				return nil
			}

			isDaemon := cCtx.Bool("daemon")
			return runMasterMode(mainOptions, isDaemon)
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("failed to run", "error", err.Error())
		os.Exit(-1)
	}
}

// runMasterMode starts the Master process which spawns and manages Worker subprocesses.
// This provides PID stability for process managers like Systemd.
func runMasterMode(mainOptions config.Options, isDaemon bool) error {
	if isDaemon {
		// Determine log output for daemon (redirect stdout/stderr)
		// If config specifies a file, use it to unify logs (like Nginx).
		// If config uses stdout/stderr, fallback to master.log to ensure we capture logs in daemon mode.
		logOutput := mainOptions.Logging.Output
		switch logOutput {
		case "":
			logOutput = "/dev/null" // User wants silence
		case "stdout", "stderr":
			logOutput = "./logs/master.log" // Fallback: Daemon cannot write to stdout/stderr
		}

		// Daemonize the process
		daemonOpts := &zero.DaemonOptions{
			PIDFile:   mainOptions.PIDFile,
			LogOutput: logOutput,
		}
		shouldExit, err := zero.Daemonize(daemonOpts)
		if err != nil {
			return fmt.Errorf("failed to daemonize: %w", err)
		}
		if shouldExit {
			return nil
		}
	}

	slog.Info("starting in Master-Worker mode", "pid", os.Getpid())

	masterOpts := &zero.MasterOptions{
		ConfigPath: mainOptions.ConfigPath(),
		PIDFile:    mainOptions.PIDFile,
	}

	master := zero.NewMaster(masterOpts)
	return master.Run(context.Background())
}

// runAsWorker handles Worker process logic.
// Workers are spawned by Master and handle actual traffic processing.
func runAsWorker() {
	// Worker inherits Master's stdout/stderr via FD inheritance (zero-copy log aggregation)
	slog.Debug("starting as worker", "pid", os.Getpid())

	_ = initialize.Bifrost()

	if err := registerMiddlewares(); err != nil {
		slog.Error("failed to register middlewares", "error", err)
		os.Exit(1)
	}

	// Get config path from command line args (passed by Master)
	configPath := ""
	for i, arg := range os.Args {
		if arg == "-c" && i+1 < len(os.Args) {
			configPath = os.Args[i+1]
			break
		}
	}

	mainOptions, err := config.Load(configPath)
	if err != nil {
		slog.Error("worker failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize logger for worker process to match master's format
	_ = initialize.Logger(mainOptions)

	// Connect to Master's control plane
	socketPath := zero.GetControlSocketPath()
	slog.Info("Worker socket path debugging",
		"env", os.Getenv("BIFROST_CONTROL_SOCKET"),
		"socketPath", socketPath,
	)

	if socketPath != "" {
		wcp := zero.NewWorkerControlPlane(socketPath)
		if err := wcp.Connect(); err != nil {
			slog.Warn("failed to connect to control plane", "error", err)
		} else {
			slog.Info("Worker connected to control plane", "socket", socketPath) // Success log
			defer wcp.Close()

			// Register with Master
			slog.Info("Worker sending register message")
			if err := wcp.Register(); err != nil {
				slog.Warn("failed to register with master", "error", err)
			} else {
				slog.Info("Worker register message sent")
			}

			// Setup FD handler
			fdHandler := zero.NewWorkerFDHandler(wcp)

			// Start control plane loop
			go func() {
				if err := wcp.Start(context.Background(), fdHandler); err != nil {
					slog.Error("control plane loop exited with error", "error", err)
				}
			}()

			// Asynchronously wait for Bifrost to be ready and register listeners
			go func() {
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
								fdHandler.RegisterListener(l.Listener, l.Key) // Note: l.Listener needs to be exported from listenInfo?
							}
							if len(listeners) > 0 {
								slog.Info("registered listeners with control plane", "count", len(listeners))
							}
						}

						// Notify Master we are ready
						if err := wcp.NotifyReady(); err != nil {
							slog.Warn("failed to notify master ready", "error", err)
						}
						return
					}
				}
			}()
		}
	}

	// Run as worker (uses inherited FDs if UPGRADE=1)
	if err := gateway.Run(mainOptions); err != nil {
		slog.Error("worker failed", "error", err)
		os.Exit(1)
	}
}

type FindMyHome struct {
}

func (f *FindMyHome) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	logger := log.FromContext(c)
	logger.Info("find my home")
	ctx.Set("$home", "default")
}

func registerMiddlewares() error {
	err := middleware.Register([]string{"find_upstream"}, func(param any) (app.HandlerFunc, error) {
		m := FindMyHome{}
		return m.ServeHTTP, nil
	})
	if err != nil {
		return err
	}

	return nil
}
