//go:build linux

package gateway

import "golang.org/x/sys/unix"

func setTCPQuickAck(fd uintptr) error {
	if err := unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_QUICKACK, 1); err != nil {
		return err
	}

	return nil
}

func setTCPFastOpen(fd uintptr) error {
	// Enable fastopen on the server side
	if err := unix.SetsockoptInt(int(fd), unix.SOL_TCP, unix.TCP_FASTOPEN, 1); err != nil {
		return err
	}

	// Enable fastopen on the client side
	if err := unix.SetsockoptInt(int(fd), unix.SOL_TCP, unix.TCP_FASTOPEN_CONNECT, 1); err != nil {
		return err
	}
	return nil
}
