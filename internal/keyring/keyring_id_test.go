package keyring

import (
	"testing"
)

func TestGeneratePasswordKeyringID_Uniqueness(t *testing.T) {
	// Test that different server names generate unique keyring IDs
	serverNames := []string{
		"web-server-1",
		"web-server-2", 
		"database-primary",
		"database-replica",
		"cache-server",
		"load-balancer",
		"monitoring",
		"backup-server",
	}

	keyringIDs := make(map[string]bool)
	
	for _, serverName := range serverNames {
		keyringID := GeneratePasswordKeyringID(serverName)
		
		// Check if this ID has been generated before
		if keyringIDs[keyringID] {
			t.Errorf("GeneratePasswordKeyringID() generated duplicate ID: %s for server: %s", keyringID, serverName)
		}
		
		keyringIDs[keyringID] = true
	}

	// Verify we have the expected number of unique IDs
	if len(keyringIDs) != len(serverNames) {
		t.Errorf("Expected %d unique keyring IDs, got %d", len(serverNames), len(keyringIDs))
	}
}

func TestGeneratePasswordKeyringID_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		expected   string
	}{
		{
			name:       "server with dots",
			serverName: "app.example.com",
			expected:   "password-app.example.com",
		},
		{
			name:       "server with dashes",
			serverName: "web-server-01",
			expected:   "password-web-server-01",
		},
		{
			name:       "server with underscores",
			serverName: "db_primary_01",
			expected:   "password-db_primary_01",
		},
		{
			name:       "server with colons (port)",
			serverName: "server.com:2222",
			expected:   "password-server.com:2222",
		},
		{
			name:       "server with at symbol",
			serverName: "user@host.com",
			expected:   "password-user@host.com",
		},
		{
			name:       "server with mixed characters",
			serverName: "web-01.staging.example.com:22",
			expected:   "password-web-01.staging.example.com:22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePasswordKeyringID(tt.serverName)
			if result != tt.expected {
				t.Errorf("GeneratePasswordKeyringID(%s) = %s, want %s", tt.serverName, result, tt.expected)
			}
		})
	}
}

func TestGeneratePasswordKeyringID_Consistency(t *testing.T) {
	// Test that the same server name always generates the same keyring ID
	serverName := "consistency-test-server"
	
	firstID := GeneratePasswordKeyringID(serverName)
	
	// Generate the ID multiple times
	for i := 0; i < 10; i++ {
		id := GeneratePasswordKeyringID(serverName)
		if id != firstID {
			t.Errorf("GeneratePasswordKeyringID() is not consistent. First: %s, Got: %s", firstID, id)
		}
	}
}

func TestGeneratePasswordKeyringID_Collision_Resistance(t *testing.T) {
	// Test edge cases that might create collisions
	tests := []struct {
		name1      string
		name2      string
		shouldDiff bool
	}{
		{
			name1:      "server",
			name2:      "server-",
			shouldDiff: true,
		},
		{
			name1:      "server-1",
			name2:      "server-2",
			shouldDiff: true,
		},
		{
			name1:      "web",
			name2:      "web.",
			shouldDiff: true,
		},
		{
			name1:      "app-server",
			name2:      "appserver",
			shouldDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run("collision_test_"+tt.name1+"_vs_"+tt.name2, func(t *testing.T) {
			id1 := GeneratePasswordKeyringID(tt.name1)
			id2 := GeneratePasswordKeyringID(tt.name2)
			
			if tt.shouldDiff && id1 == id2 {
				t.Errorf("Expected different IDs for %s and %s, both got: %s", tt.name1, tt.name2, id1)
			}
		})
	}
}

func TestGeneratePasswordKeyringID_LongNames(t *testing.T) {
	// Test with very long server names
	longName := "very-long-server-name-that-exceeds-normal-limits-and-tests-keyring-handling-of-extended-identifiers"
	
	keyringID := GeneratePasswordKeyringID(longName)
	
	// Verify the ID is properly formed
	expectedPrefix := "password-"
	if !hasPrefix(keyringID, expectedPrefix) {
		t.Errorf("Expected keyring ID to start with '%s', got: %s", expectedPrefix, keyringID)
	}
	
	// Verify the ID contains the full server name
	expectedSuffix := longName
	if !hasSuffix(keyringID, expectedSuffix) {
		t.Errorf("Expected keyring ID to end with '%s', got: %s", expectedSuffix, keyringID)
	}
}

func TestGeneratePasswordKeyringID_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		serverName  string
		expected    string
		description string
	}{
		{
			name:        "empty string",
			serverName:  "",
			expected:    "password-",
			description: "should handle empty server name",
		},
		{
			name:        "single character",
			serverName:  "a",
			expected:    "password-a",
			description: "should handle single character names",
		},
		{
			name:        "only special characters",
			serverName:  "@#$%^&*()",
			expected:    "password-@#$%^&*()",
			description: "should handle names with only special characters",
		},
		{
			name:        "unicode characters",
			serverName:  "сервер-тест",
			expected:    "password-сервер-тест",
			description: "should handle unicode characters",
		},
		{
			name:        "spaces in name",
			serverName:  "server with spaces",
			expected:    "password-server with spaces",
			description: "should handle names with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePasswordKeyringID(tt.serverName)
			if result != tt.expected {
				t.Errorf("GeneratePasswordKeyringID(%s) = %s, want %s", tt.serverName, result, tt.expected)
			}
		})
	}
}

// Helper functions since we can't import strings in tests
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}