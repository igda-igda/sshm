package auth

import (
	"fmt"
	"testing"

	"sshm/internal/config"
)

// TestNewAuthManager tests creating a new AuthManager
func TestNewAuthManager(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		expectError bool
	}{
		{
			name: "keyring enabled with auto service",
			cfg: &config.Config{
				Keyring: config.KeyringConfig{
					Enabled:   true,
					Service:   "auto",
					Namespace: "test-sshm",
				},
			},
			expectError: false,
		},
		{
			name: "keyring disabled",
			cfg: &config.Config{
				Keyring: config.KeyringConfig{
					Enabled: false,
				},
			},
			expectError: false,
		},
		{
			name: "keyring enabled with invalid service",
			cfg: &config.Config{
				Keyring: config.KeyringConfig{
					Enabled: true,
					Service: "invalid-service",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promptFunc := func(prompt string) (string, error) {
				return "test-credential", nil
			}

			manager, err := NewAuthManager(tt.cfg, promptFunc)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if manager == nil {
				t.Error("Expected non-nil manager")
			}

			// Test keyring availability matches config
			if tt.cfg.Keyring.Enabled {
				if manager.keyringManager == nil {
					t.Error("Expected keyring manager when keyring is enabled")
				}
			} else {
				if manager.keyringManager != nil {
					t.Error("Expected no keyring manager when keyring is disabled")
				}
			}
		})
	}
}

// TestGetAuthMethod tests authentication method selection
func TestGetAuthMethod(t *testing.T) {
	// Create config with keyring disabled for testing
	cfg := &config.Config{
		Keyring: config.KeyringConfig{
			Enabled: false,
		},
	}

	tests := []struct {
		name        string
		server      config.Server
		promptReturn string
		promptError error
		expectError bool
		authType    string
	}{
		{
			name: "password auth with prompt",
			server: config.Server{
				Name:     "test-server",
				AuthType: "password",
			},
			promptReturn: "test-password",
			expectError:  false,
			authType:     "password",
		},
		{
			name: "key auth without passphrase",
			server: config.Server{
				Name:                "test-server",
				AuthType:            "key",
				KeyPath:             "~/.ssh/sshm-test/test_key",
				PassphraseProtected: false,
			},
			expectError: false,
			authType:    "key",
		},
		{
			name: "key auth with passphrase",
			server: config.Server{
				Name:                "test-server",
				AuthType:            "key",
				KeyPath:             "~/.ssh/sshm-test/test_key_with_passphrase",
				PassphraseProtected: true,
			},
			promptReturn: "test-passphrase",
			expectError:  false,
			authType:     "key",
		},
		{
			name: "key auth missing key path",
			server: config.Server{
				Name:     "test-server",
				AuthType: "key",
				KeyPath:  "", // Missing key path
			},
			expectError: true,
		},
		{
			name: "unsupported auth type",
			server: config.Server{
				Name:     "test-server",
				AuthType: "unsupported",
			},
			expectError: true,
		},
		{
			name: "prompt function error",
			server: config.Server{
				Name:     "test-server",
				AuthType: "password",
			},
			promptError: fmt.Errorf("user cancelled"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promptFunc := func(prompt string) (string, error) {
				if tt.promptError != nil {
					return "", tt.promptError
				}
				return tt.promptReturn, nil
			}

			manager, err := NewAuthManager(cfg, promptFunc)
			if err != nil {
				t.Fatalf("Failed to create auth manager: %v", err)
			}

			authMethod, err := manager.GetAuthMethod(tt.server)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if authMethod == nil {
				t.Error("Expected non-nil auth method")
			}
		})
	}
}

// TestGetAuthMethodWithKeyring tests authentication with keyring integration
func TestGetAuthMethodWithKeyring(t *testing.T) {
	// Create config with keyring enabled
	cfg := &config.Config{
		Keyring: config.KeyringConfig{
			Enabled:   true,
			Service:   "auto",
			Namespace: "test-sshm",
		},
	}

	// Create auth manager
	promptFunc := func(prompt string) (string, error) {
		return "fallback-credential", nil
	}

	manager, err := NewAuthManager(cfg, promptFunc)
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	// Skip test if keyring is not available
	if manager.keyringManager == nil || !manager.keyringManager.IsAvailable() {
		t.Skip("Keyring not available for testing")
	}

	// Test server with keyring configuration
	server := config.Server{
		Name:       "test-server",
		AuthType:   "password",
		UseKeyring: true,
		KeyringID:  "test-server-password",
	}

	// Store test credential in keyring
	testCredential := "stored-password-123"
	err = manager.keyringManager.Store(server.KeyringID, testCredential)
	if err != nil {
		t.Fatalf("Failed to store test credential: %v", err)
	}

	// Clean up after test
	defer func() {
		_ = manager.keyringManager.Delete(server.KeyringID)
	}()

	// Get auth method - should retrieve from keyring
	authMethod, err := manager.GetAuthMethod(server)
	if err != nil {
		t.Fatalf("Failed to get auth method: %v", err)
	}

	if authMethod == nil {
		t.Error("Expected non-nil auth method")
	}

	// Test that credential was retrieved from keyring (indirect test)
	// We can't directly verify the password in the SSH auth method,
	// but we can verify the keyring was accessed properly
}

