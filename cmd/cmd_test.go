package cmd

import (
  "bytes"
  "fmt"
  "io"
  "os"
  "path/filepath"
  "strings"
  "testing"

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
  mockUserInput *mockInput
  mockTmuxAvailable = true
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
      name: "invalid port",
      args: []string{"test-server"},
      inputs: []string{
        "test.com", // hostname
        "invalid",  // port
      },
      expectError: true,
      contains:    "invalid port",
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
      contains:    "server 'production-api' already exists",
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
      contains:    "server 'non-existent' not found",
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
      expectError: true, // Will fail because tmux isn't available in test
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
      contains:    "server 'non-existent' not found",
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