package main

import (
	"log/slog"
	"os"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/initialize"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Version: "0.1.0",
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

				_ = initialize.Bifrost()

			configPath := cCtx.String("config")
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
				slog.Info("the config file tested successfully", "path", configPath)
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
		os.Exit(-1)
	}
}
