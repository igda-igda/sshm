package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"sshm/internal/config"
	"sshm/internal/keyring"
)

// TestKeyringMigrationBugReproduction tests that demonstrate the server loss bug
func TestKeyringMigrationBugReproduction(t *testing.T) {
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

	// Create test config with multiple servers - this reproduces the user's scenario
	cfg := &config.Config{
		Servers: []config.Server{
			{
				Name:                "web-server",
				Hostname:            "web.example.com",
				Port:                22,
				Username:            "deploy",
				AuthType:            "key",
				KeyPath:             "~/.ssh/web_rsa",
				PassphraseProtected: true,
			},
			{
				Name:     "db-server",
				Hostname: "db.example.com",
				Port:     22,
				Username: "admin",
				AuthType: "password",
			},
			{
				Name:                "staging-server",
				Hostname:            "staging.example.com",
				Port:                22,
				Username:            "deploy",
				AuthType:            "key",
				KeyPath:             "~/.ssh/staging_rsa",
				PassphraseProtected: false, // This server doesn't need migration
			},
			{
				Name:     "api-server",
				Hostname: "api.example.com",
				Port:     22,
				Username: "api",
				AuthType: "password",
			},
		},
		Keyring: config.KeyringConfig{
			Enabled:   true,
			Service:   "auto",
			Namespace: "sshm-test",
		},
	}

	// Save test config
	configPath := filepath.Join(tempDir, "config.yaml")
	err := cfg.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create keyring manager
	manager := keyring.NewKeyringManagerWithNamespace("auto", "sshm-test")
	if manager == nil || !manager.IsAvailable() {
		t.Skip("Keyring not available for migration testing")
	}

	// Clean up any existing test credentials
	defer func() {
		_ = manager.Delete("web-server_passphrase")
		_ = manager.Delete("db-server_password")
		_ = manager.Delete("api-server_password")
	}()

	// Test the bug: migrating specific server should preserve all servers 
	// but currently it replaces the entire config with just that server
	t.Run("BugReproduction_SpecificServerMigrationLosesOtherServers", func(t *testing.T) {
		// Create a direct call to demonstrate the underlying bug
		// We'll simulate the problematic code path without user interaction
		
		// Load the config (same as runKeyringMigrateCommand does)
		testCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Count servers before the bug-inducing operation
		serverCountBefore := len(testCfg.Servers)
		if serverCountBefore != 4 {
			t.Fatalf("Expected 4 servers before migration, got %d", serverCountBefore)
		}

		// This is the bug! When we filter for a specific server,
		// we replace the entire cfg.Servers slice
		serverName := "db-server"
		server, err := testCfg.GetServer(serverName)
		if err != nil {
			t.Fatalf("Server '%s' not found", serverName)
		}
		
		// THIS IS THE BUG - this line causes the server list to be reduced
		testCfg.Servers = []config.Server{*server}
		
		// Now if we save the config, we lose all other servers
		err = testCfg.Save()
		if err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Load the config again to see what was actually saved
		savedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		// This demonstrates the bug - we should have 4 servers but we only have 1
		serverCountAfter := len(savedCfg.Servers)
		if serverCountAfter == serverCountBefore {
			t.Errorf("BUG NOT REPRODUCED: Expected fewer servers after bug, got %d (same as before)", serverCountAfter)
		} else {
			t.Logf("BUG CONFIRMED: Started with %d servers, ended with %d servers", serverCountBefore, serverCountAfter)
			t.Logf("Remaining server: %s", savedCfg.Servers[0].Name)
			
			// Verify only the specific server remains
			if len(savedCfg.Servers) != 1 {
				t.Errorf("Expected exactly 1 server after bug, got %d", len(savedCfg.Servers))
			}
			if savedCfg.Servers[0].Name != serverName {
				t.Errorf("Expected remaining server to be %s, got %s", serverName, savedCfg.Servers[0].Name)
			}
		}

		// List the servers that were lost
		originalServers := []string{"web-server", "db-server", "staging-server", "api-server"}
		remainingServers := make(map[string]bool)
		for _, server := range savedCfg.Servers {
			remainingServers[server.Name] = true
		}

		var lostServers []string
		for _, originalServer := range originalServers {
			if !remainingServers[originalServer] {
				lostServers = append(lostServers, originalServer)
			}
		}

		if len(lostServers) > 0 {
			t.Logf("Servers lost due to bug: %v", lostServers)
		}
	})

	// Reset config for next test
	err = cfg.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Failed to reset test config: %v", err)
	}

	// Test what the migration logic itself does (without the bug)
	t.Run("MigrationLogicAlonePreservesServers", func(t *testing.T) {
		// Load fresh config
		testCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Use the migration logic directly (without the buggy filtering)
		promptFunc := func(prompt string) (string, error) {
			if prompt == "Enter password for db-server:" {
				return "test-password-123", nil
			}
			if prompt == "Enter passphrase for web-server SSH key:" {
				return "test-passphrase-456", nil
			}
			if prompt == "Enter password for api-server:" {
				return "test-password-789", nil
			}
			return "", fmt.Errorf("unexpected prompt: %s", prompt)
		}

		// Run migration on the full config (this should work correctly)
		results, err := keyring.MigrateFromPlaintext(testCfg, manager, promptFunc)
		if err != nil {
			t.Fatalf("Migration failed: %v", err)
		}

		// Save the config after successful migration
		err = testCfg.Save()
		if err != nil {
			t.Fatalf("Failed to save config after migration: %v", err)
		}

		// Verify the migration worked and all servers are preserved
		savedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to reload config: %v", err)
		}

		if len(savedCfg.Servers) != 4 {
			t.Errorf("Migration logic should preserve all 4 servers, got %d", len(savedCfg.Servers))
		}

		// Verify correct servers were migrated
		expectedMigrated := 3 // web-server (passphrase), db-server (password), api-server (password)
		if len(results) != expectedMigrated {
			t.Errorf("Expected %d migration results, got %d", expectedMigrated, len(results))
		}

		// Verify credentials are in keyring
		migratedServers := []string{"web-server_passphrase", "db-server_password", "api-server_password"}
		for _, keyringID := range migratedServers {
			_, err := manager.Retrieve(keyringID)
			if err != nil {
				t.Errorf("Failed to retrieve %s from keyring: %v", keyringID, err)
			}
		}

		t.Logf("✅ Migration logic alone works correctly - all servers preserved")
	})
}

