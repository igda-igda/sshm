package keyring

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"sshm/internal/config"
)

// TestMigrateFromPlaintext tests migrating plaintext credentials to keyring
func TestMigrateFromPlaintext(t *testing.T) {
	// Create a temporary config directory
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config with plaintext credentials
	cfg := &config.Config{
		Servers: []config.Server{
			{
				Name:                "server1",
				Hostname:            "host1.example.com",
				Port:                22,
				Username:            "user1",
				AuthType:            "key",
				KeyPath:             "~/.ssh/id_rsa",
				PassphraseProtected: true,
			},
			{
				Name:     "server2",
				Hostname: "host2.example.com",
				Port:     22,
				Username: "user2",
				AuthType: "password",
			},
		},
	}

	// Save config
	configPath := filepath.Join(tempDir, "config.yaml")
	err := cfg.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create keyring manager for testing
	manager := NewKeyringManager("auto")
	if manager == nil || !manager.IsAvailable() {
		t.Skip("Keyring not available for migration testing")
	}

	// Clean up any existing test credentials
	defer func() {
		_ = manager.Delete("server1_passphrase")
		_ = manager.Delete("server2_password")
	}()

	// Simulate user providing credentials during migration
	credentials := map[string]string{
		"server1_passphrase": "secret-passphrase-123",
		"server2_password":   "secret-password-456",
	}

	// Run migration
	migrated, err := MigrateFromPlaintext(cfg, manager, func(prompt string) (string, error) {
		// Mock prompt function that returns predefined credentials
		for key, value := range credentials {
			if key == "server1_passphrase" && prompt == "Enter passphrase for server1 SSH key:" {
				return value, nil
			}
			if key == "server2_password" && prompt == "Enter password for server2:" {
				return value, nil
			}
		}
		return "", nil
	})

	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	if len(migrated) != 2 {
		t.Errorf("Expected 2 migrated credentials, got %d", len(migrated))
	}

	// Verify credentials were stored in keyring
	for key, expectedValue := range credentials {
		retrievedValue, err := manager.Retrieve(key)
		if err != nil {
			t.Errorf("Failed to retrieve migrated credential %s: %v", key, err)
			continue
		}

		if retrievedValue != expectedValue {
			t.Errorf("Retrieved credential %s = %q, expected %q", key, retrievedValue, expectedValue)
		}
	}

	// Verify server configs were updated with keyring references
	for i, server := range cfg.Servers {
		if !server.UseKeyring {
			t.Errorf("Server %s was not updated to use keyring", server.Name)
		}

		expectedKeyringID := server.Name + "_"
		if server.AuthType == "password" {
			expectedKeyringID += "password"
		} else if server.PassphraseProtected {
			expectedKeyringID += "passphrase"
		}

		if server.KeyringID != expectedKeyringID {
			t.Errorf("Server %s keyring ID = %q, expected %q", server.Name, server.KeyringID, expectedKeyringID)
		}

		t.Logf("Migrated server %d: %+v", i, server)
	}
}

// TestMigrateFromPlaintext_AlreadyMigrated tests migration when credentials are already migrated
func TestMigrateFromPlaintext_AlreadyMigrated(t *testing.T) {
	// Create a temporary config directory
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config with keyring-enabled credentials
	cfg := &config.Config{
		Servers: []config.Server{
			{
				Name:        "server1",
				Hostname:    "host1.example.com",
				Port:        22,
				Username:    "user1",
				AuthType:    "key",
				KeyPath:     "~/.ssh/id_rsa",
				UseKeyring:  true,
				KeyringID:   "server1_passphrase",
			},
		},
	}

	// Create keyring manager for testing
	manager := NewKeyringManager("auto")
	if manager == nil || !manager.IsAvailable() {
		t.Skip("Keyring not available for migration testing")
	}

	// Pre-store credential in keyring
	err := manager.Store("server1_passphrase", "existing-passphrase")
	if err != nil {
		t.Fatalf("Failed to pre-store credential: %v", err)
	}

	defer func() {
		_ = manager.Delete("server1_passphrase")
	}()

	// Run migration
	migrated, err := MigrateFromPlaintext(cfg, manager, func(prompt string) (string, error) {
		t.Error("Prompt function should not be called for already migrated credentials")
		return "", nil
	})

	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Should not migrate any credentials since they're already migrated
	if len(migrated) != 0 {
		t.Errorf("Expected 0 migrated credentials, got %d", len(migrated))
	}
}

