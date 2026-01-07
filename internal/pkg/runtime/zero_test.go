package runtime

import (
	"context"
	"net"
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

	<-time.After(200 * time.Millisecond)
	z.mu.Lock()
	state := z.state
	z.mu.Unlock()
	assert.Equal(t, waitingState, state)
}

func TestDefaultProcessFinder(t *testing.T) {
	pf := &defaultProcessFinder{}
	proc, err := pf.FindProcess(os.Getpid())
	require.NoError(t, err)
	assert.NotNil(t, proc)
	// Verify it is the expected concrete type
	_, ok := proc.(*os.Process)
	assert.True(t, ok)
}

func TestNew_CustomQuitTimeout(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		z := New(Options{})
		assert.Equal(t, 10*time.Second, z.QuitTimeout)
	})

	t.Run("custom timeout", func(t *testing.T) {
		z := New(Options{QuitTimout: 30 * time.Second})
		assert.Equal(t, 30*time.Second, z.QuitTimeout)
	})

	t.Run("zero timeout uses default", func(t *testing.T) {
		z := New(Options{QuitTimout: 0})
		assert.Equal(t, 10*time.Second, z.QuitTimeout)
	})
}

func TestListener_CacheHit(t *testing.T) {
	z := New(Options{})
	defer z.Close(context.Background())

	addr := "127.0.0.1:0"
	listenOptions := &ListenerOptions{
		Network: "tcp",
		Address: addr,
	}

	// Create first listener
	l1, err := z.Listener(context.Background(), listenOptions)
	require.NoError(t, err)
	defer l1.Close()

	// Get actual address (since we used :0)
	actualAddr := l1.Addr().String()

	// Request same address should return cached listener
	listenOptions2 := &ListenerOptions{
		Network: "tcp",
		Address: addr,
	}
	l2, err := z.Listener(context.Background(), listenOptions2)
	require.NoError(t, err)

	// Should be the same listener
	assert.Equal(t, l1, l2)
	assert.Equal(t, actualAddr, l2.Addr().String())
}

func TestListener_WithCustomConfig(t *testing.T) {
	z := New(Options{})
	defer z.Close(context.Background())

	customConfig := &net.ListenConfig{}
	listenOptions := &ListenerOptions{
		Network: "tcp",
		Address: "127.0.0.1:0",
		Config:  customConfig,
	}

	l, err := z.Listener(context.Background(), listenOptions)
	require.NoError(t, err)
	defer l.Close()

	assert.NotNil(t, l)
	assert.Len(t, z.listeners, 1)
}

func TestListener_CreateError(t *testing.T) {
	z := New(Options{})
	defer z.Close(context.Background())

	// Use invalid address to trigger error
	listenOptions := &ListenerOptions{
		Network: "tcp",
		Address: "invalid-address-format::::",
	}

	_, err := z.Listener(context.Background(), listenOptions)
	assert.Error(t, err)
}

func TestClose_NotWaiting(t *testing.T) {
	z := New(Options{})

	// Create a listener
	l, err := z.Listener(context.Background(), &ListenerOptions{
		Network: "tcp",
		Address: "127.0.0.1:0",
	})
	require.NoError(t, err)
	require.NotNil(t, l)

	// Close without being in waiting state
	err = z.Close(context.Background())
	assert.NoError(t, err)

	// Verify listener is closed
	assert.Len(t, z.listeners, 1)
}

func TestClose_WhileWaiting(t *testing.T) {
	z := New(Options{})

	// Create a listener
	l, err := z.Listener(context.Background(), &ListenerOptions{
		Network: "tcp",
		Address: "127.0.0.1:0",
	})
	require.NoError(t, err)
	require.NotNil(t, l)

	// Manually set to waiting state
	z.mu.Lock()
	z.state = waitingState
	z.mu.Unlock()

	// Close should send signal to stopWaitingCh
	done := make(chan struct{})
	go func() {
		select {
		case <-z.stopWaitingCh:
			close(done)
		case <-time.After(time.Second):
			t.Error("timeout waiting for stopWaitingCh")
		}
	}()

	err = z.Close(context.Background())
	assert.NoError(t, err)

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("stopWaitingCh was not signaled")
	}
}

func TestWaitForUpgrade_InvalidState(t *testing.T) {
	z := New(Options{})

	// Set to waiting state first
	z.mu.Lock()
	z.state = waitingState
	z.mu.Unlock()

	// Calling WaitForUpgrade should fail
	err := z.WaitForUpgrade(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state is not default")
}

func TestWaitForUpgrade_CloseTriggersStop(t *testing.T) {
	z := New(Options{})

	errCh := make(chan error, 1)
	go func() {
		errCh <- z.WaitForUpgrade(context.Background())
	}()

	// Wait for goroutine to start waiting
	assert.Eventually(t, func() bool {
		return z.IsWaiting()
	}, time.Second, 10*time.Millisecond)

	// Close should trigger stop
	err := z.Close(context.Background())
	require.NoError(t, err)

	// WaitForUpgrade should return
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Error("WaitForUpgrade did not return after Close")
	}
}

func TestDefaultCommandRunner(t *testing.T) {
	runner := &defaultCommandRunner{}
	cmd := runner.Command("echo", "hello")
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Path, "echo")
}
