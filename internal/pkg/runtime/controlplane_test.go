package runtime

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewControlPlane(t *testing.T) {
	t.Run("default options uses abstract namespace", func(t *testing.T) {
		cp := NewControlPlane(nil)
		// Abstract namespace sockets start with null byte
		assert.True(t, len(cp.options.SocketPath) > 0)
		assert.Equal(t, byte(0), cp.options.SocketPath[0])
	})

	t.Run("custom socket path", func(t *testing.T) {
		opts := &ControlPlaneOptions{
			SocketPath: "/tmp/test.sock",
		}
		cp := NewControlPlane(opts)
		assert.Equal(t, "/tmp/test.sock", cp.options.SocketPath)
	})
}

func TestControlPlane_ListenClose(t *testing.T) {
	// Use abstract namespace socket for test isolation
	cp := NewControlPlane(nil)

	err := cp.Listen()
	require.NoError(t, err)
	defer cp.Close()

	assert.NotNil(t, cp.listener)
}

func TestControlPlane_WorkerConnection(t *testing.T) {
	// Create ControlPlane (Master side)
	cp := NewControlPlane(nil)
	err := cp.Listen()
	require.NoError(t, err)
	defer cp.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Track received messages
	var receivedMsgs []*ControlMessage
	var mu sync.Mutex

	cp.SetMessageHandler(func(conn net.Conn, msg *ControlMessage) {
		mu.Lock()
		receivedMsgs = append(receivedMsgs, msg)
		mu.Unlock()
	})

	// Start accepting connections
	go cp.Accept(ctx)

	// Create Worker side client
	wcp := NewWorkerControlPlane(cp.SocketPath())
	err = wcp.Connect()
	require.NoError(t, err)
	defer wcp.Close()

	// Send register message
	err = wcp.Register()
	require.NoError(t, err)

	// Send ready message
	err = wcp.NotifyReady()
	require.NoError(t, err)

	// Wait for messages to be processed
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(receivedMsgs) >= 2
	}, 2*time.Second, 10*time.Millisecond, "Expected to receive 2 messages")

	// Verify messages
	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, MessageTypeRegister, receivedMsgs[0].Type)
	assert.Equal(t, wcp.pid, receivedMsgs[0].WorkerPID)

	assert.Equal(t, MessageTypeReady, receivedMsgs[1].Type)
	assert.Equal(t, wcp.pid, receivedMsgs[1].WorkerPID)
}

func TestControlPlane_SendMessage(t *testing.T) {
	cp := NewControlPlane(nil)
	err := cp.Listen()
	require.NoError(t, err)
	defer cp.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go cp.Accept(ctx)

	// Connect Worker
	wcp := NewWorkerControlPlane(cp.SocketPath())
	err = wcp.Connect()
	require.NoError(t, err)
	defer wcp.Close()

	// Register to establish connection
	err = wcp.Register()
	require.NoError(t, err)

	// Wait for registration to be processed
	assert.Eventually(t, func() bool {
		cp.mu.RLock()
		defer cp.mu.RUnlock()
		_, ok := cp.conns[wcp.pid]
		return ok
	}, 2*time.Second, 10*time.Millisecond, "Worker should be registered")

	// Master sends message to Worker (would need Worker to read, but this tests the API)
	err = cp.SendMessage(wcp.pid, &ControlMessage{
		Type: MessageTypeShutdown,
	})
	assert.NoError(t, err)
}

func TestControlPlane_SendMessageNoConnection(t *testing.T) {
	cp := NewControlPlane(nil)

	err := cp.SendMessage(99999, &ControlMessage{Type: MessageTypeShutdown})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no connection found")
}

func TestWorkerControlPlane_NotConnected(t *testing.T) {
	wcp := NewWorkerControlPlane("\x00test.sock")

	err := wcp.Register()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")

	err = wcp.NotifyReady()
	assert.Error(t, err)

	err = wcp.SendFDs(nil, nil)
	assert.Error(t, err)
}

func TestControlPlane_ConcurrentWorkers(t *testing.T) {
	cp := NewControlPlane(nil)
	err := cp.Listen()
	require.NoError(t, err)
	defer cp.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var msgCount int
	var mu sync.Mutex

	cp.SetMessageHandler(func(conn net.Conn, msg *ControlMessage) {
		mu.Lock()
		msgCount++
		mu.Unlock()
	})

	go cp.Accept(ctx)

	// Create multiple Worker clients
	const numWorkers = 5
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			wcp := NewWorkerControlPlane(cp.SocketPath())
			if err := wcp.Connect(); err != nil {
				return
			}
			defer wcp.Close()

			_ = wcp.Register()
			_ = wcp.NotifyReady()
		}()
	}

	wg.Wait()

	// Each worker sends 2 messages
	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return msgCount >= numWorkers*2
	}, 2*time.Second, 10*time.Millisecond)
}
