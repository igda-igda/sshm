package keyring

import (
	"testing"
)

func TestPasswordStorage(t *testing.T) {
	// Use file backend for testing
	manager := NewKeyringManager("file")
	if manager == nil {
		t.Fatal("failed to create keyring manager")
	}

	tests := []struct {
		name         string
		serverName   string
		password     string
		wantError    bool
		description  string
	}{
		{
			name:        "store basic password",
			serverName:  "test-server-1",
			password:    "secret123",
			wantError:   false,
			description: "should successfully store server password",
		},
		{
			name:        "store password with special characters",
			serverName:  "test-server-2",
			password:    "P@ssw0rd!@#$%^&*()",
			wantError:   false,
			description: "should handle passwords with special characters",
		},
		{
			name:        "store empty password",
			serverName:  "test-server-3",
			password:    "",
			wantError:   false,
			description: "should allow empty passwords",
		},
		{
			name:        "store long password",
			serverName:  "test-server-4",
			password:    "verylongpasswordthatexceeds50charactersandtestshandlingoflongstrings",
			wantError:   false,
			description: "should handle long passwords",
		},
		{
			name:        "store unicode password",
			serverName:  "test-server-5",
			password:    "пароль密码パスワード",
			wantError:   false,
			description: "should handle unicode characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate keyring ID for password storage
			keyringID := GeneratePasswordKeyringID(tt.serverName)
			
			// Store the password
			err := manager.Store(keyringID, tt.password)
			if (err != nil) != tt.wantError {
				t.Errorf("Store() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				// Retrieve and verify the password
				retrieved, err := manager.Retrieve(keyringID)
				if err != nil {
					t.Errorf("Retrieve() error = %v", err)
					return
				}

				if retrieved != tt.password {
					t.Errorf("Retrieved password = %v, want %v", retrieved, tt.password)
				}
			}

			// Clean up
			_ = manager.Delete(keyringID)
		})
	}
}

func TestPasswordRetrieval(t *testing.T) {
	manager := NewKeyringManager("file")
	if manager == nil {
		t.Fatal("failed to create keyring manager")
	}

	tests := []struct {
		name         string
		serverName   string
		password     string
		shouldExist  bool
		description  string
	}{
		{
			name:        "retrieve existing password",
			serverName:  "existing-server",
			password:    "stored-password",
			shouldExist: true,
			description: "should retrieve previously stored password",
		},
		{
			name:        "retrieve non-existent password",
			serverName:  "missing-server",
			password:    "",
			shouldExist: false,
			description: "should return error for non-existent password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyringID := GeneratePasswordKeyringID(tt.serverName)

			if tt.shouldExist {
				// Store the password first
				err := manager.Store(keyringID, tt.password)
				if err != nil {
					t.Fatalf("failed to store password: %v", err)
				}
				defer manager.Delete(keyringID)
			}

			// Try to retrieve the password
			retrieved, err := manager.Retrieve(keyringID)

			if tt.shouldExist {
				if err != nil {
					t.Errorf("Retrieve() error = %v, want nil", err)
					return
				}
				if retrieved != tt.password {
					t.Errorf("Retrieved password = %v, want %v", retrieved, tt.password)
				}
			} else {
				if err == nil {
					t.Errorf("Retrieve() should return error for non-existent password")
				}
			}
		})
	}
}

