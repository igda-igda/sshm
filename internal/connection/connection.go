package connection

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"sshm/internal/auth"
	"sshm/internal/config"
	"sshm/internal/history"
	sshsdk "sshm/internal/ssh"
	"sshm/internal/tmux"
)

// Manager handles SSH connections with history tracking
type Manager struct {
	historyManager *history.HistoryManager
	tmuxManager    *tmux.Manager
}

// NewManager creates a new connection manager with history tracking
func NewManager() (*Manager, error) {
	// Initialize history manager
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".sshm")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	historyPath := filepath.Join(configDir, "history.db")
	historyManager, err := history.NewHistoryManager(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize history manager: %w", err)
	}

	return &Manager{
		historyManager: historyManager,
		tmuxManager:    tmux.NewManager(),
	}, nil
}

// Close closes the connection manager and its resources
func (m *Manager) Close() error {
	if m.historyManager != nil {
		return m.historyManager.Close()
	}
	return nil
}

// ConnectToServer connects to a single server with history tracking
func (m *Manager) ConnectToServer(server config.Server) (string, bool, error) {
	startTime := time.Now()
	
	// Record connection attempt start
	historyEntry := history.ConnectionHistoryEntry{
		ServerName:     server.Name,
		ProfileName:    "", // Empty for single server connections
		Host:           server.Hostname,
		User:           server.Username,
		Port:           server.Port,
		ConnectionType: "single",
		Status:         "attempting",
		StartTime:      startTime,
	}

	// Record the initial connection attempt
	connectionID, err := m.historyManager.RecordConnection(historyEntry)
	if err != nil {
		// Log error but don't fail the connection
		fmt.Printf("Warning: Failed to record connection history: %v\n", err)
	}

	// Test SSH connectivity first
	if err := m.testSSHConnectivity(server); err != nil {
		// Update history with failure
		if connectionID > 0 {
			m.historyManager.UpdateConnectionEnd(connectionID, time.Now(), "failed", err.Error())
		}
		return "", false, fmt.Errorf("SSH connectivity test failed: %w", err)
	}

	// Build SSH command
	sshCommand, err := buildSSHCommand(server)
	if err != nil {
		// Update history with failure
		if connectionID > 0 {
			m.historyManager.UpdateConnectionEnd(connectionID, time.Now(), "failed", err.Error())
		}
		return "", false, fmt.Errorf("failed to build SSH command: %w", err)
	}

	// Create tmux session
	sessionName, wasExisting, err := m.tmuxManager.ConnectToServer(server.Name, sshCommand)
	if err != nil {
		// Update history with failure
		if connectionID > 0 {
			m.historyManager.UpdateConnectionEnd(connectionID, time.Now(), "failed", err.Error())
		}
		return "", false, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Update history with success
	if connectionID > 0 {
		historyEntry.SessionID = sessionName
		historyEntry.Status = "success"
		m.historyManager.UpdateConnectionEnd(connectionID, time.Now(), "success", "")
	}

	return sessionName, wasExisting, nil
}

// ConnectToProfile connects to multiple servers in a profile with history tracking
func (m *Manager) ConnectToProfile(profileName string, servers []config.Server) (string, bool, error) {
	startTime := time.Now()

	// Record connection attempt for the profile
	profileHistoryEntry := history.ConnectionHistoryEntry{
		ServerName:     profileName, // Use profile name as server name for group connections
		ProfileName:    profileName,
		Host:           fmt.Sprintf("%d servers", len(servers)),
		User:           "multiple",
		Port:           0,
		ConnectionType: "group",
		Status:         "attempting",
		StartTime:      startTime,
	}

	profileConnectionID, err := m.historyManager.RecordConnection(profileHistoryEntry)
	if err != nil {
		fmt.Printf("Warning: Failed to record profile connection history: %v\n", err)
	}

	// Record individual server connection attempts and test connectivity
	serverConnectionIDs := make(map[string]int64)
	var connectivityErrors []string
	
	for _, server := range servers {
		// Record the attempt first
		serverHistoryEntry := history.ConnectionHistoryEntry{
			ServerName:     server.Name,
			ProfileName:    profileName,
			Host:           server.Hostname,
			User:           server.Username,
			Port:           server.Port,
			ConnectionType: "group",
			Status:         "attempting",
			StartTime:      startTime,
		}

		serverID, err := m.historyManager.RecordConnection(serverHistoryEntry)
		if err == nil {
			serverConnectionIDs[server.Name] = serverID
		}
		
		// Test connectivity and record failure if needed
		if connectErr := m.testSSHConnectivity(server); connectErr != nil {
			connectivityErrors = append(connectivityErrors, 
				fmt.Sprintf("%s: %v", server.Name, connectErr))
			
			// Update server history with failure
			if serverID > 0 {
				m.historyManager.UpdateConnectionEnd(serverID, time.Now(), "failed", connectErr.Error())
			}
		}
	}

	if len(connectivityErrors) > 0 {
		errorMsg := fmt.Sprintf("SSH connectivity failed for %d servers: %v", 
			len(connectivityErrors), connectivityErrors)
		
		// Update profile history with failure
		if profileConnectionID > 0 {
			m.historyManager.UpdateConnectionEnd(profileConnectionID, time.Now(), "failed", errorMsg)
		}
		
		return "", false, fmt.Errorf("SSH connectivity failed: %s", errorMsg)
	}

	// Convert config.Server slice to tmux.Server interface slice
	tmuxServers := make([]tmux.Server, len(servers))
	for i, server := range servers {
		tmuxServers[i] = &server
	}

	// Create group session
	sessionName, wasExisting, err := m.tmuxManager.ConnectToProfile(profileName, tmuxServers)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to create group session: %v", err)
		
		// Update all connection histories with failure
		if profileConnectionID > 0 {
			m.historyManager.UpdateConnectionEnd(profileConnectionID, time.Now(), "failed", errorMsg)
		}
		for _, serverID := range serverConnectionIDs {
			m.historyManager.UpdateConnectionEnd(serverID, time.Now(), "failed", 
				fmt.Sprintf("group session creation failed: %v", err))
		}
		
		return "", false, fmt.Errorf("SSH connectivity failed: %s", errorMsg)
	}

	// Update histories with success
	endTime := time.Now()
	if profileConnectionID > 0 {
		profileHistoryEntry.SessionID = sessionName
		m.historyManager.UpdateConnectionEnd(profileConnectionID, endTime, "success", "")
	}

	for _, serverID := range serverConnectionIDs {
		m.historyManager.UpdateConnectionEnd(serverID, endTime, "success", "")
	}

	return sessionName, wasExisting, nil
}

