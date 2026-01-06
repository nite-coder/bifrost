package runtime

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
	// MessageTypeFDTransferStart is sent by Worker to Master on a DEDICATED connection to start FD transfer.
	MessageTypeFDTransferStart MessageType = "fd_transfer_start"
	// MessageTypeAck is sent by Master to Worker to acknowledge readiness for raw data.
	MessageTypeAck MessageType = "ack"
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
	fdHandler  func(fds []*os.File, keys []string)      // Handler for transferred FDs with metadata
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

// ...

// SetFDHandler sets the callback for handling incoming FDs.
func (cp *ControlPlane) SetFDHandler(handler func(fds []*os.File, keys []string)) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.fdHandler = handler
}

// ...

// handleConnection handles a single Worker connection.
func (cp *ControlPlane) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

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

		slog.Debug("received message", "type", msg.Type, "workerPID", msg.WorkerPID)

		// Handle Special FD Transfer Protocol
		if msg.Type == MessageTypeFDTransferStart {
			slog.Debug("starting FD transfer protocol", "workerPID", msg.WorkerPID)

			// Extract Keys from Payload
			var keys []string
			if len(msg.Payload) > 0 {
				if err := json.Unmarshal(msg.Payload, &keys); err != nil {
					slog.Error("failed to unmarshal listener keys", "error", err)
					// Continue, but keys will be empty
				}
			}

			// 1. Send Ack to flush any buffers and signal readiness
			if err := encoder.Encode(&ControlMessage{Type: MessageTypeAck}); err != nil {
				slog.Error("failed to send Ack for FD transfer", "error", err)
				return
			}

			// 2. Switch to Raw Mode (Stop decoding JSON)
			// Read FDs from the connection immediately
			fds, err := cp.ReceiveFDsFromConn(conn)
			if err != nil {
				slog.Error("failed to receive FDs", "error", err)
				return
			}

			slog.Info("successfully received transferred FDs", "count", len(fds), "keys", len(keys))

			// 3. Dispatch FDs to handler
			cp.mu.RLock()
			fdHandler := cp.fdHandler
			cp.mu.RUnlock()

			if fdHandler != nil {
				fdHandler(fds, keys)
			}

			// Close connection after transfer is complete
			return
		}

		// Register connection if PID is present
		if msg.WorkerPID > 0 {
			cp.mu.Lock()
			cp.conns[msg.WorkerPID] = conn
			cp.mu.Unlock()
		}

		// If not FDTransferStart, then it's a regular control message
		cp.mu.RLock()
		handler := cp.msgHandler
		cp.mu.RUnlock()

		if handler != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("control message handler panicked", "recover", r)
					}
				}()
				handler(conn, &msg)
			}()
		}
	}
}