func TestPasswordKeyringIDGeneration(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		expected   string
	}{
		{
			name:       "basic server name",
			serverName: "web-server-1",
			expected:   "password-web-server-1",
		},
		{
			name:       "server with special characters",
			serverName: "server@host.com:22",
			expected:   "password-server@host.com:22",
		},
		{
			name:       "empty server name",
			serverName: "",
			expected:   "password-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePasswordKeyringID(tt.serverName)
			if result != tt.expected {
				t.Errorf("GeneratePasswordKeyringID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPasswordUpdate(t *testing.T) {
	manager := NewKeyringManager("file")
	if manager == nil {
		t.Fatal("failed to create keyring manager")
	}

	serverName := "update-test-server"
	keyringID := GeneratePasswordKeyringID(serverName)
	
	// Store initial password
	initialPassword := "initial-password"
	err := manager.Store(keyringID, initialPassword)
	if err != nil {
		t.Fatalf("failed to store initial password: %v", err)
	}
	defer manager.Delete(keyringID)

	// Verify initial storage
	retrieved, err := manager.Retrieve(keyringID)
	if err != nil {
		t.Errorf("failed to retrieve initial password: %v", err)
	}
	if retrieved != initialPassword {
		t.Errorf("initial password = %v, want %v", retrieved, initialPassword)
	}

	// Update the password
	updatedPassword := "updated-password-123"
	err = manager.Store(keyringID, updatedPassword)
	if err != nil {
		t.Fatalf("failed to update password: %v", err)
	}

	// Verify update
	retrieved, err = manager.Retrieve(keyringID)
	if err != nil {
		t.Errorf("failed to retrieve updated password: %v", err)
	}
	if retrieved != updatedPassword {
		t.Errorf("updated password = %v, want %v", retrieved, updatedPassword)
	}
}

func TestPasswordDeletion(t *testing.T) {
	manager := NewKeyringManager("file")
	if manager == nil {
		t.Fatal("failed to create keyring manager")
	}

	serverName := "delete-test-server"
	keyringID := GeneratePasswordKeyringID(serverName)
	password := "password-to-delete"

	// Store password
	err := manager.Store(keyringID, password)
	if err != nil {
		t.Fatalf("failed to store password: %v", err)
	}

	// Verify storage
	retrieved, err := manager.Retrieve(keyringID)
	if err != nil {
		t.Errorf("failed to retrieve password: %v", err)
	}
	if retrieved != password {
		t.Errorf("stored password = %v, want %v", retrieved, password)
	}

	// Delete password
	err = manager.Delete(keyringID)
	if err != nil {
		t.Errorf("failed to delete password: %v", err)
	}

	// Verify deletion
	_, err = manager.Retrieve(keyringID)
	if err == nil {
		t.Errorf("Retrieve() should return error after deletion")
	}
}

func TestMultiplePasswordStorage(t *testing.T) {
	manager := NewKeyringManager("file")
	if manager == nil {
		t.Fatal("failed to create keyring manager")
	}

	servers := []struct {
		name     string
		password string
	}{
		{"server-1", "password-1"},
		{"server-2", "password-2"},
		{"server-3", "password-3"},
	}

	// Store all passwords
	var keyringIDs []string
	for _, server := range servers {
		keyringID := GeneratePasswordKeyringID(server.name)
		keyringIDs = append(keyringIDs, keyringID)
		
		err := manager.Store(keyringID, server.password)
		if err != nil {
			t.Fatalf("failed to store password for %s: %v", server.name, err)
		}
	}

	// Retrieve and verify all passwords
	for i, server := range servers {
		retrieved, err := manager.Retrieve(keyringIDs[i])
		if err != nil {
			t.Errorf("failed to retrieve password for %s: %v", server.name, err)
		}
		if retrieved != server.password {
			t.Errorf("password for %s = %v, want %v", server.name, retrieved, server.password)
		}
	}

	// Clean up
	for _, keyringID := range keyringIDs {
		_ = manager.Delete(keyringID)
	}
}

func TestPasswordStorageEdgeCases(t *testing.T) {
	manager := NewKeyringManager("file")
	if manager == nil {
		t.Fatal("failed to create keyring manager")
	}

	tests := []struct {
		name        string
		serverName  string
		password    string
		description string
	}{
		{
			name:        "newline in password",
			serverName:  "newline-server",
			password:    "password\nwith\nnewlines",
			description: "should handle passwords with newlines",
		},
		{
			name:        "tab in password",
			serverName:  "tab-server", 
			password:    "password\twith\ttabs",
			description: "should handle passwords with tabs",
		},
		{
			name:        "binary-like password",
			serverName:  "binary-server",
			password:    string([]byte{0x00, 0x01, 0x02, 0x03, 0xFF}),
			description: "should handle binary-like content in passwords",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyringID := GeneratePasswordKeyringID(tt.serverName)
			
			// Store password
			err := manager.Store(keyringID, tt.password)
			if err != nil {
				t.Errorf("Store() error = %v", err)
				return
			}
			defer manager.Delete(keyringID)

			// Retrieve and verify
			retrieved, err := manager.Retrieve(keyringID)
			if err != nil {
				t.Errorf("Retrieve() error = %v", err)
				return
			}

			if retrieved != tt.password {
				t.Errorf("Retrieved password = %v, want %v", retrieved, tt.password)
			}
		})
	}
}