// GetConnectionHistory retrieves connection history with filtering
func (m *Manager) GetConnectionHistory(filter history.HistoryFilter) ([]history.ConnectionHistoryEntry, error) {
	return m.historyManager.GetConnectionHistory(filter)
}

// GetConnectionStats retrieves connection statistics for a server
func (m *Manager) GetConnectionStats(serverName, profileName string) (*history.ConnectionStats, error) {
	return m.historyManager.GetConnectionStats(serverName, profileName)
}

// CleanupOldHistory removes old connection history
func (m *Manager) CleanupOldHistory(retentionPeriod time.Duration) (int64, error) {
	return m.historyManager.CleanupOldHistory(retentionPeriod)
}

// GetRecentActivity gets recent connection activity summary
func (m *Manager) GetRecentActivity(hours int) (map[string]int, error) {
	return m.historyManager.GetRecentActivity(hours)
}

// IsAvailable checks if tmux is available
func (m *Manager) IsAvailable() bool {
	return m.tmuxManager.IsAvailable()
}

// AttachSession attaches to a tmux session
func (m *Manager) AttachSession(sessionName string) error {
	return m.tmuxManager.AttachSession(sessionName)
}

// GetHistoryManager returns the history manager (for testing)
func (m *Manager) GetHistoryManager() *history.HistoryManager {
	return m.historyManager
}