// ReceiveFDsFromConn extracts FDs from the given connection using SCM_RIGHTS.
func (cp *ControlPlane) ReceiveFDsFromConn(conn net.Conn) ([]*os.File, error) {
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

	// We expect the peer to send OOB data now.
	// Since we are synced via Ack, the next bytes on the wire should be the OOB msg.
	err = rawConn.Read(func(fd uintptr) bool {
		buf := make([]byte, 1)                    // Dummy byte
		oob := make([]byte, unix.CmsgSpace(4*10)) // Space for up to 10 FDs

		// Use MSG_CMSG_CLOEXEC for safety? Go usually handles it.
		n, oobn, _, _, err := unix.Recvmsg(int(fd), buf, oob, 0)
		if err != nil {
			if errors.Is(err, syscall.EAGAIN) {
				return false // Try again
			}
			recvErr = fmt.Errorf("recvmsg failed: %w", err)
			return true
		}

		if n == 0 && oobn == 0 {
			recvErr = errors.New("received empty message during FD transfer")
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
	signalFunc func(pid int, sig os.Signal) error // Mockable signal function
}

// NewWorkerControlPlane creates a Worker-side control plane client.
func NewWorkerControlPlane(socketPath string) *WorkerControlPlane {
	return &WorkerControlPlane{
		socketPath: socketPath,
		pid:        os.Getpid(),
		signalFunc: func(pid int, sig os.Signal) error {
			p, err := os.FindProcess(pid)
			if err != nil {
				return err
			}
			return p.Signal(sig)
		},
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

// SendFDs opens a dedicated connection to Master and sends listener FDs.
func (wcp *WorkerControlPlane) SendFDs(files []*os.File, keys []string) error {
	// 1. Connect to Master on a NEW connection
	d := net.Dialer{}
	conn, err := d.DialContext(context.Background(), "unix", wcp.socketPath)
	if err != nil {
		return fmt.Errorf("failed to dial dedicated FD transfer connection: %w", err)
	}
	defer conn.Close()

	// Prepare Payload (Keys)
	payload, err := json.Marshal(keys)
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %w", err)
	}

	// 2. Send Start Message with Payload
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(&ControlMessage{
		Type:      MessageTypeFDTransferStart,
		WorkerPID: wcp.pid,
		Payload:   payload,
	})
	if err != nil {
		return fmt.Errorf("failed to send FDTransferStart: %w", err)
	}

	// 3. Wait for Ack (Flush buffers)
	decoder := json.NewDecoder(conn)
	var ack ControlMessage
	if err := decoder.Decode(&ack); err != nil {
		return fmt.Errorf("failed to receive Ack: %w", err)
	}
	if ack.Type != MessageTypeAck {
		return fmt.Errorf("unexpected response type: %s", ack.Type)
	}

	// 4. Send FDs via SCM_RIGHTS (OOB) on the clean stream
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return errors.New("connection is not a unix socket")
	}

	rawConn, err := unixConn.SyscallConn()
	if err != nil {
		return fmt.Errorf("failed to get syscall conn: %w", err)
	}

	// Prepare FDs
	fds := make([]int, len(files))
	for i, f := range files {
		fds[i] = int(f.Fd())
	}
	rights := unix.UnixRights(fds...)

	var sendErr error
	err = rawConn.Write(func(fd uintptr) bool {
		// Send 1 byte of dummy data + Rights
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
		return fmt.Errorf("raw write failed: %w", err)
	}
	if sendErr != nil {
		return sendErr
	}

	return nil
}

// FDHandler is an interface for handling file descriptor requests.
type FDHandler interface {
	HandleFDRequest() error
}

// Start starts the worker control plane loop to handle messages from Master.
// It blocks until the context is cancelled or the connection is closed.
func (wcp *WorkerControlPlane) Start(ctx context.Context, fdHandler FDHandler) error {
	if wcp.conn == nil {
		return errors.New("not connected to control plane")
	}

	decoder := json.NewDecoder(wcp.conn)

	// Send ID check? No, Connect logic is simple.

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var msg ControlMessage
		if err := decoder.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}
			slog.Error("worker failed to decode message", "error", err)
			return err
		}

		slog.Debug("worker received message", "type", msg.Type)

		switch msg.Type {
		case MessageTypeFDRequest:
			if fdHandler != nil {
				if err := fdHandler.HandleFDRequest(); err != nil {
					slog.Error("worker failed to handle FD request", "error", err)
				}
			}
		case MessageTypeShutdown:
			slog.Info("worker received shutdown request")
			// Send SIGTERM to self to trigger graceful shutdown
			if err := wcp.signalFunc(wcp.pid, syscall.SIGTERM); err != nil {
				slog.Error("failed to signal self", "error", err)
			}
			return nil
		default:
			slog.Debug("worker ignored message", "type", msg.Type)
		}
	}
}

// Close closes the connection to Master.
func (wcp *WorkerControlPlane) Close() error {
	if wcp.conn != nil {
		return wcp.conn.Close()
	}
	return nil
}
