package runtime

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
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
// It collects all listener FDs and keys, then sends them via the control plane.
func (h *WorkerFDHandler) HandleFDRequest() error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.wcp == nil {
		return nil // Not connected to control plane
	}

	files := make([]*os.File, 0, len(h.listeners))
	keys := make([]string, 0, len(h.listeners))

	for _, l := range h.listeners {
		file, err := h.getListenerFile(l.listener)
		if err != nil {
			slog.Error("failed to get listener file", "error", err, "key", l.key)
			continue
		}
		files = append(files, file)
		keys = append(keys, l.key)
	}

	if len(files) == 0 {
		slog.Warn("no listener FDs to transfer")
		return nil
	}

	slog.Info("sending FDs to Master", "count", len(files))
	return h.wcp.SendFDs(files, keys)
}

// getListenerFile extracts the underlying file descriptor from a listener.
func (h *WorkerFDHandler) getListenerFile(listener net.Listener) (*os.File, error) {
	// Handle proxy protocol wrapper
	if proxyListener, ok := listener.(*proxyproto.Listener); ok {
		listener = proxyListener.Listener
	}

	// Get underlying TCPListener
	if tcpListener, ok := listener.(*net.TCPListener); ok {
		return tcpListener.File()
	}

	// Try UnixListener
	if unixListener, ok := listener.(*net.UnixListener); ok {
		return unixListener.File()
	}

	return nil, fmt.Errorf("unsupported listener type: %T", listener)
}

// ListenerCount returns the number of registered listeners.
func (h *WorkerFDHandler) ListenerCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.listeners)
}

// InheritedListeners returns the listener FDs and their keys inherited from Master.
// Worker calls this on startup when UPGRADE=1.
func InheritedListeners() (map[string]*os.File, error) {
	if os.Getenv("UPGRADE") != "1" {
		return nil, nil
	}

	// 1. Decode Keys
	keysEnv := os.Getenv("BIFROST_LISTENER_KEYS")
	keys, err := decodeListenerKeys(keysEnv)
	if err != nil {
		slog.Error("failed to decode BIFROST_LISTENER_KEYS", "error", err)
		return nil, nil
	}

	// 1.5 Sanity Check: BIFROST_FD_COUNT must match key count
	// This prevents accidental usage of FDs (like epoll FDs) in test environments
	// where only keys might be mocked but FDs are not actually passed.
	fdCountEnv := os.Getenv("BIFROST_FD_COUNT")
	if fdCountEnv == "" {
		slog.Warn("BIFROST_FD_COUNT not set, refusing to inherit FDs for safety")
		return nil, nil
	}

	fdCount, err := strconv.Atoi(fdCountEnv)
	if err != nil {
		slog.Error("invalid BIFROST_FD_COUNT", "error", err, "val", fdCountEnv)
		return nil, nil
	}

	if fdCount != len(keys) {
		slog.Error("BIFROST_FD_COUNT mismatch", "env", fdCount, "keys", len(keys))
		return nil, nil
	}

	// 2. Collect FDs (ExtraFiles start at FD 3)
	listeners := make(map[string]*os.File)
	count := 0

	// If we have keys, we map them 1:1 to FDs starting at 3
	if len(keys) > 0 {
		for i, key := range keys {
			fd := 3 + i
			file := os.NewFile(uintptr(fd), "")
			if file == nil {
				slog.Error("inherited FD is nil", "fd", fd)
				continue
			}
			// Verify FD is valid
			if _, err := file.Stat(); err != nil {
				slog.Error("inherited FD is invalid", "fd", fd, "error", err)
				continue
			}
			listeners[key] = file
			count++
			slog.Debug("inherited FD mapped", "fd", fd, "key", key)
		}
	} else {
		// Fallback: Just return list of found FDs (legacy behavior or no keys provided)
		// But return type is map... just map by index?
		// Actually, if we don't have keys, we can't map them correctly to addresses
		// which means `bind: address already in use` will likely happen.
		// However, for backward compatibility or simple cases, we might scan FDs.
		// But in this context, we rely on keys.
		slog.Warn("no listener keys found in env, FD inheritance requires keys for mapping")
		return nil, nil
	}

	if count > 0 {
		slog.Info("inherited listeners from Master", "count", count)
	}

	return listeners, nil
}

// decodeListenerKeys decodes the base64 encoded listener keys from environment variable.
func decodeListenerKeys(keysEnv string) ([]string, error) {
	if keysEnv == "" {
		return nil, nil
	}

	// Use base64 decoding for safety
	decoded, err := base64.StdEncoding.DecodeString(keysEnv)
	if err != nil {
		return nil, err
	}

	keysStr := string(decoded)
	if keysStr == "" {
		return nil, nil
	}

	return strings.Split(keysStr, ","), nil
}