// testSSHConnectivity tests SSH connectivity to a server
func (m *Manager) testSSHConnectivity(server config.Server) error {
	// Create SSH client configuration
	sshConfig := sshsdk.ClientConfig{
		Hostname: server.Hostname,
		Port:     server.Port,
		Username: server.Username,
		Timeout:  10 * time.Second, // 10 second timeout for connectivity test
	}

	// Determine authentication method
	var authMethod ssh.AuthMethod
	var err error

	switch server.AuthType {
	case "key":
		if server.KeyPath != "" {
			// For connectivity test, try without passphrase first
			authMethod, err = sshsdk.NewKeyAuth(server.KeyPath, "")
			if err != nil {
				return fmt.Errorf("failed to create key auth: %w", err)
			}
		} else {
			// Try SSH agent
			authMethod, err = sshsdk.NewAgentAuth()
			if err != nil {
				return fmt.Errorf("failed to create agent auth: %w", err)
			}
		}
	case "password":
		// For connectivity test, try to retrieve password from keyring
		if server.UseKeyring && server.KeyringID != "" {
			// Initialize password manager
			passwordManager, err := auth.NewPasswordManager("auto") // Use auto to select best available backend
			if err != nil {
				return fmt.Errorf("failed to initialize password manager: %w", err)
			}

			// Retrieve password from keyring
			password, err := passwordManager.RetrieveServerPassword(&server)
			if err != nil {
				return fmt.Errorf("failed to retrieve password from keyring: %w", err)
			}

			authMethod = sshsdk.NewPasswordAuth(password)
		} else {
			// Skip connectivity test for plaintext password auth to avoid prompting user during test
			return nil
		}
	default:
		return fmt.Errorf("unsupported auth type: %s", server.AuthType)
	}

	// Test the connection
	return sshsdk.TestConnection(sshConfig, authMethod)
}

// buildSSHCommand builds the SSH command string for a server
func buildSSHCommand(server config.Server) (string, error) {
	if err := server.Validate(); err != nil {
		return "", fmt.Errorf("invalid server configuration: %w", err)
	}

	var sshCmd string

	// Handle password authentication with keyring
	if server.AuthType == "password" && server.UseKeyring && server.KeyringID != "" {
		// Try to use sshpass for non-interactive password authentication
		// Note: This requires sshpass to be installed on the system
		
		// Initialize password manager to retrieve password
		passwordManager, err := auth.NewPasswordManager("auto")
		if err != nil {
			// Fall back to interactive SSH if password manager fails
			sshCmd = fmt.Sprintf("ssh -t %s@%s", server.Username, server.Hostname)
		} else {
			password, err := passwordManager.RetrieveServerPassword(&server)
			if err != nil {
				// Fall back to interactive SSH if password retrieval fails
				sshCmd = fmt.Sprintf("ssh -t %s@%s", server.Username, server.Hostname)
			} else {
				// Use sshpass with retrieved password
				sshCmd = fmt.Sprintf("sshpass -p '%s' ssh -t %s@%s", password, server.Username, server.Hostname)
			}
		}
	} else {
		// Build base SSH command with pseudo-terminal allocation
		sshCmd = fmt.Sprintf("ssh -t %s@%s", server.Username, server.Hostname)
	}

	// Add port if not default
	if server.Port != 22 {
		sshCmd += fmt.Sprintf(" -p %d", server.Port)
	}

	// Add key-specific options
	if server.AuthType == "key" && server.KeyPath != "" {
		sshCmd += fmt.Sprintf(" -i %s", server.KeyPath)
	}

	// Add common SSH options
	sshCmd += " -o ServerAliveInterval=60 -o ServerAliveCountMax=3"

	return sshCmd, nil
}