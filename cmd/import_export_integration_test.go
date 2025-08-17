package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"sshm/internal/config"
)

func TestImportExportIntegration(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	tests := []struct {
		name         string
		setupData    string
		setupFile    string
		exportFormat string
		roundTrip    bool
	}{
		{
			name:      "YAML round trip",
			setupFile: "initial.yaml",
			setupData: `servers:
  - name: web-server
    hostname: web.example.com
    port: 22
    username: webuser
    auth_type: key
    key_path: ~/.ssh/web_key
  - name: db-server
    hostname: db.example.com
    port: 5432
    username: dbuser
    auth_type: password
profiles:
  - name: production
    description: Production environment
    servers:
      - web-server
      - db-server`,
			exportFormat: "yaml",
			roundTrip:    true,
		},
		{
			name:      "JSON round trip",
			setupFile: "initial.json",
			setupData: `{
  "servers": [
    {
      "name": "api-server",
      "hostname": "api.example.com",
      "port": 8080,
      "username": "apiuser",
      "auth_type": "key",
      "key_path": "~/.ssh/api_key"
    }
  ],
  "profiles": [
    {
      "name": "staging",
      "description": "Staging environment",
      "servers": ["api-server"]
    }
  ]
}`,
			exportFormat: "json",
			roundTrip:    true,
		},
		{
			name:      "SSH config import then YAML export",
			setupFile: "ssh_config",
			setupData: `Host production-web
    HostName prod-web.example.com
    User produser
    Port 22
    IdentityFile ~/.ssh/prod_key

Host staging-db
    HostName staging-db.example.com
    User staginguser
    Port 3306`,
			exportFormat: "yaml",
			roundTrip:    false, // SSH config doesn't have profiles
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before test
			os.Remove(filepath.Join(tempDir, "config.yaml"))

			// Create initial data file
			initialPath := filepath.Join(tempDir, tt.setupFile)
			err := os.WriteFile(initialPath, []byte(tt.setupData), 0644)
			if err != nil {
				t.Fatalf("Failed to create initial file: %v", err)
			}

			// Import the data
			importCmd := CreateRootCommand()
			var importOutput bytes.Buffer
			importCmd.SetOut(&importOutput)
			importCmd.SetErr(&importOutput)

			importArgs := []string{"import", initialPath}
			importCmd.SetArgs(importArgs)

			err = importCmd.Execute()
			if err != nil {
				t.Fatalf("Import failed: %v", err)
			}

			// Verify import worked
			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Failed to load config after import: %v", err)
			}

			initialServers := cfg.GetServers()
			initialProfiles := cfg.GetProfiles()

			if len(initialServers) == 0 {
				t.Fatalf("No servers imported")
			}

			t.Logf("Imported %d servers and %d profiles", len(initialServers), len(initialProfiles))

			// Export the data
			exportPath := filepath.Join(tempDir, "exported."+tt.exportFormat)
			exportCmd := CreateRootCommand()
			var exportOutput bytes.Buffer
			exportCmd.SetOut(&exportOutput)
			exportCmd.SetErr(&exportOutput)

			exportArgs := []string{"export", "--format", tt.exportFormat, exportPath}
			exportCmd.SetArgs(exportArgs)

			err = exportCmd.Execute()
			if err != nil {
				t.Fatalf("Export failed: %v", err)
			}

			// Verify export file was created
			if _, err := os.Stat(exportPath); os.IsNotExist(err) {
				t.Fatalf("Export file was not created")
			}

			if !tt.roundTrip {
				// For SSH config import, just verify export worked
				return
			}

			// For round trip tests, import the exported data and compare
			// Clean config first
			os.Remove(filepath.Join(tempDir, "config.yaml"))

			// Import the exported data
			reimportCmd := CreateRootCommand()
			var reimportOutput bytes.Buffer
			reimportCmd.SetOut(&reimportOutput)
			reimportCmd.SetErr(&reimportOutput)

			reimportArgs := []string{"import", exportPath}
			reimportCmd.SetArgs(reimportArgs)

			err = reimportCmd.Execute()
			if err != nil {
				t.Fatalf("Re-import failed: %v", err)
			}

			// Load and compare
			reimportedCfg, err := config.Load()
			if err != nil {
				t.Fatalf("Failed to load config after re-import: %v", err)
			}

			reimportedServers := reimportedCfg.GetServers()
			reimportedProfiles := reimportedCfg.GetProfiles()

			// Compare server counts
			if len(reimportedServers) != len(initialServers) {
				t.Errorf("Server count changed: initial %d, after round trip %d",
					len(initialServers), len(reimportedServers))
			}

			// Compare profile counts
			if len(reimportedProfiles) != len(initialProfiles) {
				t.Errorf("Profile count changed: initial %d, after round trip %d",
					len(initialProfiles), len(reimportedProfiles))
			}

			// Compare server details
			for _, initialServer := range initialServers {
				reimportedServer, err := reimportedCfg.GetServer(initialServer.Name)
				if err != nil {
					t.Errorf("Server '%s' missing after round trip", initialServer.Name)
					continue
				}

				if reimportedServer.Hostname != initialServer.Hostname {
					t.Errorf("Server '%s' hostname changed: %s -> %s",
						initialServer.Name, initialServer.Hostname, reimportedServer.Hostname)
				}
				if reimportedServer.Port != initialServer.Port {
					t.Errorf("Server '%s' port changed: %d -> %d",
						initialServer.Name, initialServer.Port, reimportedServer.Port)
				}
				if reimportedServer.Username != initialServer.Username {
					t.Errorf("Server '%s' username changed: %s -> %s",
						initialServer.Name, initialServer.Username, reimportedServer.Username)
				}
				if reimportedServer.AuthType != initialServer.AuthType {
					t.Errorf("Server '%s' auth type changed: %s -> %s",
						initialServer.Name, initialServer.AuthType, reimportedServer.AuthType)
				}
			}

			// Compare profile details
			for _, initialProfile := range initialProfiles {
				reimportedProfile, err := reimportedCfg.GetProfile(initialProfile.Name)
				if err != nil {
					t.Errorf("Profile '%s' missing after round trip", initialProfile.Name)
					continue
				}

				if len(reimportedProfile.Servers) != len(initialProfile.Servers) {
					t.Errorf("Profile '%s' server count changed: %d -> %d",
						initialProfile.Name, len(initialProfile.Servers), len(reimportedProfile.Servers))
				}
			}
		})
	}
}

