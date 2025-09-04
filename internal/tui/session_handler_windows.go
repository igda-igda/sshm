//go:build windows

package tui

// setupPlatformSignals sets up Windows-specific signal handling
func (sh *SessionReturnHandler) setupPlatformSignals() {
	// Windows doesn't support SIGWINCH/SIGCONT, so we skip signal-based detection
	// Session detachment detection on Windows will rely on other mechanisms
}