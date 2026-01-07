//go:build !linux

package runtime

// SetProcessName is a no-op on non-Linux platforms.
// On Linux, this sets the process name visible in /proc/[pid]/comm.
func SetProcessName(name string) error {
	return nil
}
