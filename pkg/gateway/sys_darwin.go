//go:build darwin

package gateway

import (
	"context"
	"os/exec"

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
	return nil
}

func setTCPFastOpen(fd uintptr) error {
	return nil
}

func setCloExec(fd uintptr) error {
	return nil
}

func setUserAndGroup(cmd *exec.Cmd, uid, gid uint32) {

}



func DisableGopool() error {
	_ = netpoll.DisableGopool()
	runTask = func(ctx context.Context, f func()) {
		go f()
	}
	return nil
}
