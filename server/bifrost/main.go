package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

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
			&cli.BoolFlag{
				Name:    "master",
				Aliases: []string{"m"},
				Value:   false,
				Usage:   "Run in Master-Worker mode (Master spawns Worker subprocess)",
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

			// Check if Master-Worker mode is requested
			isMasterMode := cCtx.Bool("master")
			isDaemon := cCtx.Bool("daemon")

			if isMasterMode {
				// New Master-Worker architecture
				return runMasterMode(mainOptions)
			}

			// Legacy mode: direct execution (backward compatible)
			if isDaemon {
				// Ensure IsDaemon is set so that gateway.Run() activates daemon-specific logic
				// (e.g., PID file management, upgrade monitoring).
				mainOptions.IsDaemon = true
				if os.Getenv("DAEMONIZED") == "" {
					shouldExit, err := gateway.RunAsDaemon(mainOptions)
					if err != nil {
						return err
					}
					if shouldExit {
						return nil
					}
				}
			}

			return gateway.Run(mainOptions)
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("failed to run", "error", err.Error())
		os.Exit(-1)
	}
}

// runMasterMode starts the Master process which spawns and manages Worker subprocesses.
// This provides PID stability for process managers like Systemd.
func runMasterMode(mainOptions config.Options) error {
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

	// Connect to Master's control plane
	socketPath := zero.GetControlSocketPath()
	if socketPath != "" {
		wcp := zero.NewWorkerControlPlane(socketPath)
		if err := wcp.Connect(); err != nil {
			slog.Warn("failed to connect to control plane", "error", err)
		} else {
			defer wcp.Close()

			// Register with Master
			if err := wcp.Register(); err != nil {
				slog.Warn("failed to register with master", "error", err)
			}

			// Notify Master when ready (after starting servers)
			defer func() {
				if err := wcp.NotifyReady(); err != nil {
					slog.Warn("failed to notify master ready", "error", err)
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
