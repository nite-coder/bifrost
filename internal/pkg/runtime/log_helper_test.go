package runtime

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if os.Getenv("GO_WANT_LOG_HELPER") != "1" {
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGTERM)

	fmt.Fprintln(os.Stdout, "READY")

	for {
		sig := <-sigCh
		switch sig {
		case syscall.SIGUSR1:
			// After receiving USR1, test writing content to stdout and stderr
			fmt.Fprintln(os.Stdout, "STDOUT_ROTATED")
			fmt.Fprintln(os.Stderr, "STDERR_ROTATED")
		case syscall.SIGTERM:
			os.Exit(0)
		}
	}
}
