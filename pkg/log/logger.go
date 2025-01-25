package log

import (
	"bufio"
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

		// Wrap the file in a bufferedFileWriter for better performance and reopen support
		bfw := newBufferedFileWriter(file, 64*1024) // 64KB buffer size
		writer = bfw

		// Start listening for SIGUSR1 signals
		go bfw.listenForSignals()
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

// bufferedFileWriter wraps a bufio.Writer and *os.File to support buffered writing and file reopening.
type bufferedFileWriter struct {
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex // Protects concurrent access to file and writer
}

// newBufferedFileWriter creates a new bufferedFileWriter instance.
func newBufferedFileWriter(file *os.File, bufferSize int) *bufferedFileWriter {
	return &bufferedFileWriter{
		file:   file,
		writer: bufio.NewWriterSize(file, bufferSize),
	}
}

// Write writes data to the buffered writer.
func (bfw *bufferedFileWriter) Write(p []byte) (n int, err error) {
	bfw.mu.Lock()
	defer bfw.mu.Unlock()

	return bfw.writer.Write(p)
}

// Flush flushes the buffered writer to the underlying file.
func (bfw *bufferedFileWriter) Flush() error {
	bfw.mu.Lock()
	defer bfw.mu.Unlock()

	// 1. Flush bufio.Writer to *os.File
	if err := bfw.writer.Flush(); err != nil {
		fmt.Printf("Failed to flush buffer: %v\n", err)
		return fmt.Errorf("flush buffer failed: %w", err)
	}

	// 2. Sync *os.File to disk
	if bfw.file != nil {
		if err := bfw.file.Sync(); err != nil {
			fmt.Printf("Failed to sync file: %v\n", err)
			return fmt.Errorf("sync file failed: %w", err)
		}
	}

	return nil
}

// reopen closes the current file and reopens it.
func (bfw *bufferedFileWriter) reopen() error {
	bfw.mu.Lock()
	defer bfw.mu.Unlock()

	// 打印当前文件路径
	filePath := bfw.file.Name()
	fmt.Printf("Reopening file: %s\n", filePath)

	if err := bfw.writer.Flush(); err != nil {
		return fmt.Errorf("flush error: %w", err)
	}

	if bfw.file != nil {
		if err := bfw.file.Close(); err != nil {
			return fmt.Errorf("close error: %w", err)
		}
	}

	// 再次打印路径，确认没有变化
	fmt.Printf("Attempting to open file: %s\n", filePath)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("OpenFile failed: %v (path: %s)\n", err, filePath) // 明确打印错误和路径
		return fmt.Errorf("open error: %w", err)
	}

	bfw.file = file
	bfw.writer.Reset(file)
	return nil
}

// listenForSignals listens for SIGUSR1 signals to reopen the log file.
func (bfw *bufferedFileWriter) listenForSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1) // Register to receive SIGUSR1 signals

	for {
		<-sigChan // Wait for a SIGUSR1 signal
		if err := bfw.reopen(); err != nil {
			// Log the error or handle it as needed
			fmt.Printf("Failed to reopen log file: %v\n", err)
		}
	}
}
