package tui

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sshm/internal/tmux"
)

// SessionReturnHandler manages TUI suspension and resumption around tmux session operations
type SessionReturnHandler struct {
	tuiApp        *TUIApp
	tmuxManager   *tmux.Manager
	sessionName   string
	returnChannel chan struct{}
	signalChannel chan os.Signal
	isAttached    bool
	isCleanedUp   bool
}

// NewSessionReturnHandler creates a new session return handler
func NewSessionReturnHandler(tuiApp *TUIApp, tmuxManager *tmux.Manager) *SessionReturnHandler {
	return &SessionReturnHandler{
		tuiApp:        tuiApp,
		tmuxManager:   tmuxManager,
		returnChannel: make(chan struct{}, 1),
		signalChannel: make(chan os.Signal, 1),
		isAttached:    false,
	}
}

// AttachToSessionWithReturn attaches to a tmux session with TUI return capability
func (sh *SessionReturnHandler) AttachToSessionWithReturn(sessionName string) error {
	// Validate session exists
	if !sh.tmuxManager.SessionExists(sessionName) {
		return fmt.Errorf("session '%s' does not exist", sessionName)
	}

	sh.sessionName = sessionName
	sh.isAttached = true

	// Set up signal handlers for session detachment detection
	sh.setupSignalHandlers()

	// Suspend the TUI and attach to tmux session
	return sh.suspendTUIAndAttach()
}

// setupSignalHandlers sets up signal handling for session detachment detection
func (sh *SessionReturnHandler) setupSignalHandlers() {
	// Listen for signals that might indicate session detachment
	signal.Notify(sh.signalChannel, syscall.SIGWINCH, syscall.SIGCONT)
	
	go func() {
		for {
			select {
			case <-sh.signalChannel:
				// Check if we're still attached to the session
				if sh.isAttached && !sh.isSessionAttached() {
					sh.handleSessionDetachment()
				}
			case <-sh.returnChannel:
				// Explicit return signal
				sh.handleSessionDetachment()
				return
			}
		}
	}()
}

// suspendTUIAndAttach suspends the TUI and attaches to the tmux session
func (sh *SessionReturnHandler) suspendTUIAndAttach() error {
	// Stop the TUI application
	sh.tuiApp.Stop()
	
	// Give TUI time to cleanup
	time.Sleep(100 * time.Millisecond)
	
	// Attach to the tmux session
	err := sh.tmuxManager.AttachSession(sh.sessionName)
	
	// If attachment fails, resume TUI immediately
	if err != nil {
		sh.isAttached = false
		go sh.resumeTUI()
		return fmt.Errorf("failed to attach to session '%s': %w", sh.sessionName, err)
	}
	
	// Attachment completed (user detached), resume TUI
	sh.isAttached = false
	go sh.resumeTUI()
	return nil
}

// handleSessionDetachment handles when the user detaches from the tmux session
func (sh *SessionReturnHandler) handleSessionDetachment() {
	if !sh.isAttached {
		return
	}
	
	sh.isAttached = false
	
	// Resume the TUI
	go sh.resumeTUI()
}

// resumeTUI resumes the TUI after session detachment
func (sh *SessionReturnHandler) resumeTUI() {
	// Give tmux time to fully detach
	time.Sleep(200 * time.Millisecond)
	
	// Refresh session data to reflect current state
	if err := sh.tuiApp.refreshSessions(); err != nil {
		// Log error but don't fail - sessions might not be available
	}
	
	// Show return message to user
	sh.showReturnMessage()
	
	// Reset signal handlers
	signal.Stop(sh.signalChannel)
	close(sh.returnChannel)
	sh.returnChannel = make(chan struct{}, 1)
	sh.signalChannel = make(chan os.Signal, 1)
}

// showReturnMessage displays a welcome back message when returning to TUI
func (sh *SessionReturnHandler) showReturnMessage() {
	if sh.tuiApp.modalManager != nil {
		returnMessage := fmt.Sprintf("🏠 Welcome back to SSHM TUI!\n\n📋 Session '%s' has been detached.\n\nYou can now:\n• Connect to other servers\n• Manage sessions (press 's' to view)\n• Re-attach to existing sessions\n\nPress Enter to continue.", sh.sessionName)
		
		sh.tuiApp.modalManager.ShowInfoModal("Session Detached", returnMessage)
	}
}

// isSessionAttached checks if we're still attached to the current session
func (sh *SessionReturnHandler) isSessionAttached() bool {
	if sh.sessionName == "" {
		return false
	}
	
	// Check if session still exists
	if !sh.tmuxManager.SessionExists(sh.sessionName) {
		return false
	}
	
	// For now, we assume detachment after tmux.AttachSession returns
	// In a more sophisticated implementation, we could check tmux session status
	return false
}

// ForceReturn forces a return to the TUI (for emergency cases)
func (sh *SessionReturnHandler) ForceReturn() {
	sh.isAttached = false
	select {
	case sh.returnChannel <- struct{}{}:
	default:
		// Channel full or closed, handle gracefully
	}
}

// IsAttached returns whether we're currently attached to a session
func (sh *SessionReturnHandler) IsAttached() bool {
	return sh.isAttached
}

// GetCurrentSession returns the name of the currently attached session
func (sh *SessionReturnHandler) GetCurrentSession() string {
	if sh.isAttached {
		return sh.sessionName
	}
	return ""
}

// Cleanup cleans up resources used by the session return handler
func (sh *SessionReturnHandler) Cleanup() {
	if sh.isCleanedUp {
		return
	}
	
	sh.isCleanedUp = true
	sh.isAttached = false
	
	signal.Stop(sh.signalChannel)
	
	// Close channels safely
	close(sh.signalChannel)
	close(sh.returnChannel)
}