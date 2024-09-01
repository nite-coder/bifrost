//go:build linux

package gateway

import (
	"context"
	"os/exec"
	"syscall"

	"github.com/cloudwego/netpoll"
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

func setUserAndGroup(cmd *exec.Cmd, uid, gid uint32) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uid,
			Gid: gid,
		},
	}
}

func DisableGopool() error {
	_ = netpoll.DisableGopool()
	runTask = func(ctx context.Context, f func()) {
		go f()
	}
	return nil
}
