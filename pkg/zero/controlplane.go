package zero

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"syscall"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"golang.org/x/sys/unix"
)

// MessageType represents the type of control message sent via UDS.
type MessageType string

const (
	// MessageTypeRegister is sent by Worker to Master upon startup.
	MessageTypeRegister MessageType = "register"
	// MessageTypeReady is sent by Worker to Master when ready to serve traffic.
	MessageTypeReady MessageType = "ready"
	// MessageTypeFDRequest is sent by Master to Worker to request listener FDs.
	MessageTypeFDRequest MessageType = "fd_request"
	// MessageTypeFDTransfer is sent by Worker to Master to transfer listener FDs.
	MessageTypeFDTransfer MessageType = "fd_transfer"
	// MessageTypeShutdown is sent by Master to Worker to request graceful shutdown.
	MessageTypeShutdown MessageType = "shutdown"
)

// ControlMessage represents a message exchanged between Master and Worker.
type ControlMessage struct {
	// Type is the message type.
	Type MessageType `json:"type"`
	// WorkerPID is the PID of the Worker sending the message.
	WorkerPID int `json:"worker_pid,omitempty"`
	// Payload contains optional additional data.
	Payload []byte `json:"payload,omitempty"`
}

// ControlPlaneOptions contains configuration for the ControlPlane.
type ControlPlaneOptions struct {
	// SocketPath is the UDS path. If empty, uses Abstract Namespace.
	// Abstract Namespace format: "\x00bifrost-{pid}.sock"
	SocketPath string
}

// ControlPlane provides Unix Domain Socket communication between Master and Worker.
// It uses Linux Abstract Namespace to avoid filesystem issues.
type ControlPlane struct {
	options    *ControlPlaneOptions
	listener   net.Listener
	conns      map[int]net.Conn // WorkerPID -> connection
	mu         sync.RWMutex
	closed     bool
	msgHandler func(conn net.Conn, msg *ControlMessage) // Message handler callback
}

// NewControlPlane creates a new ControlPlane instance.
// The Master calls this to create the UDS server.
//
// Socket path priority:
//  1. If opts.SocketPath is provided (from config.UpgradeSock), use it as file socket
//  2. On Linux: use Abstract Namespace (no filesystem cleanup needed)
//  3. On macOS/BSD: use file socket in /tmp
func NewControlPlane(opts *ControlPlaneOptions) *ControlPlane {
	if opts == nil {
		opts = &ControlPlaneOptions{}
	}
	if opts.SocketPath == "" {
		// Use platform-specific default
		opts.SocketPath = getDefaultSocketPath()
	}
	return &ControlPlane{
		options: opts,
		conns:   make(map[int]net.Conn),
	}
}

// SetMessageHandler sets the callback for handling incoming messages.
func (cp *ControlPlane) SetMessageHandler(handler func(conn net.Conn, msg *ControlMessage)) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.msgHandler = handler
}

// Listen starts the UDS server (Master side).
// For file-based sockets, it cleans up any stale socket files first.
// Note: Go 1.12+ automatically sets CLOEXEC on all FDs, preventing leak to child processes.
func (cp *ControlPlane) Listen() error {
	// For file-based sockets (not Abstract Namespace), clean up stale socket
	if len(cp.options.SocketPath) > 0 && cp.options.SocketPath[0] != 0 {
		// Remove stale socket file if it exists
		os.Remove(cp.options.SocketPath)
	}

	lc := net.ListenConfig{}
	listener, err := lc.Listen(context.Background(), "unix", cp.options.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on UDS %s: %w", cp.options.SocketPath, err)
	}
	cp.listener = listener
	slog.Debug("control plane listening", "socket", cp.options.SocketPath)
	return nil
}

// Accept accepts incoming Worker connections and handles messages.
// This blocks and should be run in a goroutine.
func (cp *ControlPlane) Accept(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := cp.listener.Accept()
		if err != nil {
			cp.mu.RLock()
			closed := cp.closed
			cp.mu.RUnlock()
			if closed {
				return nil
			}
			slog.Error("failed to accept connection", "error", err)
			continue
		}

		go safety.Go(ctx, func() {
			cp.handleConnection(ctx, conn)
		})
	}
}

// handleConnection handles a single Worker connection.
func (cp *ControlPlane) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var msg ControlMessage
		if err := decoder.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			slog.Error("failed to decode control message", "error", err)
			return
		}

		// Register the connection by WorkerPID
		if msg.WorkerPID > 0 {
			cp.mu.Lock()
			cp.conns[msg.WorkerPID] = conn
			cp.mu.Unlock()
		}

		// Call the message handler if set
		cp.mu.RLock()
		handler := cp.msgHandler
		cp.mu.RUnlock()

		if handler != nil {
			handler(conn, &msg)
		}

		slog.Debug("received control message",
			"type", msg.Type,
			"workerPID", msg.WorkerPID,
		)
	}
}

// Close shuts down the ControlPlane.
func (cp *ControlPlane) Close() error {
	cp.mu.Lock()
	cp.closed = true
	cp.mu.Unlock()

	if cp.listener != nil {
		return cp.listener.Close()
	}
	return nil
}

