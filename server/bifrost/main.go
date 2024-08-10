package main

import (
	"context"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/gateway"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/zero"
	"log/slog"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/urfave/cli/v2"
	_ "go.uber.org/automaxprocs"
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
			defer func() {
				if bifrost != nil {
					bifrost.Shutdown()
				}
			}()

			var err error
			_ = gateway.DisableGopool()

			err = gateway.RegisterMiddleware("find_upstream", func(param map[string]any) (app.HandlerFunc, error) {
				m := FindMyHome{}
				return m.ServeHTTP, nil
			})
			if err != nil {
				return err
			}

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

				return nil
			}

			isUpgrade := cCtx.Bool("upgrade")
			if isUpgrade {
				zeroOpts := zero.Options{
					UpgradeSock: mainOpts.UpgradeSock,
					PIDFile:     mainOpts.UpgradeSock,
				}

				zeroDT := zero.New(zeroOpts)

				if err := zeroDT.Upgrade(); err != nil {
					slog.Error("fail to upgrade", "error", err)
					return err
				}

				return nil
			}

			bifrost, err = gateway.Load(mainOpts, false)
			if err != nil {
				slog.Error("fail to start bifrost", "error", err)
				return err
			}

			config.OnChanged = func() error {
				slog.Info("reloading...")

				mainOpts, err := config.Load(configPath)
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

			bifrost.Run()
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		//slog.Error("fail to start bifrost", "error", err)
		os.Exit(-1)
	}
}
