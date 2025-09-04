package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"sshm/internal/config"
	"sshm/internal/keyring"
)

// TestKeyringMigrationWithMixedAuthTypes tests migration with various authentication types
func TestKeyringMigrationWithMixedAuthTypes(t *testing.T) {
	// Create a temporary config directory
	tempDir := t.TempDir()
	originalConfigDir := os.Getenv("SSHM_CONFIG_DIR")
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer func() {
		if originalConfigDir == "" {
			os.Unsetenv("SSHM_CONFIG_DIR")
		} else {
			os.Setenv("SSHM_CONFIG_DIR", originalConfigDir)
		}
	}()

	// Create comprehensive test config with mixed authentication types
	cfg := &config.Config{
		Servers: []config.Server{
			{
				Name:                "key-with-passphrase",
				Hostname:            "keypass.example.com",
				Port:                22,
				Username:            "user1",
				AuthType:            "key",
				KeyPath:             "~/.ssh/passphrase_key",
				PassphraseProtected: true, // Needs migration
			},
			{
				Name:     "password-auth",
				Hostname: "password.example.com",
				Port:     22,
				Username: "user2",
				AuthType: "password", // Needs migration
			},
			{
				Name:                "key-no-passphrase",
				Hostname:            "keyonly.example.com",
				Port:                22,
				Username:            "user3",
				AuthType:            "key",
				KeyPath:             "~/.ssh/plain_key",
				PassphraseProtected: false, // No migration needed
			},
			{
				Name:       "already-migrated-password",
				Hostname:   "migrated.example.com",
				Port:       22,
				Username:   "user4",
				AuthType:   "password",
				UseKeyring: true,
				KeyringID:  "already-migrated-password_password", // Already migrated
			},
			{
				Name:                "key-with-passphrase-2",
				Hostname:            "keypass2.example.com",
				Port:                2222,
				Username:            "deploy",
				AuthType:            "key",
				KeyPath:             "~/.ssh/deploy_key",
				PassphraseProtected: true, // Needs migration
			},
		},
		Keyring: config.KeyringConfig{
			Enabled:   true,
			Service:   "auto",
			Namespace: "sshm-comprehensive-test",
		},
	}

	// Save test config
	configPath := filepath.Join(tempDir, "config.yaml")
	err := cfg.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create keyring manager
	manager := keyring.NewKeyringManagerWithNamespace("auto", "sshm-comprehensive-test")
	if manager == nil || !manager.IsAvailable() {
		t.Skip("Keyring not available for comprehensive migration testing")
	}

	// Clean up any existing test credentials
	defer func() {
		_ = manager.Delete("key-with-passphrase_passphrase")
		_ = manager.Delete("password-auth_password")
		_ = manager.Delete("key-with-passphrase-2_passphrase")
	}()

	// Test complete migration of all servers that need it
	t.Run("CompleteMigrationWithMixedTypes", func(t *testing.T) {
		// Create fresh config
		testCfg := &config.Config{
			Servers: make([]config.Server, len(cfg.Servers)),
			Keyring: cfg.Keyring,
		}
		copy(testCfg.Servers, cfg.Servers)

		err := testCfg.SaveToPath(configPath)
		if err != nil {
			t.Fatalf("Failed to save test config: %v", err)
		}

		// Create migration config without filtering (migrate all that need it)
		migrationCfg := &config.Config{
			Servers: testCfg.Servers,
			Keyring: testCfg.Keyring,
		}

		// Provide credentials for all servers that need migration
		promptFunc := func(prompt string) (string, error) {
			switch {
			case prompt == "Enter passphrase for key-with-passphrase SSH key:":
				return "passphrase1", nil
			case prompt == "Enter password for password-auth:":
				return "password123", nil
			case prompt == "Enter passphrase for key-with-passphrase-2 SSH key:":
				return "deploykey456", nil
			default:
				return "", fmt.Errorf("unexpected prompt: %s", prompt)
			}
		}

		// Run migration
		results, err := keyring.MigrateFromPlaintext(migrationCfg, manager, promptFunc)
		if err != nil {
			t.Fatalf("Migration failed: %v", err)
		}

		// Should migrate exactly 3 servers (2 passphrase-protected keys + 1 password)
		expectedMigrations := 3
		if len(results) != expectedMigrations {
			t.Errorf("Expected %d migration results, got %d", expectedMigrations, len(results))
		}

		// Apply migration results to original config (simulating the fix)
		updatedServers := 0
		for _, result := range results {
			if result.Success {
				for i := range testCfg.Servers {
					if testCfg.Servers[i].Name == result.ServerName {
						testCfg.Servers[i].UseKeyring = true
						testCfg.Servers[i].KeyringID = result.KeyringID
						updatedServers++
						break
					}
				}
			}
		}

		if updatedServers != expectedMigrations {
			t.Errorf("Expected to update %d servers, actually updated %d", expectedMigrations, updatedServers)
		}

		// Save and reload config
		err = testCfg.SaveToPath(configPath)
		if err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		savedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		// Verify all 5 original servers are preserved
		if len(savedCfg.Servers) != 5 {
			t.Errorf("Expected 5 servers preserved, got %d", len(savedCfg.Servers))
		}

		// Verify specific server states
		serverStates := map[string]struct {
			ShouldUseKeyring  bool
			ExpectedKeyringID string
		}{
			"key-with-passphrase":       {true, "key-with-passphrase_passphrase"},
			"password-auth":             {true, "password-auth_password"},
			"key-no-passphrase":         {false, ""},
			"already-migrated-password": {true, "already-migrated-password_password"},
			"key-with-passphrase-2":     {true, "key-with-passphrase-2_passphrase"},
		}

		for _, server := range savedCfg.Servers {
			expected, found := serverStates[server.Name]
			if !found {
				t.Errorf("Unexpected server found: %s", server.Name)
				continue
			}

			if server.UseKeyring != expected.ShouldUseKeyring {
				t.Errorf("Server %s UseKeyring = %v, expected %v",
					server.Name, server.UseKeyring, expected.ShouldUseKeyring)
			}

			if server.KeyringID != expected.ExpectedKeyringID {
				t.Errorf("Server %s KeyringID = %q, expected %q",
					server.Name, server.KeyringID, expected.ExpectedKeyringID)
			}
		}

		// Verify credentials are stored in keyring
		expectedCredentials := []string{
			"key-with-passphrase_passphrase",
			"password-auth_password",
			"key-with-passphrase-2_passphrase",
		}

		for _, keyringID := range expectedCredentials {
			_, err := manager.Retrieve(keyringID)
			if err != nil {
				t.Errorf("Failed to retrieve %s from keyring: %v", keyringID, err)
			}
		}

		t.Logf("✅ Comprehensive migration test passed: %d servers migrated, all %d servers preserved",
			expectedMigrations, len(savedCfg.Servers))
	})

	// Test partial migration (specific server) preserves all servers
	t.Run("PartialMigrationPreservesAll", func(t *testing.T) {
		// Reset config
		err := cfg.SaveToPath(configPath)
		if err != nil {
			t.Fatalf("Failed to reset config: %v", err)
		}

		testCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Create migration config with just one server
		migrationCfg := &config.Config{
			Servers: testCfg.Servers,
			Keyring: testCfg.Keyring,
		}

		// Filter for just the password server
		server, err := migrationCfg.GetServer("password-auth")
		if err != nil {
			t.Fatalf("Server not found: %v", err)
		}
		migrationCfg.Servers = []config.Server{*server}

		// Migrate just this one server
		promptFunc := func(prompt string) (string, error) {
			if prompt == "Enter password for password-auth:" {
				return "testpass123", nil
			}
			return "", fmt.Errorf("unexpected prompt: %s", prompt)
		}

		results, err := keyring.MigrateFromPlaintext(migrationCfg, manager, promptFunc)
		if err != nil {
			t.Fatalf("Partial migration failed: %v", err)
		}

		// Should migrate exactly 1 server
		if len(results) != 1 {
			t.Errorf("Expected 1 migration result, got %d", len(results))
		}

		// Apply to original config
		for _, result := range results {
			if result.Success {
				for i := range testCfg.Servers {
					if testCfg.Servers[i].Name == result.ServerName {
						testCfg.Servers[i].UseKeyring = true
						testCfg.Servers[i].KeyringID = result.KeyringID
						break
					}
				}
			}
		}

		// Save and verify
		err = testCfg.Save()
		if err != nil {
			t.Fatalf("Failed to save after partial migration: %v", err)
		}

		savedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		// Verify all 5 servers still exist
		if len(savedCfg.Servers) != 5 {
			t.Errorf("Partial migration should preserve all 5 servers, got %d", len(savedCfg.Servers))
		}

		// Verify only password-auth was migrated
		for _, server := range savedCfg.Servers {
			if server.Name == "password-auth" {
				if !server.UseKeyring {
					t.Error("password-auth should use keyring after migration")
				}
				if server.KeyringID != "password-auth_password" {
					t.Errorf("password-auth keyring ID = %q, expected %q",
						server.KeyringID, "password-auth_password")
				}
			} else if server.Name != "already-migrated-password" {
				// All others should not be using keyring (except the already migrated one)
				if server.UseKeyring {
					t.Errorf("server %s should not use keyring (was not migrated)", server.Name)
				}
			}
		}

		t.Logf("✅ Partial migration test passed: 1 server migrated, all %d servers preserved", len(savedCfg.Servers))
	})
}