// SendMessage sends a control message to a specific Worker.
func (cp *ControlPlane) SendMessage(workerPID int, msg *ControlMessage) error {
	cp.mu.RLock()
	conn, ok := cp.conns[workerPID]
	cp.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no connection found for worker PID %d", workerPID)
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("failed to send message to worker %d: %w", workerPID, err)
	}

	return nil
}

// ReceiveFDs receives file descriptors from a Worker connection.
// Uses unix.Recvmsg with unix.UnixRights for FD passing.
func (cp *ControlPlane) ReceiveFDs(workerPID int) ([]*os.File, error) {
	cp.mu.RLock()
	conn, ok := cp.conns[workerPID]
	cp.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no connection found for worker PID %d", workerPID)
	}

	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return nil, errors.New("connection is not a unix socket")
	}

	rawConn, err := unixConn.SyscallConn()
	if err != nil {
		return nil, fmt.Errorf("failed to get syscall conn: %w", err)
	}

	var files []*os.File
	var recvErr error

	err = rawConn.Read(func(fd uintptr) bool {
		buf := make([]byte, 1)
		oob := make([]byte, unix.CmsgSpace(4*10)) // Space for up to 10 FDs

		n, oobn, _, _, err := unix.Recvmsg(int(fd), buf, oob, 0)
		if err != nil {
			if errors.Is(err, syscall.EAGAIN) {
				return false // Try again
			}
			recvErr = fmt.Errorf("recvmsg failed: %w", err)
			return true
		}

		if n == 0 || oobn == 0 {
			recvErr = errors.New("received empty message")
			return true
		}

		scms, err := unix.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			recvErr = fmt.Errorf("failed to parse socket control message: %w", err)
			return true
		}

		for _, scm := range scms {
			fds, err := unix.ParseUnixRights(&scm)
			if err != nil {
				continue
			}
			for _, fd := range fds {
				files = append(files, os.NewFile(uintptr(fd), ""))
			}
		}

		return true
	})

	if err != nil {
		return nil, err
	}
	if recvErr != nil {
		return nil, recvErr
	}

	return files, nil
}

// SocketPath returns the socket path used by this ControlPlane.
func (cp *ControlPlane) SocketPath() string {
	return cp.options.SocketPath
}

// WorkerControlPlane is the Worker-side control plane client.
type WorkerControlPlane struct {
	conn       net.Conn
	socketPath string
	pid        int
}

// NewWorkerControlPlane creates a Worker-side control plane client.
func NewWorkerControlPlane(socketPath string) *WorkerControlPlane {
	return &WorkerControlPlane{
		socketPath: socketPath,
		pid:        os.Getpid(),
	}
}

// Connect connects to the Master's ControlPlane.
func (wcp *WorkerControlPlane) Connect() error {
	d := net.Dialer{}
	conn, err := d.DialContext(context.Background(), "unix", wcp.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to control plane: %w", err)
	}
	wcp.conn = conn
	return nil
}

// Register sends a register message to Master.
func (wcp *WorkerControlPlane) Register() error {
	return wcp.sendMessage(&ControlMessage{
		Type:      MessageTypeRegister,
		WorkerPID: wcp.pid,
	})
}

// NotifyReady sends a ready message to Master.
func (wcp *WorkerControlPlane) NotifyReady() error {
	return wcp.sendMessage(&ControlMessage{
		Type:      MessageTypeReady,
		WorkerPID: wcp.pid,
	})
}

// sendMessage sends a control message to Master.
func (wcp *WorkerControlPlane) sendMessage(msg *ControlMessage) error {
	if wcp.conn == nil {
		return errors.New("not connected to control plane")
	}

	encoder := json.NewEncoder(wcp.conn)
	if err := encoder.Encode(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// SendFDs sends listener file descriptors to Master.
// Uses unix.Sendmsg with unix.UnixRights for FD passing.
func (wcp *WorkerControlPlane) SendFDs(files []*os.File) error {
	if wcp.conn == nil {
		return errors.New("not connected to control plane")
	}

	unixConn, ok := wcp.conn.(*net.UnixConn)
	if !ok {
		return errors.New("connection is not a unix socket")
	}

	rawConn, err := unixConn.SyscallConn()
	if err != nil {
		return fmt.Errorf("failed to get syscall conn: %w", err)
	}

	// First send the fd_transfer message
	err = wcp.sendMessage(&ControlMessage{
		Type:      MessageTypeFDTransfer,
		WorkerPID: wcp.pid,
	})
	if err != nil {
		return err
	}

	// Extract file descriptors
	fds := make([]int, len(files))
	for i, f := range files {
		fds[i] = int(f.Fd())
	}

	// Send FDs via SCM_RIGHTS
	rights := unix.UnixRights(fds...)

	var sendErr error
	err = rawConn.Write(func(fd uintptr) bool {
		err := unix.Sendmsg(int(fd), []byte{0}, rights, nil, 0)
		if err != nil {
			if errors.Is(err, syscall.EAGAIN) {
				return false // Try again
			}
			sendErr = fmt.Errorf("sendmsg failed: %w", err)
		}
		return true
	})

	if err != nil {
		return err
	}
	return sendErr
}

// Close closes the connection to Master.
func (wcp *WorkerControlPlane) Close() error {
	if wcp.conn != nil {
		return wcp.conn.Close()
	}
	return nil
}
