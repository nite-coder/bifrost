package zero

import (
	"context"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	t.Run("upgraded", func(t *testing.T) {
		z := New(Options{})
		z.envGetter = func(k string) string {
			if k == "UPGRADE" {
				return "1"
			}
			return ""
		}
		if !z.IsUpgraded() {
			t.Error("Expected upgraded state")
		}
	})

	t.Run("not upgraded", func(t *testing.T) {
		z := New(Options{})
		z.envGetter = func(string) string { return "" }
		if z.IsUpgraded() {
			t.Error("Expected normal state")
		}
	})
}

func TestListener(t *testing.T) {
	t.Run("new listener creation", func(t *testing.T) {
		z := New(Options{})
		l, err := z.Listener(context.Background(), "tcp", "localhost:0", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		if len(z.listeners) != 1 {
			t.Errorf("Expected 1 listener, got %d", len(z.listeners))
		}
	})

	t.Run("reuse listener when upgraded", func(t *testing.T) {
		// mock upgraded
		z := New(Options{})
		z.envGetter = func(k string) string {
			if k == "UPGRADE" {
				return "1"
			}
			if k == "LISTENERS" {
				return `[{"Key":"localhost:1234"}]`
			}
			return ""
		}

		// mock file descriptor
		z.fileOpener = func(name string) (*os.File, error) {
			return os.NewFile(uintptr(3), ""), nil
		}

		l, err := z.Listener(context.Background(), "tcp", "localhost:1234", nil)
		if err != nil {
			t.Fatal(err)
		}
		if l == nil {
			t.Error("Expected existing listener")
		}
	})
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

func TestPIDOperations(t *testing.T) {
	tempFile, _ := os.CreateTemp("", "pidfile")
	defer os.Remove(tempFile.Name())

	z := New(Options{PIDFile: tempFile.Name()})

	t.Run("write pid", func(t *testing.T) {
		err := z.WritePID()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("read pid", func(t *testing.T) {
		pid, err := z.GetPID()
		if err != nil {
			t.Fatal(err)
		}
		if pid != os.Getpid() {
			t.Errorf("Expected PID %d, got %d", os.Getpid(), pid)
		}
	})
}

type mockProcess struct {
	signals []os.Signal
	pid     int
	killed  bool
	err     error
	wait    bool
}

func (m *mockProcess) Signal(sig os.Signal) error {
	if m.killed {
		return os.ErrProcessDone
	}

	if sig == syscall.SIGTERM && !m.wait {
		m.killed = true
	}

	m.signals = append(m.signals, sig)
	return m.err
}

func (m *mockProcess) Kill() error {
	if !m.wait {
		m.killed = true
	}
	return nil
}

func (m *mockProcess) Wait() (*os.ProcessState, error) {
	return nil, m.err
}

func (m *mockProcess) Release() error {
	return m.err
}

type mockProcessFinder struct {
	proc process
}

func (m *mockProcessFinder) FindProcess(pid int) (process, error) {
	return m.proc, nil
}

func TestKillProcess(t *testing.T) {
	t.Run("normal kill", func(t *testing.T) {
		mp := &mockProcess{pid: 123}

		z := New(Options{})
		z.processFinder = &mockProcessFinder{proc: mp}

		err := z.Quit(context.Background(), 123, false)
		if err != nil {
			t.Fatal(err)
		}

		if len(mp.signals) == 0 || mp.signals[0] != syscall.SIGTERM {
			t.Error("Expected SIGTERM signal")
		}
	})

	t.Run("kill timeout", func(t *testing.T) {
		mp := &mockProcess{pid: 123, wait: true}

		z := New(Options{KillTimout: 2 * time.Second})
		z.processFinder = &mockProcessFinder{proc: mp}

		err := z.Quit(context.Background(), 123, false)
		assert.ErrorIs(t, err, ErrKillTimeout)
	})
}
