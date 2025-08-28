package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"sshm/internal/connection"
	"sshm/internal/history"
)

func TestHistoryListCommand(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-history-cmd-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create connection manager and add some test history
	manager, err := connection.NewManager()
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer manager.Close()

	// Add test history entries
	baseTime := time.Now().Add(-2 * time.Hour)
	testEntries := []history.ConnectionHistoryEntry{
		{
			ServerName:      "web-server",
			ProfileName:     "production",
			Host:           "web.prod.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "success",
			StartTime:      baseTime,
			EndTime:        baseTime.Add(10 * time.Minute),
			DurationSeconds: 600,
			SessionID:      "web-server-session",
		},
		{
			ServerName:      "db-server",
			ProfileName:     "production",
			Host:           "db.prod.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "failed",
			StartTime:      baseTime.Add(30 * time.Minute),
			ErrorMessage:   "Connection timeout",
		},
	}

	for _, entry := range testEntries {
		_, err := manager.GetHistoryManager().RecordConnection(entry)
		if err != nil {
			t.Fatalf("Failed to record test entry: %v", err)
		}
	}

	tests := []struct {
		name       string
		serverName string
		profile    string
		status     string
		days       int
		limit      int
		expectErr  bool
		contains   []string
	}{
		{
			name:      "list all history",
			limit:     20,
			expectErr: false,
			contains:  []string{"web-server", "db-server", "SUCCESS", "FAILED"},
		},
		{
			name:       "filter by server",
			serverName: "web-server",
			limit:      20,
			expectErr:  false,
			contains:   []string{"web-server", "SUCCESS"},
		},
		{
			name:      "filter by status",
			status:    "failed",
			limit:     20,
			expectErr: false,
			contains:  []string{"db-server", "FAILED"},
		},
		{
			name:      "filter by profile",
			profile:   "production",
			limit:     20,
			expectErr: false,
			contains:  []string{"web-server", "db-server"},
		},
		{
			name:      "limit results",
			limit:     1,
			expectErr: false,
			contains:  []string{"entries"}, // Should show "Showing 1 entries"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := runHistoryListCommand(&buf, tt.serverName, tt.profile, tt.status, tt.days, tt.limit)
			
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			output := buf.String()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't. Output:\n%s", expected, output)
				}
			}
		})
	}
}

func TestHistoryStatsCommand(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-history-stats-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create connection manager and add some test history
	manager, err := connection.NewManager()
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer manager.Close()

	// Add test history entries for statistics
	baseTime := time.Now().Add(-24 * time.Hour)
	testEntries := []history.ConnectionHistoryEntry{
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
			StartTime:      baseTime.Add(2 * time.Hour),
			ErrorMessage:   "Connection refused",
		},
	}

	historyManager := manager.GetHistoryManager()
	for _, entry := range testEntries {
		_, err := historyManager.RecordConnection(entry)
		if err != nil {
			t.Fatalf("Failed to record test entry: %v", err)
		}
	}

	tests := []struct {
		name        string
		serverName  string
		profileName string
		expectErr   bool
		contains    []string
	}{
		{
			name:       "server stats",
			serverName: "test-server",
			expectErr:  false,
			contains:   []string{"Statistics for test-server", "Total Connections:", "Success Rate:"},
		},
		{
			name:      "recent activity",
			expectErr: false,
			contains:  []string{"Recent Activity", "Total Connections:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := runHistoryStatsCommand(&buf, tt.serverName, tt.profileName)
			
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			output := buf.String()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't. Output:\n%s", expected, output)
				}
			}
		})
	}
}

