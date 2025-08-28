package tui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"sshm/internal/tmux"
)

// SessionEventType represents the type of session event
type SessionEventType int

const (
	SessionDetached SessionEventType = iota
	SessionReattached
	SessionKilled
)

// SessionEvent represents a tmux session event
type SessionEvent struct {
	Type        SessionEventType
	SessionName string
	Timestamp   time.Time
}

// SessionDetachEvent represents a tmux session detachment event
type SessionDetachEvent struct {
	SessionName string
	DetachTime  time.Time
}

// TUIState represents the state of the TUI that can be preserved
type TUIState struct {
	SelectedProfile      string
	SelectedProfileIndex int
	CurrentView          string
	ScrollPosition       int
	SelectedRow          int
	SelectedSession      int
	FocusedPanel         string
	CurrentFilter        string
}

// SessionMonitor monitors tmux sessions for detachment events
type SessionMonitor struct {
	sessionName   string
	isActive      bool
	stopChannel   chan struct{}
	eventChannel  chan SessionEvent
	wg            sync.WaitGroup
	mu            sync.RWMutex
}

// SessionReturnHandler manages TUI suspension and resumption around tmux session operations
type SessionReturnHandler struct {
	tuiApp        *TUIApp
	tmuxManager   *tmux.Manager
	sessionName   string
	returnChannel chan struct{}
	signalChannel chan os.Signal
	isAttached    bool
	isCleanedUp   bool
	sessionMonitor *SessionMonitor
	eventChannel  chan SessionEvent
	tuiState      *TUIState
	mu            sync.RWMutex
}

// NewSessionReturnHandler creates a new session return handler
func NewSessionReturnHandler(tuiApp *TUIApp, tmuxManager *tmux.Manager) *SessionReturnHandler {
	return &SessionReturnHandler{
		tuiApp:        tuiApp,
		tmuxManager:   tmuxManager,
		returnChannel: make(chan struct{}, 1),
		signalChannel: make(chan os.Signal, 1),
		isAttached:    false,
		eventChannel:  make(chan SessionEvent, 10),
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

	// Save current TUI state before attaching
	sh.SaveTUIState()

	// Start session monitoring before attachment
	if err := sh.StartSessionMonitoring(sessionName); err != nil {
		return fmt.Errorf("failed to start session monitoring: %w", err)
	}

	// Set up signal handlers for session detachment detection
	sh.setupSignalHandlers()

	// Start event processing in background
	go sh.processEventLoop()

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
	
	// Attach to the tmux session (this will block until user detaches)
	err := sh.tmuxManager.AttachSession(sh.sessionName)
	
	// If attachment fails, restart TUI immediately
	if err != nil {
		sh.isAttached = false
		sh.StopSessionMonitoring()
		return sh.restartTUIAfterError(fmt.Errorf("failed to attach to session '%s': %w", sh.sessionName, err))
	}
	
	// Attachment completed successfully - this means user detached from tmux (Ctrl+B D)
	// Now we need to restart the TUI application to return user to SSHM interface
	sh.isAttached = false
	
	// Stop monitoring as we're returning to TUI
	sh.StopSessionMonitoring()
	
	// Restart TUI with state restoration - this is the key fix!
	return sh.restartTUIWithStateRestoration()
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

// resumeTUIWithStateRestoration resumes the TUI with state restoration
func (sh *SessionReturnHandler) resumeTUIWithStateRestoration() {
	// Give tmux time to fully detach
	time.Sleep(200 * time.Millisecond)
	
	// Restore saved TUI state with error handling
	func() {
		defer func() {
			if r := recover(); r != nil {
				// State restoration failed, continue without state
			}
		}()
		if sh.tuiState != nil {
			sh.RestoreTUIState(sh.tuiState)
		}
	}()
	
	// Refresh session data to reflect current state with error handling
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Session refresh failed, continue without refresh
			}
		}()
		if err := sh.tuiApp.refreshSessions(); err != nil {
			// Log error but don't fail - sessions might not be available
		}
	}()
	
	// Show enhanced return message with state information
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Message display failed, continue silently
			}
		}()
		sh.showEnhancedReturnMessage()
	}()
	
	// Reset signal handlers safely
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Signal handler reset failed, continue
			}
		}()
		signal.Stop(sh.signalChannel)
		
		// Close channels safely
		if sh.returnChannel != nil {
			select {
			case <-sh.returnChannel:
			default:
			}
			close(sh.returnChannel)
			sh.returnChannel = make(chan struct{}, 1)
		}
		
		if sh.signalChannel != nil {
			sh.signalChannel = make(chan os.Signal, 1)
		}
	}()
}

