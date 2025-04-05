//go:build windows

package gateway

import (
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
