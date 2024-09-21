package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/urfave/cli/v2"
)

var (
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
			var err error

			configPath := cCtx.String("config")
			isTest := cCtx.Bool("test")

			mainOptions, err := config.Load(configPath)
			if err != nil {
				slog.Error(err.Error())
				if isTest {
					slog.Info("config file tested failed")
				}
				return err
			}

			if isTest {
				slog.Info("config file tested successfully", "path", configPath)
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

			err = registerMiddlewares()
			if err != nil {
				return err
			}

			return gateway.Run(mainOptions)
		},
	}

	if err := app.Run(os.Args); err != nil {
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
	err := gateway.RegisterMiddleware("find_upstream", func(param map[string]any) (app.HandlerFunc, error) {
		m := FindMyHome{}
		return m.ServeHTTP, nil
	})
	if err != nil {
		return err
	}

	return nil
}