// showReturnMessage displays a welcome back message when returning to TUI
func (sh *SessionReturnHandler) showReturnMessage() {
	if sh.tuiApp.modalManager != nil {
		returnMessage := fmt.Sprintf("🏠 Welcome back to SSHM TUI!\n\n📋 Session '%s' has been detached.\n\nYou can now:\n• Connect to other servers\n• Manage sessions (press 's' to view)\n• Re-attach to existing sessions\n\nPress Enter to continue.", sh.sessionName)
		
		sh.tuiApp.modalManager.ShowInfoModal("Session Detached", returnMessage)
	}
}

// showEnhancedReturnMessage displays an enhanced welcome back message with state information
func (sh *SessionReturnHandler) showEnhancedReturnMessage() {
	if sh.tuiApp.modalManager != nil {
		var stateInfo string
		if sh.tuiState != nil {
			if sh.tuiState.SelectedProfile != "" {
				stateInfo = fmt.Sprintf("\n🔄 Profile '%s' restored", sh.tuiState.SelectedProfile)
			} else {
				stateInfo = "\n🔄 TUI state restored to 'All' servers view"
			}
		}
		
		returnMessage := fmt.Sprintf("🏠 Welcome back to SSHM TUI!\n\n📋 Session '%s' has been detached.%s\n\nYou can now:\n• Connect to other servers\n• Manage sessions (press 's' to view)\n• Re-attach to existing sessions\n\nPress Enter to continue.", sh.sessionName, stateInfo)
		
		sh.tuiApp.modalManager.ShowInfoModal("Session Detached - State Restored", returnMessage)
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

// StartSessionMonitoring starts monitoring a tmux session for detachment events
func (sh *SessionReturnHandler) StartSessionMonitoring(sessionName string) error {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Validate session exists
	if !sh.tmuxManager.SessionExists(sessionName) {
		return fmt.Errorf("session '%s' does not exist", sessionName)
	}

	// Stop existing monitoring if any
	if sh.sessionMonitor != nil {
		sh.stopMonitoringLocked()
	}

	// Create new session monitor
	sh.sessionMonitor = &SessionMonitor{
		sessionName:  sessionName,
		isActive:     true,
		stopChannel:  make(chan struct{}),
		eventChannel: sh.eventChannel,
	}

	// Start monitoring goroutine
	sh.sessionMonitor.wg.Add(1)
	go sh.monitorSessionLoop()

	return nil
}

// StopSessionMonitoring stops monitoring the current session
func (sh *SessionReturnHandler) StopSessionMonitoring() {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.stopMonitoringLocked()
}

// stopMonitoringLocked stops monitoring (internal method, assumes lock held)
func (sh *SessionReturnHandler) stopMonitoringLocked() {
	if sh.sessionMonitor == nil {
		return
	}

	sh.sessionMonitor.mu.Lock()
	if sh.sessionMonitor.isActive {
		sh.sessionMonitor.isActive = false
		close(sh.sessionMonitor.stopChannel)
	}
	sh.sessionMonitor.mu.Unlock()

	// Wait for monitoring goroutine to finish
	sh.sessionMonitor.wg.Wait()
	sh.sessionMonitor = nil
}

// IsMonitoringSession returns whether session monitoring is active
func (sh *SessionReturnHandler) IsMonitoringSession() bool {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	
	if sh.sessionMonitor == nil {
		return false
	}
	
	sh.sessionMonitor.mu.RLock()
	defer sh.sessionMonitor.mu.RUnlock()
	return sh.sessionMonitor.isActive
}

// monitorSessionLoop runs the session monitoring loop
func (sh *SessionReturnHandler) monitorSessionLoop() {
	defer sh.sessionMonitor.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sh.sessionMonitor.stopChannel:
			return
		case <-ticker.C:
			sh.checkSessionStatus()
		}
	}
}

