package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"sshm/internal/config"
	"sshm/internal/tmux"
)

// TestTUICLIConfigurationSharing tests that configuration changes in TUI reflect in CLI
func TestTUICLIConfigurationSharing(t *testing.T) {
	// Create temporary config directory
	tempDir, err := os.MkdirTemp("", "sshm_tui_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set config directory for this test
	originalConfigDir := os.Getenv("SSHM_CONFIG_DIR")
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer func() {
		if originalConfigDir != "" {
			os.Setenv("SSHM_CONFIG_DIR", originalConfigDir)
		} else {
			os.Unsetenv("SSHM_CONFIG_DIR")
		}
	}()

	// Ensure config directory exists
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Get default config path (which will use our test directory due to SSHM_CONFIG_DIR)
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	// Create initial configuration
	cfg := &config.Config{
		Servers: []config.Server{
			{
				Name:     "test-server",
				Hostname: "test.example.com",
				Port:     22,
				Username: "testuser",
				AuthType: "key",
				KeyPath:  "/path/to/key",
			},
		},
		Profiles: []config.Profile{
			{
				Name:    "test",
				Servers: []string{"test-server"},
			},
		},
	}
	
	err = cfg.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test that TUI loaded the configuration
	servers := tuiApp.config.GetServers()
	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}
	if servers[0].Name != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%s'", servers[0].Name)
	}

	// Test configuration refresh in TUI
	err = tuiApp.RefreshConfig()
	if err != nil {
		t.Errorf("Failed to refresh config in TUI: %v", err)
	}

	// Verify config consistency
	refreshedServers := tuiApp.config.GetServers()
	if len(refreshedServers) != len(servers) {
		t.Errorf("Config refresh changed server count from %d to %d", len(servers), len(refreshedServers))
	}
}

