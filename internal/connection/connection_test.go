package connection

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sshm/internal/config"
	"sshm/internal/history"
)

func TestNewManager(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-connection-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer manager.Close()

	// Verify history database was created
	expectedHistoryPath := filepath.Join(tempDir, ".sshm", "history.db")
	if _, err := os.Stat(expectedHistoryPath); os.IsNotExist(err) {
		t.Error("History database file was not created")
	}

	// Verify manager is functional
	if !manager.IsAvailable() {
		t.Skip("tmux not available for testing")
	}
}

func TestConnectionHistoryRecording(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-connection-history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer manager.Close()

	// Test server configuration (this will fail connection but should record history)
	server := config.Server{
		Name:     "test-server",
		Hostname: "nonexistent.example.com", // This should fail
		Port:     22,
		Username: "testuser",
		AuthType: "key",
		KeyPath:  "/nonexistent/key",
	}

	// Try to connect (should fail but record history)
	_, _, err = manager.ConnectToServer(server)
	if err == nil {
		t.Error("Expected connection to fail for nonexistent server")
	}

	// Check that history was recorded
	historyFilter := history.HistoryFilter{
		ServerName: "test-server",
		Limit:      10,
	}

	historyEntries, err := manager.GetConnectionHistory(historyFilter)
	if err != nil {
		t.Fatalf("Failed to get connection history: %v", err)
	}

	if len(historyEntries) == 0 {
		t.Error("Expected at least one history entry")
	}

	entry := historyEntries[0]
	if entry.ServerName != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%s'", entry.ServerName)
	}

	if entry.Status != "failed" {
		t.Errorf("Expected status 'failed', got '%s'", entry.Status)
	}

	if entry.ConnectionType != "single" {
		t.Errorf("Expected connection type 'single', got '%s'", entry.ConnectionType)
	}

	if entry.ErrorMessage == "" {
		t.Error("Expected error message for failed connection")
	}
}

func TestConnectionStatistics(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-connection-stats-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer manager.Close()

	// Manually add some history entries for testing
	baseTime := time.Now().Add(-24 * time.Hour)
	entries := []history.ConnectionHistoryEntry{
		{
			ServerName:      "test-server",
			Host:           "test.example.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "success",
			StartTime:      baseTime,
			EndTime:        baseTime.Add(10 * time.Minute),
			DurationSeconds: 600,
		},
		{
			ServerName:      "test-server",
			Host:           "test.example.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "failed",
			StartTime:      baseTime.Add(1 * time.Hour),
			ErrorMessage:   "Connection timeout",
		},
	}

	for _, entry := range entries {
		_, err := manager.historyManager.RecordConnection(entry)
		if err != nil {
			t.Fatalf("Failed to record test history entry: %v", err)
		}
	}

	// Get statistics
	stats, err := manager.GetConnectionStats("test-server", "")
	if err != nil {
		t.Fatalf("Failed to get connection stats: %v", err)
	}

	if stats.TotalConnections != 2 {
		t.Errorf("Expected 2 total connections, got %d", stats.TotalConnections)
	}

	if stats.SuccessfulConnections != 1 {
		t.Errorf("Expected 1 successful connection, got %d", stats.SuccessfulConnections)
	}

	if stats.SuccessRate != 0.5 {
		t.Errorf("Expected success rate 0.5, got %f", stats.SuccessRate)
	}
}

