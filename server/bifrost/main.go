package main

import (
	"context"
	"http-benchmark/pkg/gateway"
	"http-benchmark/pkg/log"
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
				panic(err)
			}

			configPath := cCtx.String("config")
			bifrost, err = gateway.LoadFromConfig(configPath)
			if err != nil {
				slog.Error("fail to start bifrost", "error", err)
				return nil
			}

			bifrost.Run()
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("fail to start bifrost", "error", err)
		os.Exit(-1)
	}
}
