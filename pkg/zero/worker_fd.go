package zero

import (
	"log/slog"
	"net"
	"os"
	"sync"

	proxyproto "github.com/pires/go-proxyproto"
)

// WorkerFDHandler handles FD requests from Master in Worker processes.
// It collects listener file descriptors and sends them to Master via ControlPlane.
type WorkerFDHandler struct {
	listeners []*listenerInfo
	wcp       *WorkerControlPlane
	mu        sync.RWMutex
}

// listenerInfo holds a listener and its key for FD transfer.
type listenerInfo struct {
	listener net.Listener
	key      string
}

// NewWorkerFDHandler creates a new WorkerFDHandler.
func NewWorkerFDHandler(wcp *WorkerControlPlane) *WorkerFDHandler {
	return &WorkerFDHandler{
		wcp:       wcp,
		listeners: make([]*listenerInfo, 0),
	}
}

// RegisterListener registers a listener for FD transfer.
// Call this for each listener the Worker creates or inherits.
func (h *WorkerFDHandler) RegisterListener(listener net.Listener, key string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.listeners = append(h.listeners, &listenerInfo{
		listener: listener,
		key:      key,
	})

	slog.Debug("registered listener for FD transfer", "key", key)
}

// HandleFDRequest handles an FD request from Master.
// It collects all listener FDs and sends them via the control plane.
func (h *WorkerFDHandler) HandleFDRequest() error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.wcp == nil {
		return nil // Not connected to control plane
	}

	files := make([]*os.File, 0, len(h.listeners))

	for _, l := range h.listeners {
		file, err := h.getListenerFile(l.listener)
		if err != nil {
			slog.Error("failed to get listener file", "error", err, "key", l.key)
			continue
		}
		files = append(files, file)
		slog.Debug("collected listener FD", "key", l.key, "fd", file.Fd())
	}

	if len(files) == 0 {
		slog.Warn("no listener FDs to transfer")
		return nil
	}

	slog.Info("sending FDs to Master", "count", len(files))
	return h.wcp.SendFDs(files)
}

// getListenerFile extracts the underlying file descriptor from a listener.
func (h *WorkerFDHandler) getListenerFile(listener net.Listener) (*os.File, error) {
	// Handle proxy protocol wrapper
	if proxyListener, ok := listener.(*proxyproto.Listener); ok {
		listener = proxyListener.Listener
	}

	// Get underlying TCPListener
	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		// Try UnixListener
		if unixListener, ok := listener.(*net.UnixListener); ok {
			return unixListener.File()
		}
		slog.Warn("unsupported listener type for FD extraction")
		return nil, nil
	}

	return tcpListener.File()
}

// ListenerCount returns the number of registered listeners.
func (h *WorkerFDHandler) ListenerCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.listeners)
}

// InheritedFDs returns the listener FDs inherited from Master via ExtraFiles.
// Worker calls this on startup when UPGRADE=1 to get inherited listeners.
func InheritedFDs() []*os.File {
	if os.Getenv("UPGRADE") != "1" {
		return nil
	}

	// ExtraFiles are passed as FD 3, 4, 5, ...
	// FD 0=stdin, 1=stdout, 2=stderr, 3+ are ExtraFiles
	files := make([]*os.File, 0)

	// Try to detect how many FDs were passed
	// Look for BIFROST_FD_COUNT env var or just try opening FDs
	for i := 3; i < 20; i++ { // Reasonable upper limit
		file := os.NewFile(uintptr(i), "")
		if file == nil {
			break
		}

		// Check if the FD is actually valid
		_, err := file.Stat()
		if err != nil {
			file.Close()
			break
		}

		files = append(files, file)
		slog.Debug("inherited FD from Master", "fd", i)
	}

	if len(files) > 0 {
		slog.Info("inherited FDs from Master", "count", len(files))
	}

	return files
}
