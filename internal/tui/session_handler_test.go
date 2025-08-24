package tui

import (
	"os"
	"testing"
	"time"

	"sshm/internal/tmux"
)

func TestSessionReturnHandler_Creation(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app and tmux manager
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tmuxMgr := tmux.NewManager()

	// Create session handler
	handler := NewSessionReturnHandler(tuiApp, tmuxMgr)
	
	// Verify initialization
	if handler == nil {
		t.Fatal("Expected session handler to be created")
	}
	
	if handler.tuiApp != tuiApp {
		t.Error("Expected TUI app to be set correctly")
	}
	
	if handler.tmuxManager != tmuxMgr {
		t.Error("Expected tmux manager to be set correctly")
	}
	
	if handler.isAttached {
		t.Error("Expected handler to start in detached state")
	}
	
	if handler.GetCurrentSession() != "" {
		t.Error("Expected current session to be empty initially")
	}
}

func TestSessionReturnHandler_StateManagement(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tmuxMgr := tmux.NewManager()
	handler := NewSessionReturnHandler(tuiApp, tmuxMgr)

	// Test initial state
	if handler.IsAttached() {
		t.Error("Expected handler to start in detached state")
	}

	if handler.GetCurrentSession() != "" {
		t.Error("Expected no current session initially")
	}

	// Test force return
	handler.ForceReturn()
	
	// Handler should still be in detached state after force return
	if handler.IsAttached() {
		t.Error("Expected handler to remain detached after force return")
	}
}

func TestSessionReturnHandler_AttachmentValidation(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tmuxMgr := tmux.NewManager()
	handler := NewSessionReturnHandler(tuiApp, tmuxMgr)

	// Test attachment to non-existent session
	err = handler.AttachToSessionWithReturn("non-existent-session")
	if err == nil {
		t.Error("Expected error when attaching to non-existent session")
	}

	// Verify error message contains session name
	if err != nil && err.Error() != "session 'non-existent-session' does not exist" {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	// Handler should remain detached after failed attachment
	if handler.IsAttached() {
		t.Error("Expected handler to remain detached after failed attachment")
	}
}

func TestSessionReturnHandler_SessionDetection(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tmuxMgr := tmux.NewManager()
	handler := NewSessionReturnHandler(tuiApp, tmuxMgr)

	// Test session attachment detection without actual session
	if handler.isSessionAttached() {
		t.Error("Expected isSessionAttached to return false with no session")
	}

	// Set a mock session name and test
	handler.sessionName = "test-session"
	if handler.isSessionAttached() {
		t.Error("Expected isSessionAttached to return false for non-existent session")
	}

	// Clear session name
	handler.sessionName = ""
	if handler.isSessionAttached() {
		t.Error("Expected isSessionAttached to return false with empty session name")
	}
}

func TestSessionReturnHandler_Cleanup(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tmuxMgr := tmux.NewManager()
	handler := NewSessionReturnHandler(tuiApp, tmuxMgr)

	// Test cleanup doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Cleanup panicked: %v", r)
		}
	}()

	handler.Cleanup()

	// Test multiple cleanups don't panic
	handler.Cleanup()
}

func TestSessionReturnHandler_SignalHandling(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tmuxMgr := tmux.NewManager()
	handler := NewSessionReturnHandler(tuiApp, tmuxMgr)

	// Test signal handlers setup and cleanup
	handler.setupSignalHandlers()
	
	// Give signal handlers time to start
	time.Sleep(10 * time.Millisecond)
	
	// Test force return works
	handler.ForceReturn()
	
	// Give signal handlers time to process
	time.Sleep(10 * time.Millisecond)
	
	// Cleanup should work without panicking
	handler.Cleanup()
}

func TestSessionReturnHandler_ErrorHandling(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tmuxMgr := tmux.NewManager()
	handler := NewSessionReturnHandler(tuiApp, tmuxMgr)

	// Test various error conditions
	
	// Empty session name
	err = handler.AttachToSessionWithReturn("")
	if err == nil {
		t.Error("Expected error for empty session name")
	}

	// Invalid session name characters (though tmux might accept them)
	err = handler.AttachToSessionWithReturn("invalid session name with spaces")
	if err == nil {
		t.Error("Expected error for session with spaces (depending on tmux configuration)")
	}

	// Handler should remain detached after all failed attempts
	if handler.IsAttached() {
		t.Error("Expected handler to remain detached after failed attachment attempts")
	}
}

func TestSessionReturnHandler_Integration(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app (this will initialize session handler)
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Verify session handler was initialized
	if tuiApp.sessionHandler == nil {
		t.Fatal("Expected session handler to be initialized in TUI app")
	}

	// Test that session handler is properly connected
	if tuiApp.sessionHandler.tuiApp != tuiApp {
		t.Error("Expected session handler to reference TUI app correctly")
	}

	if tuiApp.sessionHandler.tmuxManager != tuiApp.tmuxManager {
		t.Error("Expected session handler to reference tmux manager correctly")
	}

	// Test cleanup integration
	tuiApp.Stop() // This should call session handler cleanup
	
	// No crash should occur
}

func TestSessionReturnHandler_ShowReturnMessage(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tmuxMgr := tmux.NewManager()
	handler := NewSessionReturnHandler(tuiApp, tmuxMgr)

	// Test show return message doesn't crash
	handler.sessionName = "test-session"
	
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("showReturnMessage panicked: %v", r)
		}
	}()
	
	handler.showReturnMessage()
}

func TestSessionReturnHandler_HandleDetachment(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tmuxMgr := tmux.NewManager()
	handler := NewSessionReturnHandler(tuiApp, tmuxMgr)

	// Set up attached state for testing
	handler.isAttached = true
	handler.sessionName = "test-session"

	// Test detachment handling
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("handleSessionDetachment panicked: %v", r)
		}
	}()

	handler.handleSessionDetachment()

	// Should be marked as detached
	if handler.IsAttached() {
		t.Error("Expected handler to be detached after handleSessionDetachment")
	}

	// Test double detachment (should be safe)
	handler.handleSessionDetachment()
	
	if handler.IsAttached() {
		t.Error("Expected handler to remain detached after double detachment")
	}
}