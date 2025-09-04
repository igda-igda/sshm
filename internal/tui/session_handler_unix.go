//go:build !windows

package tui

import (
	"os/signal"
	"syscall"
)

// setupPlatformSignals sets up Unix-specific signal handling
func (sh *SessionReturnHandler) setupPlatformSignals() {
	signal.Notify(sh.signalChannel, syscall.SIGWINCH, syscall.SIGCONT)
}