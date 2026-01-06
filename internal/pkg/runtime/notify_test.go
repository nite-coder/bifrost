package runtime

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifySystemdReady(t *testing.T) {
	t.Run("no notify socket", func(t *testing.T) {
		os.Unsetenv("NOTIFY_SOCKET")
		// Should not panic or error
		NotifySystemdReady()
	})

	t.Run("with notify socket", func(t *testing.T) {
		// Create a temporary socket
		tmpDir := t.TempDir()
		socketPath := filepath.Join(tmpDir, "notify.sock")

		// Listen on the socket (Unixgram)
		addr := &net.UnixAddr{Name: socketPath, Net: "unixgram"}
		conn, err := net.ListenUnixgram("unixgram", addr)
		require.NoError(t, err)
		defer conn.Close()

		// Set env var
		os.Setenv("NOTIFY_SOCKET", socketPath)
		defer os.Unsetenv("NOTIFY_SOCKET")

		// Call notify
		NotifySystemdReady()

		// Check if we received data
		buf := make([]byte, 1024)
		conn.SetReadDeadline(func() time.Time { return time.Now().Add(1 * time.Second) }())
		n, _, err := conn.ReadFrom(buf)
		require.NoError(t, err)

		msg := string(buf[:n])
		assert.Contains(t, msg, "READY=1")
	})
}
