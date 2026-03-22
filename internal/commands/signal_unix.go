//go:build !windows

package commands

import (
	"os"
	"syscall"
)

func resizeSignals() []os.Signal {
	return []os.Signal{syscall.SIGWINCH}
}

func forwardingSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}

func signalName(sig os.Signal) string {
	switch sig {
	case os.Interrupt:
		return "INT"
	case syscall.SIGTERM:
		return "TERM"
	default:
		return ""
	}
}
