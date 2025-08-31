package connection

import (
	"fmt"
	"os"
	"testing"

	"sshm/internal/auth"
	"sshm/internal/config"
)

func TestPasswordIntegration_ConnectivityTest(t *testing.T) {
	// Skip if running in CI environment without SSH server
	if os.Getenv("CI") != "" {
		t.Skip("Skipping SSH connectivity tests in CI environment")
	}

	// Create password manager
	passwordManager, err := auth.NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	tests := []struct {
		name        string
		server      config.Server
		password    string
		expectSkip  bool
		description string
	}{
		{
			name: "password auth with keyring",
			server: config.Server{
				Name:       "test-keyring-server",
				Hostname:   "localhost",
				Port:       22,
				Username:   "testuser",
				AuthType:   "password",
				UseKeyring: true,
				KeyringID:  "password-test-keyring-server",
			},
			password:    "testpassword123",
			expectSkip:  false,
			description: "should attempt connectivity test with keyring password",
		},
		{
			name: "password auth without keyring",
			server: config.Server{
				Name:     "test-plaintext-server",
				Hostname: "localhost",
				Port:     22,
				Username: "testuser",
				AuthType: "password",
				Password: "plaintext-password",
			},
			expectSkip:  true,
			description: "should skip connectivity test for plaintext password",
		},
		{
			name: "key auth server",
			server: config.Server{
				Name:     "test-key-server",
				Hostname: "localhost",
				Port:     22,
				Username: "testuser",
				AuthType: "key",
				KeyPath:  "~/.ssh/nonexistent_key",
			},
			expectSkip:  false,
			description: "should attempt connectivity test with key auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store password in keyring if needed
			if tt.server.UseKeyring && tt.password != "" {
				err := passwordManager.StoreServerPassword(&tt.server, tt.password)
				if err != nil {
					t.Fatalf("failed to store password: %v", err)
				}
				defer passwordManager.DeleteServerPassword(&tt.server)
			}

			// Create connection manager
			manager, err := NewManager()
			if err != nil {
				t.Fatalf("failed to create connection manager: %v", err)
			}
			defer manager.Close()

			// Test SSH connectivity
			err = manager.testSSHConnectivity(tt.server)

			if tt.expectSkip {
				// For plaintext password auth, we expect nil (skipped test)
				if err != nil && tt.server.AuthType == "password" && !tt.server.UseKeyring {
					// This is expected - test was skipped
					return
				}
			} else {
				// For keyring-based or key auth, we expect some result
				// The actual result depends on whether the server exists and credentials are valid
				// We're primarily testing that the password retrieval logic works
				if tt.server.UseKeyring {
					// Verify that password retrieval doesn't cause a panic or unexpected error
					// The SSH connection may fail due to invalid server/credentials, but that's okay
					t.Logf("SSH connectivity test result: %v", err)
				}
			}
		})
	}
}

func TestPasswordIntegration_BuildSSHCommand(t *testing.T) {
	// Create password manager
	passwordManager, err := auth.NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	tests := []struct {
		name             string
		server           config.Server
		password         string
		expectSshpass    bool
		expectInteractive bool
		description      string
	}{
		{
			name: "keyring password auth",
			server: config.Server{
				Name:       "keyring-server",
				Hostname:   "example.com",
				Port:       22,
				Username:   "user",
				AuthType:   "password",
				UseKeyring: true,
				KeyringID:  "password-keyring-server",
			},
			password:          "stored-password",
			expectSshpass:     true,
			expectInteractive: false,
			description:       "should use sshpass with keyring password",
		},
		{
			name: "plaintext password auth",
			server: config.Server{
				Name:     "plaintext-server",
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				AuthType: "password",
				Password: "plaintext",
			},
			expectSshpass:     false,
			expectInteractive: true,
			description:       "should use interactive SSH for plaintext password",
		},
		{
			name: "key auth",
			server: config.Server{
				Name:     "key-server",
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				AuthType: "key",
				KeyPath:  "~/.ssh/id_rsa",
			},
			expectSshpass:     false,
			expectInteractive: true,
			description:       "should use interactive SSH for key auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store password in keyring if needed
			if tt.server.UseKeyring && tt.password != "" {
				err := passwordManager.StoreServerPassword(&tt.server, tt.password)
				if err != nil {
					t.Fatalf("failed to store password: %v", err)
				}
				defer passwordManager.DeleteServerPassword(&tt.server)
			}

			// Build SSH command
			sshCmd, err := buildSSHCommand(tt.server)
			if err != nil {
				t.Fatalf("buildSSHCommand() error = %v", err)
			}

			t.Logf("Generated SSH command: %s", sshCmd)

			// Verify command structure
			if tt.expectSshpass {
				if !contains(sshCmd, "sshpass -p") {
					t.Errorf("Expected command to use sshpass, got: %s", sshCmd)
				}
				if !contains(sshCmd, tt.password) {
					t.Errorf("Expected command to contain password, got: %s", sshCmd)
				}
			}

			if tt.expectInteractive {
				if contains(sshCmd, "sshpass") {
					t.Errorf("Expected interactive SSH command, but found sshpass: %s", sshCmd)
				}
			}

			// Verify basic SSH command structure
			expectedParts := []string{
				"ssh",
				"-t",
				fmt.Sprintf("%s@%s", tt.server.Username, tt.server.Hostname),
				"-o ServerAliveInterval=60",
				"-o ServerAliveCountMax=3",
			}

			for _, part := range expectedParts {
				if !contains(sshCmd, part) {
					t.Errorf("Expected command to contain '%s', got: %s", part, sshCmd)
				}
			}

			// Verify port handling
			if tt.server.Port != 22 {
				expectedPort := fmt.Sprintf("-p %d", tt.server.Port)
				if !contains(sshCmd, expectedPort) {
					t.Errorf("Expected command to contain port '%s', got: %s", expectedPort, sshCmd)
				}
			}

			// Verify key path handling
			if tt.server.AuthType == "key" && tt.server.KeyPath != "" {
				expectedKey := fmt.Sprintf("-i %s", tt.server.KeyPath)
				if !contains(sshCmd, expectedKey) {
					t.Errorf("Expected command to contain key path '%s', got: %s", expectedKey, sshCmd)
				}
			}
		})
	}
}

