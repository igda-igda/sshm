package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// execCommand is a variable to allow mocking in tests
var execCommand = exec.Command

// GetExecCommand returns the current execCommand function for testing
func GetExecCommand() func(string, ...string) *exec.Cmd {
	return execCommand
}

// SetExecCommand sets the execCommand function for testing
func SetExecCommand(fn func(string, ...string) *exec.Cmd) {
	execCommand = fn
}

// Manager handles tmux session operations
type Manager struct {
	// For testing purposes, we can inject existing sessions
	existingSessions []string
}

// NewManager creates a new tmux manager instance
func NewManager() *Manager {
	return &Manager{}
}

// IsAvailable checks if tmux is installed and available on the system
func (m *Manager) IsAvailable() bool {
	cmd := execCommand("tmux", "-V")
	err := cmd.Run()
	return err == nil
}

// CreateSession creates a new tmux session with the given name
func (m *Manager) CreateSession(sessionName string) error {
	cmd := execCommand("tmux", "new-session", "-d", "-s", sessionName)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create tmux session '%s': %w", sessionName, err)
	}
	return nil
}

// ListSessions returns a list of existing tmux session names
func (m *Manager) ListSessions() ([]string, error) {
	// If we have mock sessions for testing, use those
	if m.existingSessions != nil {
		return m.existingSessions, nil
	}

	cmd := execCommand("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// If tmux list-sessions fails, it might be because no sessions exist
		// Check if it's a "no server running" error
		if strings.Contains(err.Error(), "no server running") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	sessionNames := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(sessionNames) == 1 && sessionNames[0] == "" {
		return []string{}, nil
	}
	return sessionNames, nil
}

// normalizeSessionName converts session names to match tmux's naming behavior
// tmux converts dots and other special characters to underscores
func normalizeSessionName(name string) string {
	// Replace dots with underscores to match tmux behavior
	normalized := strings.ReplaceAll(name, ".", "_")
	// Add other character replacements as needed for tmux compatibility
	return normalized
}

// generateUniqueSessionName creates a unique session name by appending a counter if needed
func (m *Manager) generateUniqueSessionName(baseName string) string {
	// Normalize the base name to match tmux behavior
	normalizedBaseName := normalizeSessionName(baseName)
	
	sessions, err := m.ListSessions()
	if err != nil {
		// If we can't list sessions, just return the normalized base name
		return normalizedBaseName
	}

	// Check if normalized base name is available
	if !contains(sessions, normalizedBaseName) {
		return normalizedBaseName
	}

	// Find the lowest available counter
	counter := 1
	for {
		candidateName := fmt.Sprintf("%s-%d", normalizedBaseName, counter)
		if !contains(sessions, candidateName) {
			return candidateName
		}
		counter++
	}
}

// SendKeys sends a command to a tmux session
func (m *Manager) SendKeys(sessionName, command string) error {
	cmd := execCommand("tmux", "send-keys", "-t", sessionName, command, "Enter")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to send keys to session '%s': %w", sessionName, err)
	}
	return nil
}

// AttachSession attaches to a tmux session
func (m *Manager) AttachSession(sessionName string) error {
	cmd := execCommand("tmux", "attach-session", "-t", sessionName)
	// Set up the command to inherit stdin, stdout, stderr so it can take over the terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to attach to session '%s': %w", sessionName, err)
	}
	return nil
}

// ConnectToServer creates a tmux session and connects to a server via SSH, or reattaches to existing session
func (m *Manager) ConnectToServer(serverName, sshCommand string) (string, bool, error) {
	// Check if tmux is available
	if !m.IsAvailable() {
		return "", false, fmt.Errorf("tmux is not available on this system")
	}

	// Normalize the session name to match tmux behavior
	normalizedSessionName := normalizeSessionName(serverName)
	
	// Check if session already exists
	if m.SessionExists(normalizedSessionName) {
		// Session exists, just return it for reattachment
		return normalizedSessionName, true, nil
	}

	// Session doesn't exist, create a new one
	// Generate unique session name (this will handle conflicts with other sessions)
	sessionName := m.generateUniqueSessionName(serverName)

	// Create the tmux session
	err := m.CreateSession(sessionName)
	if err != nil {
		return "", false, err
	}

	// Send the SSH command to the session
	err = m.SendKeys(sessionName, sshCommand)
	if err != nil {
		return "", false, err
	}

	return sessionName, false, nil
}

// SessionExists checks if a session with the given name exists
func (m *Manager) SessionExists(sessionName string) bool {
	sessions, err := m.ListSessions()
	if err != nil {
		return false
	}
	return contains(sessions, sessionName)
}

