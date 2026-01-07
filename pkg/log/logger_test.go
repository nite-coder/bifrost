package log

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestLogging(t *testing.T) {
	options := config.LoggingOtions{
		Level:   "",
		Handler: "json",
		Output:  "",
	}

	logger, err := NewLogger(options)
	assert.NoError(t, err)

	logger.Info("test")
}

func TestLoggingStdout(t *testing.T) {
	options := config.LoggingOtions{
		Level:   "info",
		Handler: "text",
		Output:  "stdout",
	}

	logger, err := NewLogger(options)
	assert.NoError(t, err)
	assert.NotNil(t, logger)

	logger.Info("test stdout output")
}

// TestSIGUSR1Reopen tests the SIGUSR1 signal handling to reopen the log file.
func TestSIGUSR1Reopen(t *testing.T) {
	// Create a temporary log file for testing
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up the temp file after the test

	// Configure the logger to write to the temp file
	opts := config.LoggingOtions{
		Output:  tmpFile.Name(),
		Level:   "info",
		Handler: "text",
	}

	logger, err := NewLogger(opts)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write some logs to the file
	logger.Info("Log before SIGUSR1")

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

	// Wait for the signal to be processed and log file to be recreated
	assert.Eventually(t, func() bool {
		_, err := os.Stat(tmpFile.Name())
		return err == nil
	}, 5*time.Second, 100*time.Millisecond, "Failed to detect log file recreation")

	// Write more logs after the file has been reopened
	logger.Info("Log after SIGUSR1")

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
