package keyring

import (
	"fmt"

	"sshm/internal/config"
)

// PromptFunc is a function type for prompting the user for credentials
type PromptFunc func(prompt string) (string, error)

// MigrationResult contains information about a credential migration
type MigrationResult struct {
	ServerName    string
	CredentialType string
	KeyringID     string
	Success       bool
	Error         error
}

// MigrateFromPlaintext migrates plaintext credentials from config to encrypted keyring storage
func MigrateFromPlaintext(cfg *config.Config, manager KeyringManager, promptFunc PromptFunc) ([]MigrationResult, error) {
	var results []MigrationResult

	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if manager == nil {
		return nil, fmt.Errorf("keyring manager cannot be nil")
	}

	if promptFunc == nil {
		return nil, fmt.Errorf("prompt function cannot be nil")
	}

	// Identify credentials that need migration
	for i := range cfg.Servers {
		server := &cfg.Servers[i]
		
		needsMigration, credType := identifyCredentialToMigrate(*server)
		if !needsMigration {
			continue
		}

		result := MigrationResult{
			ServerName:     server.Name,
			CredentialType: credType,
		}

		// Generate keyring ID for this credential
		keyringID := generateKeyringID(server.Name, credType)
		result.KeyringID = keyringID

		// Check if credential is already in keyring
		_, err := manager.Retrieve(keyringID)
		if err == nil {
			// Credential already exists in keyring, just update config
			server.UseKeyring = true
			server.KeyringID = keyringID
			result.Success = true
			results = append(results, result)
			continue
		}

		// Prompt user for credential
		prompt := generatePrompt(server.Name, credType)
		credential, err := promptFunc(prompt)
		if err != nil {
			result.Error = fmt.Errorf("failed to get credential from user: %w", err)
			results = append(results, result)
			return results, result.Error
		}

		// Store credential in keyring
		err = manager.Store(keyringID, credential)
		if err != nil {
			result.Error = fmt.Errorf("failed to store credential in keyring: %w", err)
			results = append(results, result)
			continue
		}

		// Update server config to use keyring
		server.UseKeyring = true
		server.KeyringID = keyringID

		result.Success = true
		results = append(results, result)
	}

	return results, nil
}

// identifyCredentialToMigrate determines if a server needs credential migration
func identifyCredentialToMigrate(server config.Server) (bool, string) {
	// Skip if already using keyring
	if server.UseKeyring && server.KeyringID != "" {
		return false, ""
	}

	switch server.AuthType {
	case "password":
		return true, "password"
	case "key":
		if server.PassphraseProtected {
			return true, "passphrase"
		}
		// Key without passphrase doesn't need keyring storage
		return false, ""
	default:
		return false, ""
	}
}

// generateKeyringID creates a unique keyring ID for a server credential
func generateKeyringID(serverName, credType string) string {
	return serverName + "_" + credType
}

// generatePrompt creates a user-friendly prompt for credential input
func generatePrompt(serverName, credType string) string {
	switch credType {
	case "password":
		return fmt.Sprintf("Enter password for %s:", serverName)
	case "passphrase":
		return fmt.Sprintf("Enter passphrase for %s SSH key:", serverName)
	default:
		return fmt.Sprintf("Enter credential for %s:", serverName)
	}
}

// ValidateMigration checks if a migration was successful by verifying keyring storage
func ValidateMigration(manager KeyringManager, results []MigrationResult) error {
	for _, result := range results {
		if !result.Success {
			continue
		}

		// Verify credential exists in keyring
		_, err := manager.Retrieve(result.KeyringID)
		if err != nil {
			return fmt.Errorf("validation failed for %s: credential not found in keyring: %w", 
				result.ServerName, err)
		}
	}
	return nil
}

// RollbackMigration removes migrated credentials from keyring and reverts config changes
func RollbackMigration(cfg *config.Config, manager KeyringManager, results []MigrationResult) error {
	var rollbackErrors []error

	for _, result := range results {
		if !result.Success {
			continue
		}

		// Remove credential from keyring
		err := manager.Delete(result.KeyringID)
		if err != nil {
			rollbackErrors = append(rollbackErrors, 
				fmt.Errorf("failed to delete %s from keyring: %w", result.KeyringID, err))
		}

		// Find and update server config
		for i := range cfg.Servers {
			if cfg.Servers[i].Name == result.ServerName {
				cfg.Servers[i].UseKeyring = false
				cfg.Servers[i].KeyringID = ""
				break
			}
		}
	}

	if len(rollbackErrors) > 0 {
		return fmt.Errorf("rollback encountered %d errors: %v", len(rollbackErrors), rollbackErrors)
	}

	return nil
}

// GetMigrationStatus returns information about which servers need migration
func GetMigrationStatus(cfg *config.Config) []MigrationStatus {
	var status []MigrationStatus

	if cfg == nil {
		return status
	}

	for _, server := range cfg.Servers {
		needsMigration, credType := identifyCredentialToMigrate(server)
		
		serverStatus := MigrationStatus{
			ServerName:      server.Name,
			NeedsMigration:  needsMigration,
			CredentialType:  credType,
			UsingKeyring:    server.UseKeyring,
			KeyringID:       server.KeyringID,
		}

		status = append(status, serverStatus)
	}

	return status
}

// MigrationStatus contains information about a server's migration status
type MigrationStatus struct {
	ServerName      string
	NeedsMigration  bool
	CredentialType  string
	UsingKeyring    bool
	KeyringID       string
}