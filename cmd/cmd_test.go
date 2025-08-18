package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"sshm/internal/config"
	"sshm/internal/tmux"

	"github.com/spf13/cobra"
)

// Test helpers and mocks
type mockInput struct {
	inputs []string
	index  int
}

func (m *mockInput) readLine() string {
	if m.index >= len(m.inputs) {
		return ""
	}
	result := m.inputs[m.index]
	m.index++
	return result
}

// Mock functions for testing
var (
	mockUserInput      *mockInput
	mockTmuxAvailable  = true
	mockConnectSuccess = true
)

func TestAddCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		inputs      []string
		setupFn     func(string) // function to setup test data
		expectError bool
		contains    string
	}{
		{
			name: "successful add with key auth",
			args: []string{"production-api"},
			inputs: []string{
				"api.prod.company.com", // hostname
				"22",                   // port
				"deploy",               // username
				"key",                  // auth type
				"~/.ssh/prod_rsa",      // key path
				"n",                    // passphrase protected
			},
			expectError: false,
			contains:    "Server 'production-api' added successfully",
		},
		{
			name: "successful add with password auth",
			args: []string{"staging-db"},
			inputs: []string{
				"db.staging.company.com", // hostname
				"2222",                   // port
				"admin",                  // username
				"password",               // auth type
			},
			expectError: false,
			contains:    "Server 'staging-db' added successfully",
		},
		{
			name:        "missing server name",
			args:        []string{},
			expectError: true,
			contains:    "accepts 1 arg(s), received 0",
		},
		{
			name:        "empty server name",
			args:        []string{""},
			expectError: true,
			contains:    "Server name cannot be empty",
		},
		{
			name:        "whitespace-only server name",
			args:        []string{"   "},
			expectError: true,
			contains:    "Server name cannot be empty",
		},
		{
			name: "invalid port",
			args: []string{"test-server"},
			inputs: []string{
				"test.com", // hostname
				"invalid",  // port
			},
			expectError: true,
			contains:    "Invalid port",
		},
		{
			name: "duplicate server name",
			args: []string{"production-api"},
			inputs: []string{
				"api2.prod.company.com",
				"22",
				"deploy2",
				"key",
				"~/.ssh/prod_rsa2",
				"n",
			},
			setupFn: func(configDir string) {
				setupTestServers(configDir) // This will create a production-api server
			},
			expectError: true,
			contains:    "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temporary config directory
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			// Setup test data if needed
			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			// Setup mock stdin
			restoreStdin := setupMockStdin(tt.inputs)
			defer restoreStdin()

			// Capture output
			var output bytes.Buffer

			// Create a new root command for this test
			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(addCmd)

			args := append([]string{"add"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}
			if tt.contains != "" && !strings.Contains(outputStr, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, outputStr)
			}
		})
	}
}

func TestListCommand(t *testing.T) {
	tests := []struct {
		name     string
		setupFn  func(string) // function to setup test data
		contains []string
	}{
		{
			name:     "empty configuration",
			setupFn:  func(configDir string) {}, // no setup, empty config
			contains: []string{"No servers configured"},
		},
		{
			name: "list with servers",
			setupFn: func(configDir string) {
				setupTestServers(configDir)
			},
			contains: []string{"production-api", "staging-db", "api.prod.company.com", "deploy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			var output bytes.Buffer

			// Create a new root command for this test
			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(listCmd)

			testRootCmd.SetArgs([]string{"list"})
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			outputStr := output.String()
			for _, expectedContent := range tt.contains {
				if !strings.Contains(outputStr, expectedContent) {
					t.Errorf("Expected output to contain '%s', got: %s", expectedContent, outputStr)
				}
			}
		})
	}
}

func TestRemoveCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFn     func(string)
		inputs      []string // for confirmation prompts
		expectError bool
		contains    string
	}{
		{
			name: "successful removal with confirmation",
			args: []string{"production-api"},
			setupFn: func(configDir string) {
				setupTestServers(configDir)
			},
			inputs:      []string{"y"},
			expectError: false,
			contains:    "Server 'production-api' removed successfully",
		},
		{
			name: "cancelled removal",
			args: []string{"production-api"},
			setupFn: func(configDir string) {
				setupTestServers(configDir)
			},
			inputs:      []string{"n"},
			expectError: false,
			contains:    "Removal cancelled",
		},
		{
			name:        "missing server name",
			args:        []string{},
			expectError: true,
			contains:    "accepts 1 arg(s), received 0",
		},
		{
			name:        "non-existent server",
			args:        []string{"non-existent"},
			expectError: true,
			contains:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			restoreStdin := setupMockStdin(tt.inputs)
			defer restoreStdin()

			var output bytes.Buffer

			// Create a new root command for this test
			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(removeCmd)

			args := append([]string{"remove"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}
			if tt.contains != "" && !strings.Contains(outputStr, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, outputStr)
			}
		})
	}
}

func TestConnectCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFn     func(string)
		expectError bool
		contains    string
	}{
		{
			name: "connect attempt",
			args: []string{"production-api"},
			setupFn: func(configDir string) {
				setupTestServers(configDir)
			},
			expectError: false, // Should succeed and gracefully handle tmux attach failure
			contains:    "Connecting to production-api",
		},
		{
			name:        "missing server name",
			args:        []string{},
			expectError: true,
			contains:    "accepts 1 arg(s), received 0",
		},
		{
			name:        "non-existent server",
			args:        []string{"non-existent"},
			expectError: true,
			contains:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			var output bytes.Buffer

			// Create a new root command for this test
			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(connectCmd)

			args := append([]string{"connect"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}
			if tt.contains != "" && !strings.Contains(outputStr, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, outputStr)
			}
		})
	}
}

// Test helper functions
func setupTestConfig(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "sshm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set environment variable to use test config directory
	os.Setenv("SSHM_CONFIG_DIR", tmpDir)

	return tmpDir
}

func setupTestServers(configDir string) {
	configContent := `servers:
  - name: "production-api"
    hostname: "api.prod.company.com"
    port: 22
    username: "deploy"
    auth_type: "key"
    key_path: "~/.ssh/prod_rsa"
    passphrase_protected: false
  - name: "staging-db"
    hostname: "db.staging.company.com"
    port: 2222
    username: "admin"
    auth_type: "password"
`

	configPath := filepath.Join(configDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		panic(fmt.Sprintf("Failed to write test config: %v", err))
	}
}

// Mock stdin for testing
type mockStdin struct {
	inputs []string
	index  int
}

func (m *mockStdin) Read(p []byte) (n int, err error) {
	if m.index >= len(m.inputs) {
		return 0, io.EOF
	}

	input := m.inputs[m.index] + "\n"
	m.index++

	copy(p, []byte(input))
	return len(input), nil
}

// Override stdin for testing
func setupMockStdin(inputs []string) func() {
	original := os.Stdin

	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		defer w.Close()
		for _, input := range inputs {
			fmt.Fprintln(w, input)
		}
	}()

	return func() {
		os.Stdin = original
	}
}

// Profile-related test helpers

func setupTestProfiles(configDir string) {
	configContent := `servers:
  - name: "web-dev"
    hostname: "dev.example.com"
    port: 22
    username: "devuser"
    auth_type: "key"
    key_path: "~/.ssh/dev_key"
  - name: "db-dev"
    hostname: "db-dev.example.com"
    port: 22
    username: "devuser"
    auth_type: "key"
    key_path: "~/.ssh/dev_key"
  - name: "web-prod"
    hostname: "prod.example.com"
    port: 22
    username: "produser"
    auth_type: "key"
    key_path: "~/.ssh/prod_key"
profiles:
  - name: "development"
    description: "Development environment servers"
    servers: ["web-dev", "db-dev"]
  - name: "production"
    description: "Production environment servers"
    servers: ["web-prod"]
`

	configPath := filepath.Join(configDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		panic(fmt.Sprintf("Failed to write test config with profiles: %v", err))
	}
}

// Profile command tests

func TestProfileCreateCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		inputs      []string
		setupFn     func(string)
		expectError bool
		contains    string
	}{
		{
			name: "successful profile creation",
			args: []string{"staging"},
			inputs: []string{
				"Staging environment servers", // description
			},
			setupFn:     func(configDir string) { setupTestServers(configDir) },
			expectError: false,
			contains:    "Profile 'staging' created successfully",
		},
		{
			name: "profile creation with empty description",
			args: []string{"testing"},
			inputs: []string{
				"", // empty description
			},
			setupFn:     func(configDir string) { setupTestServers(configDir) },
			expectError: false,
			contains:    "Profile 'testing' created successfully",
		},
		{
			name:        "profile creation with --description flag",
			args:        []string{"production", "--description", "Production servers"},
			setupFn:     func(configDir string) { setupTestServers(configDir) },
			expectError: false,
			contains:    "Profile 'production' created successfully",
		},
		{
			name:        "profile creation with -d short flag",
			args:        []string{"dev", "-d", "Development environment"},
			setupFn:     func(configDir string) { setupTestServers(configDir) },
			expectError: false,
			contains:    "Profile 'dev' created successfully",
		},
		{
			name:        "profile creation with empty --description flag",
			args:        []string{"empty-desc", "--description", ""},
			setupFn:     func(configDir string) { setupTestServers(configDir) },
			expectError: false,
			contains:    "Profile 'empty-desc' created successfully",
		},
		{
			name:        "missing profile name",
			args:        []string{},
			expectError: true,
			contains:    "accepts 1 arg(s), received 0",
		},
		{
			name: "duplicate profile name",
			args: []string{"development"},
			inputs: []string{
				"Another development environment",
			},
			setupFn:     func(configDir string) { setupTestProfiles(configDir) },
			expectError: true,
			contains:    "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			restoreStdin := setupMockStdin(tt.inputs)
			defer restoreStdin()

			var output bytes.Buffer

			// Create a new root command for this test
			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(profileCmd)

			args := append([]string{"profile", "create"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}
			if tt.contains != "" && !strings.Contains(outputStr, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, outputStr)
			}
		})
	}
}

func TestProfileCreateWithDescriptionFlag(t *testing.T) {
	// Test that the description flag actually saves the correct description
	tmpDir := setupTestConfig(t)
	defer os.RemoveAll(tmpDir)

	// Create profile with description flag
	testRootCmd := &cobra.Command{Use: "sshm"}
	testRootCmd.AddCommand(profileCmd)

	var output bytes.Buffer
	testRootCmd.SetOut(&output)
	testRootCmd.SetErr(&output)

	testRootCmd.SetArgs([]string{"profile", "create", "test-desc", "--description", "Test Description"})
	err := testRootCmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Load config and verify description was saved
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	profile, err := cfg.GetProfile("test-desc")
	if err != nil {
		t.Fatalf("Failed to get created profile: %v", err)
	}

	if profile.Description != "Test Description" {
		t.Errorf("Expected description 'Test Description', got '%s'", profile.Description)
	}
}

func TestProfileListCommand(t *testing.T) {
	tests := []struct {
		name        string
		setupFn     func(string)
		contains    []string
		notContains []string
	}{
		{
			name:     "empty profiles",
			setupFn:  func(configDir string) { setupTestServers(configDir) },
			contains: []string{"No profiles configured"},
		},
		{
			name: "list with profiles",
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			contains: []string{"development", "production", "Development environment servers", "Production environment servers"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			var output bytes.Buffer

			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(profileCmd)

			testRootCmd.SetArgs([]string{"profile", "list"})
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			outputStr := output.String()
			for _, expectedContent := range tt.contains {
				if !strings.Contains(outputStr, expectedContent) {
					t.Errorf("Expected output to contain '%s', got: %s", expectedContent, outputStr)
				}
			}
			for _, notExpected := range tt.notContains {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output to NOT contain '%s', got: %s", notExpected, outputStr)
				}
			}
		})
	}
}

func TestProfileDeleteCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFn     func(string)
		inputs      []string
		expectError bool
		contains    string
	}{
		{
			name: "successful profile deletion with confirmation",
			args: []string{"development"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			inputs:      []string{"y"},
			expectError: false,
			contains:    "Profile 'development' deleted successfully",
		},
		{
			name: "cancelled profile deletion",
			args: []string{"development"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			inputs:      []string{"n"},
			expectError: false,
			contains:    "Deletion cancelled",
		},
		{
			name:        "missing profile name",
			args:        []string{},
			expectError: true,
			contains:    "accepts 1 arg(s), received 0",
		},
		{
			name:        "non-existent profile",
			args:        []string{"non-existent"},
			setupFn:     func(configDir string) { setupTestProfiles(configDir) },
			expectError: true,
			contains:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			restoreStdin := setupMockStdin(tt.inputs)
			defer restoreStdin()

			var output bytes.Buffer

			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(profileCmd)

			args := append([]string{"profile", "delete"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}
			if tt.contains != "" && !strings.Contains(outputStr, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, outputStr)
			}
		})
	}
}

func TestProfileAssignCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFn     func(string)
		expectError bool
		contains    string
	}{
		{
			name: "successful server assignment",
			args: []string{"web-dev", "production"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false,
			contains:    "Server 'web-dev' assigned to profile 'production'",
		},
		{
			name: "assign already assigned server",
			args: []string{"web-dev", "development"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false,
			contains:    "Server 'web-dev' assigned to profile 'development'",
		},
		{
			name:        "missing arguments",
			args:        []string{"web-dev"},
			expectError: true,
			contains:    "accepts 2 arg(s), received 1",
		},
		{
			name: "non-existent server",
			args: []string{"non-existent", "development"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: true,
			contains:    "not found",
		},
		{
			name: "non-existent profile",
			args: []string{"web-dev", "non-existent"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: true,
			contains:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			var output bytes.Buffer

			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(profileCmd)

			args := append([]string{"profile", "assign"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}
			if tt.contains != "" && !strings.Contains(outputStr, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, outputStr)
			}
		})
	}
}

func TestProfileUnassignCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFn     func(string)
		expectError bool
		contains    string
	}{
		{
			name: "successful server unassignment",
			args: []string{"web-dev", "development"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false,
			contains:    "Server 'web-dev' unassigned from profile 'development'",
		},
		{
			name:        "missing arguments",
			args:        []string{"web-dev"},
			expectError: true,
			contains:    "accepts 2 arg(s), received 1",
		},
		{
			name: "server not in profile",
			args: []string{"web-prod", "development"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: true,
			contains:    "is not assigned to profile",
		},
		{
			name: "non-existent profile",
			args: []string{"web-dev", "non-existent"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: true,
			contains:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			var output bytes.Buffer

			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(profileCmd)

			args := append([]string{"profile", "unassign"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}
			if tt.contains != "" && !strings.Contains(outputStr, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, outputStr)
			}
		})
	}
}

// Tests for profile-filtered operations

func TestListCommandWithProfileFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFn     func(string)
		expectError bool
		contains    []string
		notContains []string
	}{
		{
			name: "list servers in development profile",
			args: []string{"--profile", "development"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false,
			contains:    []string{"web-dev", "db-dev", "dev.example.com", "db-dev.example.com"},
			notContains: []string{"web-prod", "prod.example.com"},
		},
		{
			name: "list servers in production profile",
			args: []string{"--profile", "production"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false,
			contains:    []string{"web-prod", "prod.example.com"},
			notContains: []string{"web-dev", "db-dev", "dev.example.com"},
		},
		{
			name: "non-existent profile",
			args: []string{"--profile", "non-existent"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: true,
			contains:    []string{"Profile 'non-existent' not found"},
		},
		{
			name: "empty profile",
			args: []string{"--profile", "empty"},
			setupFn: func(configDir string) {
				setupTestProfilesWithEmpty(configDir)
			},
			expectError: false,
			contains:    []string{"No servers found in profile 'empty'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			var output bytes.Buffer

			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(listCmd)

			args := append([]string{"list"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}

			for _, expectedContent := range tt.contains {
				if !strings.Contains(outputStr, expectedContent) {
					t.Errorf("Expected output to contain '%s', got: %s", expectedContent, outputStr)
				}
			}
			for _, notExpected := range tt.notContains {
				if strings.Contains(outputStr, notExpected) {
					t.Errorf("Expected output to NOT contain '%s', got: %s", notExpected, outputStr)
				}
			}
		})
	}
}

func TestBatchCommandWithProfileFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFn     func(string)
		expectError bool
		contains    []string
	}{
		{
			name: "batch connect to development profile",
			args: []string{"--profile", "development"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false, // Should succeed and gracefully handle tmux attach failure
			contains:    []string{"Creating group session for profile 'development'"},
		},
		{
			name: "batch connect to production profile",
			args: []string{"--profile", "production"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false, // Should succeed and gracefully handle tmux attach failure
			contains:    []string{"Creating group session for profile 'production'"},
		},
		{
			name: "non-existent profile",
			args: []string{"--profile", "non-existent"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: true,
			contains:    []string{"Profile 'non-existent' not found"},
		},
		{
			name: "empty profile",
			args: []string{"--profile", "empty"},
			setupFn: func(configDir string) {
				setupTestProfilesWithEmpty(configDir)
			},
			expectError: true,
			contains:    []string{"No servers found in profile 'empty'"},
		},
		{
			name: "missing profile argument",
			args: []string{"--profile"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: true,
			contains:    []string{"flag needs an argument"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			var output bytes.Buffer

			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(batchCmd)

			args := append([]string{"batch"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}

			for _, expectedContent := range tt.contains {
				if !strings.Contains(outputStr, expectedContent) {
					t.Errorf("Expected output to contain '%s', got: %s", expectedContent, outputStr)
				}
			}
		})
	}
}

// Additional test helper for profiles with empty profile
func setupTestProfilesWithEmpty(configDir string) {
	configContent := `servers:
  - name: "web-dev"
    hostname: "dev.example.com"
    port: 22
    username: "devuser"
    auth_type: "key"
    key_path: "~/.ssh/dev_key"
  - name: "db-dev"
    hostname: "db-dev.example.com"
    port: 22
    username: "devuser"
    auth_type: "key"
    key_path: "~/.ssh/dev_key"
  - name: "web-prod"
    hostname: "prod.example.com"
    port: 22
    username: "produser"
    auth_type: "key"
    key_path: "~/.ssh/prod_key"
profiles:
  - name: "development"
    description: "Development environment servers"
    servers: ["web-dev", "db-dev"]
  - name: "production"
    description: "Production environment servers"
    servers: ["web-prod"]
  - name: "empty"
    description: "Empty profile for testing"
    servers: []
`

	configPath := filepath.Join(configDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		panic(fmt.Sprintf("Failed to write test config with empty profile: %v", err))
	}
}

func TestSessionsListCommand(t *testing.T) {
	tests := []struct {
		name            string
		mockSessions    []string
		expectError     bool
		expectedContent string
	}{
		{
			name:            "no active sessions",
			mockSessions:    []string{},
			expectError:     false,
			expectedContent: "No active tmux sessions found",
		},
		{
			name:            "single session",
			mockSessions:    []string{"production-web"},
			expectError:     false,
			expectedContent: "Active sessions: 1",
		},
		{
			name:            "multiple sessions with different types",
			mockSessions:    []string{"development", "staging_server", "production-api"},
			expectError:     false,
			expectedContent: "Active sessions: 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test directory
			testDir := t.TempDir()
			t.Setenv("SSHM_CONFIG_DIR", testDir)

			// Mock tmux command for sessions list
			oldExecCommand := tmux.GetExecCommand()
			defer tmux.SetExecCommand(oldExecCommand)

			tmux.SetExecCommand(func(name string, arg ...string) *exec.Cmd {
				if name == "tmux" && len(arg) > 0 {
					if arg[0] == "-V" {
						// tmux version check
						return exec.Command("echo", "tmux version")
					} else if arg[0] == "list-sessions" {
						// list sessions
						if len(tt.mockSessions) == 0 {
							return exec.Command("echo", "")
						}
						output := ""
						for i, session := range tt.mockSessions {
							if i > 0 {
								output += "\n"
							}
							output += session
						}
						return exec.Command("echo", output)
					}
				}
				return exec.Command("echo", "")
			})

			var output strings.Builder
			err := runSessionsListCommand(&output)

			if (err != nil) != tt.expectError {
				t.Errorf("runSessionsListCommand() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if !strings.Contains(output.String(), tt.expectedContent) {
				t.Errorf("runSessionsListCommand() output = %v, expected to contain %v", output.String(), tt.expectedContent)
			}
		})
	}
}

func TestSessionsKillCommand(t *testing.T) {
	tests := []struct {
		name            string
		sessionName     string
		mockSessions    []string
		expectError     bool
		expectedContent string
	}{
		{
			name:            "kill existing session",
			sessionName:     "test-session",
			mockSessions:    []string{"test-session", "other-session"},
			expectError:     false,
			expectedContent: "terminated successfully",
		},
		{
			name:            "kill non-existent session",
			sessionName:     "non-existent",
			mockSessions:    []string{"test-session"},
			expectError:     true,
			expectedContent: "Session 'non-existent' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test directory
			testDir := t.TempDir()
			t.Setenv("SSHM_CONFIG_DIR", testDir)

			// Mock tmux command
			oldExecCommand := tmux.GetExecCommand()
			defer tmux.SetExecCommand(oldExecCommand)

			tmux.SetExecCommand(func(name string, arg ...string) *exec.Cmd {
				if name == "tmux" && len(arg) > 0 {
					if arg[0] == "-V" {
						// tmux version check
						return exec.Command("echo", "tmux version")
					} else if arg[0] == "list-sessions" {
						// list sessions
						if len(tt.mockSessions) == 0 {
							return exec.Command("echo", "")
						}
						output := ""
						for i, session := range tt.mockSessions {
							if i > 0 {
								output += "\n"
							}
							output += session
						}
						return exec.Command("echo", output)
					} else if arg[0] == "kill-session" {
						// kill session - just succeed
						return exec.Command("echo", "session killed")
					}
				}
				return exec.Command("echo", "")
			})

			var output strings.Builder
			err := runSessionsKillCommand(tt.sessionName, &output)

			if (err != nil) != tt.expectError {
				t.Errorf("runSessionsKillCommand() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if tt.expectError {
				// For error cases, check the error message
				if err != nil && !strings.Contains(err.Error(), tt.expectedContent) {
					t.Errorf("runSessionsKillCommand() error = %v, expected to contain %v", err.Error(), tt.expectedContent)
				}
			} else {
				// For success cases, check the output
				if !strings.Contains(output.String(), tt.expectedContent) {
					t.Errorf("runSessionsKillCommand() output = %v, expected to contain %v", output.String(), tt.expectedContent)
				}
			}
		})
	}
}

func TestIsGroupSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		expected    bool
	}{
		{
			name:        "simple profile name",
			sessionName: "development",
			expected:    true,
		},
		{
			name:        "profile name with conflict resolution",
			sessionName: "staging-1",
			expected:    false,
		},
		{
			name:        "server with normalized dots",
			sessionName: "api_server_com",
			expected:    false,
		},
		{
			name:        "simple server name",
			sessionName: "webserver",
			expected:    true,
		},
		{
			name:        "server with underscores",
			sessionName: "db_primary",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGroupSession(tt.sessionName)
			if result != tt.expected {
				t.Errorf("isGroupSession() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// CLI Flag Tests

func TestAddCommandWithCLIFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFn     func(string)
		expectError bool
		contains    string
	}{
		{
			name: "successful add with CLI flags - key auth",
			args: []string{"cli-test-key", "--hostname", "test.example.com", "--username", "testuser", "--auth-type", "key", "--key-path", "~/.ssh/test_key"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: false,
			contains: "Server 'cli-test-key' added successfully",
		},
		{
			name: "successful add with CLI flags - key auth with passphrase",
			args: []string{"cli-test-passphrase", "--hostname", "secure.example.com", "--username", "secureuser", "--auth-type", "key", "--key-path", "~/.ssh/secure_key", "--passphrase-protected"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: false,
			contains: "Server 'cli-test-passphrase' added successfully",
		},
		{
			name: "successful add with CLI flags - password auth",
			args: []string{"cli-test-password", "--hostname", "db.example.com", "--username", "dbuser", "--auth-type", "password", "--port", "5432"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: false,
			contains: "Server 'cli-test-password' added successfully",
		},
		{
			name: "successful add with CLI flags - short flags",
			args: []string{"cli-test-short", "-H", "short.example.com", "-u", "shortuser", "-a", "password", "-p", "3000"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: false,
			contains: "Server 'cli-test-short' added successfully",
		},
		{
			name: "missing hostname flag",
			args: []string{"cli-test-missing-hostname", "--username", "testuser", "--auth-type", "password"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: true,
			contains: "--hostname is required for non-interactive mode",
		},
		{
			name: "missing username flag",
			args: []string{"cli-test-missing-username", "--hostname", "test.example.com", "--auth-type", "password"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: true,
			contains: "--username is required for non-interactive mode",
		},
		{
			name: "missing auth-type flag",
			args: []string{"cli-test-missing-authtype", "--hostname", "test.example.com", "--username", "testuser"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: true,
			contains: "--auth-type is required for non-interactive mode",
		},
		{
			name: "invalid auth type",
			args: []string{"cli-test-invalid", "--hostname", "test.example.com", "--username", "testuser", "--auth-type", "invalid"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: true,
			contains: "Authentication type must be 'key' or 'password'",
		},
		{
			name: "key auth without key path",
			args: []string{"cli-test-nokey", "--hostname", "test.example.com", "--username", "testuser", "--auth-type", "key"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: true,
			contains: "--key-path is required when auth-type is 'key'",
		},
		{
			name: "invalid port",
			args: []string{"cli-test-badport", "--hostname", "test.example.com", "--username", "testuser", "--auth-type", "password", "--port", "99999"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: true,
			contains: "Invalid port: 99999",
		},
		{
			name: "empty hostname value",
			args: []string{"cli-test-empty-hostname", "--hostname", "", "--username", "testuser", "--auth-type", "password"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: true,
			contains: "Hostname cannot be empty",
		},
		{
			name: "empty username value",
			args: []string{"cli-test-empty-username", "--hostname", "test.example.com", "--username", "", "--auth-type", "password"},
			setupFn: func(configDir string) { setupTestServers(configDir) },
			expectError: true,
			contains: "Username cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			var output bytes.Buffer

			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(addCmd)

			args := append([]string{"add"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}
			if tt.contains != "" && !strings.Contains(outputStr, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, outputStr)
			}
		})
	}
}

func TestProfileDeleteWithYesFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFn     func(string)
		expectError bool
		contains    string
	}{
		{
			name: "successful profile deletion with --yes flag",
			args: []string{"development", "--yes"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false,
			contains: "Profile 'development' deleted successfully",
		},
		{
			name: "successful profile deletion with -y short flag",
			args: []string{"production", "-y"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false,
			contains: "Profile 'production' deleted successfully",
		},
		{
			name: "non-existent profile with --yes flag",
			args: []string{"non-existent", "--yes"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: true,
			contains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			var output bytes.Buffer

			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(profileCmd)

			args := append([]string{"profile", "delete"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}
			if tt.contains != "" && !strings.Contains(outputStr, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, outputStr)
			}

			// Verify that the profile was actually deleted (for success cases)
			if !tt.expectError {
				cfg, err := config.Load()
				if err != nil {
					t.Fatalf("Failed to load config: %v", err)
				}
				profileName := tt.args[0]
				_, err = cfg.GetProfile(profileName)
				if err == nil {
					t.Errorf("Expected profile '%s' to be deleted, but it still exists", profileName)
				}
			}
		})
	}
}

func TestRemoveCommandWithYesFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFn     func(string)
		expectError bool
		contains    string
	}{
		{
			name: "successful server removal with --yes flag",
			args: []string{"web-dev", "--yes"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false,
			contains: "Server 'web-dev' removed successfully",
		},
		{
			name: "successful server removal with -y short flag",
			args: []string{"db-dev", "-y"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: false,
			contains: "Server 'db-dev' removed successfully",
		},
		{
			name: "non-existent server with --yes flag",
			args: []string{"non-existent", "--yes"},
			setupFn: func(configDir string) {
				setupTestProfiles(configDir)
			},
			expectError: true,
			contains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestConfig(t)
			defer os.RemoveAll(tmpDir)

			if tt.setupFn != nil {
				tt.setupFn(tmpDir)
			}

			var output bytes.Buffer

			testRootCmd := &cobra.Command{Use: "sshm"}
			testRootCmd.AddCommand(removeCmd)

			args := append([]string{"remove"}, tt.args...)
			testRootCmd.SetArgs(args)
			testRootCmd.SetOut(&output)
			testRootCmd.SetErr(&output)

			err := testRootCmd.Execute()
			outputStr := output.String()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Output: %s", outputStr)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, outputStr)
			}
			if tt.contains != "" && !strings.Contains(outputStr, tt.contains) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.contains, outputStr)
			}

			// Verify that the server was actually removed (for success cases)
			if !tt.expectError {
				cfg, err := config.Load()
				if err != nil {
					t.Fatalf("Failed to load config: %v", err)
				}
				serverName := tt.args[0]
				_, err = cfg.GetServer(serverName)
				if err == nil {
					t.Errorf("Expected server '%s' to be removed, but it still exists", serverName)
				}
			}
		})
	}
}
