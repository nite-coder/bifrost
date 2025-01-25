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

func TestBufferedFileWriterReopen(t *testing.T) {
	// Create a temporary log file for testing
	tmpFile, err := os.CreateTemp("", "test-log-*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up the temp file after the test

	// Initialize the bufferedFileWriter
	bfw := newBufferedFileWriter(tmpFile, 64*1024) // 64KB buffer size
	go bfw.listenForSignals()

	// Write some data to the file
	data := "Log before reopen\n"
	if _, err := bfw.Write([]byte(data)); err != nil {
		t.Fatalf("Failed to write to file: %v", err)
	}

	// Flush the buffer to ensure data is written to the file
	if err := bfw.Flush(); err != nil {
		t.Fatalf("Failed to flush buffer: %v", err)
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

	// Write more data after the file has been reopened
	data = "Log after reopen\n"
	if _, err := bfw.Write([]byte(data)); err != nil {
		t.Fatalf("Failed to write to file after reopen: %v", err)
	}

	// Flush the buffer again to ensure all data is written to the file
	if err := bfw.Flush(); err != nil {
		t.Fatalf("Failed to flush buffer: %v", err)
	}

	// Read the contents of the new log file
	newLogContent, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read new log file: %v", err)
	}

	// Verify that the new log file contains the expected logs
	expectedLog := "Log after reopen"
	if !strings.Contains(string(newLogContent), expectedLog) {
		t.Errorf("New log file does not contain expected log. Got: %s", string(newLogContent))
	}

	// Read the contents of the rotated log file
	rotatedLogContent, err := os.ReadFile(rotatedFile)
	if err != nil {
		t.Fatalf("Failed to read rotated log file: %v", err)
	}

	// Verify that the rotated log file contains the expected logs
	expectedRotatedLog := "Log before reopen"
	if !strings.Contains(string(rotatedLogContent), expectedRotatedLog) {
		t.Errorf("Rotated log file does not contain expected log. Got: %s", string(rotatedLogContent))
	}
}
