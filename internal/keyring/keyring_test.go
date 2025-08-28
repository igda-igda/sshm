package keyring

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestKeyringManager_ServiceDetection tests keyring service detection across platforms
func TestKeyringManager_ServiceDetection(t *testing.T) {
	tests := []struct {
		name           string
		service        string
		expectError    bool
		skipOnPlatform string // "windows", "darwin", "linux"
	}{
		{
			name:        "auto service detection",
			service:     "auto",
			expectError: false,
		},
		{
			name:           "keychain service on macOS",
			service:        "keychain",
			expectError:    false,
			skipOnPlatform: "windows,linux",
		},
		{
			name:           "wincred service on Windows",
			service:        "wincred",
			expectError:    false,
			skipOnPlatform: "darwin,linux",
		},
		{
			name:           "secret-service on Linux",
			service:        "secret-service",
			expectError:    false,
			skipOnPlatform: "windows,darwin",
		},
		{
			name:        "invalid service type",
			service:     "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip platform-specific tests
			if tt.skipOnPlatform != "" {
				platformsToSkip := strings.Split(tt.skipOnPlatform, ",")
				currentPlatform := getCurrentPlatform()
				for _, platform := range platformsToSkip {
					if currentPlatform == strings.TrimSpace(platform) {
						t.Skip("Skipping platform-specific test")
					}
				}
			}

			manager := NewKeyringManager(tt.service)
			
			if tt.expectError {
				if manager != nil {
					t.Error("Expected error for invalid service, but got manager")
				}
			} else {
				if manager == nil {
					t.Error("Expected valid manager, but got nil")
				}
			}
		})
	}
}

// TestKeyringManager_IsAvailable tests keyring availability detection
func TestKeyringManager_IsAvailable(t *testing.T) {
	manager := NewKeyringManager("auto")
	if manager == nil {
		t.Skip("No keyring service available on this platform")
	}

	available := manager.IsAvailable()
	t.Logf("Keyring available: %v, Service: %s", available, manager.ServiceName())
	
	// On CI systems, keyring might not be available, so we don't fail the test
	if !available {
		t.Log("Keyring not available - this is expected in headless environments")
	}
}

// TestKeyringManager_ServiceName tests service name reporting
func TestKeyringManager_ServiceName(t *testing.T) {
	manager := NewKeyringManager("auto")
	if manager == nil {
		t.Skip("No keyring service available on this platform")
	}

	serviceName := manager.ServiceName()
	if serviceName == "" {
		t.Error("Service name should not be empty")
	}
	
	t.Logf("Detected service: %s", serviceName)
}

