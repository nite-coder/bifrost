package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/nite-coder/bifrost/pkg/config"
)

// fileWriter is a wrapper around os.File to support reopening the file on SIGUSR1.
type fileWriter struct {
	file *os.File
	mu   sync.Mutex // Protects concurrent access to the file
}

// Write implements the io.Writer interface.
func (fw *fileWriter) Write(p []byte) (n int, err error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	return fw.file.Write(p)
}

// reopen closes the current file and reopens it.
func (fw *fileWriter) reopen() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Close the current file
	if fw.file != nil {
		fw.file.Close()
	}

	// Reopen the file
	file, err := os.OpenFile(fw.file.Name(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	fw.file = file

	return nil
}

// NewLogger creates a new slog.Logger instance with the specified options.
func NewLogger(opts config.LoggingOtions) (*slog.Logger, error) {
	var writer io.Writer

	// Configure the slog.HandlerOptions
	logOptions := &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize the name of the level key and the output string
			if a.Key == slog.LevelKey {
				// Handle custom level values
				level := a.Value.Any().(slog.Level)

				if level == LevelNotice {
					a.Value = slog.StringValue("NOTICE")
				}
			}
			return a
		},
	}

	// Parse the log level
	level := strings.TrimSpace(opts.Level)
	level = strings.ToLower(level)

	switch level {
	case "debug":
		logOptions.Level = slog.LevelDebug
	case "info", "":
		logOptions.Level = slog.LevelInfo
	case "warn":
		logOptions.Level = slog.LevelWarn
	case "error":
		logOptions.Level = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	// Determine the output destination
	output := strings.TrimSpace(opts.Output)
	output = strings.ToLower(output)

	switch output {
	case "":
		writer = io.Discard // Discard logs if no output is specified
	case "stderr":
		writer = os.Stderr // Write logs to stderr
	default:
		// Open the log file for appending, creating it if it doesn't exist
		file, err := os.OpenFile(opts.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}

		// Wrap the file in a fileWriter to support reopening
		fw := &fileWriter{file: file}
		writer = fw

		// Listen for SIGUSR1 signals to reopen the log file
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGUSR1) // Register to receive SIGUSR1 signals

			for {
				<-sigChan       // Wait for a SIGUSR1 signal
				_ = fw.reopen() // Reopen the log file
			}
		}()
	}

	// Determine the log handler type
	handler := strings.TrimSpace(opts.Handler)
	handler = strings.ToLower(handler)

	var logHandler slog.Handler

	switch handler {
	case "text", "":
		logHandler = slog.NewTextHandler(writer, logOptions)
	case "json":
		logHandler = slog.NewJSONHandler(writer, logOptions)
	default:
		return nil, fmt.Errorf("handler '%s' is not supported", handler)
	}

	// Create the slog.Logger instance
	logger := slog.New(logHandler)
	return logger, nil
}
