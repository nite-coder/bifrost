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
	// Enable fastopen on both client and server side
	if err := unix.SetsockoptInt(int(fd), unix.SOL_TCP, unix.TCP_FASTOPEN, 3); err != nil {
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

func setCloExec(fd int) error {
	unix.CloseOnExec(fd)
	return nil
}

func DisableGopool() error {
	_ = netpoll.DisableGopool()
	runTask = func(ctx context.Context, f func()) {
		go f()
	}
	return nil
}
