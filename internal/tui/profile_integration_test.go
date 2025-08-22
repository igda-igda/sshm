package tui

import (
	"os"
	"path/filepath"
	"testing"

	"sshm/internal/config"
)

// TestProfileManagementIntegration tests the complete profile management workflow
func TestProfileManagementIntegration(t *testing.T) {
	// Setup test configuration directory
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create initial configuration with servers
	cfg := &config.Config{
		Servers: []config.Server{
			{Name: "web-server", Hostname: "web.example.com", Port: 22, Username: "ubuntu", AuthType: "key", KeyPath: "/home/user/.ssh/web_key"},
			{Name: "db-server", Hostname: "db.example.com", Port: 22, Username: "postgres", AuthType: "key", KeyPath: "/home/user/.ssh/db_key"},
			{Name: "api-server", Hostname: "api.example.com", Port: 22, Username: "deploy", AuthType: "key", KeyPath: "/home/user/.ssh/api_key"},
		},
		Profiles: []config.Profile{},
	}
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := cfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Create TUI application
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test 1: Create a new profile
	profileForm := tuiApp.CreateProfileForm()
	if err := profileForm.SetFieldValue("name", "production"); err != nil {
		t.Errorf("Failed to set profile name: %v", err)
	}
	if err := profileForm.SetFieldValue("description", "Production environment servers"); err != nil {
		t.Errorf("Failed to set profile description: %v", err)
	}

	// Validate and submit form data
	data, err := profileForm.CollectFormData()
	if err != nil {
		t.Errorf("Profile form validation failed: %v", err)
	}

	// Simulate profile creation (form submission would normally handle this)
	profile := config.Profile{
		Name:        data["name"].(string),
		Description: data["description"].(string),
		Servers:     []string{},
	}
	if err := tuiApp.config.AddProfile(profile); err != nil {
		t.Errorf("Failed to add profile: %v", err)
	}
	if err := tuiApp.config.Save(); err != nil {
		t.Errorf("Failed to save config after adding profile: %v", err)
	}

	// Test 2: Assign servers to profile
	if err := tuiApp.config.AssignServerToProfile("web-server", "production"); err != nil {
		t.Errorf("Failed to assign web-server to production profile: %v", err)
	}
	if err := tuiApp.config.AssignServerToProfile("api-server", "production"); err != nil {
		t.Errorf("Failed to assign api-server to production profile: %v", err)
	}

	// Test 3: Verify profile has assigned servers
	prodProfile, err := tuiApp.config.GetProfile("production")
	if err != nil {
		t.Errorf("Failed to retrieve production profile: %v", err)
	}
	if len(prodProfile.Servers) != 2 {
		t.Errorf("Expected 2 servers in production profile, got %d", len(prodProfile.Servers))
	}

	// Test 4: Get servers by profile
	prodServers, err := tuiApp.config.GetServersByProfile("production")
	if err != nil {
		t.Errorf("Failed to get servers by profile: %v", err)
	}
	if len(prodServers) != 2 {
		t.Errorf("Expected 2 servers for production profile, got %d", len(prodServers))
	}

	// Verify correct servers are assigned
	serverNames := make(map[string]bool)
	for _, server := range prodServers {
		serverNames[server.Name] = true
	}
	if !serverNames["web-server"] || !serverNames["api-server"] {
		t.Error("Expected web-server and api-server in production profile")
	}

	// Test 5: Edit profile
	editForm := tuiApp.CreateEditProfileForm("production")
	if err := editForm.SetFieldValue("description", "Updated production environment"); err != nil {
		t.Errorf("Failed to update profile description: %v", err)
	}

	editData, err := editForm.CollectFormData()
	if err != nil {
		t.Errorf("Edit form validation failed: %v", err)
	}

	// Simulate profile update
	updatedProfile := config.Profile{
		Name:        editData["name"].(string),
		Description: editData["description"].(string),
		Servers:     prodProfile.Servers, // Keep existing assignments
	}
	
	// Update profile in config
	for i, p := range tuiApp.config.Profiles {
		if p.Name == "production" {
			tuiApp.config.Profiles[i] = updatedProfile
			break
		}
	}

	// Test 6: Unassign server from profile
	if err := tuiApp.config.UnassignServerFromProfile("web-server", "production"); err != nil {
		t.Errorf("Failed to unassign web-server from production profile: %v", err)
	}

	// Verify unassignment
	prodProfile, err = tuiApp.config.GetProfile("production")
	if err != nil {
		t.Errorf("Failed to retrieve production profile after unassignment: %v", err)
	}
	if len(prodProfile.Servers) != 1 {
		t.Errorf("Expected 1 server in production profile after unassignment, got %d", len(prodProfile.Servers))
	}
	if prodProfile.Servers[0] != "api-server" {
		t.Errorf("Expected api-server to remain in production profile, got %s", prodProfile.Servers[0])
	}

	// Test 7: Create second profile
	devProfile := config.Profile{
		Name:        "development",
		Description: "Development environment",
		Servers:     []string{"db-server"},
	}
	if err := tuiApp.config.AddProfile(devProfile); err != nil {
		t.Errorf("Failed to add development profile: %v", err)
	}

	// Test 8: Verify profile listing
	profiles := tuiApp.config.GetProfiles()
	if len(profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(profiles))
	}

	profileNames := make(map[string]bool)
	for _, profile := range profiles {
		profileNames[profile.Name] = true
	}
	if !profileNames["production"] || !profileNames["development"] {
		t.Error("Expected production and development profiles")
	}

	// Test 9: Delete profile
	if err := tuiApp.config.RemoveProfile("development"); err != nil {
		t.Errorf("Failed to remove development profile: %v", err)
	}

	// Verify deletion
	profiles = tuiApp.config.GetProfiles()
	if len(profiles) != 1 {
		t.Errorf("Expected 1 profile after deletion, got %d", len(profiles))
	}
	if profiles[0].Name != "production" {
		t.Errorf("Expected production profile to remain, got %s", profiles[0].Name)
	}

	// Test 10: Final save and reload
	if err := tuiApp.config.Save(); err != nil {
		t.Errorf("Failed to save final configuration: %v", err)
	}

	// Reload configuration to verify persistence
	reloadedConfig, err := config.Load()
	if err != nil {
		t.Errorf("Failed to reload configuration: %v", err)
	}

	reloadedProfiles := reloadedConfig.GetProfiles()
	if len(reloadedProfiles) != 1 {
		t.Errorf("Expected 1 profile after reload, got %d", len(reloadedProfiles))
	}
	
	if reloadedProfiles[0].Name != "production" {
		t.Errorf("Expected production profile after reload, got %s", reloadedProfiles[0].Name)
	}
	
	if reloadedProfiles[0].Description != "Updated production environment" {
		t.Errorf("Expected updated description after reload, got %s", reloadedProfiles[0].Description)
	}
}

