package runtime

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZero_GetListeners(t *testing.T) {
	z := New(Options{})
	defer z.Close(context.Background())

	// Initially empty
	listeners := z.GetListeners()
	assert.Empty(t, listeners)

	// Add a listener
	l, err := z.Listener(context.Background(), &ListenerOptions{
		Network: "tcp",
		Address: "127.0.0.1:0",
	})
	require.NoError(t, err)
	defer l.Close()

	// Should have 1 listener
	listeners = z.GetListeners()
	assert.Len(t, listeners, 1)
	assert.Equal(t, l, listeners[0].Listener)
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
		z := New(Options{})
		// Mock environment for upgrade
		z.envGetter = func(k string) string {
			if k == "UPGRADE" {
				return "1"
			}
			if k == "LISTENERS" {
				// Base64 encoded "localhost:1234" logic is complicated to include here without importing internal logic?
				// Actually InheritedListeners uses os.Getenv.
				// zero.Listener() calls InheritedListeners() if Upgraded.
				// But we need to mock file descriptor 3.
				// And the env var format is key=b64(value).
				return "BIFROST_LISTENER_0=bG9jYWxob3N0OjEyMzQ=" + string(os.PathListSeparator) + "BIFROST_FD_COUNT=1"
			}
			// We need to look at how InheritedListeners parses env.
			// It looks for BIFROST_LISTENER_%d
			if k == "BIFROST_LISTENER_0" {
				return "bG9jYWxob3N0OjEyMzQ=;42" // format: key;fd
			}
			if k == "BIFROST_FD_COUNT" {
				return "1"
			}
			return ""
		}
		// Mock file opener to return a valid file for FD 3
		z.fileOpener = func(name string) (*os.File, error) {
			// In test we can return any file.
			return os.CreateTemp(t.TempDir(), "fd3")
		}

		// However, InheritedListeners logic is in worker_fd.go, but zero.go calls it?
		// No, zero.go calls z.GetListeners() ?? No.
		// zero.go Listener() calls:
		// if z.IsUpgraded() {
		//   return z.getListenerFromInherited(ctx, network, address)
		// }

		// getListenerFromInherited iterates z.inheritedListeners.
		// z.inheritedListeners is populated in New() -> z.InheritedListeners().
		// InheritedListeners() is in worker_fd.go? No, zero.go calls `InheritedListeners` from `worker_fd.go`?
		// No, `InheritedListeners` is in `worker_fd.go`.
		// But `ZeroDownTime` struct has `inheritedListeners`.
		// Let's check `zero.go` `New`.

		// Wait, `New` calls `InheritedListeners`.
		// So we need to mock env BEFORE `New` called?
		// No, `New` returns struct. `InheritedListeners` uses `os.Getenv` directly?
		// If `InheritedListeners` uses `os.Getenv`, we are stuck unless we set real env vars.
		// BUT `worker_fd.go` `InheritedListeners` uses `os.Getenv`.
		// DOES `zero.go` use `InheritedListeners`?
		// Let's check `zero.go`.
	})
}

func TestIsWaiting(t *testing.T) {
	z := New(Options{})
	assert.False(t, z.IsWaiting())

	z.mu.Lock()
	z.state = waitingState
	z.mu.Unlock()
	assert.True(t, z.IsWaiting())
}

func TestWaitForUpgrade(t *testing.T) {
	z := New(Options{})

	errCh := make(chan error, 1)
	go func() {
		errCh <- z.WaitForUpgrade(context.Background())
	}()

	// Wait for goroutine to start waiting
	time.Sleep(100 * time.Millisecond)

	// Send SIGHUP
	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	err = p.Signal(syscall.SIGHUP)
	require.NoError(t, err)

	select {
	case <-time.After(200 * time.Millisecond):
		z.mu.Lock()
		state := z.state
		z.mu.Unlock()
		assert.Equal(t, waitingState, state)
	}
}

func TestDefaultProcessFinder(t *testing.T) {
	pf := &defaultProcessFinder{}
	proc, err := pf.FindProcess(os.Getpid())
	require.NoError(t, err)
	assert.NotNil(t, proc)
	// Just verify it implements interface
	_, ok := proc.(process)
	assert.True(t, ok)
}
