package config

import (
	"os"
	"path/filepath"
	"reflect"
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

// Profile-related tests

func TestProfileValidation(t *testing.T) {
	tests := []struct {
		name    string
		profile Profile
		wantErr bool
	}{
		{
			name: "valid profile",
			profile: Profile{
				Name:        "development",
				Description: "Development environment servers",
				Servers:     []string{"web-dev", "db-dev"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			profile: Profile{
				Description: "Development environment servers",
				Servers:     []string{"web-dev", "db-dev"},
			},
			wantErr: true,
		},
		{
			name: "empty name",
			profile: Profile{
				Name:        "",
				Description: "Development environment servers",
				Servers:     []string{"web-dev", "db-dev"},
			},
			wantErr: true,
		},
		{
			name: "whitespace only name",
			profile: Profile{
				Name:        "   ",
				Description: "Development environment servers",
				Servers:     []string{"web-dev", "db-dev"},
			},
			wantErr: true,
		},
		{
			name: "valid profile without description",
			profile: Profile{
				Name:    "prod",
				Servers: []string{"web-prod"},
			},
			wantErr: false,
		},
		{
			name: "valid profile without servers",
			profile: Profile{
				Name:        "empty",
				Description: "Empty profile",
				Servers:     []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Profile.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigAddProfile(t *testing.T) {
	config := &Config{
		Servers:  []Server{},
		Profiles: []Profile{},
	}

	profile := Profile{
		Name:        "development",
		Description: "Development environment",
		Servers:     []string{"web-dev"},
	}

	err := config.AddProfile(profile)
	if err != nil {
		t.Fatalf("Expected no error adding profile, got: %v", err)
	}

	if len(config.Profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(config.Profiles))
	}

	if config.Profiles[0].Name != "development" {
		t.Errorf("Expected profile name 'development', got '%s'", config.Profiles[0].Name)
	}
}

func TestConfigAddDuplicateProfile(t *testing.T) {
	config := &Config{
		Servers: []Server{},
		Profiles: []Profile{
			{Name: "existing", Description: "Existing profile", Servers: []string{}},
		},
	}

	duplicateProfile := Profile{
		Name:        "existing",
		Description: "Another profile with same name",
		Servers:     []string{},
	}

	err := config.AddProfile(duplicateProfile)
	if err == nil {
		t.Fatal("Expected error when adding duplicate profile name")
	}

	if len(config.Profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(config.Profiles))
	}
}

func TestConfigRemoveProfile(t *testing.T) {
	config := &Config{
		Servers: []Server{},
		Profiles: []Profile{
			{Name: "profile1", Description: "First profile", Servers: []string{}},
			{Name: "profile2", Description: "Second profile", Servers: []string{}},
		},
	}

	err := config.RemoveProfile("profile1")
	if err != nil {
		t.Fatalf("Expected no error removing profile, got: %v", err)
	}

	if len(config.Profiles) != 1 {
		t.Errorf("Expected 1 profile remaining, got %d", len(config.Profiles))
	}

	if config.Profiles[0].Name != "profile2" {
		t.Errorf("Expected remaining profile to be 'profile2', got '%s'", config.Profiles[0].Name)
	}
}

func TestConfigRemoveNonExistentProfile(t *testing.T) {
	config := &Config{
		Servers: []Server{},
		Profiles: []Profile{
			{Name: "profile1", Description: "First profile", Servers: []string{}},
		},
	}

	err := config.RemoveProfile("non-existent")
	if err == nil {
		t.Fatal("Expected error when removing non-existent profile")
	}

	if len(config.Profiles) != 1 {
		t.Errorf("Expected 1 profile remaining, got %d", len(config.Profiles))
	}
}

func TestConfigGetProfile(t *testing.T) {
	config := &Config{
		Servers: []Server{},
		Profiles: []Profile{
			{Name: "dev", Description: "Development", Servers: []string{"web-dev"}},
			{Name: "prod", Description: "Production", Servers: []string{"web-prod"}},
		},
	}

	profile, err := config.GetProfile("prod")
	if err != nil {
		t.Fatalf("Expected no error getting profile, got: %v", err)
	}

	if profile.Name != "prod" {
		t.Errorf("Expected profile name 'prod', got '%s'", profile.Name)
	}
	if profile.Description != "Production" {
		t.Errorf("Expected description 'Production', got '%s'", profile.Description)
	}
}

func TestConfigGetNonExistentProfile(t *testing.T) {
	config := &Config{
		Servers:  []Server{},
		Profiles: []Profile{},
	}

	_, err := config.GetProfile("non-existent")
	if err == nil {
		t.Fatal("Expected error when getting non-existent profile")
	}
}

func TestConfigGetServersByProfile(t *testing.T) {
	config := &Config{
		Servers: []Server{
			{Name: "web-dev", Hostname: "dev.example.com", Port: 22, Username: "user", AuthType: "key"},
			{Name: "db-dev", Hostname: "db-dev.example.com", Port: 22, Username: "user", AuthType: "key"},
			{Name: "web-prod", Hostname: "prod.example.com", Port: 22, Username: "user", AuthType: "key"},
		},
		Profiles: []Profile{
			{Name: "development", Description: "Dev environment", Servers: []string{"web-dev", "db-dev"}},
			{Name: "production", Description: "Prod environment", Servers: []string{"web-prod"}},
		},
	}

	servers, err := config.GetServersByProfile("development")
	if err != nil {
		t.Fatalf("Expected no error getting servers by profile, got: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}

	expectedNames := map[string]bool{"web-dev": true, "db-dev": true}
	for _, server := range servers {
		if !expectedNames[server.Name] {
			t.Errorf("Unexpected server name: %s", server.Name)
		}
		delete(expectedNames, server.Name)
	}

	if len(expectedNames) > 0 {
		t.Errorf("Missing expected servers: %v", expectedNames)
	}
}

func TestConfigGetServersByNonExistentProfile(t *testing.T) {
	config := &Config{
		Servers:  []Server{},
		Profiles: []Profile{},
	}

	_, err := config.GetServersByProfile("non-existent")
	if err == nil {
		t.Fatal("Expected error when getting servers by non-existent profile")
	}
}

func TestConfigGetServersByProfileWithMissingServer(t *testing.T) {
	config := &Config{
		Servers: []Server{
			{Name: "web-dev", Hostname: "dev.example.com", Port: 22, Username: "user", AuthType: "key"},
		},
		Profiles: []Profile{
			{Name: "development", Description: "Dev environment", Servers: []string{"web-dev", "missing-server"}},
		},
	}

	servers, err := config.GetServersByProfile("development")
	if err != nil {
		t.Fatalf("Expected no error getting servers by profile, got: %v", err)
	}

	// Should only return existing servers, skip missing ones
	if len(servers) != 1 {
		t.Errorf("Expected 1 server (existing only), got %d", len(servers))
	}

	if servers[0].Name != "web-dev" {
		t.Errorf("Expected server name 'web-dev', got '%s'", servers[0].Name)
	}
}

func TestConfigAssignServerToProfile(t *testing.T) {
	config := &Config{
		Servers: []Server{
			{Name: "web-server", Hostname: "web.example.com", Port: 22, Username: "user", AuthType: "key"},
		},
		Profiles: []Profile{
			{Name: "development", Description: "Dev environment", Servers: []string{}},
		},
	}

	err := config.AssignServerToProfile("web-server", "development")
	if err != nil {
		t.Fatalf("Expected no error assigning server to profile, got: %v", err)
	}

	profile, _ := config.GetProfile("development")
	if len(profile.Servers) != 1 {
		t.Errorf("Expected 1 server in profile, got %d", len(profile.Servers))
	}

	if profile.Servers[0] != "web-server" {
		t.Errorf("Expected server 'web-server' in profile, got '%s'", profile.Servers[0])
	}
}

func TestConfigAssignNonExistentServerToProfile(t *testing.T) {
	config := &Config{
		Servers: []Server{},
		Profiles: []Profile{
			{Name: "development", Description: "Dev environment", Servers: []string{}},
		},
	}

	err := config.AssignServerToProfile("non-existent", "development")
	if err == nil {
		t.Fatal("Expected error when assigning non-existent server to profile")
	}
}

func TestConfigAssignServerToNonExistentProfile(t *testing.T) {
	config := &Config{
		Servers: []Server{
			{Name: "web-server", Hostname: "web.example.com", Port: 22, Username: "user", AuthType: "key"},
		},
		Profiles: []Profile{},
	}

	err := config.AssignServerToProfile("web-server", "non-existent")
	if err == nil {
		t.Fatal("Expected error when assigning server to non-existent profile")
	}
}

func TestConfigUnassignServerFromProfile(t *testing.T) {
	config := &Config{
		Servers: []Server{
			{Name: "web-server", Hostname: "web.example.com", Port: 22, Username: "user", AuthType: "key"},
		},
		Profiles: []Profile{
			{Name: "development", Description: "Dev environment", Servers: []string{"web-server", "other-server"}},
		},
	}

	err := config.UnassignServerFromProfile("web-server", "development")
	if err != nil {
		t.Fatalf("Expected no error unassigning server from profile, got: %v", err)
	}

	profile, _ := config.GetProfile("development")
	if len(profile.Servers) != 1 {
		t.Errorf("Expected 1 server remaining in profile, got %d", len(profile.Servers))
	}

	if profile.Servers[0] != "other-server" {
		t.Errorf("Expected remaining server 'other-server', got '%s'", profile.Servers[0])
	}
}

func TestConfigUnassignServerFromNonExistentProfile(t *testing.T) {
	config := &Config{
		Servers:  []Server{},
		Profiles: []Profile{},
	}

	err := config.UnassignServerFromProfile("web-server", "non-existent")
	if err == nil {
		t.Fatal("Expected error when unassigning server from non-existent profile")
	}
}

func TestConfigUnassignNonAssignedServer(t *testing.T) {
	config := &Config{
		Servers: []Server{},
		Profiles: []Profile{
			{Name: "development", Description: "Dev environment", Servers: []string{"other-server"}},
		},
	}

	err := config.UnassignServerFromProfile("web-server", "development")
	if err == nil {
		t.Fatal("Expected error when unassigning server not in profile")
	}
}

func TestConfigWithProfilesYAMLSerialization(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create test config with profiles
	config := &Config{
		Servers: []Server{
			{
				Name:     "web-dev",
				Hostname: "dev.example.com",
				Port:     22,
				Username: "devuser",
				AuthType: "key",
				KeyPath:  "~/.ssh/dev_key",
			},
			{
				Name:     "db-prod",
				Hostname: "prod-db.example.com",
				Port:     22,
				Username: "produser",
				AuthType: "password",
			},
		},
		Profiles: []Profile{
			{
				Name:        "development",
				Description: "Development environment servers",
				Servers:     []string{"web-dev"},
			},
			{
				Name:        "production",
				Description: "Production environment servers",
				Servers:     []string{"db-prod"},
			},
		},
	}

	// Save config
	err := config.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Expected no error saving config with profiles, got: %v", err)
	}

	// Load config back
	loadedConfig, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("Expected no error loading config with profiles, got: %v", err)
	}

	// Verify servers are preserved
	if len(loadedConfig.Servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(loadedConfig.Servers))
	}

	// Verify profiles are preserved
	if len(loadedConfig.Profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(loadedConfig.Profiles))
	}

	// Check specific profile data
	devProfile, err := loadedConfig.GetProfile("development")
	if err != nil {
		t.Fatalf("Expected to find development profile, got error: %v", err)
	}

	if devProfile.Description != "Development environment servers" {
		t.Errorf("Expected development profile description to be preserved, got: %s", devProfile.Description)
	}

	if !reflect.DeepEqual(devProfile.Servers, []string{"web-dev"}) {
		t.Errorf("Expected development profile servers to be preserved, got: %v", devProfile.Servers)
	}
}

func TestConfigBackwardCompatibility(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "legacy_config.yaml")

	// Create a legacy config file (without profiles)
	legacyConfigContent := `servers:
- name: legacy-server
  hostname: legacy.example.com
  port: 22
  username: legacyuser
  auth_type: key
  key_path: ~/.ssh/legacy_key
`

	err := os.WriteFile(configPath, []byte(legacyConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create legacy config file: %v", err)
	}

	// Load the legacy config
	config, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("Expected no error loading legacy config, got: %v", err)
	}

	// Should have servers but empty profiles
	if len(config.Servers) != 1 {
		t.Errorf("Expected 1 server from legacy config, got %d", len(config.Servers))
	}

	if len(config.Profiles) != 0 {
		t.Errorf("Expected 0 profiles from legacy config, got %d", len(config.Profiles))
	}

	// Should be able to add profiles to legacy config
	profile := Profile{
		Name:        "legacy-profile",
		Description: "Profile for legacy servers",
		Servers:     []string{"legacy-server"},
	}

	err = config.AddProfile(profile)
	if err != nil {
		t.Fatalf("Expected no error adding profile to legacy config, got: %v", err)
	}

	// Save should work and preserve both servers and profiles
	err = config.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Expected no error saving upgraded config, got: %v", err)
	}

	// Reload and verify everything is preserved
	reloadedConfig, err := LoadFromPath(configPath)
	if err != nil {
		t.Fatalf("Expected no error reloading upgraded config, got: %v", err)
	}

	if len(reloadedConfig.Servers) != 1 {
		t.Errorf("Expected 1 server after reload, got %d", len(reloadedConfig.Servers))
	}

	if len(reloadedConfig.Profiles) != 1 {
		t.Errorf("Expected 1 profile after reload, got %d", len(reloadedConfig.Profiles))
	}
}