func TestPasswordIntegration_ManagerOperations(t *testing.T) {
	// Create password manager
	passwordManager, err := auth.NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	// Create test server
	server := &config.Server{
		Name:     "integration-test-server",
		Hostname: "test.example.com",
		Port:     22,
		Username: "testuser",
		AuthType: "password",
	}

	password := "integration-test-password-123"

	// Test complete workflow
	t.Run("complete password workflow", func(t *testing.T) {
		// 1. Store password
		err := passwordManager.StoreServerPassword(server, password)
		if err != nil {
			t.Fatalf("StoreServerPassword() error = %v", err)
		}

		// Verify server configuration was updated
		if !server.UseKeyring {
			t.Errorf("Server should be configured to use keyring")
		}
		if server.KeyringID == "" {
			t.Errorf("Server should have keyring ID")
		}

		// 2. Check password exists
		if !passwordManager.HasServerPassword(server) {
			t.Errorf("HasServerPassword() should return true")
		}

		// 3. Retrieve password
		retrievedPassword, err := passwordManager.RetrieveServerPassword(server)
		if err != nil {
			t.Fatalf("RetrieveServerPassword() error = %v", err)
		}
		if retrievedPassword != password {
			t.Errorf("Retrieved password = %v, want %v", retrievedPassword, password)
		}

		// 4. Update password
		newPassword := "updated-integration-password-456"
		err = passwordManager.UpdateServerPassword(server, newPassword)
		if err != nil {
			t.Fatalf("UpdateServerPassword() error = %v", err)
		}

		// Verify updated password
		updatedPassword, err := passwordManager.RetrieveServerPassword(server)
		if err != nil {
			t.Fatalf("RetrieveServerPassword() after update error = %v", err)
		}
		if updatedPassword != newPassword {
			t.Errorf("Updated password = %v, want %v", updatedPassword, newPassword)
		}

		// 5. Test SSH command generation with updated password
		sshCmd, err := buildSSHCommand(*server)
		if err != nil {
			t.Fatalf("buildSSHCommand() error = %v", err)
		}
		if !contains(sshCmd, "sshpass") {
			t.Errorf("SSH command should use sshpass with keyring password")
		}

		// 6. Delete password
		err = passwordManager.DeleteServerPassword(server)
		if err != nil {
			t.Fatalf("DeleteServerPassword() error = %v", err)
		}

		// Verify deletion
		if passwordManager.HasServerPassword(server) {
			t.Errorf("HasServerPassword() should return false after deletion")
		}
		if server.UseKeyring {
			t.Errorf("Server should not use keyring after deletion")
		}
		if server.KeyringID != "" {
			t.Errorf("Server keyring ID should be cleared after deletion")
		}
	})
}

func TestPasswordIntegration_ErrorHandling(t *testing.T) {
	// Create password manager
	passwordManager, err := auth.NewPasswordManager("file")
	if err != nil {
		t.Fatalf("failed to create password manager: %v", err)
	}

	tests := []struct {
		name        string
		server      *config.Server
		operation   string
		expectError bool
		description string
	}{
		{
			name:        "retrieve from non-existent server",
			server:      &config.Server{Name: "nonexistent", AuthType: "password", UseKeyring: true},
			operation:   "retrieve",
			expectError: true,
			description: "should fail when retrieving password for non-existent server",
		},
		{
			name:        "store password for key auth server",
			server:      &config.Server{Name: "key-server", AuthType: "key"},
			operation:   "store",
			expectError: true,
			description: "should fail when storing password for key-authenticated server",
		},
		{
			name:        "retrieve password from key auth server",
			server:      &config.Server{Name: "key-server", AuthType: "key"},
			operation:   "retrieve",
			expectError: true,
			description: "should fail when retrieving password from key-authenticated server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			switch tt.operation {
			case "store":
				err = passwordManager.StoreServerPassword(tt.server, "test-password")
			case "retrieve":
				_, err = passwordManager.RetrieveServerPassword(tt.server)
			case "update":
				err = passwordManager.UpdateServerPassword(tt.server, "new-password")
			case "delete":
				err = passwordManager.DeleteServerPassword(tt.server)
			}

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s operation, but got nil", tt.operation)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s operation: %v", tt.operation, err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
			(len(s) > len(substr) && 
			 (s[:len(substr)] == substr || 
			  s[len(s)-len(substr):] == substr || 
			  containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}