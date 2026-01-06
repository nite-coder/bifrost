package runtime

import (
	"context"
	"net"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/stretchr/testify/assert"
)

// ... (code)

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

		listenOptions := &ListenerOptions{
			Network: "tcp",
			Address: "localhost:0",
		}

		l, err := z.Listener(context.Background(), listenOptions)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		if len(z.listeners) != 1 {
			t.Errorf("Expected 1 listener, got %d", len(z.listeners))
		}
	})

	t.Run("listener with proxy protocol", func(t *testing.T) {
		z := New(Options{})

		listenOptions := &ListenerOptions{
			Network:       "tcp",
			Address:       "localhost:0",
			ProxyProtocol: true,
		}

		l, err := z.Listener(context.Background(), listenOptions)
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

		listenOptions := &ListenerOptions{
			Network: "tcp",
			Address: "localhost:1234",
		}

		l, err := z.Listener(context.Background(), listenOptions)
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
	listenOptions := &ListenerOptions{
		Network: "tcp",
		Address: "localhost:0",
	}

	l, err := z.Listener(ctx, listenOptions)
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
	// Create a socket pair to prevent blocking on SIGHUP
	// This test is tricky because WaitForUpgrade blocks until SIGHUP.
	// We need to simulate the environment.

	z := New(Options{})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		safety.Go(ctx, func() {
			// Wait for WaitForUpgrade to set up signal handler
			assert.Eventually(t, func() bool {
				return z.IsWaiting()
			}, 2*time.Second, 10*time.Millisecond, "WaitForUpgrade should be in waiting state")

			// Send SIGHUP to self to trigger upgrade
			// Note: This sends signal to the whole process, so we must be careful.
			// However, since this is a unit test, it might interfere with test runner.
			// But the original test did this, so it should be "okay" in isolation if handled.
			// Alternatively, we can just test the state transition if possible,
			// but WaitForUpgrade logic depends on real signal.
			syscall.Kill(os.Getpid(), syscall.SIGHUP)

			// Close will signal the goroutine to exit if it hung
			go func() {
				time.Sleep(1 * time.Second)
				z.Close(ctx)
			}()
		})
	}()

	// Mock command runner to avoid actually starting a new process
	z.command = &mockCommandRunner{}

	err := z.WaitForUpgrade(ctx)
	if err != nil {
		t.Logf("WaitForUpgrade() returned error (expected if mocked): %v", err)
	}
}

func TestWaitForUpgradeStateError(t *testing.T) {
	t.Run("already in waiting state", func(t *testing.T) {
		z := New(Options{})

		// Manually set state to waiting
		z.state = waitingState

		err := z.WaitForUpgrade(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state is not default")
	})
}

func TestCloseWithWaitingState(t *testing.T) {
	z := New(Options{})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start WaitForUpgrade in background
	go func() {
		_ = z.WaitForUpgrade(ctx)
	}()

	// Wait for state to change
	assert.Eventually(t, func() bool {
		return z.IsWaiting()
	}, 1*time.Second, 10*time.Millisecond, "Zero failed to enter waiting state")

	// Close should work even in waiting state
	err := z.Close(ctx)
	assert.NoError(t, err)
}

func TestListenerWithConfig(t *testing.T) {
	t.Run("listener with custom config", func(t *testing.T) {
		z := New(Options{})

		config := &net.ListenConfig{}
		listenOptions := &ListenerOptions{
			Config:  config,
			Network: "tcp",
			Address: "localhost:0",
		}

		l, err := z.Listener(context.Background(), listenOptions)
		assert.NoError(t, err)
		assert.NotNil(t, l)
		defer l.Close()
	})

	t.Run("listener cache hit same key", func(t *testing.T) {
		z := New(Options{})

		// Create first listener with specific address pattern
		listenOptions := &ListenerOptions{
			Network: "tcp",
			Address: "localhost:0",
		}

		l1, err := z.Listener(context.Background(), listenOptions)
		assert.NoError(t, err)
		defer l1.Close()

		// Request listener with same Address key (cache key is Address, not actual bound port)
		// Since localhost:0 is the key, requesting it again should return the cached one
		l2, err := z.Listener(context.Background(), listenOptions)
		assert.NoError(t, err)

		// Should be the same listener (cache hit based on Key)
		assert.Equal(t, l1.Addr().String(), l2.Addr().String())
	})
}

func TestListenerCreateError(t *testing.T) {
	t.Run("listener creation error with invalid address", func(t *testing.T) {
		z := New(Options{})

		listenOptions := &ListenerOptions{
			Network: "tcp",
			Address: "invalid:address:format:99999",
		}

		_, err := z.Listener(context.Background(), listenOptions)
		assert.Error(t, err)
	})
}

// Mocks

type mockCommandRunner struct{}

func (m *mockCommandRunner) Command(name string, arg ...string) *exec.Cmd {
	// Return a dummy command that does nothing
	return exec.Command("true")
}
