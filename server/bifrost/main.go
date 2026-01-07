package main

import (
	"log/slog"
	"os"

	"github.com/nite-coder/bifrost"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/urfave/cli/v2"
)

func main() {
	err := bifrost.Run(
		bifrost.WithVersion("1.0.0"),
		bifrost.WithDebugMode(false), // true: change to single process mode
		bifrost.WithInit(func(c *cli.Context, opts config.Options) error {
			slog.Info("executing init hook")
			return nil
		}),
	)

	if err != nil {
		slog.Error("failed to run bifrost", "error", err)
		os.Exit(1)
	}
}