func TestImportExportProfileFiltering(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create a configuration with multiple profiles
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Add servers
	servers := []config.Server{
		{Name: "web1", Hostname: "web1.example.com", Port: 22, Username: "user1", AuthType: "key", KeyPath: "~/.ssh/key1"},
		{Name: "web2", Hostname: "web2.example.com", Port: 22, Username: "user2", AuthType: "key", KeyPath: "~/.ssh/key2"},
		{Name: "db1", Hostname: "db1.example.com", Port: 5432, Username: "dbuser", AuthType: "password"},
	}

	for _, server := range servers {
		err := cfg.AddServer(server)
		if err != nil {
			t.Fatalf("Failed to add server: %v", err)
		}
	}

	// Add profiles
	profiles := []config.Profile{
		{Name: "web", Description: "Web servers", Servers: []string{"web1", "web2"}},
		{Name: "database", Description: "Database servers", Servers: []string{"db1"}},
	}

	for _, profile := range profiles {
		err := cfg.AddProfile(profile)
		if err != nil {
			t.Fatalf("Failed to add profile: %v", err)
		}
	}

	err = cfg.Save()
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Export specific profile
	exportPath := filepath.Join(tempDir, "web_profile.yaml")
	cmd := CreateRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	args := []string{"export", "--profile", "web", exportPath}
	cmd.SetArgs(args)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import into new config
	os.Remove(filepath.Join(tempDir, "config.yaml"))

	importCmd := CreateRootCommand()
	var importOutput bytes.Buffer
	importCmd.SetOut(&importOutput)
	importCmd.SetErr(&importOutput)

	importArgs := []string{"import", exportPath}
	importCmd.SetArgs(importArgs)

	err = importCmd.Execute()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify only web servers were imported
	newCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	importedServers := newCfg.GetServers()
	if len(importedServers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(importedServers))
	}

	importedProfiles := newCfg.GetProfiles()
	if len(importedProfiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(importedProfiles))
	}

	if importedProfiles[0].Name != "web" {
		t.Errorf("Expected profile 'web', got '%s'", importedProfiles[0].Name)
	}

	// Verify web servers are present
	webServerNames := []string{"web1", "web2"}
	for _, name := range webServerNames {
		_, err := newCfg.GetServer(name)
		if err != nil {
			t.Errorf("Expected server '%s' not found", name)
		}
	}

	// Verify database server is not present
	_, err = newCfg.GetServer("db1")
	if err == nil {
		t.Error("Database server should not be present in web profile export")
	}
}

func TestImportWithProfileFlag(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create SSH config data
	sshConfigData := `Host server1
    HostName server1.example.com
    User user1
    Port 22
    IdentityFile ~/.ssh/key1

Host server2
    HostName server2.example.com
    User user2
    Port 2222`

	sshConfigPath := filepath.Join(tempDir, "ssh_config")
	err := os.WriteFile(sshConfigPath, []byte(sshConfigData), 0644)
	if err != nil {
		t.Fatalf("Failed to create SSH config: %v", err)
	}

	// Import with profile flag
	cmd := CreateRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)

	args := []string{"import", "--profile", "imported", sshConfigPath}
	cmd.SetArgs(args)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify import results
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check that servers were imported
	servers := cfg.GetServers()
	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}

	// Check that profile was created
	profile, err := cfg.GetProfile("imported")
	if err != nil {
		t.Fatalf("Profile 'imported' not found: %v", err)
	}

	if len(profile.Servers) != 2 {
		t.Errorf("Expected 2 servers in profile, got %d", len(profile.Servers))
	}

	expectedServers := []string{"server1", "server2"}
	profileServerMap := make(map[string]bool)
	for _, serverName := range profile.Servers {
		profileServerMap[serverName] = true
	}

	for _, expected := range expectedServers {
		if !profileServerMap[expected] {
			t.Errorf("Expected server '%s' not found in profile", expected)
		}
	}
}