package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/nite-coder/bifrost/pkg/config"
)

func NewLogger(opts config.LoggingOtions) (*slog.Logger, error) {
	var err error

	logOptions := &slog.HandlerOptions{}

	level := strings.TrimSpace(opts.Level)
	level = strings.ToLower(level)

	switch level {
	case "":
	case "debug":
		logOptions.Level = slog.LevelDebug
	case "info":
		logOptions.Level = slog.LevelInfo
	case "warn":
		logOptions.Level = slog.LevelWarn
	case "error":
		logOptions.Level = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	var writer io.Writer

	output := strings.TrimSpace(opts.Output)
	output = strings.ToLower(output)

	switch output {
	case "stderr", "":
		writer = os.Stderr
	default:
		writer, err = os.OpenFile(opts.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
	}

	if level == "none" {
		logOptions.Level = slog.LevelError
		writer = io.Discard
	}

	var logHandler slog.Handler

	handler := strings.TrimSpace(opts.Handler)
	handler = strings.ToLower(handler)

	switch handler {
	case "text", "":
		logHandler = slog.NewTextHandler(writer, logOptions)
	case "json":
		logHandler = slog.NewJSONHandler(writer, logOptions)
	default:
		return nil, fmt.Errorf("handler '%s' is not supported", handler)
	}

	logger := slog.New(logHandler)
	return logger, nil
}
