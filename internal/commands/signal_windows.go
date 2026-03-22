//go:build windows

package commands

import "os"

func resizeSignals() []os.Signal {
	return nil
}

func forwardingSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}

func signalName(sig os.Signal) string {
	if sig == os.Interrupt {
		return "INT"
	}
	return ""
}