// KillSession terminates a tmux session
func (m *Manager) KillSession(sessionName string) error {
	cmd := execCommand("tmux", "kill-session", "-t", sessionName)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to kill session '%s': %w", sessionName, err)
	}
	return nil
}

// Server interface for tmux operations - avoiding circular import
type Server interface {
	GetName() string
	GetHostname() string
	GetPort() int
	GetUsername() string
	GetAuthType() string
	GetKeyPath() string
	Validate() error
}

// ConnectToProfile creates a tmux session for a profile with multiple windows for servers
func (m *Manager) ConnectToProfile(profileName string, servers []Server) (string, bool, error) {
	// Check if tmux is available
	if !m.IsAvailable() {
		return "", false, fmt.Errorf("tmux is not available on this system")
	}

	// Normalize the session name to match tmux behavior  
	normalizedSessionName := normalizeSessionName(profileName)

	// Check if session already exists
	if m.SessionExists(normalizedSessionName) {
		// Session exists, just return it for reattachment
		return normalizedSessionName, true, nil
	}

	// Session doesn't exist, create a new one
	// Generate unique session name (this will handle conflicts with other sessions)
	sessionName := m.generateUniqueSessionName(profileName)

	// Create the tmux session
	err := m.CreateSession(sessionName)
	if err != nil {
		return "", false, err
	}

	// Create windows for each server and send SSH commands
	for i, server := range servers {
		serverName := server.GetName()
		
		// Build SSH command for this server
		sshCommand, err := m.buildSSHCommand(server)
		if err != nil {
			return "", false, fmt.Errorf("failed to build SSH command for %s: %w", serverName, err)
		}

		// Create new window for this server (except for the first one which uses the default window)
		if i > 0 {
			err = m.CreateWindow(sessionName, serverName)
			if err != nil {
				return "", false, fmt.Errorf("failed to create window for server %s: %w", serverName, err)
			}
		} else {
			// Rename the default window to the first server name
			err = m.RenameWindow(sessionName, "0", serverName)
			if err != nil {
				return "", false, fmt.Errorf("failed to rename first window to %s: %w", serverName, err)
			}
		}

		// Send the SSH command to the appropriate window
		windowTarget := fmt.Sprintf("%s:%d", sessionName, i)
		err = m.SendKeysToWindow(windowTarget, sshCommand)
		if err != nil {
			return "", false, fmt.Errorf("failed to send SSH command to window %s: %w", windowTarget, err)
		}
	}

	return sessionName, false, nil
}

// buildSSHCommand builds an SSH command string for a server
func (m *Manager) buildSSHCommand(server Server) (string, error) {
	// Validate server configuration
	if err := server.Validate(); err != nil {
		return "", fmt.Errorf("invalid server configuration: %w", err)
	}

	// Build base SSH command with pseudo-terminal allocation
	sshCmd := fmt.Sprintf("ssh -t %s@%s", server.GetUsername(), server.GetHostname())

	// Add port if not default
	if server.GetPort() != 22 {
		sshCmd += fmt.Sprintf(" -p %d", server.GetPort())
	}

	// Add key-specific options
	if server.GetAuthType() == "key" && server.GetKeyPath() != "" {
		sshCmd += fmt.Sprintf(" -i %s", server.GetKeyPath())
	}

	// Add common SSH options
	sshCmd += " -o ServerAliveInterval=60 -o ServerAliveCountMax=3"

	return sshCmd, nil
}

// CreateWindow creates a new window in an existing tmux session
func (m *Manager) CreateWindow(sessionName, windowName string) error {
	cmd := execCommand("tmux", "new-window", "-t", sessionName, "-n", windowName)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create window '%s' in session '%s': %w", windowName, sessionName, err)
	}
	return nil
}

// RenameWindow renames an existing window in a tmux session
func (m *Manager) RenameWindow(sessionName, windowNumber, newName string) error {
	windowTarget := fmt.Sprintf("%s:%s", sessionName, windowNumber)
	cmd := execCommand("tmux", "rename-window", "-t", windowTarget, newName)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to rename window %s to '%s': %w", windowTarget, newName, err)
	}
	return nil
}

// SendKeysToWindow sends a command to a specific window in a tmux session
func (m *Manager) SendKeysToWindow(windowTarget, command string) error {
	cmd := execCommand("tmux", "send-keys", "-t", windowTarget, command, "Enter")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to send keys to window '%s': %w", windowTarget, err)
	}
	return nil
}

// contains checks if a string slice contains a specific value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