// TestProfileValidationEdgeCases tests edge cases in profile validation
func TestProfileValidationEdgeCases(t *testing.T) {
	// Setup test configuration directory
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tests := []struct {
		name        string
		profileName string
		wantErr     bool
		errType     string
	}{
		{
			name:        "NameWithSpaces",
			profileName: "test profile",
			wantErr:     false,
		},
		{
			name:        "NameWithDashes",
			profileName: "test-profile",
			wantErr:     false,
		},
		{
			name:        "NameWithUnderscores", 
			profileName: "test_profile",
			wantErr:     false,
		},
		{
			name:        "NameWithNumbers",
			profileName: "profile123",
			wantErr:     false,
		},
		{
			name:        "NameWithSpecialChars",
			profileName: "profile@#$",
			wantErr:     true,
			errType:     "invalid characters",
		},
		{
			name:        "SingleCharacterName",
			profileName: "p",
			wantErr:     true,
			errType:     "too short",
		},
		{
			name:        "WhitespaceOnly",
			profileName: "   ",
			wantErr:     true,
			errType:     "required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tuiApp.validateProfileName(tt.profileName, "")
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected validation error for profile name '%s', got none", tt.profileName)
				}
			} else if err != nil {
				t.Errorf("Unexpected validation error for profile name '%s': %v", tt.profileName, err)
			}
		})
	}
}