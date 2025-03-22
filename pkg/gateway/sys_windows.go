//go:build windows

package gateway

import (
	"context"
	"os/exec"
)

func setTCPReusePort(fd uintptr) error {
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

func DisableGopool() error {
	runTask = func(ctx context.Context, f func()) {
		go f()
	}
	return nil
}