func TestHistoryCleanupCommand(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-history-cleanup-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create connection manager and add some test history
	manager, err := connection.NewManager()
	if err != nil {
		t.Fatalf("Failed to create connection manager: %v", err)
	}
	defer manager.Close()

	// Add old and new test entries
	oldTime := time.Now().Add(-40 * 24 * time.Hour) // 40 days ago
	newTime := time.Now().Add(-5 * 24 * time.Hour)  // 5 days ago

	testEntries := []history.ConnectionHistoryEntry{
		{
			ServerName:     "old-server",
			Host:           "old.example.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "success",
			StartTime:      oldTime,
		},
		{
			ServerName:     "new-server",
			Host:           "new.example.com",
			User:           "admin",
			Port:           22,
			ConnectionType: "single",
			Status:         "success",
			StartTime:      newTime,
		},
	}

	historyManager := manager.GetHistoryManager()
	for _, entry := range testEntries {
		_, err := historyManager.RecordConnection(entry)
		if err != nil {
			t.Fatalf("Failed to record test entry: %v", err)
		}
	}

	tests := []struct {
		name      string
		days      int
		expectErr bool
		contains  []string
	}{
		{
			name:      "cleanup old entries",
			days:      30,
			expectErr: false,
			contains:  []string{"Cleaned up", "old history entries"},
		},
		{
			name:      "invalid days",
			days:      0,
			expectErr: true,
		},
		{
			name:      "no entries to cleanup",
			days:      1, // Keep only 1 day - should clean up everything
			expectErr: false,
			contains:  []string{"history entries"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := runHistoryCleanupCommand(&buf, tt.days)
			
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			output := buf.String()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't. Output:\n%s", expected, output)
				}
			}
		})
	}
}

func TestHistoryHealthCommand(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "sshm-history-health-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set HOME to temp directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	var buf bytes.Buffer
	err = runHistoryHealthCommand(&buf)
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	output := buf.String()
	expectedContent := []string{
		"Session health monitoring",
		"Health monitoring data",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't. Output:\n%s", expected, output)
		}
	}
}

func TestDisplayHistoryEntry(t *testing.T) {
	entry := history.ConnectionHistoryEntry{
		ServerName:      "test-server",
		ProfileName:     "production",
		Host:           "test.example.com",
		User:           "admin",
		Port:           22,
		ConnectionType: "single",
		Status:         "success",
		StartTime:      time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC),
		EndTime:        time.Date(2023, 12, 25, 10, 35, 0, 0, time.UTC),
		DurationSeconds: 300,
		SessionID:      "test-session",
	}

	var buf bytes.Buffer
	displayHistoryEntry(&buf, entry, true)

	output := buf.String()
	expectedContent := []string{
		"test-server",
		"admin@test.example.com:22",
		"2023-12-25 10:30:00",
		"Duration: 5m0s",
		"Session: test-session",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't. Output:\n%s", expected, output)
		}
	}
}

func TestDisplayServerStats(t *testing.T) {
	stats := &history.ConnectionStats{
		ServerName:            "test-server",
		ProfileName:           "production",
		TotalConnections:      10,
		SuccessfulConnections: 8,
		SuccessRate:           0.8,
		AverageDuration:       300.5,
		FirstConnection:       time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC),
		LastConnection:        time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
	}

	var buf bytes.Buffer
	displayServerStats(&buf, stats)

	output := buf.String()
	expectedContent := []string{
		"Statistics for test-server",
		"Total Connections: 10",
		"Successful Connections: 8",
		"Failed Connections: 2",
		"Success Rate: 80.0%",
		"Average Duration: 5m0.5s",
		"First Connection: 2023-12-01 10:00:00",
		"Last Connection: 2023-12-25 15:30:00",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't. Output:\n%s", expected, output)
		}
	}
}

func TestDisplayActivityStats(t *testing.T) {
	activity := map[string]int{
		"success":   5,
		"failed":    2,
		"timeout":   1,
		"cancelled": 1,
	}

	var buf bytes.Buffer
	displayActivityStats(&buf, activity)

	output := buf.String()
	expectedContent := []string{
		"Recent Activity (Last 24 Hours)",
		"Total Connections: 9",
		"Successful: 5",
		"Failed: 2",
		"Timeout: 1",
		"Cancelled: 1",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't. Output:\n%s", expected, output)
		}
	}

	// Test empty activity
	var buf2 bytes.Buffer
	displayActivityStats(&buf2, map[string]int{})

	output2 := buf2.String()
	if !strings.Contains(output2, "No recent connection activity") {
		t.Errorf("Expected empty activity message, but got: %s", output2)
	}
}