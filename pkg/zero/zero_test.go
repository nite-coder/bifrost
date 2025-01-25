package zero

import (
	"context"
	"net"
	"os"
	"testing"
	"time"
)

func TestGetPIDFile(t *testing.T) {
	tests := []struct {
		name     string
		pidFile  string
		expected string
	}{
		{"Default", "", "./logs/bifrost.pid"},
		{"Custom", "./custom.pid", "./custom.pid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{PIDFile: tt.pidFile}
			if got := opts.GetPIDFile(); got != tt.expected {
				t.Errorf("GetPIDFile() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetUpgradeSock(t *testing.T) {
	tests := []struct {
		name        string
		upgradeSock string
		expected    string
	}{
		{"Default", "", "./logs/bifrost.sock"},
		{"Custom", "./custom.sock", "./custom.sock"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{UpgradeSock: tt.upgradeSock}
			if got := opts.GetUpgradeSock(); got != tt.expected {
				t.Errorf("GetUpgradeSock() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsUpgraded(t *testing.T) {
	z := New(Options{})

	// Test when UPGRADE env var is not set
	if z.IsUpgraded() {
		t.Error("IsUpgraded() should return false when UPGRADE env var is not set")
	}

	// Test when UPGRADE env var is set
	os.Setenv("UPGRADE", "1")
	defer os.Unsetenv("UPGRADE")
	if !z.IsUpgraded() {
		t.Error("IsUpgraded() should return true when UPGRADE env var is set")
	}
}

func TestListener(t *testing.T) {
	z := New(Options{})
	ctx := context.Background()

	// Test creating a new listener
	l, err := z.Listener(ctx, "tcp", "localhost:7788", nil)
	if err != nil {
		t.Fatalf("Listener() error = %v", err)
	}
	defer l.Close()

	if l == nil {
		t.Error("Listener() returned nil")
	}

	// Test retrieving the same listener
	l2, err := z.Listener(ctx, "tcp", "localhost:7788", nil)
	if err != nil {
		t.Fatalf("Listener() error = %v", err)
	}

	if l != l2 {
		t.Error("Listener() did not return the same listener for the same address")
	}
}

func TestClose(t *testing.T) {
	z := New(Options{})
	ctx := context.Background()

	// Create a listener
	l, err := z.Listener(ctx, "tcp", "localhost:0", nil)
	if err != nil {
		t.Fatalf("Listener() error = %v", err)
	}

	// Close ZeroDownTime
	err = z.Close(ctx)
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Try to use the closed listener
	_, err = l.Accept()
	if err == nil {
		t.Error("Listener should be closed")
	}
}

func TestWaitForUpgrade(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zero_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := Options{
		UpgradeSock: tmpDir + "/test.sock",
		PIDFile:     tmpDir + "/test.pid",
	}
	z := New(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(500 * time.Millisecond)
		conn, err := net.Dial("unix", opts.GetUpgradeSock())
		if err != nil {
			t.Errorf("Failed to connect to upgrade socket: %v", err)
			return
		}
		conn.Close()
		z.Close(ctx)
	}()

	err = z.WaitForUpgrade(ctx)
	if err != nil {
		t.Fatalf("WaitForUpgrade() error = %v", err)
	}
}
