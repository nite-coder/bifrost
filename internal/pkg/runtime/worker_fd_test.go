package runtime

import (
	"encoding/base64"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	proxyproto "github.com/pires/go-proxyproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeListenerKeys(t *testing.T) {
	t.Run("valid keys", func(t *testing.T) {
		keys := []string{"127.0.0.1:8080", "127.0.0.1:9090"}
		encoded := base64.StdEncoding.EncodeToString([]byte(strings.Join(keys, ",")))

		decodedKeys, err := decodeListenerKeys(encoded)
		assert.NoError(t, err)
		assert.NotNil(t, decodedKeys)
		assert.Contains(t, decodedKeys, "127.0.0.1:8080")
		assert.Contains(t, decodedKeys, "127.0.0.1:9090")
	})

	t.Run("invalid base64", func(t *testing.T) {
		decodedKeys, err := decodeListenerKeys("invalid-base64")
		assert.Error(t, err)
		assert.Nil(t, decodedKeys)
	})

	t.Run("empty environment variable", func(t *testing.T) {
		decodedKeys, err := decodeListenerKeys("")
		assert.NoError(t, err)
		assert.Nil(t, decodedKeys)
	})
}

// Mock ControlPlaneClient
type mockControlPlaneClient struct {
	sentFDs  []*os.File
	sentKeys []string
	err      error
}

func (m *mockControlPlaneClient) SendFDs(files []*os.File, keys []string) error {
	m.sentFDs = files
	m.sentKeys = keys
	return m.err
}

func TestWorkerFDHandler(t *testing.T) {
	t.Run("register listener", func(t *testing.T) {
		mockClient := &mockControlPlaneClient{}
		handler := NewWorkerFDHandler(mockClient)

		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer l.Close()

		handler.RegisterListener(l, "test-key")

		assert.Equal(t, 1, handler.ListenerCount())
	})

	t.Run("handle valid fd request", func(t *testing.T) {
		mockClient := &mockControlPlaneClient{}
		handler := NewWorkerFDHandler(mockClient)

		// Create real listeners
		l1, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer l1.Close()

		l2, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer l2.Close()

		handler.RegisterListener(l1, "key1")
		handler.RegisterListener(l2, "key2")

		err = handler.HandleFDRequest()
		assert.NoError(t, err)

		assert.Len(t, mockClient.sentFDs, 2)
		assert.Len(t, mockClient.sentKeys, 2)
		assert.Contains(t, mockClient.sentKeys, "key1")
		assert.Contains(t, mockClient.sentKeys, "key2")
	})

	t.Run("handle request with no listeners", func(t *testing.T) {
		mockClient := &mockControlPlaneClient{}
		handler := NewWorkerFDHandler(mockClient)

		err := handler.HandleFDRequest()
		assert.NoError(t, err)
		assert.Empty(t, mockClient.sentFDs)
	})

	t.Run("client error propagation", func(t *testing.T) {
		mockClient := &mockControlPlaneClient{err: assert.AnError}
		handler := NewWorkerFDHandler(mockClient)

		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer l.Close()

		handler.RegisterListener(l, "key1")

		err = handler.HandleFDRequest()
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestInheritedListeners(t *testing.T) {
	// Override startFD for this test to avoid conflicting with test runner
	oldStartFD := startFD
	startFD = 42
	defer func() { startFD = oldStartFD }()

	// Create a valid FD at startFD (42) using Dup2
	f, err := os.CreateTemp(t.TempDir(), "test_fd")
	require.NoError(t, err)
	defer f.Close()

	// Dup to 42
	err = syscall.Dup2(int(f.Fd()), 42)
	require.NoError(t, err)
	// We should close 42 when done, or rely on OS cleanup, but explicit is better for test hygiene
	defer syscall.Close(42)

	// Setup env
	keys := []string{"localhost:1234"}
	encoded := base64.StdEncoding.EncodeToString([]byte(strings.Join(keys, ",")))
	t.Setenv("BIFROST_LISTENER_KEYS", encoded)
	t.Setenv("BIFROST_FD_COUNT", "1")
	t.Setenv("UPGRADE", "1")

	listeners, err := InheritedListeners()
	require.NoError(t, err)
	require.NotNil(t, listeners)
	assert.Len(t, listeners, 1)

	file := listeners["localhost:1234"]
	require.NotNil(t, file)
	assert.Equal(t, uintptr(42), file.Fd())

	// Sanity check invalid count
	t.Setenv("BIFROST_FD_COUNT", "0")
	listeners, err = InheritedListeners()
	assert.Nil(t, listeners)
}

func TestWorkerFDHandler_GetListenerFile(t *testing.T) {
	h := NewWorkerFDHandler(nil)

	// Case 1: TCP Listener (Covered by other tests efficiently, but good to have explicit)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	file, err := h.getListenerFile(l)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	file.Close()

	// Case 2: Unix Listener
	tmpSocket := filepath.Join(t.TempDir(), "test.sock")
	ul, err := net.Listen("unix", tmpSocket)
	require.NoError(t, err)
	defer ul.Close()
	file, err = h.getListenerFile(ul)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	file.Close()

	// Case 3: Proxy Protocol Listener
	pl := &proxyproto.Listener{Listener: l}
	file, err = h.getListenerFile(pl)
	assert.NoError(t, err) // Should unwrap and succeed
	assert.NotNil(t, file)
	file.Close()

	// Case 4: Unsupported Listener
	badListener := &dummyListener{}
	file, err = h.getListenerFile(badListener)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported listener type")
	assert.Nil(t, file)
}

type dummyListener struct{}

func (d *dummyListener) Accept() (net.Conn, error) { return nil, nil }
func (d *dummyListener) Close() error              { return nil }
func (d *dummyListener) Addr() net.Addr            { return nil }
