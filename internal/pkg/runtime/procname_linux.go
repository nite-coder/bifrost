//go:build linux

package runtime

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// SetProcessName sets the process name visible in /proc/[pid]/comm.
// This name appears in tools like top, htop, and ps -o comm.
// Maximum 15 characters (Linux kernel limitation for PR_SET_NAME).
func SetProcessName(name string) error {
	if len(name) > 15 {
		name = name[:15]
	}
	bytes := append([]byte(name), 0)
	return unix.Prctl(unix.PR_SET_NAME, uintptr(unsafe.Pointer(&bytes[0])), 0, 0, 0)
}

// init sets the initial process name based on BIFROST_ROLE environment variable.
// This runs on the main OS thread before main() is called, so prctl will
// affect the thread group leader (which is what shows up in ps -o comm).
func init() {
	role := os.Getenv(EnvBifrostRole)
	if role == RoleWorker {
		_ = SetProcessName("bifrost-worker")
	} else {
		// Master process (or standalone mode)
		_ = SetProcessName("bifrost-master")
	}
}
