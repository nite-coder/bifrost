package accesslog

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
)

func TestBufferedLoggerReopen(t *testing.T) {
	// Create a temporary log file for testing
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up the temp file after the test

	// Configure the BufferedLogger
	opts := config.AccessLogOptions{
		Output:     tmpFile.Name(),
		BufferSize: 64 * 1024, // 64KB
		Flush:      1 * time.Second,
	}

	logger, err := NewBufferedLogger(opts)
	if err != nil {
		t.Fatalf("Failed to create BufferedLogger: %v", err)
	}
	defer logger.Close()

	// Write some logs to the file
	logger.Write("Log before SIGUSR1\n")

	// Flush the buffer to ensure logs are written to the file
	if err := logger.Flush(); err != nil {
		t.Fatalf("Failed to flush logs: %v", err)
	}

	// Simulate log rotation by renaming the current log file
	rotatedFile := tmpFile.Name() + ".rotated"
	if err := os.Rename(tmpFile.Name(), rotatedFile); err != nil {
		t.Fatalf("Failed to rename log file: %v", err)
	}

	// Send SIGUSR1 signal to the current process to trigger log file reopening
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find current process: %v", err)
	}
	if err := process.Signal(syscall.SIGUSR1); err != nil {
		t.Fatalf("Failed to send SIGUSR1 signal: %v", err)
	}

	// Wait for the signal to be processed
	time.Sleep(100 * time.Millisecond)

	// Write more logs after the file has been reopened
	logger.Write("Log after SIGUSR1\n")

	// Flush the buffer again to ensure all logs are written to the file
	if err := logger.Flush(); err != nil {
		t.Fatalf("Failed to flush logs: %v", err)
	}

	// Read the contents of the new log file
	newLogContent, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read new log file: %v", err)
	}

	// Verify that the new log file contains the expected logs
	expectedLog := "Log after SIGUSR1"
	if !strings.Contains(string(newLogContent), expectedLog) {
		t.Errorf("New log file does not contain expected log. Got: %s", string(newLogContent))
	}

	// Read the contents of the rotated log file
	rotatedLogContent, err := os.ReadFile(rotatedFile)
	if err != nil {
		t.Fatalf("Failed to read rotated log file: %v", err)
	}

	// Verify that the rotated log file contains the expected logs
	expectedRotatedLog := "Log before SIGUSR1"
	if !strings.Contains(string(rotatedLogContent), expectedRotatedLog) {
		t.Errorf("Rotated log file does not contain expected log. Got: %s", string(rotatedLogContent))
	}
}