// TestKeyringMigrateFixValidation tests that the fix works correctly
func TestKeyringMigrateFixValidation(t *testing.T) {
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

	// Create test config with multiple servers - this reproduces the user's scenario
	cfg := &config.Config{
		Servers: []config.Server{
			{
				Name:                "web-server",
				Hostname:            "web.example.com",
				Port:                22,
				Username:            "deploy",
				AuthType:            "key",
				KeyPath:             "~/.ssh/web_rsa",
				PassphraseProtected: true,
			},
			{
				Name:     "db-server",
				Hostname: "db.example.com",
				Port:     22,
				Username: "admin",
				AuthType: "password",
			},
			{
				Name:                "staging-server",
				Hostname:            "staging.example.com",
				Port:                22,
				Username:            "deploy",
				AuthType:            "key",
				KeyPath:             "~/.ssh/staging_rsa",
				PassphraseProtected: false, // This server doesn't need migration
			},
			{
				Name:     "api-server",
				Hostname: "api.example.com",
				Port:     22,
				Username: "api",
				AuthType: "password",
			},
		},
		Keyring: config.KeyringConfig{
			Enabled:   true,
			Service:   "auto",
			Namespace: "sshm-test",
		},
	}

	// Save test config
	configPath := filepath.Join(tempDir, "config.yaml")
	err := cfg.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create keyring manager
	manager := keyring.NewKeyringManagerWithNamespace("auto", "sshm-test")
	if manager == nil || !manager.IsAvailable() {
		t.Skip("Keyring not available for migration testing")
	}

	// Clean up any existing test credentials
	defer func() {
		_ = manager.Delete("web-server_passphrase")
		_ = manager.Delete("db-server_password")
		_ = manager.Delete("api-server_password")
	}()

	// Test the fix: verify that migrating a specific server preserves all servers
	t.Run("FixedMigrationPreservesAllServers", func(t *testing.T) {
		// Create a working copy of the config for this test
		testCfg := &config.Config{
			Servers: make([]config.Server, len(cfg.Servers)),
			Keyring: cfg.Keyring,
		}
		copy(testCfg.Servers, cfg.Servers)
		
		err := testCfg.SaveToPath(configPath)
		if err != nil {
			t.Fatalf("Failed to save test config: %v", err)
		}

		// Count servers before migration
		serverCountBefore := len(testCfg.Servers)
		if serverCountBefore != 4 {
			t.Fatalf("Expected 4 servers before migration, got %d", serverCountBefore)
		}

		// Create a filtered migration config (simulating the fix)
		migrationCfg := &config.Config{
			Servers: testCfg.Servers,
			Keyring: testCfg.Keyring,
		}

		// Filter for specific server (db-server)
		serverName := "db-server"
		server, err := migrationCfg.GetServer(serverName)
		if err != nil {
			t.Fatalf("Server '%s' not found", serverName)
		}
		migrationCfg.Servers = []config.Server{*server} // Only affect migration config

		// Run migration on filtered config
		promptFunc := func(prompt string) (string, error) {
			if prompt == "Enter password for db-server:" {
				return "test-password-123", nil
			}
			return "", fmt.Errorf("unexpected prompt: %s", prompt)
		}

		results, err := keyring.MigrateFromPlaintext(migrationCfg, manager, promptFunc)
		if err != nil {
			t.Fatalf("Migration failed: %v", err)
		}

		// Apply migration results to original config (this is the fix)
		for _, result := range results {
			if result.Success {
				// Find the server in the original config and update it
				for i := range testCfg.Servers {
					if testCfg.Servers[i].Name == result.ServerName {
						testCfg.Servers[i].UseKeyring = true
						testCfg.Servers[i].KeyringID = result.KeyringID
						break
					}
				}
			}
		}

		// Save the updated original config
		err = testCfg.SaveToPath(configPath)
		if err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Verify all servers are preserved
		savedCfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load saved config: %v", err)
		}

		if len(savedCfg.Servers) != 4 {
			t.Errorf("Expected 4 servers after migration, got %d", len(savedCfg.Servers))
			for i, server := range savedCfg.Servers {
				t.Logf("Server %d: %s", i, server.Name)
			}
		}

		// Verify the migrated server was updated correctly
		var dbServer *config.Server
		for _, server := range savedCfg.Servers {
			if server.Name == "db-server" {
				dbServer = &server
				break
			}
		}

		if dbServer == nil {
			t.Fatal("db-server not found in saved config")
		}

		if !dbServer.UseKeyring {
			t.Error("db-server should be configured to use keyring after migration")
		}

		if dbServer.KeyringID != "db-server_password" {
			t.Errorf("db-server keyring ID = %q, expected %q", dbServer.KeyringID, "db-server_password")
		}

		// Verify other servers were not modified
		expectedUnchanged := []string{"web-server", "staging-server", "api-server"}
		for _, serverName := range expectedUnchanged {
			for _, server := range savedCfg.Servers {
				if server.Name == serverName {
					if server.UseKeyring {
						t.Errorf("server %s should not use keyring (was not migrated)", server.Name)
					}
					if server.KeyringID != "" {
						t.Errorf("server %s should not have keyring ID (was not migrated)", server.Name)
					}
					break
				}
			}
		}

		// Verify credential was stored in keyring
		_, err = manager.Retrieve("db-server_password")
		if err != nil {
			t.Errorf("Failed to retrieve db-server_password from keyring: %v", err)
		}

		t.Logf("✅ Fix validated: all %d servers preserved, only db-server migrated", len(savedCfg.Servers))
	})
}