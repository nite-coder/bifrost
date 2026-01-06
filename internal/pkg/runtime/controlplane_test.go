package runtime

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewControlPlane(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		cp := NewControlPlane(nil)
		assert.NotNil(t, cp)
		// On Linux it uses Abstract Namespace which starts with @ (or \0)
		// We can't easily assert the default random path without inspecting it, but it shouldn't be empty
		assert.NotEmpty(t, cp.SocketPath())
	})

	t.Run("custom socket path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.sock")
		opts := &ControlPlaneOptions{SocketPath: path}
		cp := NewControlPlane(opts)
		assert.Equal(t, path, cp.SocketPath())
	})
}

func TestControlPlane_Listen(t *testing.T) {
	t.Run("listen creates socket file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "listen.sock")

		cp := NewControlPlane(&ControlPlaneOptions{SocketPath: path})
		err := cp.Listen()
		require.NoError(t, err)
		defer cp.Close()

		// Verify file exists
		_, err = os.Stat(path)
		assert.NoError(t, err)

		// Verify we can connect
		conn, err := net.Dial("unix", path)
		require.NoError(t, err)
		conn.Close()
	})

	t.Run("listen cleans up existing socket", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "cleanup.sock")

		// Create a dummy file
		err := os.WriteFile(path, []byte("dummy"), 0600)
		require.NoError(t, err)

		cp := NewControlPlane(&ControlPlaneOptions{SocketPath: path})
		err = cp.Listen()
		require.NoError(t, err)
		defer cp.Close()

		// Should be a socket now
		fi, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.ModeSocket|0600, fi.Mode()&os.ModeSocket|0600) // approximate check
	})
}

func TestControlPlane_Connection(t *testing.T) {
	t.Run("connection and register", func(t *testing.T) {
		socketPath := filepath.Join(t.TempDir(), "conn.sock")
		cp := NewControlPlane(&ControlPlaneOptions{SocketPath: socketPath})

		// Setup message handler
		msgCh := make(chan *ControlMessage, 1)
		cp.SetMessageHandler(func(conn net.Conn, msg *ControlMessage) {
			msgCh <- msg
		})

		require.NoError(t, cp.Listen())
		defer cp.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Accept connections in background
		go func() {
			_ = cp.Accept(ctx)
		}()

		// Wait for socket to be ready
		time.Sleep(100 * time.Millisecond)

		wcp := NewWorkerControlPlane(socketPath)
		err := wcp.Connect()
		require.NoError(t, err)
		defer wcp.Close()

		// Send Register
		err = wcp.Register()
		require.NoError(t, err)

		// Verify Master received it
		select {
		case msg := <-msgCh:
			assert.Equal(t, MessageTypeRegister, msg.Type)
			assert.Equal(t, os.Getpid(), msg.WorkerPID)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for register message")
		}

		// Send NotifyReady
		err = wcp.NotifyReady()
		require.NoError(t, err)

		// Verify Master received it
		select {
		case msg := <-msgCh:
			assert.Equal(t, MessageTypeReady, msg.Type)
			assert.Equal(t, os.Getpid(), msg.WorkerPID)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for ready message")
		}
	})
}

func TestControlPlane_SendReceiveFDs(t *testing.T) {
	// Create a pair of connected sockets to simulate transport

	socketPath := filepath.Join(t.TempDir(), "fd.sock")
	cp := NewControlPlane(&ControlPlaneOptions{SocketPath: socketPath})

	// Channel to receive FDs
	fdCh := make(chan []*os.File, 1)
	cp.SetFDHandler(func(fds []*os.File, keys []string) {
		fdCh <- fds
	})

	require.NoError(t, cp.Listen())
	defer cp.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = cp.Accept(ctx)
	}()

	// Connect worker
	wcp := NewWorkerControlPlane(socketPath)
	require.NoError(t, wcp.Connect())
	defer wcp.Close()

	// Prepare FDs to send (dummy)
	f1, err := os.CreateTemp(t.TempDir(), "f1")
	require.NoError(t, err)
	defer f1.Close()

	fds := []*os.File{f1}
	keys := []string{"test_key"}

	// Send FDs
	err = wcp.SendFDs(fds, keys)
	require.NoError(t, err)

	// Receive FDs
	select {
	case receivedFDs := <-fdCh:
		require.Len(t, receivedFDs, 1)
		_, err := receivedFDs[0].Stat()
		assert.NoError(t, err)
		receivedFDs[0].Close()
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for FDs")
	}
}

type mockFDHandler struct {
	called chan struct{}
}

func (m *mockFDHandler) HandleFDRequest() error {
	m.called <- struct{}{}
	return nil
}

func TestWorkerControlPlane_Start(t *testing.T) {
	// Setup Master
	socketPath := filepath.Join(t.TempDir(), "worker_start.sock")
	cp := NewControlPlane(&ControlPlaneOptions{SocketPath: socketPath})
	require.NoError(t, cp.Listen())
	defer cp.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = cp.Accept(ctx)
	}()

	// Setup Worker
	wcp := NewWorkerControlPlane(socketPath)
	require.NoError(t, wcp.Connect())
	defer wcp.Close()

	// Mock Signal
	signalCh := make(chan os.Signal, 1)
	wcp.signalFunc = func(pid int, sig os.Signal) error {
		signalCh <- sig
		return nil
	}

	// Mock FDHandler
	fdHandler := &mockFDHandler{called: make(chan struct{}, 1)}

	// Start Worker Loop
	errCh := make(chan error, 1)
	go func() {
		errCh <- wcp.Start(ctx, fdHandler)
	}()

	// Wait for connection to be registered in Master
	require.NoError(t, wcp.Register())
	time.Sleep(100 * time.Millisecond) // Allow master to process registration

	// Test 1: FD Request
	err := cp.SendMessage(wcp.pid, &ControlMessage{Type: MessageTypeFDRequest})
	require.NoError(t, err)

	select {
	case <-fdHandler.called:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("FDHandler not called")
	}

	// Test 2: Shutdown
	err = cp.SendMessage(wcp.pid, &ControlMessage{Type: MessageTypeShutdown})
	require.NoError(t, err)

	select {
	case sig := <-signalCh:
		assert.Equal(t, syscall.SIGTERM, sig)
	case <-time.After(1 * time.Second):
		t.Fatal("Signal not received")
	}

	// Loop should exit nil
	// Note: Start loop returns nil when shutdown message is received
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("Start loop did not exit")
	}
}
