//go:build linux

package gateway

import (
	"golang.org/x/sys/unix"
)

func setTCPReusePort(fd uintptr) error {
	if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
		return err
	}

	return nil
}

func setTCPQuickAck(fd uintptr) error {
	if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_QUICKACK, 1); err != nil {
		return err
	}

	return nil
}

func setTCPFastOpen(fd uintptr) error {
	// Enable fastopen on both client and server side
	if err := unix.SetsockoptInt(int(fd), unix.SOL_TCP, unix.TCP_FASTOPEN, 3); err != nil {
		return err
	}
	return nil
}

func setCloExec(fd int) error {
	unix.CloseOnExec(fd)
	return nil
}
