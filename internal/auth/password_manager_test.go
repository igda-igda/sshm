package auth

import (
	"testing"
	"sshm/internal/config"
	"sshm/internal/keyring"
)

func TestNewPasswordManager(t *testing.T) {
	tests := []struct {
		name        string
		service     string
		wantError   bool
		description string
	}{
		{
			name:        "file backend",
			service:     "file",
			wantError:   false,
			description: "should create password manager with file backend",
		},
		{
			name:        "auto backend",
			service:     "auto",
			wantError:   false,
			description: "should create password manager with auto backend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, err := NewPasswordManager(tt.service)
			
			if tt.wantError {
				if err == nil {
					t.Errorf("NewPasswordManager() should return error")
				}
				return
			}

			if err != nil {
				t.Errorf("NewPasswordManager() error = %v", err)
				return
			}

			if pm == nil {
				t.Errorf("NewPasswordManager() returned nil")
			}

			if !pm.IsAvailable() {
				t.Errorf("Password manager should be available")
			}
		})
	}
}

func TestPasswordManager_StoreServerPassword(t *testing.T) {
	pm, err := NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	tests := []struct {
		name        string
		server      *config.Server
		password    string
		wantError   bool
		description string
	}{
		{
			name: "store password for password auth server",
			server: &config.Server{
				Name:     "test-server-1",
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				AuthType: "password",
			},
			password:    "secret123",
			wantError:   false,
			description: "should store password for password-authenticated server",
		},
		{
			name: "attempt store on key auth server",
			server: &config.Server{
				Name:     "test-server-2",
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				AuthType: "key",
				KeyPath:  "~/.ssh/id_rsa",
			},
			password:    "secret123",
			wantError:   true,
			description: "should fail for key-authenticated server",
		},
		{
			name:        "nil server",
			server:      nil,
			password:    "secret123",
			wantError:   true,
			description: "should fail for nil server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pm.StoreServerPassword(tt.server, tt.password)
			
			if (err != nil) != tt.wantError {
				t.Errorf("StoreServerPassword() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.server != nil {
				// Verify server configuration was updated
				if !tt.server.UseKeyring {
					t.Errorf("Server should be configured to use keyring")
				}
				if tt.server.KeyringID == "" {
					t.Errorf("Server should have keyring ID")
				}
				if tt.server.Password != "" {
					t.Errorf("Server plaintext password should be cleared")
				}

				// Clean up
				pm.DeleteServerPassword(tt.server)
			}
		})
	}
}

func TestPasswordManager_RetrieveServerPassword(t *testing.T) {
	pm, err := NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	// Create test server
	server := &config.Server{
		Name:     "test-retrieve-server",
		Hostname: "example.com",
		Port:     22,
		Username: "user",
		AuthType: "password",
	}

	password := "retrieve-test-password"

	// Store password first
	err = pm.StoreServerPassword(server, password)
	if err != nil {
		t.Fatalf("failed to store password: %v", err)
	}
	defer pm.DeleteServerPassword(server)

	tests := []struct {
		name        string
		server      *config.Server
		expected    string
		wantError   bool
		description string
	}{
		{
			name:        "retrieve stored password",
			server:      server,
			expected:    password,
			wantError:   false,
			description: "should retrieve previously stored password",
		},
		{
			name: "retrieve from key auth server",
			server: &config.Server{
				Name:     "key-server",
				AuthType: "key",
			},
			wantError:   true,
			description: "should fail for key-authenticated server",
		},
		{
			name:        "nil server",
			server:      nil,
			wantError:   true,
			description: "should fail for nil server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := pm.RetrieveServerPassword(tt.server)
			
			if (err != nil) != tt.wantError {
				t.Errorf("RetrieveServerPassword() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && retrieved != tt.expected {
				t.Errorf("RetrieveServerPassword() = %v, want %v", retrieved, tt.expected)
			}
		})
	}
}

