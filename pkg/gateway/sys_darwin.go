//go:build darwin

package gateway

func setTCPQuickAck(fd uintptr) error {
	return nil
}

func setTCPFastOpen(fd uintptr) error {
	return nil
}
