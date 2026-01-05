//go:build !linux

package zero

import (
	"fmt"
	"os"
)

// getDefaultSocketPath returns the default socket path for the control plane.
// On non-Linux systems (macOS, BSD), uses a file-based socket in /tmp.
func getDefaultSocketPath() string {
	return fmt.Sprintf("/tmp/bifrost-%d.sock", os.Getpid())
}

// supportsAbstractNamespace returns true if the platform supports Abstract Namespace sockets.
func supportsAbstractNamespace() bool {
	return false
}