// TestKeyringManager_StoreRetrieveDelete tests basic credential operations
func TestKeyringManager_StoreRetrieveDelete(t *testing.T) {
	manager := NewKeyringManager("auto")
	if manager == nil {
		t.Skip("No keyring service available on this platform")
	}

	if !manager.IsAvailable() {
		t.Skip("Keyring not available in this environment")
	}

	testKey := "sshm-test-key"
	testValue := "test-secret-password-123"

	// Clean up any existing test data
	defer func() {
		_ = manager.Delete(testKey)
	}()

	// Test Store
	err := manager.Store(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Test Retrieve
	retrievedValue, err := manager.Retrieve(testKey)
	if err != nil {
		t.Fatalf("Failed to retrieve credential: %v", err)
	}

	if retrievedValue != testValue {
		t.Errorf("Retrieved value %q does not match stored value %q", retrievedValue, testValue)
	}

	// Test Delete
	err = manager.Delete(testKey)
	if err != nil {
		t.Fatalf("Failed to delete credential: %v", err)
	}

	// Verify deletion
	_, err = manager.Retrieve(testKey)
	if err == nil {
		t.Error("Expected error when retrieving deleted credential")
	}
}

// TestKeyringManager_StoreEmpty tests handling of empty values
func TestKeyringManager_StoreEmpty(t *testing.T) {
	manager := NewKeyringManager("auto")
	if manager == nil {
		t.Skip("No keyring service available on this platform")
	}

	if !manager.IsAvailable() {
		t.Skip("Keyring not available in this environment")
	}

	testKey := "sshm-test-empty"

	// Clean up
	defer func() {
		_ = manager.Delete(testKey)
	}()

	// Test storing empty string
	err := manager.Store(testKey, "")
	if err != nil {
		t.Fatalf("Failed to store empty credential: %v", err)
	}

	// Test retrieving empty string
	retrievedValue, err := manager.Retrieve(testKey)
	if err != nil {
		t.Fatalf("Failed to retrieve empty credential: %v", err)
	}

	if retrievedValue != "" {
		t.Errorf("Expected empty string, got %q", retrievedValue)
	}
}

// TestKeyringManager_List tests listing stored keys
func TestKeyringManager_List(t *testing.T) {
	manager := NewKeyringManager("auto")
	if manager == nil {
		t.Skip("No keyring service available on this platform")
	}

	if !manager.IsAvailable() {
		t.Skip("Keyring not available in this environment")
	}

	testKeys := []string{"sshm-test-list-1", "sshm-test-list-2", "sshm-test-list-3"}

	// Clean up
	defer func() {
		for _, key := range testKeys {
			_ = manager.Delete(key)
		}
	}()

	// Store test credentials
	for i, key := range testKeys {
		err := manager.Store(key, fmt.Sprintf("value-%d", i+1))
		if err != nil {
			t.Fatalf("Failed to store test credential %s: %v", key, err)
		}
	}

	// List keys
	keys, err := manager.List()
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	// Verify our test keys are in the list
	keysMap := make(map[string]bool)
	for _, key := range keys {
		keysMap[key] = true
	}

	for _, testKey := range testKeys {
		if !keysMap[testKey] {
			t.Errorf("Expected key %s not found in list", testKey)
		}
	}
}

// TestKeyringManager_RetrieveNonExistent tests retrieving non-existent keys
func TestKeyringManager_RetrieveNonExistent(t *testing.T) {
	manager := NewKeyringManager("auto")
	if manager == nil {
		t.Skip("No keyring service available on this platform")
	}

	if !manager.IsAvailable() {
		t.Skip("Keyring not available in this environment")
	}

	_, err := manager.Retrieve("sshm-nonexistent-key")
	if err == nil {
		t.Error("Expected error when retrieving non-existent key")
	}
}

// TestKeyringManager_DeleteNonExistent tests deleting non-existent keys
func TestKeyringManager_DeleteNonExistent(t *testing.T) {
	manager := NewKeyringManager("auto")
	if manager == nil {
		t.Skip("No keyring service available on this platform")
	}

	if !manager.IsAvailable() {
		t.Skip("Keyring not available in this environment")
	}

	// Deleting non-existent keys should not error (idempotent operation)
	err := manager.Delete("sshm-nonexistent-key")
	if err != nil {
		t.Logf("Delete non-existent key returned error (may be expected): %v", err)
	}
}

// TestKeyringManager_LargeValue tests handling of large credential values
func TestKeyringManager_LargeValue(t *testing.T) {
	manager := NewKeyringManager("auto")
	if manager == nil {
		t.Skip("No keyring service available on this platform")
	}

	if !manager.IsAvailable() {
		t.Skip("Keyring not available in this environment")
	}

	testKey := "sshm-test-large"
	// Create a large value (1KB)
	testValue := strings.Repeat("A", 1024)

	// Clean up
	defer func() {
		_ = manager.Delete(testKey)
	}()

	// Test storing large value
	err := manager.Store(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to store large credential: %v", err)
	}

	// Test retrieving large value
	retrievedValue, err := manager.Retrieve(testKey)
	if err != nil {
		t.Fatalf("Failed to retrieve large credential: %v", err)
	}

	if retrievedValue != testValue {
		t.Error("Retrieved large value does not match stored value")
	}
}

// TestKeyringManager_SpecialCharacters tests handling of special characters
func TestKeyringManager_SpecialCharacters(t *testing.T) {
	manager := NewKeyringManager("auto")
	if manager == nil {
		t.Skip("No keyring service available on this platform")
	}

	if !manager.IsAvailable() {
		t.Skip("Keyring not available in this environment")
	}

	testKey := "sshm-test-special"
	// Test value with special characters, newlines, and unicode
	testValue := "password!@#$%^&*(){}[]|\\:;\"'<>,.?/~`\nline2\n🔐🗝️"

	// Clean up
	defer func() {
		_ = manager.Delete(testKey)
	}()

	// Test storing special characters
	err := manager.Store(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to store credential with special characters: %v", err)
	}

	// Test retrieving special characters
	retrievedValue, err := manager.Retrieve(testKey)
	if err != nil {
		t.Fatalf("Failed to retrieve credential with special characters: %v", err)
	}

	if retrievedValue != testValue {
		t.Errorf("Retrieved value with special characters does not match stored value")
	}
}

// Helper function to get current platform for testing
func getCurrentPlatform() string {
	switch {
	case strings.Contains(strings.ToLower(os.Getenv("OS")), "windows"):
		return "windows"
	case fileExists("/System/Library/CoreServices/SystemVersion.plist"):
		return "darwin"
	default:
		return "linux"
	}
}

// Helper function to check file existence
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}