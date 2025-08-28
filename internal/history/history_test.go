package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewHistoryManager(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "sshm-history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "history.db")
	
	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestConnectionHistoryRecording(t *testing.T) {
	// Setup test database
	tempDir, err := os.MkdirTemp("", "sshm-history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewHistoryManager(filepath.Join(tempDir, "history.db"))
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Test data
	entry := ConnectionHistoryEntry{
		ServerName:     "test-server",
		ProfileName:    "development",
		Host:           "192.168.1.100",
		User:           "admin",
		Port:           22,
		ConnectionType: "single",
		Status:         "success",
		StartTime:      time.Now(),
		EndTime:        time.Now().Add(5 * time.Minute),
		DurationSeconds: 300,
		SessionID:      "sshm-test-session",
	}

	// Test recording connection
	id, err := manager.RecordConnection(entry)
	if err != nil {
		t.Fatalf("Failed to record connection: %v", err)
	}

	if id <= 0 {
		t.Error("Expected positive connection ID")
	}

	// Test retrieving connection history
	history, err := manager.GetConnectionHistory(HistoryFilter{
		ServerName: "test-server",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("Failed to get connection history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(history))
	}

	if history[0].ServerName != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%s'", history[0].ServerName)
	}

	if history[0].Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", history[0].Status)
	}
}

func TestConnectionStatistics(t *testing.T) {
	// Setup test database
	tempDir, err := os.MkdirTemp("", "sshm-history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewHistoryManager(filepath.Join(tempDir, "history.db"))
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Add test connection history entries
	baseTime := time.Now().Add(-24 * time.Hour)
	entries := []ConnectionHistoryEntry{
		{
			ServerName:      "server1",
			ProfileName:     "prod",
			Host:           "prod1.example.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "success",
			StartTime:      baseTime,
			EndTime:        baseTime.Add(10 * time.Minute),
			DurationSeconds: 600,
		},
		{
			ServerName:      "server1",
			ProfileName:     "prod",
			Host:           "prod1.example.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "failed",
			StartTime:      baseTime.Add(1 * time.Hour),
			ErrorMessage:   "Connection timeout",
		},
		{
			ServerName:      "server2",
			ProfileName:     "dev",
			Host:           "dev1.example.com",
			User:           "developer",
			Port:           22,
			ConnectionType: "single",
			Status:         "success",
			StartTime:      baseTime.Add(2 * time.Hour),
			EndTime:        baseTime.Add(2 * time.Hour).Add(5 * time.Minute),
			DurationSeconds: 300,
		},
	}

	for _, entry := range entries {
		_, err := manager.RecordConnection(entry)
		if err != nil {
			t.Fatalf("Failed to record connection: %v", err)
		}
	}

	// Test getting statistics for server1
	stats, err := manager.GetConnectionStats("server1", "prod")
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

	if stats.AverageDuration != 600.0 {
		t.Errorf("Expected average duration 600s, got %f", stats.AverageDuration)
	}
}

func TestSessionHealthMonitoring(t *testing.T) {
	// Setup test database
	tempDir, err := os.MkdirTemp("", "sshm-history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewHistoryManager(filepath.Join(tempDir, "history.db"))
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Test recording session health
	healthEntry := SessionHealthEntry{
		SessionID:      "test-session-123",
		ServerName:     "test-server",
		CheckTime:      time.Now(),
		Status:         "healthy",
		ResponseTimeMs: 45,
	}

	err = manager.RecordSessionHealth(healthEntry)
	if err != nil {
		t.Fatalf("Failed to record session health: %v", err)
	}

	// Test retrieving session health
	health, err := manager.GetSessionHealth("test-session-123", 10)
	if err != nil {
		t.Fatalf("Failed to get session health: %v", err)
	}

	if len(health) != 1 {
		t.Errorf("Expected 1 health entry, got %d", len(health))
	}

	if health[0].Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health[0].Status)
	}

	if health[0].ResponseTimeMs != 45 {
		t.Errorf("Expected response time 45ms, got %d", health[0].ResponseTimeMs)
	}
}

func TestHistoryCleanup(t *testing.T) {
	// Setup test database
	tempDir, err := os.MkdirTemp("", "sshm-history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewHistoryManager(filepath.Join(tempDir, "history.db"))
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Add old and new entries
	oldTime := time.Now().Add(-40 * 24 * time.Hour) // 40 days ago
	newTime := time.Now().Add(-5 * 24 * time.Hour)  // 5 days ago

	oldEntry := ConnectionHistoryEntry{
		ServerName:     "old-server",
		Host:           "old.example.com",
		User:           "admin",
		Port:           22,
		ConnectionType: "single",
		Status:         "success",
		StartTime:      oldTime,
	}

	newEntry := ConnectionHistoryEntry{
		ServerName:     "new-server",
		Host:           "new.example.com",
		User:           "admin",
		Port:           22,
		ConnectionType: "single",
		Status:         "success",
		StartTime:      newTime,
	}

	_, err = manager.RecordConnection(oldEntry)
	if err != nil {
		t.Fatalf("Failed to record old connection: %v", err)
	}

	_, err = manager.RecordConnection(newEntry)
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
	history, err := manager.GetConnectionHistory(HistoryFilter{Limit: 10})
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

func TestDatabaseMigrations(t *testing.T) {
	// Setup test database
	tempDir, err := os.MkdirTemp("", "sshm-history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "history.db")

	// Test initial database creation
	manager1, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create initial history manager: %v", err)
	}
	manager1.Close()

	// Test reopening existing database (should run any necessary migrations)
	manager2, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen existing history manager: %v", err)
	}
	defer manager2.Close()

	// Verify database is functional after reopening
	entry := ConnectionHistoryEntry{
		ServerName:     "migration-test",
		Host:           "test.example.com",
		User:           "admin",
		Port:           22,
		ConnectionType: "single",
		Status:         "success",
		StartTime:      time.Now(),
	}

	_, err = manager2.RecordConnection(entry)
	if err != nil {
		t.Fatalf("Failed to record connection after migration: %v", err)
	}
}

func TestHistoryFiltering(t *testing.T) {
	// Setup test database
	tempDir, err := os.MkdirTemp("", "sshm-history-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewHistoryManager(filepath.Join(tempDir, "history.db"))
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Add test entries with different attributes
	baseTime := time.Now().Add(-24 * time.Hour)
	entries := []ConnectionHistoryEntry{
		{
			ServerName:     "web-server",
			ProfileName:    "production",
			Host:           "web.prod.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "success",
			StartTime:      baseTime,
		},
		{
			ServerName:     "db-server",
			ProfileName:    "production",
			Host:           "db.prod.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "failed",
			StartTime:      baseTime.Add(1 * time.Hour),
		},
		{
			ServerName:     "web-server",
			ProfileName:    "development",
			Host:           "web.dev.com",
			User:           "developer",
			Port:           2222,
			ConnectionType: "group",
			Status:         "success",
			StartTime:      baseTime.Add(2 * time.Hour),
		},
	}

	for _, entry := range entries {
		_, err := manager.RecordConnection(entry)
		if err != nil {
			t.Fatalf("Failed to record connection: %v", err)
		}
	}

	// Test filtering by server name
	history, err := manager.GetConnectionHistory(HistoryFilter{
		ServerName: "web-server",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("Failed to get filtered history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 entries for web-server, got %d", len(history))
	}

	// Test filtering by profile
	history, err = manager.GetConnectionHistory(HistoryFilter{
		ProfileName: "production",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("Failed to get filtered history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 entries for production profile, got %d", len(history))
	}

	// Test filtering by status
	history, err = manager.GetConnectionHistory(HistoryFilter{
		Status: "failed",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Failed to get filtered history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 failed connection, got %d", len(history))
	}

	if history[0].ServerName != "db-server" {
		t.Errorf("Expected failed connection to be db-server, got %s", history[0].ServerName)
	}

	// Test date range filtering
	history, err = manager.GetConnectionHistory(HistoryFilter{
		StartTime: baseTime.Add(30 * time.Minute),
		EndTime:   baseTime.Add(3 * time.Hour),
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("Failed to get date-filtered history: %v", err)
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 entries in date range, got %d", len(history))
	}
}