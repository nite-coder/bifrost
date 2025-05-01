package accesslog

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/variable"
)

// BufferedLogger is a logger that buffers log entries and writes them to a file or stderr.
type BufferedLogger struct {
	writer     *bufio.Writer
	file       *os.File
	mu         sync.Mutex // Protects concurrent access to the writer and file
	flushTimer *time.Timer
	options    *config.AccessLogOptions
}

// NewBufferedLogger creates a new BufferedLogger instance.
func NewBufferedLogger(opts config.AccessLogOptions) (*BufferedLogger, error) {
	var writer io.Writer

	logger := &BufferedLogger{
		options: &opts,
	}

	// Determine the output destination
	output := strings.ToLower(opts.Output)
	switch output {
	case "":
		writer = io.Discard // Discard logs if no output is specified
	case "stderr":
		writer = os.Stderr // Write logs to stderr
	default:
		// Open the log file for appending, creating it if it doesn't exist
		file, err := os.OpenFile(opts.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY|syscall.O_CLOEXEC, 0600)
		if err != nil {
			return nil, err
		}
		logger.file = file
		writer = file
	}

	// Set the buffer size (default to 64KB if not specified)
	if opts.BufferSize <= 0 {
		opts.BufferSize = 64 * variable.KB
	}
	logger.writer = bufio.NewWriterSize(writer, opts.BufferSize)

	// Set up periodic flushing if a flush interval is specified
	if opts.Flush.Seconds() > 0 {
		logger.flushTimer = time.AfterFunc(opts.Flush, logger.periodicFlush)
	}

	// Start listening for SIGUSR1 signals to reopen the log file
	go safety.Go(context.Background(), logger.listenForSignals)

	return logger, nil
}

// Write writes a log entry to the buffer.
func (l *BufferedLogger) Write(log string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, _ = l.writer.WriteString(log)
}

// Flush flushes the buffer to the underlying file.
func (l *BufferedLogger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Flush the buffer
	err := l.writer.Flush()
	if err != nil {
		return err
	}

	// Sync the file to ensure data is written to disk
	if l.file != nil {
		err = l.file.Sync()
		if err != nil {
			return err
		}
	}

	return nil
}

// periodicFlush is called periodically to flush the buffer.
func (l *BufferedLogger) periodicFlush() {
	_ = l.Flush()
	l.flushTimer.Reset(l.options.Flush) // Reset the timer for the next flush
}

// Close closes the logger and releases resources.
func (l *BufferedLogger) Close() error {
	if l.flushTimer != nil {
		l.flushTimer.Stop() // Stop the periodic flush timer
	}

	// Flush any remaining logs in the buffer
	_ = l.Flush()

	// Close the file if it's not stderr
	if l.file != nil && l.file != os.Stderr {
		return l.file.Close()
	}

	return nil
}

// listenForSignals listens for SIGUSR1 signals to reopen the log file.
func (l *BufferedLogger) listenForSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1) // Register to receive SIGUSR1 signals

	for {
		<-sigChan          // Wait for a SIGUSR1 signal
		_ = l.reopenFile() // Reopen the log file
	}
}

// reopenFile closes the current log file and reopens it.
func (l *BufferedLogger) reopenFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Flush the buffer to ensure all logs are written to the file
	if err := l.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %w", err)
	}

	// Close the current file if it's not stderr
	if l.file != nil && l.file != os.Stderr {
		if err := l.file.Close(); err != nil {
			return fmt.Errorf("failed to close file: %w", err)
		}
	}

	// Reopen the file
	file, err := os.OpenFile(l.options.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return fmt.Errorf("failed to reopen file: %w", err)
	}
	l.file = file

	// Update the writer to use the new file
	l.writer.Reset(file)

	return nil
}
