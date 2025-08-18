package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sshm/internal/config"
	"gopkg.in/yaml.v3"
)

func TestExportCommand(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Add test servers
	testServers := []config.Server{
		{
			Name:     "web-server",
			Hostname: "web.example.com",
			Port:     22,
			Username: "webuser",
			AuthType: "key",
			KeyPath:  "~/.ssh/web_key",
		},
		{
			Name:     "db-server",
			Hostname: "db.example.com",
			Port:     3306,
			Username: "dbuser",
			AuthType: "password",
		},
	}

	for _, server := range testServers {
		err := cfg.AddServer(server)
		if err != nil {
			t.Fatalf("Failed to add test server: %v", err)
		}
	}

	// Add test profiles
	testProfiles := []config.Profile{
		{
			Name:        "production",
			Description: "Production servers",
			Servers:     []string{"web-server", "db-server"},
		},
		{
			Name:        "staging",
			Description: "Staging environment",
			Servers:     []string{"web-server"},
		},
	}

	for _, profile := range testProfiles {
		err := cfg.AddProfile(profile)
		if err != nil {
			t.Fatalf("Failed to add test profile: %v", err)
		}
	}

	err = cfg.Save()
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	tests := []struct {
		name         string
		args         []string
		expectedFile string
		format       string
		checkContent bool
	}{
		{
			name:         "export to YAML",
			args:         []string{"export", "output.yaml"},
			expectedFile: "output.yaml",
			format:       "yaml",
			checkContent: true,
		},
		{
			name:         "export to JSON",
			args:         []string{"export", "output.json"},
			expectedFile: "output.json",
			format:       "json",
			checkContent: true,
		},
		{
			name:         "export with explicit format",
			args:         []string{"export", "--format", "yaml", "custom.txt"},
			expectedFile: "custom.txt",
			format:       "yaml",
			checkContent: true,
		},
		{
			name:         "export specific profile",
			args:         []string{"export", "--profile", "production", "prod.yaml"},
			expectedFile: "prod.yaml",
			format:       "yaml",
			checkContent: false, // Will check separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tempDir, tt.expectedFile)
			
			// Update args to use full path
			if len(tt.args) > 1 {
				tt.args[len(tt.args)-1] = outputPath
			}

			cmd := CreateRootCommand()
			
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check if file was created
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("Expected output file %s was not created", outputPath)
				return
			}

			if !tt.checkContent {
				return
			}

			// Read and validate file content
			data, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			// Parse based on format and verify structure
			switch tt.format {
			case "yaml":
				var exportedConfig config.Config
				err := yaml.Unmarshal(data, &exportedConfig)
				if err != nil {
					t.Errorf("Failed to parse exported YAML: %v", err)
				}
				
				if len(exportedConfig.Servers) != len(testServers) {
					t.Errorf("Expected %d servers in export, got %d", len(testServers), len(exportedConfig.Servers))
				}
				
				if len(exportedConfig.Profiles) != len(testProfiles) {
					t.Errorf("Expected %d profiles in export, got %d", len(testProfiles), len(exportedConfig.Profiles))
				}

			case "json":
				var exportedConfig config.Config
				err := json.Unmarshal(data, &exportedConfig)
				if err != nil {
					t.Errorf("Failed to parse exported JSON: %v", err)
				}
				
				if len(exportedConfig.Servers) != len(testServers) {
					t.Errorf("Expected %d servers in export, got %d", len(testServers), len(exportedConfig.Servers))
				}
				
				if len(exportedConfig.Profiles) != len(testProfiles) {
					t.Errorf("Expected %d profiles in export, got %d", len(testProfiles), len(exportedConfig.Profiles))
				}
			}
			
			// Clean up
			os.Remove(outputPath)
		})
	}
}

func TestExportCommandProfileFilter(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Add test servers
	servers := []config.Server{
		{Name: "server1", Hostname: "s1.example.com", Port: 22, Username: "user1", AuthType: "key", KeyPath: "~/.ssh/key1"},
		{Name: "server2", Hostname: "s2.example.com", Port: 22, Username: "user2", AuthType: "key", KeyPath: "~/.ssh/key2"},
		{Name: "server3", Hostname: "s3.example.com", Port: 22, Username: "user3", AuthType: "password"},
	}

	for _, server := range servers {
		err := cfg.AddServer(server)
		if err != nil {
			t.Fatalf("Failed to add server: %v", err)
		}
	}

	// Add profiles
	prodProfile := config.Profile{
		Name:        "production",
		Description: "Production servers",
		Servers:     []string{"server1", "server2"},
	}
	
	stagingProfile := config.Profile{
		Name:        "staging", 
		Description: "Staging servers",
		Servers:     []string{"server3"},
	}

	err = cfg.AddProfile(prodProfile)
	if err != nil {
		t.Fatalf("Failed to add production profile: %v", err)
	}
	
	err = cfg.AddProfile(stagingProfile)
	if err != nil {
		t.Fatalf("Failed to add staging profile: %v", err)
	}

	err = cfg.Save()
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Test profile-specific export
	outputPath := filepath.Join(tempDir, "prod_export.yaml")
	
	cmd := CreateRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	
	args := []string{"export", "--profile", "production", outputPath}
	cmd.SetArgs(args)
	
	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Read and verify exported content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportedConfig config.Config
	err = yaml.Unmarshal(data, &exportedConfig)
	if err != nil {
		t.Fatalf("Failed to parse exported YAML: %v", err)
	}

	// Should only have servers from production profile
	if len(exportedConfig.Servers) != 2 {
		t.Errorf("Expected 2 servers in profile export, got %d", len(exportedConfig.Servers))
	}

	// Should include the exported profile
	if len(exportedConfig.Profiles) != 1 {
		t.Errorf("Expected 1 profile in export, got %d", len(exportedConfig.Profiles))
	}

	if exportedConfig.Profiles[0].Name != "production" {
		t.Errorf("Expected profile name 'production', got '%s'", exportedConfig.Profiles[0].Name)
	}

	// Verify server names
	serverNames := make(map[string]bool)
	for _, server := range exportedConfig.Servers {
		serverNames[server.Name] = true
	}

	expectedServers := []string{"server1", "server2"}
	for _, expected := range expectedServers {
		if !serverNames[expected] {
			t.Errorf("Expected server '%s' not found in export", expected)
		}
	}
}

func TestExportCommandFlags(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create minimal config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	server := config.Server{
		Name:     "test-server",
		Hostname: "test.example.com",
		Port:     22,
		Username: "testuser",
		AuthType: "key",
		KeyPath:  "~/.ssh/test_key",
	}

	err = cfg.AddServer(server)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}

	err = cfg.Save()
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name: "export with valid format",
			args: []string{"export", "--format", "json", filepath.Join(tempDir, "test.json")},
		},
		{
			name:        "export with invalid format",
			args:        []string{"export", "--format", "xml", filepath.Join(tempDir, "test.xml")},
			shouldError: true,
		},
		{
			name:        "export without file argument",
			args:        []string{"export"},
			shouldError: true,
		},
		{
			name:        "export with non-existent profile",
			args:        []string{"export", "--profile", "nonexistent", filepath.Join(tempDir, "test.yaml")},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := CreateRootCommand()
			
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDetectExportFormat(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"config.yaml", "yaml"},
		{"config.yml", "yaml"},
		{"config.json", "json"},
		{"servers.txt", "yaml"}, // default
		{"backup", "yaml"},      // default
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detectExportFormat(tt.filename)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}