// TestTUIConnectionIntegration tests that TUI can connect to servers using CLI logic
func TestTUIConnectionIntegration(t *testing.T) {
	// Create temporary config directory
	tempDir, err := os.MkdirTemp("", "sshm_tui_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set config directory for this test
	originalConfigDir := os.Getenv("SSHM_CONFIG_DIR")
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer func() {
		if originalConfigDir != "" {
			os.Setenv("SSHM_CONFIG_DIR", originalConfigDir)
		} else {
			os.Unsetenv("SSHM_CONFIG_DIR")
		}
	}()

	// Ensure config directory exists
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create test configuration
	cfg := &config.Config{
		Servers: []config.Server{
			{
				Name:     "integration-test-server",
				Hostname: "localhost",
				Port:     22,
				Username: "testuser",
				AuthType: "key",
				KeyPath:  "/dev/null", // Use /dev/null as a dummy key path
			},
		},
		Profiles: []config.Profile{
			{
				Name:    "test",
				Servers: []string{"integration-test-server"},
			},
		},
	}
	
	// Get default config path (which will use our test directory due to SSHM_CONFIG_DIR)
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	err = cfg.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Mock tmux commands for testing
	oldExecCommand := tmux.GetExecCommand()
	defer tmux.SetExecCommand(oldExecCommand)

	mockCalls := make(map[string]int)
	tmux.SetExecCommand(func(command string, args ...string) *exec.Cmd {
		cmdString := fmt.Sprintf("%s %s", command, strings.Join(args, " "))
		mockCalls[cmdString]++
		
		// Create a mock command that will succeed
		cmd := exec.Command("true") // 'true' command always succeeds
		return cmd
	})

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test that TUI can build SSH commands like CLI
	server, err := tuiApp.config.GetServer("integration-test-server")
	if err != nil {
		t.Fatalf("Failed to get server config: %v", err)
	}

	sshCommand, err := tuiApp.buildSSHCommand(*server)
	if err != nil {
		t.Errorf("Failed to build SSH command: %v", err)
	}

	expectedCommand := "ssh -t testuser@localhost -i /dev/null -o ServerAliveInterval=60 -o ServerAliveCountMax=3"
	if sshCommand != expectedCommand {
		t.Errorf("Expected SSH command '%s', got '%s'", expectedCommand, sshCommand)
	}

	// Test tmux session creation through TUI
	sessionName, wasExisting, err := tuiApp.tmuxManager.ConnectToServer("integration-test-server", sshCommand)
	if err != nil {
		t.Errorf("Failed to connect to server through TUI: %v", err)
	}

	if wasExisting {
		t.Error("Expected new session, but got existing session")
	}

	expectedSessionName := "integration-test-server"
	if sessionName != expectedSessionName {
		t.Errorf("Expected session name '%s', got '%s'", expectedSessionName, sessionName)
	}

	// Verify that tmux commands were called
	if mockCalls["tmux -V"] == 0 {
		t.Error("Expected tmux version check to be called")
	}
	if mockCalls["tmux new-session -d -s integration-test-server"] == 0 {
		t.Error("Expected tmux session creation to be called")
	}
}

// TestTUISessionManagement tests TUI integration with tmux session operations
func TestTUISessionManagement(t *testing.T) {
	// Mock tmux for testing
	oldExecCommand := tmux.GetExecCommand()
	defer tmux.SetExecCommand(oldExecCommand)

	// Mock existing sessions
	mockSessions := []string{"test-session-1", "test-session-2", "other-session"}
	tmux.SetExecCommand(func(command string, args ...string) *exec.Cmd {
		if command == "tmux" && len(args) > 0 && args[0] == "list-sessions" {
			// Return mock session list
			cmd := exec.Command("echo", strings.Join(mockSessions, "\n"))
			return cmd
		}
		// Default to success for other commands
		cmd := exec.Command("true")
		return cmd
	})

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test session listing
	sessions, err := tuiApp.tmuxManager.ListSessions()
	if err != nil {
		t.Errorf("Failed to list sessions: %v", err)
	}

	if len(sessions) != len(mockSessions) {
		t.Errorf("Expected %d sessions, got %d", len(mockSessions), len(sessions))
	}

	// Verify session names
	for i, expectedSession := range mockSessions {
		if i < len(sessions) && sessions[i] != expectedSession {
			t.Errorf("Expected session '%s' at index %d, got '%s'", expectedSession, i, sessions[i])
		}
	}

	// Test session refresh functionality
	err = tuiApp.refreshSessions()
	if err != nil {
		t.Errorf("Failed to refresh sessions: %v", err)
	}

	// Verify sessions were updated
	if len(tuiApp.sessions) != len(mockSessions) {
		t.Errorf("Expected %d sessions after refresh, got %d", len(mockSessions), len(tuiApp.sessions))
	}
}

// TestTUIErrorHandling tests error handling for CLI integration failures
func TestTUIErrorHandling(t *testing.T) {
	// Create temporary config directory
	tempDir, err := os.MkdirTemp("", "sshm_tui_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set config directory for this test
	originalConfigDir := os.Getenv("SSHM_CONFIG_DIR")
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer func() {
		if originalConfigDir != "" {
			os.Setenv("SSHM_CONFIG_DIR", originalConfigDir)
		} else {
			os.Unsetenv("SSHM_CONFIG_DIR")
		}
	}()

	// Test with invalid configuration file
	invalidConfigPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(invalidConfigPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Try to create TUI app with invalid config
	_, err = NewTUIApp()
	if err == nil {
		t.Error("Expected error when loading invalid configuration, but got none")
	}

	// Test with valid but empty configuration
	emptyConfig := &config.Config{Servers: []config.Server{}}
	// Get default config path (which will use our test directory due to SSHM_CONFIG_DIR)
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	err = emptyConfig.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Failed to save empty config: %v", err)
	}

	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app with empty config: %v", err)
	}

	// Test error handling for missing server
	server, err := tuiApp.config.GetServer("nonexistent-server")
	if err == nil {
		t.Error("Expected error when getting nonexistent server, but got none")
	}
	if server != nil {
		t.Error("Expected nil server for nonexistent server, but got a server")
	}

	// Test error handling for tmux unavailable
	oldExecCommand := tmux.GetExecCommand()
	defer tmux.SetExecCommand(oldExecCommand)

	// Mock tmux as unavailable
	tmux.SetExecCommand(func(command string, args ...string) *exec.Cmd {
		if command == "tmux" && len(args) > 0 && args[0] == "-V" {
			// Return failure for version check
			cmd := exec.Command("false")
			return cmd
		}
		cmd := exec.Command("true")
		return cmd
	})

	// Test that TUI handles tmux unavailability gracefully
	if tuiApp.tmuxManager.IsAvailable() {
		t.Error("Expected tmux to be unavailable, but IsAvailable returned true")
	}

	// Test connection attempt with tmux unavailable
	_, _, err = tuiApp.tmuxManager.ConnectToServer("test", "ssh test")
	if err == nil {
		t.Error("Expected error when connecting with tmux unavailable, but got none")
	}
	if !strings.Contains(err.Error(), "tmux is not available") {
		t.Errorf("Expected 'tmux is not available' error, got: %v", err)
	}
}

// TestConfigurationConsistency tests that configuration remains consistent between TUI operations
func TestConfigurationConsistency(t *testing.T) {
	// Create temporary config directory
	tempDir, err := os.MkdirTemp("", "sshm_consistency_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set config directory for this test
	originalConfigDir := os.Getenv("SSHM_CONFIG_DIR")
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer func() {
		if originalConfigDir != "" {
			os.Setenv("SSHM_CONFIG_DIR", originalConfigDir)
		} else {
			os.Unsetenv("SSHM_CONFIG_DIR")
		}
	}()

	// Create initial configuration
	initialServers := []config.Server{
		{
			Name:     "server1",
			Hostname: "host1.example.com",
			Port:     22,
			Username: "user1",
			AuthType: "key",
			KeyPath:  "/path/to/key1",
		},
		{
			Name:     "server2", 
			Hostname: "host2.example.com",
			Port:     2222,
			Username: "user2",
			AuthType: "password",
		},
	}

	cfg := &config.Config{
		Servers: initialServers,
		Profiles: []config.Profile{
			{
				Name:    "test",
				Servers: []string{"server1", "server2"},
			},
			{
				Name:    "dev",
				Servers: []string{"server1"},
			},
		},
	}
	// Get default config path (which will use our test directory due to SSHM_CONFIG_DIR)
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	err = cfg.SaveToPath(configPath)
	if err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Create TUI app
	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test that TUI loaded all servers correctly
	loadedServers := tuiApp.config.GetServers()
	if len(loadedServers) != len(initialServers) {
		t.Errorf("Expected %d servers, got %d", len(initialServers), len(loadedServers))
	}

	// Test server deletion consistency
	err = tuiApp.deleteServerFromConfig("server1")
	if err != nil {
		t.Errorf("Failed to delete server: %v", err)
	}

	// Verify server was removed from config
	remainingServers := tuiApp.config.GetServers()
	if len(remainingServers) != 1 {
		t.Errorf("Expected 1 server after deletion, got %d", len(remainingServers))
	}
	if len(remainingServers) > 0 && remainingServers[0].Name != "server2" {
		t.Errorf("Expected remaining server to be 'server2', got '%s'", remainingServers[0].Name)
	}

	// Test that configuration file was updated
	reloadedCfg, err := config.Load()
	if err != nil {
		t.Errorf("Failed to reload config: %v", err)
	}
	
	reloadedServers := reloadedCfg.GetServers()
	if len(reloadedServers) != 1 {
		t.Errorf("Expected 1 server in reloaded config, got %d", len(reloadedServers))
	}

	// Test that profiles were updated after server deletion
	profiles := reloadedCfg.GetProfiles()
	// After deleting server1, "dev" profile should no longer exist since it only contained server1
	for _, profile := range profiles {
		if profile.Name == "dev" {
			t.Error("Expected 'dev' profile to be removed after deleting server1, but it still exists")
		}
	}
	
	// Also check that dev profile was cleaned up in the profiles list
	for _, prof := range reloadedCfg.Profiles {
		if prof.Name == "dev" && len(prof.Servers) == 0 {
			t.Error("Expected 'dev' profile to be removed completely, but empty profile still exists")
		}
	}
}