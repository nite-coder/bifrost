package main

import (
	"context"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/gateway"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/zero"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/urfave/cli/v2"
)

type FindMyHome struct {
}

func (f *FindMyHome) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	logger := log.FromContext(c)
	logger.Info("find my home")
	ctx.Set("$home", "default")
}

var (
	bifrost  *gateway.Bifrost
	Revision = "fafafaf"
)

func main() {
	app := &cli.App{
		Version: "0.0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "./config.yaml",
				Usage:   "The path to the configuration file",
			},
			&cli.BoolFlag{
				Name:    "daemon",
				Aliases: []string{"d"},
				Value:   false,
				Usage:   "Daemonize the server",
			},
			&cli.BoolFlag{
				Name:    "upgrade",
				Aliases: []string{"u"},
				Value:   false,
				Usage:   "This server should gracefully upgrade a running server",
			},
			&cli.BoolFlag{
				Name:    "test",
				Aliases: []string{"t"},
				Value:   false,
				Usage:   "Test the server conf and then exit",
			},
			&cli.BoolFlag{
				Name:    "stop",
				Aliases: []string{"s"},
				Value:   false,
				Usage:   "This server should gracefully shutdown a running server",
			},
		},
		Action: func(cCtx *cli.Context) error {
			ctx := context.Background()
			var err error
			_ = gateway.DisableGopool()

			configPath := cCtx.String("config")
			mainOpts, err := config.Load(configPath)
			if err != nil {
				slog.Error("fail to load config", "error", err, "path", configPath)
				return err
			}

			isTest := cCtx.Bool("test")
			if isTest {
				err = config.Validate(mainOpts)
				if err != nil {
					slog.Error("fail to validate config", "error", err)
					return err
				}

				slog.Info("config is tested successfully")
				return nil
			}

			isStop := cCtx.Bool("stop")
			if isStop {
				zeroOpts := zero.Options{
					UpgradeSock: mainOpts.UpgradeSock,
					PIDFile:     mainOpts.PIDFile,
				}

				zeroDT := zero.New(zeroOpts)

				err = zeroDT.Shutdown(ctx)
				if err != nil {
					slog.Error("fail to stop", "error", err)
					return err
				}
				return nil
			}

			isUpgrade := cCtx.Bool("upgrade")
			if isUpgrade {
				zeroOpts := zero.Options{
					UpgradeSock: mainOpts.UpgradeSock,
					PIDFile:     mainOpts.PIDFile,
				}

				zeroDT := zero.New(zeroOpts)

				if err := zeroDT.Upgrade(); err != nil {
					slog.Error("fail to upgrade", "error", err)
					return err
				}

				return nil
			}

			isDaemon := cCtx.Bool("daemon")
			if isDaemon && os.Getenv("DAEMONIZED") == "" {
				cmd := exec.Command(os.Args[0], os.Args[1:]...)
				cmd.Stdin = nil
				cmd.Stdout = nil
				cmd.Stderr = nil
				cmd.Env = append(os.Environ(), "DAEMONIZED=1")

				err := cmd.Start()
				if err != nil {
					slog.Error("fail to start daemon", "error", err)
					return err
				}

				slog.Info("daemon process started", "pid", cmd.Process.Pid)
				return nil
			}

			err = registerMiddlewares()
			if err != nil {
				return err
			}

			bifrost, err = gateway.Load(mainOpts, false)
			if err != nil {
				slog.Error("fail to start bifrost", "error", err)
				return err
			}

			// shutdown bifrost
			defer func() {
				_ = shutdown(ctx)
			}()

			config.OnChanged = func() error {
				slog.Debug("reloading...")

				_, mainOpts, err := config.LoadDynamic(mainOpts)
				if err != nil {
					slog.Error("fail to load config", "error", err, "path", configPath)
					return err
				}

				err = config.Validate(mainOpts)
				if err != nil {
					slog.Error("fail to validate config", "error", err)
					return err
				}

				newBifrost, err := gateway.Load(mainOpts, true)
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

					time.Sleep(5 * time.Second)
				}

				if isDaemon {
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
		},
	}

	if err := app.Run(os.Args); err != nil {
		//slog.Error("fail to start bifrost", "error", err)
		os.Exit(-1)
	}
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

func registerMiddlewares() error {
	err := gateway.RegisterMiddleware("find_upstream", func(param map[string]any) (app.HandlerFunc, error) {
		m := FindMyHome{}
		return m.ServeHTTP, nil
	})
	if err != nil {
		return err
	}

	return nil
}
