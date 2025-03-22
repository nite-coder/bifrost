package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/initialize"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/urfave/cli/v2"
)

var (
	Build = "commit_id"
)

func main() {
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
				Usage:   "This server should gracefully shutdown a running server",
			},
		},
		Action: func(cCtx *cli.Context) error {
			defer func() {
				if err := recover(); err != nil {
					slog.Error("unknown error", "error", err)
				}
			}()

			var err error

			_ = initialize.Middleware()

			err = registerMiddlewares()
			if err != nil {
				return err
			}

			configPath := cCtx.String("config")

			isTestAndSkip := cCtx.Bool("testskip")
			if isTestAndSkip {
				path, err := config.TestAndSkipResovler(configPath)
				if err != nil {
					slog.Error("fail to load config", "error", err.Error())
					slog.Info("the configuration file test has failed")
					return err
				}

				slog.Info("the config file tested successfully", "path", path)
				return nil
			}

			isTest := cCtx.Bool("test")
			mainOptions, err := config.Load(configPath)
			if err != nil {
				slog.Error("fail to load config", "error", err.Error())
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

			isDaemon := cCtx.Bool("daemon")
			if isDaemon {
				mainOptions.IsDaemon = true
				if os.Getenv("DAEMONIZED") == "" {
					return gateway.RunAsDaemon(mainOptions)
				}
			}

			return gateway.Run(mainOptions)
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("fail to run", "error", err.Error())
		os.Exit(-1)
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
	err := middleware.RegisterMiddleware("find_upstream", func(param any) (app.HandlerFunc, error) {
		m := FindMyHome{}
		return m.ServeHTTP, nil
	})
	if err != nil {
		return err
	}

	return nil
}
