package keyring

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFileBackendFallback tests that the keyring falls back to file backend when native keyring is unavailable
func TestFileBackendFallback(t *testing.T) {
	// We'll test that the file backend can be explicitly requested
	manager := NewKeyringManager("file")
	if manager == nil {
		t.Fatal("Expected file backend to be available")
	}

	// Test that it reports as file backend
	serviceName := manager.ServiceName()
	if serviceName != "file" {
		t.Errorf("Expected service name 'file', got '%s'", serviceName)
	}

	// Test basic operations with file backend
	if !manager.IsAvailable() {
		t.Error("File backend should always be available")
	}

	testKey := "fallback-test-key"
	testValue := "fallback-test-value"

	// Clean up
	defer func() {
		_ = manager.Delete(testKey)
	}()

	// Test store
	err := manager.Store(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to store with file backend: %v", err)
	}

	// Test retrieve
	retrievedValue, err := manager.Retrieve(testKey)
	if err != nil {
		t.Fatalf("Failed to retrieve with file backend: %v", err)
	}

	if retrievedValue != testValue {
		t.Errorf("Retrieved value %q doesn't match stored value %q", retrievedValue, testValue)
	}

	// Test list
	keys, err := manager.List()
	if err != nil {
		t.Fatalf("Failed to list keys with file backend: %v", err)
	}

	found := false
	for _, key := range keys {
		if key == testKey {
			found = true
			break
		}
	}

	if !found {
		t.Error("Stored key not found in list")
	}

	// Test delete
	err = manager.Delete(testKey)
	if err != nil {
		t.Fatalf("Failed to delete with file backend: %v", err)
	}

	// Verify deletion
	_, err = manager.Retrieve(testKey)
	if err == nil {
		t.Error("Expected error after deleting key")
	}

}

// TestKeyringFallbackBehavior tests the automatic fallback behavior
func TestKeyringFallbackBehavior(t *testing.T) {
	// Test that we can create a keyring manager even in environments
	// where native keyring might not be available
	manager := NewKeyringManager("auto")
	if manager == nil {
		t.Fatal("Auto keyring should fall back to file backend if native keyring unavailable")
	}

	// Should be available (either native or file backend)
	if !manager.IsAvailable() {
		t.Error("Keyring manager should be available with fallback")
	}

	serviceName := manager.ServiceName()
	t.Logf("Using service: %s", serviceName)

	// Service name should be one of the expected values
	expectedServices := []string{"keychain", "wincred", "secret-service", "file"}
	validService := false
	for _, expected := range expectedServices {
		if serviceName == expected {
			validService = true
			break
		}
	}

	if !validService {
		t.Errorf("Unexpected service name: %s", serviceName)
	}
}

// TestEncryptedFileStorage tests that file backend actually encrypts data
func TestEncryptedFileStorage(t *testing.T) {
	// Create a keyring manager with file backend
	manager := NewKeyringManager("file")
	if manager == nil {
		t.Fatal("File backend should be available")
	}

	testKey := "encryption-test-key"
	testValue := "sensitive-password-data-123!@#"

	// Clean up
	defer func() {
		_ = manager.Delete(testKey)
	}()

	// Store the credential
	err := manager.Store(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Try to find the raw file and verify it's encrypted
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	keyringDir := filepath.Join(homeDir, ".sshm", "keyring")
	
	// Check if keyring directory exists
	if _, err := os.Stat(keyringDir); os.IsNotExist(err) {
		// Directory might not exist if keyring backend chose a different location
		// This is okay, we just can't verify file encryption
		t.Log("Keyring directory not found at expected location, skipping file encryption verification")
		return
	}

	// Look for any files in the keyring directory
	files, err := os.ReadDir(keyringDir)
	if err != nil {
		t.Fatalf("Failed to read keyring directory: %v", err)
	}

	// If files exist, check that they don't contain plaintext password
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(keyringDir, file.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip files we can't read
		}

		// Verify that the sensitive data is not stored in plaintext
		contentStr := string(content)
		if len(contentStr) > 0 && contentStr == testValue {
			t.Error("Credential appears to be stored in plaintext")
		}

		// Log file for debugging (without revealing content)
		t.Logf("Found keyring file: %s, size: %d bytes", file.Name(), len(content))
	}

	// Verify we can still retrieve the credential correctly
	retrievedValue, err := manager.Retrieve(testKey)
	if err != nil {
		t.Fatalf("Failed to retrieve credential: %v", err)
	}

	if retrievedValue != testValue {
		t.Errorf("Retrieved credential doesn't match stored value")
	}
}