// checkSessionStatus checks the current session status and generates events
func (sh *SessionReturnHandler) checkSessionStatus() {
	sh.sessionMonitor.mu.RLock()
	sessionName := sh.sessionMonitor.sessionName
	isActive := sh.sessionMonitor.isActive
	sh.sessionMonitor.mu.RUnlock()

	// If monitoring is no longer active, stop checking
	if !isActive {
		return
	}

	// Check if session still exists (with error handling)
	sessionExists := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Tmux command panicked, assume session doesn't exist
				sessionExists = false
			}
		}()
		sessionExists = sh.tmuxManager.SessionExists(sessionName)
	}()

	if !sessionExists {
		// Session was killed or tmux is not available
		event := SessionEvent{
			Type:        SessionKilled,
			SessionName: sessionName,
			Timestamp:   time.Now(),
		}
		sh.sendEvent(event)
		
		// Stop monitoring since session no longer exists
		go sh.StopSessionMonitoring()
		return
	}

	// Check if session is attached (with error handling)
	isAttached, err := sh.tmuxManager.IsSessionAttached(sessionName)
	if err != nil {
		// Error checking attachment status - this could be a temporary tmux issue
		// Don't generate events but continue monitoring
		return
	}

	// Generate appropriate event based on attachment status
	sh.mu.RLock()
	currentlyAttached := sh.isAttached
	sh.mu.RUnlock()

	if !isAttached && currentlyAttached {
		// Session was detached
		event := SessionEvent{
			Type:        SessionDetached,
			SessionName: sessionName,
			Timestamp:   time.Now(),
		}
		sh.sendEvent(event)
	} else if isAttached && !currentlyAttached {
		// Session was reattached
		event := SessionEvent{
			Type:        SessionReattached,
			SessionName: sessionName,
			Timestamp:   time.Now(),
		}
		sh.sendEvent(event)
	}
}

// sendEvent sends a session event to the event channel
func (sh *SessionReturnHandler) sendEvent(event SessionEvent) {
	select {
	case sh.eventChannel <- event:
		// Event sent successfully
	default:
		// Event channel full, event dropped
	}
}

// ProcessDetachmentEvent processes a detachment event
func (sh *SessionReturnHandler) ProcessDetachmentEvent(event SessionDetachEvent) {
	if event.SessionName == sh.sessionName && sh.isAttached {
		sh.handleSessionDetachment()
	}
}

// SaveTUIState saves the current TUI state
func (sh *SessionReturnHandler) SaveTUIState() *TUIState {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Get comprehensive current state from TUI app
	state := &TUIState{
		SelectedProfile:      sh.tuiApp.GetSelectedProfile(),
		SelectedProfileIndex: sh.tuiApp.GetSelectedProfileIndex(),
		CurrentView:          "main", // Could be enhanced to track actual view
		ScrollPosition:       0,      // Could be enhanced to track scroll
		SelectedRow:          sh.tuiApp.GetSelectedRow(),
		SelectedSession:      sh.tuiApp.GetSelectedSession(),
		FocusedPanel:         sh.tuiApp.GetFocusedPanel(),
		CurrentFilter:        sh.tuiApp.GetCurrentFilter(),
	}

	sh.tuiState = state
	return state
}

