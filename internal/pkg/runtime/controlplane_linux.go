//go:build linux

package runtime

import (
	"fmt"
	"os"
)

// getDefaultSocketPath returns the default socket path for the control plane.
// On Linux, uses Abstract Namespace (memory-based, no filesystem cleanup needed).
func getDefaultSocketPath() string {
	return fmt.Sprintf("\x00bifrost-%d.sock", os.Getpid())
}
