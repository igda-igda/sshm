package connection

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sshm/internal/history"
	"sshm/internal/tmux"
)

func TestNewHealthMonitor(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-health-monitor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create history manager
	historyPath := filepath.Join(tempDir, "history.db")
	historyManager, err := history.NewHistoryManager(historyPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer historyManager.Close()

	// Create tmux manager
	tmuxManager := tmux.NewManager()

	// Create health monitor
	monitor := NewHealthMonitor(historyManager, tmuxManager)
	if monitor == nil {
		t.Fatal("Expected health monitor to be created")
	}

	if monitor.interval != 30*time.Second {
		t.Errorf("Expected default interval 30s, got %v", monitor.interval)
	}

	if len(monitor.sessions) != 0 {
		t.Errorf("Expected empty sessions map, got %d sessions", len(monitor.sessions))
	}
}

func TestHealthMonitor_SessionManagement(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-session-mgmt-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create history manager
	historyPath := filepath.Join(tempDir, "history.db")
	historyManager, err := history.NewHistoryManager(historyPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer historyManager.Close()

	// Create tmux manager
	tmuxManager := tmux.NewManager()

	// Create health monitor
	monitor := NewHealthMonitor(historyManager, tmuxManager)

	// Test adding session
	sessionID := "test-session"
	serverName := "test-server"
	monitor.AddSession(sessionID, serverName)

	// Verify session was added
	sessions := monitor.GetActiveSessions()
	if len(sessions) != 1 {
		t.Errorf("Expected 1 active session, got %d", len(sessions))
	}

	session, exists := sessions[sessionID]
	if !exists {
		t.Error("Expected test-session to exist in active sessions")
	} else {
		if session.ServerName != serverName {
			t.Errorf("Expected server name '%s', got '%s'", serverName, session.ServerName)
		}
		if session.LastStatus != "healthy" {
			t.Errorf("Expected initial status 'healthy', got '%s'", session.LastStatus)
		}
	}

	// Test getting session info
	info, exists := monitor.GetSessionInfo(sessionID)
	if !exists {
		t.Error("Expected session info to exist")
	} else {
		if info.SessionID != sessionID {
			t.Errorf("Expected session ID '%s', got '%s'", sessionID, info.SessionID)
		}
	}

	// Test removing session
	monitor.RemoveSession(sessionID)
	sessions = monitor.GetActiveSessions()
	if len(sessions) != 0 {
		t.Errorf("Expected 0 active sessions after removal, got %d", len(sessions))
	}

	// Verify session info no longer exists
	_, exists = monitor.GetSessionInfo(sessionID)
	if exists {
		t.Error("Expected session info to not exist after removal")
	}
}

func TestHealthMonitor_CheckInterval(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-interval-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create history manager
	historyPath := filepath.Join(tempDir, "history.db")
	historyManager, err := history.NewHistoryManager(historyPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer historyManager.Close()

	// Create tmux manager
	tmuxManager := tmux.NewManager()

	// Create health monitor
	monitor := NewHealthMonitor(historyManager, tmuxManager)

	// Test setting check interval
	newInterval := 60 * time.Second
	monitor.SetCheckInterval(newInterval)

	if monitor.interval != newInterval {
		t.Errorf("Expected interval %v, got %v", newInterval, monitor.interval)
	}
}

func TestHealthMonitor_HealthStats(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-stats-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create history manager
	historyPath := filepath.Join(tempDir, "history.db")
	historyManager, err := history.NewHistoryManager(historyPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer historyManager.Close()

	// Create tmux manager
	tmuxManager := tmux.NewManager()

	// Create health monitor
	monitor := NewHealthMonitor(historyManager, tmuxManager)

	// Add sessions with different statuses
	monitor.AddSession("healthy-session", "server1")
	monitor.AddSession("degraded-session", "server2")
	monitor.AddSession("failed-session", "server3")

	// Manually set different statuses for testing
	monitor.mutex.Lock()
	monitor.sessions["degraded-session"].LastStatus = "degraded"
	monitor.sessions["degraded-session"].FailureCount = 1
	monitor.sessions["failed-session"].LastStatus = "failed"
	monitor.sessions["failed-session"].FailureCount = 3
	monitor.mutex.Unlock()

	// Get health stats
	stats := monitor.GetHealthStats()

	if stats.TotalSessions != 3 {
		t.Errorf("Expected 3 total sessions, got %d", stats.TotalSessions)
	}

	if stats.HealthySessions != 1 {
		t.Errorf("Expected 1 healthy session, got %d", stats.HealthySessions)
	}

	if stats.DegradedSessions != 1 {
		t.Errorf("Expected 1 degraded session, got %d", stats.DegradedSessions)
	}

	if stats.FailedSessions != 1 {
		t.Errorf("Expected 1 failed session, got %d", stats.FailedSessions)
	}

	if stats.SessionsWithFailures != 2 {
		t.Errorf("Expected 2 sessions with failures, got %d", stats.SessionsWithFailures)
	}
}

func TestHealthMonitor_StartStopMonitoring(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-start-stop-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create history manager
	historyPath := filepath.Join(tempDir, "history.db")
	historyManager, err := history.NewHistoryManager(historyPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer historyManager.Close()

	// Create tmux manager
	tmuxManager := tmux.NewManager()

	// Create health monitor with short interval for testing
	monitor := NewHealthMonitor(historyManager, tmuxManager)
	monitor.SetCheckInterval(100 * time.Millisecond)

	// Start monitoring in background
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- monitor.StartMonitoring(ctx)
	}()

	// Let it run for a short time
	time.Sleep(200 * time.Millisecond)

	// Test manual stop
	monitor.Stop()

	// Wait for completion
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("StartMonitoring returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("StartMonitoring did not stop within timeout")
	}
}

func TestHealthMonitor_GetDetailedSessionInfo(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-detailed-session-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create history manager
	historyPath := filepath.Join(tempDir, "history.db")
	historyManager, err := history.NewHistoryManager(historyPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer historyManager.Close()

	// Create tmux manager
	tmuxManager := tmux.NewManager()

	// Create health monitor
	monitor := NewHealthMonitor(historyManager, tmuxManager)

	// Test with no sessions (tmux not available or no sessions)
	sessions, err := monitor.getDetailedSessionInfo()
	if err != nil {
		// This is expected if tmux is not available
		t.Logf("getDetailedSessionInfo failed (expected without tmux): %v", err)
		return
	}

	// If tmux is available but no sessions exist, should return empty slice
	if sessions == nil {
		t.Error("Expected empty slice, got nil")
	}

	t.Logf("Found %d detailed sessions", len(sessions))
	
	// If sessions exist, verify structure
	for i, session := range sessions {
		if session.Name == "" {
			t.Errorf("Session %d has empty name", i)
		}
		
		if session.Created.IsZero() {
			t.Errorf("Session %d has zero creation time", i)
		}
		
		t.Logf("Session %d: name=%s, created=%v, windows=%d", 
			i, session.Name, session.Created, session.Windows)
	}
}

func TestTmuxSessionInfo_Structure(t *testing.T) {
	// Test TmuxSessionInfo structure
	session := TmuxSessionInfo{
		Name:    "test-session",
		Created: time.Now(),
		Windows: 2,
	}

	if session.Name != "test-session" {
		t.Errorf("Expected name 'test-session', got '%s'", session.Name)
	}

	if session.Windows != 2 {
		t.Errorf("Expected 2 windows, got %d", session.Windows)
	}

	if session.Created.IsZero() {
		t.Error("Expected created time to be set")
	}
}