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
		bifrost.WithVersion("0.1.0"),
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
