package auth

import (
	"fmt"
	"sshm/internal/config"
	"sshm/internal/keyring"
)

// PasswordManager handles secure storage and retrieval of server passwords
type PasswordManager struct {
	keyringManager keyring.KeyringManager
}

// NewPasswordManager creates a new password manager with keyring backend
func NewPasswordManager(service string) (*PasswordManager, error) {
	keyringManager := keyring.NewKeyringManager(service)
	if keyringManager == nil {
		return nil, fmt.Errorf("failed to initialize keyring manager with service: %s", service)
	}

	if !keyringManager.IsAvailable() {
		return nil, fmt.Errorf("keyring service %s is not available", service)
	}

	return &PasswordManager{
		keyringManager: keyringManager,
	}, nil
}

// StoreServerPassword stores a password for a server configuration
func (pm *PasswordManager) StoreServerPassword(server *config.Server, password string) error {
	if server == nil {
		return fmt.Errorf("server configuration is required")
	}

	if server.AuthType != "password" {
		return fmt.Errorf("server %s is not configured for password authentication", server.Name)
	}

	err := pm.keyringManager.StoreServerPassword(server.Name, password)
	if err != nil {
		return fmt.Errorf("failed to store password for server %s: %w", server.Name, err)
	}

	// Update server configuration to use keyring
	server.UseKeyring = true
	server.KeyringID = keyring.GeneratePasswordKeyringID(server.Name)
	server.Password = "" // Clear plaintext password

	return nil
}

// RetrieveServerPassword retrieves a password for a server configuration
func (pm *PasswordManager) RetrieveServerPassword(server *config.Server) (string, error) {
	if server == nil {
		return "", fmt.Errorf("server configuration is required")
	}

	if server.AuthType != "password" {
		return "", fmt.Errorf("server %s is not configured for password authentication", server.Name)
	}

	// First check if password is stored in keyring
	if server.UseKeyring && server.KeyringID != "" {
		password, err := pm.keyringManager.RetrieveServerPassword(server.Name)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve password from keyring for server %s: %w", server.Name, err)
		}
		return password, nil
	}

	// Fallback to plaintext password (legacy support)
	if server.Password != "" {
		return server.Password, nil
	}

	return "", fmt.Errorf("no password found for server %s", server.Name)
}

// HasServerPassword checks if a password is available for a server
func (pm *PasswordManager) HasServerPassword(server *config.Server) bool {
	if server == nil || server.AuthType != "password" {
		return false
	}

	// Check keyring first
	if server.UseKeyring {
		return pm.keyringManager.HasServerPassword(server.Name)
	}

	// Check plaintext password
	return server.Password != ""
}

// UpdateServerPassword updates the password for a server
func (pm *PasswordManager) UpdateServerPassword(server *config.Server, newPassword string) error {
	if server == nil {
		return fmt.Errorf("server configuration is required")
	}

	if server.AuthType != "password" {
		return fmt.Errorf("server %s is not configured for password authentication", server.Name)
	}

	// Store the new password
	err := pm.keyringManager.StoreServerPassword(server.Name, newPassword)
	if err != nil {
		return fmt.Errorf("failed to update password for server %s: %w", server.Name, err)
	}

	// Ensure server is configured to use keyring
	server.UseKeyring = true
	server.KeyringID = keyring.GeneratePasswordKeyringID(server.Name)
	server.Password = "" // Clear plaintext password

	return nil
}

// DeleteServerPassword deletes the password for a server
func (pm *PasswordManager) DeleteServerPassword(server *config.Server) error {
	if server == nil {
		return fmt.Errorf("server configuration is required")
	}

	// Delete from keyring if stored there
	if server.UseKeyring {
		err := pm.keyringManager.DeleteServerPassword(server.Name)
		if err != nil {
			return fmt.Errorf("failed to delete password from keyring for server %s: %w", server.Name, err)
		}
	}

	// Clear server configuration
	server.UseKeyring = false
	server.KeyringID = ""
	server.Password = ""

	return nil
}

// MigrateServerToKeyring migrates a server's password from plaintext to keyring storage
func (pm *PasswordManager) MigrateServerToKeyring(server *config.Server) error {
	if server == nil {
		return fmt.Errorf("server configuration is required")
	}

	if server.AuthType != "password" {
		return fmt.Errorf("server %s is not configured for password authentication", server.Name)
	}

	// Skip if already using keyring
	if server.UseKeyring {
		return nil
	}

	// Get the current password
	currentPassword := server.Password
	if currentPassword == "" {
		return fmt.Errorf("no password to migrate for server %s", server.Name)
	}

	// Store in keyring
	err := pm.keyringManager.StoreServerPassword(server.Name, currentPassword)
	if err != nil {
		return fmt.Errorf("failed to migrate password to keyring for server %s: %w", server.Name, err)
	}

	// Update server configuration
	server.UseKeyring = true
	server.KeyringID = keyring.GeneratePasswordKeyringID(server.Name)
	server.Password = "" // Clear plaintext password

	return nil
}

// IsAvailable checks if the password manager's keyring backend is available
func (pm *PasswordManager) IsAvailable() bool {
	return pm.keyringManager.IsAvailable()
}

// ServiceName returns the name of the keyring service being used
func (pm *PasswordManager) ServiceName() string {
	return pm.keyringManager.ServiceName()
}

// GetKeyringManager returns the underlying keyring manager (for testing)
func (pm *PasswordManager) GetKeyringManager() keyring.KeyringManager {
	return pm.keyringManager
}