//go:build linux

package gateway

import (
	"golang.org/x/sys/unix"
)

func setTCPReusePort(fd uintptr) error {
	err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	if err != nil {
		return err
	}

	return nil
}

func setTCPQuickAck(fd uintptr) error {
	err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_QUICKACK, 1)
	if err != nil {
		return err
	}

	return nil
}

func setTCPFastOpen(fd uintptr) error {
	// Enable fastopen on both client and server side
	const tcpFastopenQueueSize = 3
	err := unix.SetsockoptInt(int(fd), unix.SOL_TCP, unix.TCP_FASTOPEN, tcpFastopenQueueSize)
	if err != nil {
		return err
	}
	return nil
}

func setCloExec(fd int) error {
	unix.CloseOnExec(fd)
	return nil
}
