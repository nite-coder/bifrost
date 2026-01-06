package runtime

import (
	"log/slog"
	"os"

	"github.com/coreos/go-systemd/v22/daemon"
)

// NotifySystemdReady notifies systemd that the service is ready.
// This should be called after the Worker has confirmed it is ready.
// If not running under systemd (NOTIFY_SOCKET not set), this is a no-op.
func NotifySystemdReady() {
	if os.Getenv("NOTIFY_SOCKET") == "" {
		// Not running under systemd with Type=notify
		return
	}

	sent, err := daemon.SdNotify(false, daemon.SdNotifyReady)
	if err != nil {
		slog.Warn("failed to notify systemd", "error", err)
		return
	}
	if sent {
		slog.Info("notified systemd: service is ready")
	}
}