// RestoreTUIState restores a previously saved TUI state
func (sh *SessionReturnHandler) RestoreTUIState(state *TUIState) {
	if state == nil {
		return
	}

	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Restore comprehensive state to TUI app
	sh.tuiApp.SetSelectedProfile(state.SelectedProfile)
	sh.tuiApp.SetSelectedRow(state.SelectedRow)
	sh.tuiApp.SetSelectedSession(state.SelectedSession)
	sh.tuiApp.SetFocusedPanel(state.FocusedPanel)
	sh.tuiApp.SetCurrentFilter(state.CurrentFilter)
	
	// Store state reference
	sh.tuiState = state
}

// HandleSessionKilled handles when a session is killed externally
func (sh *SessionReturnHandler) HandleSessionKilled(sessionName string) error {
	if !sh.IsMonitoringSession() {
		return fmt.Errorf("session monitoring not active")
	}

	if sh.sessionName == sessionName {
		sh.isAttached = false
		sh.handleSessionDetachment()
	}

	return nil
}

// RestoreTUIFromSession restores the TUI from a session
func (sh *SessionReturnHandler) RestoreTUIFromSession(sessionName string) {
	// Restore TUI state if available
	if sh.tuiState != nil {
		sh.RestoreTUIState(sh.tuiState)
	}

	// Show return message
	sh.showReturnMessage()
}

// SetEventChannel sets the event channel for session events
func (sh *SessionReturnHandler) SetEventChannel(eventChannel chan SessionEvent) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.eventChannel = eventChannel
}

// ProcessEventQueue processes all events in the event queue
func (sh *SessionReturnHandler) ProcessEventQueue() {
	for {
		select {
		case event := <-sh.eventChannel:
			sh.processEvent(event)
		default:
			// No more events to process
			return
		}
	}
}

// processEvent processes a single session event
func (sh *SessionReturnHandler) processEvent(event SessionEvent) {
	switch event.Type {
	case SessionDetached:
		if event.SessionName == sh.sessionName {
			sh.handleSessionDetachment()
		}
	case SessionReattached:
		// Handle reattachment if needed
	case SessionKilled:
		if event.SessionName == sh.sessionName {
			sh.isAttached = false
			sh.handleSessionDetachment()
		}
	}
}

// processEventLoop continuously processes session events
func (sh *SessionReturnHandler) processEventLoop() {
	for {
		select {
		case event, ok := <-sh.eventChannel:
			if !ok {
				// Channel closed, stop processing
				return
			}
			sh.processEvent(event)
		case <-time.After(5 * time.Second):
			// Check if we should continue processing
			if sh.isCleanedUp || !sh.IsMonitoringSession() {
				return
			}
		}
	}
}

// restartTUIWithStateRestoration restarts the TUI application with state restoration after successful tmux detachment
func (sh *SessionReturnHandler) restartTUIWithStateRestoration() error {
	// Small delay to ensure tmux has fully released the terminal
	time.Sleep(300 * time.Millisecond)
	
	// Create a new TUI application instance
	newTUIApp, err := NewTUIApp()
	if err != nil {
		return fmt.Errorf("failed to create new TUI application: %w", err)
	}
	
	// Restore the saved state to the new TUI instance
	if sh.tuiState != nil {
		newTUIApp.SetSelectedProfile(sh.tuiState.SelectedProfile)
		newTUIApp.SetSelectedRow(sh.tuiState.SelectedRow)
		newTUIApp.SetSelectedSession(sh.tuiState.SelectedSession)
		newTUIApp.SetFocusedPanel(sh.tuiState.FocusedPanel)
		newTUIApp.SetCurrentFilter(sh.tuiState.CurrentFilter)
	}
	
	// Show the return message with state information
	go func() {
		time.Sleep(500 * time.Millisecond) // Give TUI time to initialize
		sh.showTUIReturnMessage(newTUIApp)
	}()
	
	// Start the new TUI application (this will take over the terminal)
	return newTUIApp.Run(context.Background())
}

