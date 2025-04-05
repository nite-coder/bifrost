//go:build darwin

package gateway

import (
	"os/exec"

	"golang.org/x/sys/unix"
)

func setTCPReusePort(fd uintptr) error {
	if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		return err
	}

	return nil
}

func setTCPQuickAck(fd uintptr) error {
	return nil
}

func setTCPFastOpen(fd uintptr) error {
	return nil
}

func setCloExec(fd int) error {
	return nil
}

func setUserAndGroup(cmd *exec.Cmd, uid, gid uint32) {

}