func TestProfileConnectionHistory(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-profile-connection-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer manager.Close()

	// Test servers configuration (these will fail but should record history)
	servers := []config.Server{
		{
			Name:     "web-server",
			Hostname: "web.example.com",
			Port:     22,
			Username: "admin",
			AuthType: "key",
			KeyPath:  "/nonexistent/web-key",
		},
		{
			Name:     "db-server",
			Hostname: "db.example.com",
			Port:     22,
			Username: "admin",
			AuthType: "key",
			KeyPath:  "/nonexistent/db-key",
		},
	}

	// Try to connect to profile (should fail but record history)
	_, _, err = manager.ConnectToProfile("development", servers)
	if err == nil {
		t.Error("Expected profile connection to fail for nonexistent servers")
	}

	// Check that history was recorded for the profile
	profileFilter := history.HistoryFilter{
		ProfileName: "development",
		Limit:       10,
	}

	historyEntries, err := manager.GetConnectionHistory(profileFilter)
	if err != nil {
		t.Fatalf("Failed to get profile connection history: %v", err)
	}

	// Should have entries for the profile and each server
	if len(historyEntries) < 2 {
		t.Errorf("Expected at least 2 history entries (profile + servers), got %d", len(historyEntries))
	}

	// Check profile entry
	var profileEntry *history.ConnectionHistoryEntry
	var serverEntries []history.ConnectionHistoryEntry

	for _, entry := range historyEntries {
		if entry.ConnectionType == "group" && entry.ServerName == "development" {
			profileEntry = &entry
		} else if entry.ConnectionType == "group" {
			serverEntries = append(serverEntries, entry)
		}
	}

	if profileEntry == nil {
		t.Error("Expected to find profile history entry")
	} else {
		if profileEntry.ProfileName != "development" {
			t.Errorf("Expected profile name 'development', got '%s'", profileEntry.ProfileName)
		}
		if profileEntry.Status != "failed" {
			t.Errorf("Expected profile status 'failed', got '%s'", profileEntry.Status)
		}
	}

	if len(serverEntries) != 2 {
		t.Errorf("Expected 2 server entries, got %d", len(serverEntries))
	}
}

func TestHistoryCleanup(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-cleanup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer manager.Close()

	// Add old and new entries
	oldTime := time.Now().Add(-40 * 24 * time.Hour) // 40 days ago
	newTime := time.Now().Add(-5 * 24 * time.Hour)  // 5 days ago

	oldEntry := history.ConnectionHistoryEntry{
		ServerName:     "old-server",
		Host:           "old.example.com",
		User:           "admin",
		Port:           22,
		ConnectionType: "single",
		Status:         "success",
		StartTime:      oldTime,
	}

	newEntry := history.ConnectionHistoryEntry{
		ServerName:     "new-server",
		Host:           "new.example.com",
		User:           "admin",
		Port:           22,
		ConnectionType: "single",
		Status:         "success",
		StartTime:      newTime,
	}

	_, err = manager.historyManager.RecordConnection(oldEntry)
	if err != nil {
		t.Fatalf("Failed to record old connection: %v", err)
	}

	_, err = manager.historyManager.RecordConnection(newEntry)
	if err != nil {
		t.Fatalf("Failed to record new connection: %v", err)
	}

	// Test cleanup with 30-day retention
	deleted, err := manager.CleanupOldHistory(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("Failed to cleanup old history: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 deleted entry, got %d", deleted)
	}

	// Verify only new entry remains
	history, err := manager.GetConnectionHistory(history.HistoryFilter{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to get connection history after cleanup: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 remaining entry after cleanup, got %d", len(history))
	}

	if history[0].ServerName != "new-server" {
		t.Errorf("Expected remaining server to be 'new-server', got '%s'", history[0].ServerName)
	}
}

func TestRecentActivity(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-activity-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer manager.Close()

	// Add recent activity
	recentTime := time.Now().Add(-2 * time.Hour)
	entries := []history.ConnectionHistoryEntry{
		{
			ServerName:     "server1",
			Host:           "server1.example.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "success",
			StartTime:      recentTime,
		},
		{
			ServerName:     "server2",
			Host:           "server2.example.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "failed",
			StartTime:      recentTime.Add(30 * time.Minute),
		},
		{
			ServerName:     "server3",
			Host:           "server3.example.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "success",
			StartTime:      recentTime.Add(60 * time.Minute),
		},
	}

	for _, entry := range entries {
		_, err := manager.historyManager.RecordConnection(entry)
		if err != nil {
			t.Fatalf("Failed to record activity entry: %v", err)
		}
	}

	// Get recent activity for last 6 hours
	activity, err := manager.GetRecentActivity(6)
	if err != nil {
		t.Fatalf("Failed to get recent activity: %v", err)
	}

	if activity["success"] != 2 {
		t.Errorf("Expected 2 successful connections, got %d", activity["success"])
	}

	if activity["failed"] != 1 {
		t.Errorf("Expected 1 failed connection, got %d", activity["failed"])
	}
}