func TestPasswordManager_HasServerPassword(t *testing.T) {
	pm, err := NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	// Create test servers
	serverWithPassword := &config.Server{
		Name:     "server-with-password",
		AuthType: "password",
	}
	serverWithoutPassword := &config.Server{
		Name:     "server-without-password",
		AuthType: "password",
	}
	keyAuthServer := &config.Server{
		Name:     "key-auth-server",
		AuthType: "key",
	}

	// Store password for one server
	err = pm.StoreServerPassword(serverWithPassword, "test-password")
	if err != nil {
		t.Fatalf("failed to store password: %v", err)
	}
	defer pm.DeleteServerPassword(serverWithPassword)

	tests := []struct {
		name        string
		server      *config.Server
		expected    bool
		description string
	}{
		{
			name:        "server with stored password",
			server:      serverWithPassword,
			expected:    true,
			description: "should return true for server with stored password",
		},
		{
			name:        "server without password",
			server:      serverWithoutPassword,
			expected:    false,
			description: "should return false for server without password",
		},
		{
			name:        "key auth server",
			server:      keyAuthServer,
			expected:    false,
			description: "should return false for key-authenticated server",
		},
		{
			name:        "nil server",
			server:      nil,
			expected:    false,
			description: "should return false for nil server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pm.HasServerPassword(tt.server)
			if result != tt.expected {
				t.Errorf("HasServerPassword() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPasswordManager_UpdateServerPassword(t *testing.T) {
	pm, err := NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	server := &config.Server{
		Name:     "update-test-server",
		AuthType: "password",
	}

	// Store initial password
	initialPassword := "initial-password"
	err = pm.StoreServerPassword(server, initialPassword)
	if err != nil {
		t.Fatalf("failed to store initial password: %v", err)
	}
	defer pm.DeleteServerPassword(server)

	// Verify initial password
	retrieved, err := pm.RetrieveServerPassword(server)
	if err != nil {
		t.Errorf("failed to retrieve initial password: %v", err)
	}
	if retrieved != initialPassword {
		t.Errorf("initial password = %v, want %v", retrieved, initialPassword)
	}

	// Update password
	updatedPassword := "updated-password"
	err = pm.UpdateServerPassword(server, updatedPassword)
	if err != nil {
		t.Errorf("UpdateServerPassword() error = %v", err)
	}

	// Verify updated password
	retrieved, err = pm.RetrieveServerPassword(server)
	if err != nil {
		t.Errorf("failed to retrieve updated password: %v", err)
	}
	if retrieved != updatedPassword {
		t.Errorf("updated password = %v, want %v", retrieved, updatedPassword)
	}
}

func TestPasswordManager_DeleteServerPassword(t *testing.T) {
	pm, err := NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	server := &config.Server{
		Name:     "delete-test-server",
		AuthType: "password",
	}

	// Store password
	err = pm.StoreServerPassword(server, "password-to-delete")
	if err != nil {
		t.Fatalf("failed to store password: %v", err)
	}

	// Verify password exists
	if !pm.HasServerPassword(server) {
		t.Errorf("Password should exist before deletion")
	}

	// Delete password
	err = pm.DeleteServerPassword(server)
	if err != nil {
		t.Errorf("DeleteServerPassword() error = %v", err)
	}

	// Verify password is deleted
	if pm.HasServerPassword(server) {
		t.Errorf("Password should not exist after deletion")
	}

	// Verify server configuration is cleared
	if server.UseKeyring {
		t.Errorf("Server should not use keyring after deletion")
	}
	if server.KeyringID != "" {
		t.Errorf("Server keyring ID should be cleared")
	}
	if server.Password != "" {
		t.Errorf("Server password should be cleared")
	}
}

func TestPasswordManager_MigrateServerToKeyring(t *testing.T) {
	pm, err := NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	tests := []struct {
		name        string
		server      *config.Server
		wantError   bool
		description string
	}{
		{
			name: "migrate password auth server with plaintext password",
			server: &config.Server{
				Name:     "migrate-server-1",
				AuthType: "password",
				Password: "plaintext-password",
			},
			wantError:   false,
			description: "should migrate server with plaintext password",
		},
		{
			name: "migrate password auth server already using keyring",
			server: &config.Server{
				Name:       "migrate-server-2",
				AuthType:   "password",
				UseKeyring: true,
				KeyringID:  keyring.GeneratePasswordKeyringID("migrate-server-2"),
			},
			wantError:   false,
			description: "should skip server already using keyring",
		},
		{
			name: "migrate password auth server with no password",
			server: &config.Server{
				Name:     "migrate-server-3",
				AuthType: "password",
				Password: "",
			},
			wantError:   true,
			description: "should fail for server with no password",
		},
		{
			name: "migrate key auth server",
			server: &config.Server{
				Name:     "migrate-server-4",
				AuthType: "key",
				KeyPath:  "~/.ssh/id_rsa",
			},
			wantError:   true,
			description: "should fail for key-authenticated server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store password for keyring servers
			if tt.server.UseKeyring && !tt.wantError {
				pm.GetKeyringManager().StoreServerPassword(tt.server.Name, "stored-password")
			}
			
			originalPassword := tt.server.Password
			err := pm.MigrateServerToKeyring(tt.server)
			
			if (err != nil) != tt.wantError {
				t.Errorf("MigrateServerToKeyring() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && tt.server.AuthType == "password" {
				// Verify server configuration
				if !tt.server.UseKeyring {
					t.Errorf("Server should use keyring after migration")
				}
				if tt.server.KeyringID == "" {
					t.Errorf("Server should have keyring ID after migration")
				}
				if tt.server.Password != "" {
					t.Errorf("Server plaintext password should be cleared")
				}

				// Verify password is accessible
				if originalPassword != "" {
					retrieved, err := pm.RetrieveServerPassword(tt.server)
					if err != nil {
						t.Errorf("failed to retrieve password after migration: %v", err)
					}
					if retrieved != originalPassword {
						t.Errorf("migrated password = %v, want %v", retrieved, originalPassword)
					}
				}

				// Clean up
				pm.DeleteServerPassword(tt.server)
			}
		})
	}
}

func TestPasswordManager_LegacyPasswordSupport(t *testing.T) {
	pm, err := NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	// Create server with legacy plaintext password
	server := &config.Server{
		Name:     "legacy-server",
		AuthType: "password",
		Password: "legacy-plaintext-password",
	}

	// Should be able to retrieve legacy password
	password, err := pm.RetrieveServerPassword(server)
	if err != nil {
		t.Errorf("failed to retrieve legacy password: %v", err)
	}
	if password != "legacy-plaintext-password" {
		t.Errorf("legacy password = %v, want %v", password, "legacy-plaintext-password")
	}

	// Should detect that password exists
	if !pm.HasServerPassword(server) {
		t.Errorf("should detect legacy password exists")
	}
}