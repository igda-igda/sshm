package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

// execCommand is a variable to allow mocking in tests
var execCommand = exec.Command

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
	// For attach, we want to run the command and let it take over the terminal
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to attach to session '%s': %w", sessionName, err)
	}
	return nil
}

// ConnectToServer creates a tmux session and connects to a server via SSH
func (m *Manager) ConnectToServer(serverName, sshCommand string) (string, error) {
	// Check if tmux is available
	if !m.IsAvailable() {
		return "", fmt.Errorf("tmux is not available on this system")
	}

	// Generate unique session name
	sessionName := m.generateUniqueSessionName(serverName)

	// Create the tmux session
	err := m.CreateSession(sessionName)
	if err != nil {
		return "", err
	}

	// Send the SSH command to the session
	err = m.SendKeys(sessionName, sshCommand)
	if err != nil {
		return "", err
	}

	return sessionName, nil
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

// contains checks if a string slice contains a specific value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
