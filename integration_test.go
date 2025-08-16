package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sshm/cmd"
	"sshm/internal/config"
)

// Integration tests for end-to-end workflows
func TestIntegrationWorkflow(t *testing.T) {
	t.Run("complete workflow: add → list → remove", func(t *testing.T) {
		// Setup temporary config directory
		tmpDir, err := os.MkdirTemp("", "sshm-integration-workflow-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Set environment variable to use test config directory
		originalConfigDir := os.Getenv("SSHM_CONFIG_DIR")
		os.Setenv("SSHM_CONFIG_DIR", tmpDir)
		defer func() {
			if originalConfigDir != "" {
				os.Setenv("SSHM_CONFIG_DIR", originalConfigDir)
			} else {
				os.Unsetenv("SSHM_CONFIG_DIR")
			}
		}()
		// Step 1: Start with empty list
		output := runCLICommand(t, []string{"list"})
		if !strings.Contains(output, "No servers configured") {
			t.Errorf("Expected empty server list, got: %s", output)
		}

		// Step 2: Add a server using mock input
		addInputs := []string{
			"test.example.com",  // hostname
			"22",                // port
			"testuser",          // username
			"key",               // auth type
			"~/.ssh/test_rsa",   // key path
			"n",                 // passphrase protected
		}
		
		restoreStdin := setupMockStdin(addInputs)
		output = runCLICommand(t, []string{"add", "test-server"})
		restoreStdin()
		
		if !strings.Contains(output, "Server 'test-server' added successfully") {
			t.Errorf("Expected success message, got: %s", output)
		}

		// Step 3: Verify server appears in list
		output = runCLICommand(t, []string{"list"})
		expectedContent := []string{"test-server", "test.example.com", "testuser", "key"}
		for _, content := range expectedContent {
			if !strings.Contains(output, content) {
				t.Errorf("Expected list to contain '%s', got: %s", content, output)
			}
		}

		// Step 4: Remove the server
		removeInputs := []string{"y"} // confirm removal
		restoreStdin = setupMockStdin(removeInputs)
		output = runCLICommand(t, []string{"remove", "test-server"})
		restoreStdin()
		
		if !strings.Contains(output, "Server 'test-server' removed successfully") {
			t.Errorf("Expected removal success message, got: %s", output)
		}

		// Step 5: Verify server is gone
		output = runCLICommand(t, []string{"list"})
		if !strings.Contains(output, "No servers configured") {
			t.Errorf("Expected empty server list after removal, got: %s", output)
		}
	})

	t.Run("error handling workflow", func(t *testing.T) {
		// Setup temporary config directory
		tmpDir, err := os.MkdirTemp("", "sshm-integration-error-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Set environment variable to use test config directory
		originalConfigDir := os.Getenv("SSHM_CONFIG_DIR")
		os.Setenv("SSHM_CONFIG_DIR", tmpDir)
		defer func() {
			if originalConfigDir != "" {
				os.Setenv("SSHM_CONFIG_DIR", originalConfigDir)
			} else {
				os.Unsetenv("SSHM_CONFIG_DIR")
			}
		}()
		// Test adding duplicate server
		addInputs := []string{
			"duplicate.example.com",
			"22",
			"user",
			"key",
			"~/.ssh/key",
			"n",
		}
		
		// Add first server
		restoreStdin := setupMockStdin(addInputs)
		output := runCLICommand(t, []string{"add", "error-test-server"})
		restoreStdin()
		
		if !strings.Contains(output, "added successfully") {
			t.Errorf("Expected first add to succeed, got: %s", output)
		}

		// Try to add duplicate
		restoreStdin = setupMockStdin(addInputs)
		output = runCLICommandExpectError(t, []string{"add", "error-test-server"})
		restoreStdin()
		
		if !strings.Contains(output, "already exists") {
			t.Errorf("Expected duplicate error, got: %s", output)
		}

		// Test removing non-existent server
		output = runCLICommandExpectError(t, []string{"remove", "non-existent"})
		if !strings.Contains(output, "not found") {
			t.Errorf("Expected not found error, got: %s", output)
		}

		// Test connecting to non-existent server
		output = runCLICommandExpectError(t, []string{"connect", "non-existent"})
		if !strings.Contains(output, "not found") {
			t.Errorf("Expected not found error, got: %s", output)
		}
	})

	t.Run("configuration persistence", func(t *testing.T) {
		// Setup temporary config directory
		tmpDir, err := os.MkdirTemp("", "sshm-integration-persist-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Set environment variable to use test config directory
		originalConfigDir := os.Getenv("SSHM_CONFIG_DIR")
		os.Setenv("SSHM_CONFIG_DIR", tmpDir)
		defer func() {
			if originalConfigDir != "" {
				os.Setenv("SSHM_CONFIG_DIR", originalConfigDir)
			} else {
				os.Unsetenv("SSHM_CONFIG_DIR")
			}
		}()
		// Add multiple servers
		servers := []struct {
			name   string
			inputs []string
		}{
			{
				name: "prod-web",
				inputs: []string{"web.prod.com", "22", "deploy", "key", "~/.ssh/prod", "n"},
			},
			{
				name: "staging-db",
				inputs: []string{"db.staging.com", "5432", "admin", "password"},
			},
		}

		for _, server := range servers {
			restoreStdin := setupMockStdin(server.inputs)
			output := runCLICommand(t, []string{"add", server.name})
			restoreStdin()
			
			if !strings.Contains(output, "added successfully") {
				t.Errorf("Failed to add server %s: %s", server.name, output)
			}
		}

		// Verify both servers persist
		output := runCLICommand(t, []string{"list"})
		if !strings.Contains(output, "prod-web") || !strings.Contains(output, "staging-db") {
			t.Errorf("Servers not persisted correctly: %s", output)
		}

		// Check config file was created with correct permissions
		configPath := filepath.Join(tmpDir, "config.yaml")
		info, err := os.Stat(configPath)
		if err != nil {
			t.Errorf("Config file not created: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("Expected file permissions 0600, got %v", info.Mode().Perm())
		}

		// Verify config file content
		cfg, err := config.LoadFromPath(configPath)
		if err != nil {
			t.Errorf("Failed to load config: %v", err)
		}
		if len(cfg.Servers) != 2 {
			t.Errorf("Expected 2 servers in config, got %d", len(cfg.Servers))
		}
	})
}

func TestServerValidationIntegration(t *testing.T) {

	tests := []struct {
		name        string
		inputs      []string
		expectError bool
		errorText   string
	}{
		{
			name:        "invalid port",
			inputs:      []string{"test.com", "invalid-port"},
			expectError: true,
			errorText:   "invalid port",
		},
		{
			name:        "empty hostname",
			inputs:      []string{"", "22"},
			expectError: true,
			errorText:   "Hostname is required",
		},
		{
			name:        "valid key server",
			inputs:      []string{"valid.com", "22", "user", "key", "~/.ssh/key", "n"},
			expectError: false,
		},
		{
			name:        "valid password server",
			inputs:      []string{"valid.com", "22", "user", "password"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temporary config directory for each test
			tmpDir, err := os.MkdirTemp("", "sshm-validation-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			originalConfigDir := os.Getenv("SSHM_CONFIG_DIR")
			os.Setenv("SSHM_CONFIG_DIR", tmpDir)
			defer func() {
				if originalConfigDir != "" {
					os.Setenv("SSHM_CONFIG_DIR", originalConfigDir)
				} else {
					os.Unsetenv("SSHM_CONFIG_DIR")
				}
			}()

			restoreStdin := setupMockStdin(tt.inputs)
			defer restoreStdin()

			if tt.expectError {
				output := runCLICommandExpectError(t, []string{"add", "test-" + tt.name})
				if !strings.Contains(output, tt.errorText) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorText, output)
				}
			} else {
				output := runCLICommand(t, []string{"add", "test-" + tt.name})
				if !strings.Contains(output, "added successfully") {
					t.Errorf("Expected success, got: %s", output)
				}
			}
		})
	}
}

// Helper functions for integration tests
func runCLICommand(t *testing.T, args []string) string {
	var output bytes.Buffer
	
	// Create a new root command for this test to avoid state conflicts
	testRootCmd := cmd.CreateRootCommand()
	testRootCmd.SetArgs(args)
	testRootCmd.SetOut(&output)
	testRootCmd.SetErr(&output)
	
	// Execute command
	err := testRootCmd.Execute()
	if err != nil {
		t.Fatalf("Command failed: %v, output: %s", err, output.String())
	}
	
	return output.String()
}

func runCLICommandExpectError(t *testing.T, args []string) string {
	var output bytes.Buffer
	
	// Create a new root command for this test to avoid state conflicts
	testRootCmd := cmd.CreateRootCommand()
	testRootCmd.SetArgs(args)
	testRootCmd.SetOut(&output)
	testRootCmd.SetErr(&output)
	
	// Execute command (expecting error)
	err := testRootCmd.Execute()
	if err == nil {
		t.Fatalf("Expected command to fail, but it succeeded. Output: %s", output.String())
	}
	
	return output.String()
}

// Mock stdin for testing
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