// TestMigrateFromPlaintext_SkipKeyOnly tests skipping key-only authentication
func TestMigrateFromPlaintext_SkipKeyOnly(t *testing.T) {
	// Create a temporary config directory
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config with key-only authentication (no passphrase)
	cfg := &config.Config{
		Servers: []config.Server{
			{
				Name:                "server1",
				Hostname:            "host1.example.com",
				Port:                22,
				Username:            "user1",
				AuthType:            "key",
				KeyPath:             "~/.ssh/id_rsa",
				PassphraseProtected: false, // No passphrase
			},
		},
	}

	// Create keyring manager for testing
	manager := NewKeyringManager("auto")
	if manager == nil || !manager.IsAvailable() {
		t.Skip("Keyring not available for migration testing")
	}

	// Run migration
	migrated, err := MigrateFromPlaintext(cfg, manager, func(prompt string) (string, error) {
		t.Error("Prompt function should not be called for key-only authentication")
		return "", nil
	})

	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Should not migrate any credentials since key-only auth doesn't need keyring
	if len(migrated) != 0 {
		t.Errorf("Expected 0 migrated credentials, got %d", len(migrated))
	}

	// Server config should not be modified
	server := cfg.Servers[0]
	if server.UseKeyring {
		t.Error("Server should not be configured to use keyring for key-only auth")
	}
	if server.KeyringID != "" {
		t.Errorf("Server keyring ID should be empty, got %q", server.KeyringID)
	}
}

// TestMigrateFromPlaintext_PromptError tests handling of prompt errors
func TestMigrateFromPlaintext_PromptError(t *testing.T) {
	// Create a temporary config directory
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config with password authentication
	cfg := &config.Config{
		Servers: []config.Server{
			{
				Name:     "server1",
				Hostname: "host1.example.com",
				Port:     22,
				Username: "user1",
				AuthType: "password",
			},
		},
	}

	// Create keyring manager for testing
	manager := NewKeyringManager("auto")
	if manager == nil || !manager.IsAvailable() {
		t.Skip("Keyring not available for migration testing")
	}

	// Run migration with prompt function that returns error
	_, err := MigrateFromPlaintext(cfg, manager, func(prompt string) (string, error) {
		return "", fmt.Errorf("user cancelled")
	})

	if err == nil {
		t.Error("Expected migration to fail when prompt function returns error")
	}

	// Verify server config was not modified
	server := cfg.Servers[0]
	if server.UseKeyring {
		t.Error("Server should not be configured to use keyring after failed migration")
	}
}

// TestIdentifyCredentialsToMigrate tests identification of credentials that need migration
func TestIdentifyCredentialsToMigrate(t *testing.T) {
	tests := []struct {
		name                string
		server              config.Server
		expectNeedsMigration bool
		expectedType        string
	}{
		{
			name: "password authentication",
			server: config.Server{
				Name:     "server1",
				AuthType: "password",
			},
			expectNeedsMigration: true,
			expectedType:        "password",
		},
		{
			name: "key with passphrase",
			server: config.Server{
				Name:                "server2",
				AuthType:            "key",
				PassphraseProtected: true,
			},
			expectNeedsMigration: true,
			expectedType:        "passphrase",
		},
		{
			name: "key without passphrase",
			server: config.Server{
				Name:                "server3",
				AuthType:            "key",
				PassphraseProtected: false,
			},
			expectNeedsMigration: false,
			expectedType:        "",
		},
		{
			name: "already using keyring",
			server: config.Server{
				Name:       "server4",
				AuthType:   "password",
				UseKeyring: true,
				KeyringID:  "server4_password",
			},
			expectNeedsMigration: false,
			expectedType:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsMigration, credType := identifyCredentialToMigrate(tt.server)
			
			if needsMigration != tt.expectNeedsMigration {
				t.Errorf("Expected needsMigration=%v, got %v", tt.expectNeedsMigration, needsMigration)
			}
			
			if credType != tt.expectedType {
				t.Errorf("Expected credType=%q, got %q", tt.expectedType, credType)
			}
		})
	}
}