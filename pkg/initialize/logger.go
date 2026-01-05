package initialize

import (
	"log/slog"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
)

// Logger initializes the system logger with the provided configuration.
func Logger(mainOptions config.Options) error {
	// system logger
	logger, err := log.NewLogger(mainOptions.Logging)
	if err != nil {
		return err
	}
	slog.SetDefault(logger)
	return nil
}