// restartTUIAfterError restarts the TUI application after an attachment error
func (sh *SessionReturnHandler) restartTUIAfterError(attachError error) error {
	// Small delay to ensure any error output is visible
	time.Sleep(500 * time.Millisecond)
	
	// Create a new TUI application instance
	newTUIApp, err := NewTUIApp()
	if err != nil {
		return fmt.Errorf("failed to create new TUI application after attachment error: %w", err)
	}
	
	// Restore the saved state to the new TUI instance
	if sh.tuiState != nil {
		newTUIApp.SetSelectedProfile(sh.tuiState.SelectedProfile)
		newTUIApp.SetSelectedRow(sh.tuiState.SelectedRow)
		newTUIApp.SetSelectedSession(sh.tuiState.SelectedSession)
		newTUIApp.SetFocusedPanel(sh.tuiState.FocusedPanel)
		newTUIApp.SetCurrentFilter(sh.tuiState.CurrentFilter)
	}
	
	// Show the error message
	go func() {
		time.Sleep(500 * time.Millisecond) // Give TUI time to initialize
		sh.showTUIErrorMessage(newTUIApp, attachError)
	}()
	
	// Start the new TUI application (this will take over the terminal)
	return newTUIApp.Run(context.Background())
}

// showTUIReturnMessage displays the return message in the new TUI instance
func (sh *SessionReturnHandler) showTUIReturnMessage(tuiApp *TUIApp) {
	if tuiApp.modalManager != nil {
		var stateInfo string
		if sh.tuiState != nil {
			if sh.tuiState.SelectedProfile != "" {
				stateInfo = fmt.Sprintf("\n🔄 Profile '%s' restored", sh.tuiState.SelectedProfile)
			} else {
				stateInfo = "\n🔄 TUI state restored to 'All' servers view"
			}
		}
		
		returnMessage := fmt.Sprintf("🏠 Welcome back to SSHM TUI!\n\n📋 Session '%s' has been detached.%s\n\nYou can now:\n• Connect to other servers\n• Manage sessions (press 's' to view)\n• Re-attach to existing sessions\n\nPress Enter to continue.", sh.sessionName, stateInfo)
		
		tuiApp.modalManager.ShowInfoModal("Session Detached - Returned to TUI", returnMessage)
	}
}

// showTUIErrorMessage displays an error message in the new TUI instance
func (sh *SessionReturnHandler) showTUIErrorMessage(tuiApp *TUIApp, attachError error) {
	errorMessage := fmt.Sprintf("❌ Failed to attach to tmux session '%s'\n\nError: %s\n\nYou can now:\n• Try connecting again\n• Check if the session exists\n• Create a new session", sh.sessionName, attachError.Error())
	
	tuiApp.showErrorModal(errorMessage)
}

// UpdateSessionActivity updates session activity timestamp
func (sh *SessionReturnHandler) UpdateSessionActivity(sessionName string) {
	// This is a placeholder for session activity tracking
	// In a full implementation, this would update activity timestamps
}

// Cleanup cleans up resources used by the session return handler
func (sh *SessionReturnHandler) Cleanup() {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	
	if sh.isCleanedUp {
		return
	}
	
	sh.isCleanedUp = true
	sh.isAttached = false
	
	// Stop session monitoring safely (call internal method directly to avoid deadlock)
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Monitoring cleanup failed, continue
			}
		}()
		sh.stopMonitoringLocked()
	}()
	
	// Stop signal handlers safely
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Signal cleanup failed, continue
			}
		}()
		if sh.signalChannel != nil {
			signal.Stop(sh.signalChannel)
		}
	}()
	
	// Close channels safely
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Channel cleanup failed, continue
			}
		}()
		
		if sh.signalChannel != nil {
			select {
			case <-sh.signalChannel:
			default:
			}
			close(sh.signalChannel)
			sh.signalChannel = nil
		}
		
		if sh.returnChannel != nil {
			select {
			case <-sh.returnChannel:
			default:
			}
			close(sh.returnChannel)
			sh.returnChannel = nil
		}
		
		if sh.eventChannel != nil {
			// Drain the channel before closing
			for len(sh.eventChannel) > 0 {
				select {
				case <-sh.eventChannel:
				default:
					break
				}
			}
			close(sh.eventChannel)
			sh.eventChannel = nil
		}
	}()
}