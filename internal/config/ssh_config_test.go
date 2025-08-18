package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSSHConfig(t *testing.T) {
	tests := []struct {
		name        string
		configData  string
		expected    []Server
		expectError bool
	}{
		{
			name: "simple host configuration",
			configData: `Host web-server
    HostName 192.168.1.100
    User ubuntu
    Port 22
    IdentityFile ~/.ssh/id_rsa`,
			expected: []Server{
				{
					Name:     "web-server",
					Hostname: "192.168.1.100",
					Username: "ubuntu",
					Port:     22,
					AuthType: "key",
					KeyPath:  "~/.ssh/id_rsa",
				},
			},
		},
		{
			name: "multiple hosts",
			configData: `Host production
    HostName prod.example.com
    User admin
    Port 2222
    IdentityFile ~/.ssh/prod_key

Host staging
    HostName staging.example.com
    User deploy
    Port 22
    IdentityFile ~/.ssh/staging_key`,
			expected: []Server{
				{
					Name:     "production",
					Hostname: "prod.example.com",
					Username: "admin",
					Port:     2222,
					AuthType: "key",
					KeyPath:  "~/.ssh/prod_key",
				},
				{
					Name:     "staging",
					Hostname: "staging.example.com",
					Username: "deploy",
					Port:     22,
					AuthType: "key",
					KeyPath:  "~/.ssh/staging_key",
				},
			},
		},
		{
			name: "host with no identity file (password auth)",
			configData: `Host database
    HostName db.example.com
    User postgres
    Port 5432`,
			expected: []Server{
				{
					Name:     "database",
					Hostname: "db.example.com",
					Username: "postgres",
					Port:     5432,
					AuthType: "password",
				},
			},
		},
		{
			name: "host with case variations",
			configData: `Host test-server
    hostname test.example.com
    user testuser
    port 2222
    identityfile ~/.ssh/test_key`,
			expected: []Server{
				{
					Name:     "test-server",
					Hostname: "test.example.com",
					Username: "testuser",
					Port:     2222,
					AuthType: "key",
					KeyPath:  "~/.ssh/test_key",
				},
			},
		},
		{
			name: "config with comments and global settings",
			configData: `# Global settings
StrictHostKeyChecking no

# Production server
Host prod
    HostName 10.0.1.100
    User root
    IdentityFile ~/.ssh/prod

# Staging server  
Host staging
    HostName 10.0.1.200
    User deploy`,
			expected: []Server{
				{
					Name:     "prod",
					Hostname: "10.0.1.100",
					Username: "root",
					Port:     22, // default port
					AuthType: "key",
					KeyPath:  "~/.ssh/prod",
				},
				{
					Name:     "staging",
					Hostname: "10.0.1.200",
					Username: "deploy",
					Port:     22, // default port
					AuthType: "password", // no identity file
				},
			},
		},
		{
			name: "wildcard hosts should be ignored",
			configData: `Host *
    ServerAliveInterval 60

Host web
    HostName web.example.com
    User webuser`,
			expected: []Server{
				{
					Name:     "web",
					Hostname: "web.example.com",
					Username: "webuser",
					Port:     22,
					AuthType: "password",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary SSH config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "ssh_config")
			
			err := os.WriteFile(configPath, []byte(tt.configData), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			servers, err := ParseSSHConfig(configPath)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(servers) != len(tt.expected) {
				t.Fatalf("Expected %d servers, got %d", len(tt.expected), len(servers))
			}

			for i, expected := range tt.expected {
				if i >= len(servers) {
					t.Fatalf("Missing server at index %d", i)
				}
				
				actual := servers[i]
				
				if actual.Name != expected.Name {
					t.Errorf("Server %d: expected name %q, got %q", i, expected.Name, actual.Name)
				}
				if actual.Hostname != expected.Hostname {
					t.Errorf("Server %d: expected hostname %q, got %q", i, expected.Hostname, actual.Hostname)
				}
				if actual.Username != expected.Username {
					t.Errorf("Server %d: expected username %q, got %q", i, expected.Username, actual.Username)
				}
				if actual.Port != expected.Port {
					t.Errorf("Server %d: expected port %d, got %d", i, expected.Port, actual.Port)
				}
				if actual.AuthType != expected.AuthType {
					t.Errorf("Server %d: expected auth type %q, got %q", i, expected.AuthType, actual.AuthType)
				}
				if actual.KeyPath != expected.KeyPath {
					t.Errorf("Server %d: expected key path %q, got %q", i, expected.KeyPath, actual.KeyPath)
				}
			}
		})
	}
}

func TestParseSSHConfigFromDefaultLocation(t *testing.T) {
	// Test loading from default SSH config location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get user home directory")
	}
	
	sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
	
	// Check if SSH config exists
	if _, err := os.Stat(sshConfigPath); os.IsNotExist(err) {
		t.Skip("No SSH config file found at default location")
	}
	
	// Try to parse - should not fail even if there are no valid hosts
	servers, err := ParseSSHConfig(sshConfigPath)
	if err != nil {
		t.Fatalf("Failed to parse SSH config: %v", err)
	}
	
	// Just verify it returns a slice (could be empty)
	if servers == nil {
		t.Error("Expected non-nil slice of servers")
	}
}

func TestParseSSHConfigErrors(t *testing.T) {
	tests := []struct {
		name       string
		configData string
	}{
		{
			name: "host without hostname",
			configData: `Host incomplete
    User testuser`,
		},
		{
			name: "host without user",
			configData: `Host incomplete
    HostName test.example.com`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "ssh_config")
			
			err := os.WriteFile(configPath, []byte(tt.configData), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			servers, err := ParseSSHConfig(configPath)
			
			// Should not error, but should skip incomplete hosts
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			// Should return empty slice for incomplete hosts
			if len(servers) != 0 {
				t.Errorf("Expected 0 servers for incomplete config, got %d", len(servers))
			}
		})
	}
}