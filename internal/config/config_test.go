package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	// Create temporary config directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Test loading non-existent config (should create default)
	config, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("Expected no error loading non-existent config, got: %v", err)
	}
	if config == nil {
		t.Fatal("Expected config to be initialized")
	}
	if len(config.Servers) != 0 {
		t.Errorf("Expected empty servers list, got %d servers", len(config.Servers))
	}
}

func TestConfigSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create test config
	config := &Config{
		Servers: []Server{
			{
				Name:               "test-server",
				Hostname:           "example.com",
				Port:               22,
				Username:           "testuser",
				AuthType:           "key",
				KeyPath:            "~/.ssh/id_rsa",
				PassphraseProtected: false,
			},
		},
	}

	// Save config
	err := config.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Expected no error saving config, got: %v", err)
	}

	// Check file exists and has correct permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Config file was not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %v", info.Mode().Perm())
	}
}

func TestConfigAddServer(t *testing.T) {
	config := &Config{Servers: []Server{}}

	server := Server{
		Name:     "test-server",
		Hostname: "example.com",
		Port:     22,
		Username: "testuser",
		AuthType: "key",
		KeyPath:  "~/.ssh/id_rsa",
	}

	err := config.AddServer(server)
	if err != nil {
		t.Fatalf("Expected no error adding server, got: %v", err)
	}

	if len(config.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(config.Servers))
	}

	if config.Servers[0].Name != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%s'", config.Servers[0].Name)
	}
}

func TestConfigAddDuplicateServer(t *testing.T) {
	config := &Config{
		Servers: []Server{
			{Name: "existing-server", Hostname: "example.com", Port: 22, Username: "user", AuthType: "key"},
		},
	}

	duplicateServer := Server{
		Name:     "existing-server",
		Hostname: "other.com",
		Port:     22,
		Username: "user2",
		AuthType: "key",
	}

	err := config.AddServer(duplicateServer)
	if err == nil {
		t.Fatal("Expected error when adding duplicate server name")
	}

	if len(config.Servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(config.Servers))
	}
}

func TestConfigRemoveServer(t *testing.T) {
	config := &Config{
		Servers: []Server{
			{Name: "server1", Hostname: "example1.com", Port: 22, Username: "user1", AuthType: "key"},
			{Name: "server2", Hostname: "example2.com", Port: 22, Username: "user2", AuthType: "key"},
		},
	}

	err := config.RemoveServer("server1")
	if err != nil {
		t.Fatalf("Expected no error removing server, got: %v", err)
	}

	if len(config.Servers) != 1 {
		t.Errorf("Expected 1 server remaining, got %d", len(config.Servers))
	}

	if config.Servers[0].Name != "server2" {
		t.Errorf("Expected remaining server to be 'server2', got '%s'", config.Servers[0].Name)
	}
}

func TestConfigRemoveNonExistentServer(t *testing.T) {
	config := &Config{
		Servers: []Server{
			{Name: "server1", Hostname: "example1.com", Port: 22, Username: "user1", AuthType: "key"},
		},
	}

	err := config.RemoveServer("non-existent")
	if err == nil {
		t.Fatal("Expected error when removing non-existent server")
	}

	if len(config.Servers) != 1 {
		t.Errorf("Expected 1 server remaining, got %d", len(config.Servers))
	}
}

func TestConfigGetServer(t *testing.T) {
	config := &Config{
		Servers: []Server{
			{Name: "server1", Hostname: "example1.com", Port: 22, Username: "user1", AuthType: "key"},
			{Name: "server2", Hostname: "example2.com", Port: 22, Username: "user2", AuthType: "password"},
		},
	}

	server, err := config.GetServer("server2")
	if err != nil {
		t.Fatalf("Expected no error getting server, got: %v", err)
	}

	if server.Name != "server2" {
		t.Errorf("Expected server name 'server2', got '%s'", server.Name)
	}
	if server.AuthType != "password" {
		t.Errorf("Expected auth type 'password', got '%s'", server.AuthType)
	}
}

func TestConfigGetNonExistentServer(t *testing.T) {
	config := &Config{Servers: []Server{}}

	_, err := config.GetServer("non-existent")
	if err == nil {
		t.Fatal("Expected error when getting non-existent server")
	}
}

func TestServerValidation(t *testing.T) {
	tests := []struct {
		name    string
		server  Server
		wantErr bool
	}{
		{
			name: "valid key auth server",
			server: Server{
				Name:     "valid-server",
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				AuthType: "key",
				KeyPath:  "~/.ssh/id_rsa",
			},
			wantErr: false,
		},
		{
			name: "valid password auth server",
			server: Server{
				Name:     "valid-server",
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				AuthType: "password",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			server: Server{
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				AuthType: "key",
			},
			wantErr: true,
		},
		{
			name: "missing hostname",
			server: Server{
				Name:     "test",
				Port:     22,
				Username: "user",
				AuthType: "key",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			server: Server{
				Name:     "test",
				Hostname: "example.com",
				Port:     22,
				AuthType: "key",
			},
			wantErr: true,
		},
		{
			name: "invalid auth type",
			server: Server{
				Name:     "test",
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				AuthType: "invalid",
			},
			wantErr: true,
		},
		{
			name: "key auth missing key path",
			server: Server{
				Name:     "test",
				Hostname: "example.com",
				Port:     22,
				Username: "user",
				AuthType: "key",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			server: Server{
				Name:     "test",
				Hostname: "example.com",
				Port:     0,
				Username: "user",
				AuthType: "key",
				KeyPath:  "~/.ssh/id_rsa",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.server.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Server.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}