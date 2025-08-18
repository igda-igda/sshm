package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"sshm/internal/config"
)

func TestImportCommand(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	tests := []struct {
		name          string
		inputFile     string
		inputData     string
		fileType      string
		expectedCount int
		shouldError   bool
		expectedNames []string
	}{
		{
			name:      "import from YAML file",
			inputFile: "config.yaml",
			inputData: `servers:
  - name: web-server
    hostname: 192.168.1.100
    port: 22
    username: ubuntu
    auth_type: key
    key_path: ~/.ssh/id_rsa
  - name: db-server
    hostname: 192.168.1.200
    port: 3306
    username: mysql
    auth_type: password
profiles:
  - name: production
    description: Production servers
    servers:
      - web-server
      - db-server`,
			fileType:      "yaml",
			expectedCount: 2,
			expectedNames: []string{"web-server", "db-server"},
		},
		{
			name:      "import from JSON file",
			inputFile: "config.json",
			inputData: `{
  "servers": [
    {
      "name": "api-server",
      "hostname": "api.example.com",
      "port": 22,
      "username": "deploy",
      "auth_type": "key",
      "key_path": "~/.ssh/deploy_key"
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
			fileType:      "json",
			expectedCount: 1,
			expectedNames: []string{"api-server"},
		},
		{
			name:      "import from SSH config",
			inputFile: "ssh_config",
			inputData: `Host web1
    HostName web1.example.com
    User webuser
    Port 22
    IdentityFile ~/.ssh/web_key

Host db1
    HostName db1.example.com  
    User dbuser
    Port 5432`,
			fileType:      "ssh",
			expectedCount: 2,
			expectedNames: []string{"web1", "db1"},
		},
		{
			name:      "import with conflicting server names",
			inputFile: "conflict.yaml",
			inputData: `servers:
  - name: existing-server
    hostname: new.example.com
    port: 22
    username: newuser
    auth_type: key
    key_path: ~/.ssh/new_key`,
			fileType:      "yaml",
			expectedCount: 1,
			shouldError:   false, // Should handle conflicts gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create input file
			inputPath := filepath.Join(tempDir, tt.inputFile)
			err := os.WriteFile(inputPath, []byte(tt.inputData), 0644)
			if err != nil {
				t.Fatalf("Failed to create input file: %v", err)
			}

			// For conflict test, pre-populate config with existing server
			if tt.name == "import with conflicting server names" {
				cfg, err := config.Load()
				if err != nil {
					t.Fatalf("Failed to load config: %v", err)
				}

				existingServer := config.Server{
					Name:     "existing-server",
					Hostname: "old.example.com",
					Port:     22,
					Username: "olduser",
					AuthType: "password",
				}

				err = cfg.AddServer(existingServer)
				if err != nil {
					t.Fatalf("Failed to add existing server: %v", err)
				}

				err = cfg.Save()
				if err != nil {
					t.Fatalf("Failed to save config: %v", err)
				}
			}

			// Execute import command
			cmd := CreateRootCommand()

			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			args := []string{"import", inputPath}
			cmd.SetArgs(args)

			err = cmd.Execute()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify import results
			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Failed to load config after import: %v", err)
			}

			servers := cfg.GetServers()

			// For conflict test, verify original server was updated
			if tt.name == "import with conflicting server names" {
				if len(servers) != 1 {
					t.Errorf("Expected 1 server after conflict resolution, got %d", len(servers))
				}

				server, err := cfg.GetServer("existing-server")
				if err != nil {
					t.Fatalf("Failed to get existing server: %v", err)
				}

				// Verify server was updated with new values
				if server.Hostname != "new.example.com" {
					t.Errorf("Expected hostname to be updated to 'new.example.com', got '%s'", server.Hostname)
				}
				return
			}

			if len(servers) != tt.expectedCount {
				t.Errorf("Expected %d servers, got %d", tt.expectedCount, len(servers))
			}

			// Verify server names
			serverNames := make(map[string]bool)
			for _, server := range servers {
				serverNames[server.Name] = true
			}

			for _, expectedName := range tt.expectedNames {
				if !serverNames[expectedName] {
					t.Errorf("Expected server '%s' not found", expectedName)
				}
			}

			// Clean up for next test
			os.Remove(filepath.Join(tempDir, "config.yaml"))
		})
	}
}

func TestImportCommandFlags(t *testing.T) {
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test YAML file
	yamlData := `servers:
  - name: test-server
    hostname: test.example.com
    port: 22
    username: testuser
    auth_type: key
    key_path: ~/.ssh/test_key`

	inputPath := filepath.Join(tempDir, "test.yaml")
	err := os.WriteFile(inputPath, []byte(yamlData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name: "import with --type flag",
			args: []string{"import", "--type", "yaml", inputPath},
		},
		{
			name: "import with --profile flag",
			args: []string{"import", "--profile", "imported", inputPath},
		},
		{
			name: "import with both flags",
			args: []string{"import", "--type", "yaml", "--profile", "test-profile", inputPath},
		},
		{
			name:        "import with invalid type",
			args:        []string{"import", "--type", "invalid", inputPath},
			shouldError: true,
		},
		{
			name:        "import without file argument",
			args:        []string{"import"},
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

			// Clean up
			os.Remove(filepath.Join(tempDir, "config.yaml"))
		})
	}
}

func TestDetectFileType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"config.yaml", "yaml"},
		{"config.yml", "yaml"},
		{"config.json", "json"},
		{"ssh_config", "ssh"},
		{"config", "ssh"},
		{"servers.unknown", "yaml"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detectFileType(tt.filename)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