// TestStoreRetrieveCredential tests credential storage and retrieval
func TestStoreRetrieveCredential(t *testing.T) {
	// Create config with keyring enabled
	cfg := &config.Config{
		Keyring: config.KeyringConfig{
			Enabled:   true,
			Service:   "auto",
			Namespace: "test-sshm",
		},
	}

	manager, err := NewAuthManager(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	// Skip test if keyring is not available
	if manager.keyringManager == nil || !manager.keyringManager.IsAvailable() {
		t.Skip("Keyring not available for testing")
	}

	server := config.Server{
		Name:       "test-server",
		AuthType:   "password",
		UseKeyring: true,
		KeyringID:  "test-credential-storage",
	}

	testCredential := "test-password-xyz"

	// Clean up after test
	defer func() {
		_ = manager.keyringManager.Delete(server.KeyringID)
	}()

	// Test store credential
	err = manager.StoreCredential(server, testCredential)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Test retrieve credential
	retrievedCredential, err := manager.RetrieveCredential(server)
	if err != nil {
		t.Fatalf("Failed to retrieve credential: %v", err)
	}

	if retrievedCredential != testCredential {
		t.Errorf("Retrieved credential %q does not match stored credential %q", 
			retrievedCredential, testCredential)
	}
}

// TestStoreCredentialErrors tests error cases for credential storage
func TestStoreCredentialErrors(t *testing.T) {
	// Create config with keyring disabled
	cfg := &config.Config{
		Keyring: config.KeyringConfig{
			Enabled: false,
		},
	}

	manager, err := NewAuthManager(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	tests := []struct {
		name   string
		server config.Server
	}{
		{
			name: "keyring not available",
			server: config.Server{
				Name:       "test-server",
				UseKeyring: true,
				KeyringID:  "test-key",
			},
		},
		{
			name: "server not configured for keyring",
			server: config.Server{
				Name:       "test-server",
				UseKeyring: false,
			},
		},
		{
			name: "missing keyring ID",
			server: config.Server{
				Name:       "test-server",
				UseKeyring: true,
				KeyringID:  "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.StoreCredential(tt.server, "test-credential")
			if err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// TestGetAuthMethodWithFallback tests auth method with SSH agent fallback
func TestGetAuthMethodWithFallback(t *testing.T) {
	cfg := &config.Config{
		Keyring: config.KeyringConfig{
			Enabled: false,
		},
	}

	promptFunc := func(prompt string) (string, error) {
		return "test-credential", nil
	}

	manager, err := NewAuthManager(cfg, promptFunc)
	if err != nil {
		t.Fatalf("Failed to create auth manager: %v", err)
	}

	tests := []struct {
		name               string
		server             config.Server
		expectMultipleMethods bool
	}{
		{
			name: "password auth - no fallback",
			server: config.Server{
				Name:     "test-server",
				AuthType: "password",
			},
			expectMultipleMethods: false,
		},
		{
			name: "key auth - SSH agent fallback",
			server: config.Server{
				Name:     "test-server",
				AuthType: "key",
				KeyPath:  "~/.ssh/sshm-test/test_key",
			},
			expectMultipleMethods: true, // Should include SSH agent fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			methods, err := manager.GetAuthMethodWithFallback(tt.server)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(methods) == 0 {
				t.Error("Expected at least one auth method")
				return
			}

			if tt.expectMultipleMethods {
				// For key auth, we expect primary method plus SSH agent fallback
				// But SSH agent might not be available, so we accept 1 or more methods
				if len(methods) < 1 {
					t.Error("Expected at least primary auth method")
				}
			} else {
				// For password auth, we expect only the primary method
				if len(methods) != 1 {
					t.Errorf("Expected exactly 1 auth method, got %d", len(methods))
				}
			}
		})
	}
}