package auth

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
	"sshm/internal/config"
	"sshm/internal/keyring"
	sshssh "sshm/internal/ssh"
)

// AuthManager handles authentication for SSH connections
type AuthManager struct {
	keyringManager keyring.KeyringManager
	promptFunc     PromptFunc
}

// PromptFunc is a function type for prompting the user for credentials
type PromptFunc func(prompt string) (string, error)

// NewAuthManager creates a new AuthManager
func NewAuthManager(cfg *config.Config, promptFunc PromptFunc) (*AuthManager, error) {
	var keyringManager keyring.KeyringManager

	if cfg.Keyring.Enabled {
		service := cfg.Keyring.Service
		if service == "" {
			service = "auto"
		}

		namespace := cfg.Keyring.Namespace
		if namespace == "" {
			namespace = "sshm"
		}

		keyringManager = keyring.NewKeyringManagerWithNamespace(service, namespace)
		if keyringManager == nil {
			return nil, fmt.Errorf("failed to initialize keyring with service: %s", service)
		}
	}

	return &AuthManager{
		keyringManager: keyringManager,
		promptFunc:     promptFunc,
	}, nil
}

// GetAuthMethod returns an SSH authentication method for the given server
func (a *AuthManager) GetAuthMethod(server config.Server) (ssh.AuthMethod, error) {
	switch server.AuthType {
	case "password":
		return a.getPasswordAuth(server)
	case "key":
		return a.getKeyAuth(server)
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", server.AuthType)
	}
}

// getPasswordAuth handles password authentication
func (a *AuthManager) getPasswordAuth(server config.Server) (ssh.AuthMethod, error) {
	var password string
	var err error

	// Try to get password from keyring if available
	if server.UseKeyring && server.KeyringID != "" && a.keyringManager != nil {
		password, err = a.keyringManager.Retrieve(server.KeyringID)
		if err != nil {
			// Keyring retrieval failed, fall back to prompting
			if a.promptFunc == nil {
				return nil, fmt.Errorf("keyring retrieval failed and no prompt function available: %w", err)
			}
			
			prompt := fmt.Sprintf("Enter password for %s (keyring unavailable):", server.Name)
			password, err = a.promptFunc(prompt)
			if err != nil {
				return nil, fmt.Errorf("failed to get password from user: %w", err)
			}
		}
	} else {
		// No keyring or not configured to use keyring
		if a.promptFunc == nil {
			return nil, fmt.Errorf("no keyring configuration and no prompt function available")
		}
		
		prompt := fmt.Sprintf("Enter password for %s:", server.Name)
		password, err = a.promptFunc(prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to get password from user: %w", err)
		}
	}

	return sshssh.NewPasswordAuth(password), nil
}

// getKeyAuth handles key-based authentication
func (a *AuthManager) getKeyAuth(server config.Server) (ssh.AuthMethod, error) {
	if strings.TrimSpace(server.KeyPath) == "" {
		return nil, fmt.Errorf("key path is required for key authentication")
	}

	var passphrase string
	var err error

	// Handle passphrase if key is passphrase-protected
	if server.PassphraseProtected {
		// Try to get passphrase from keyring if available
		if server.UseKeyring && server.KeyringID != "" && a.keyringManager != nil {
			passphrase, err = a.keyringManager.Retrieve(server.KeyringID)
			if err != nil {
				// Keyring retrieval failed, fall back to prompting
				if a.promptFunc == nil {
					return nil, fmt.Errorf("keyring retrieval failed and no prompt function available: %w", err)
				}
				
				prompt := fmt.Sprintf("Enter passphrase for %s SSH key (keyring unavailable):", server.Name)
				passphrase, err = a.promptFunc(prompt)
				if err != nil {
					return nil, fmt.Errorf("failed to get passphrase from user: %w", err)
				}
			}
		} else {
			// No keyring or not configured to use keyring
			if a.promptFunc == nil {
				// Try without passphrase first, let SSH library handle the prompt
				return sshssh.NewKeyAuth(server.KeyPath, "")
			}
			
			prompt := fmt.Sprintf("Enter passphrase for %s SSH key:", server.Name)
			passphrase, err = a.promptFunc(prompt)
			if err != nil {
				return nil, fmt.Errorf("failed to get passphrase from user: %w", err)
			}
		}
	}

	return sshssh.NewKeyAuth(server.KeyPath, passphrase)
}

// GetAuthMethodWithFallback gets auth method with fallback to SSH agent
func (a *AuthManager) GetAuthMethodWithFallback(server config.Server) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// Try primary authentication method
	primaryAuth, err := a.GetAuthMethod(server)
	if err != nil {
		return nil, fmt.Errorf("primary auth method failed: %w", err)
	}
	methods = append(methods, primaryAuth)

	// Add SSH agent as fallback for key authentication
	if server.AuthType == "key" {
		agentAuth, err := sshssh.NewAgentAuth()
		if err == nil {
			methods = append(methods, agentAuth)
		}
		// Don't fail if agent is not available, just continue without it
	}

	return methods, nil
}

// TestConnection tests if authentication works for a server
func (a *AuthManager) TestConnection(server config.Server) error {
	authMethods, err := a.GetAuthMethodWithFallback(server)
	if err != nil {
		return fmt.Errorf("failed to get auth methods: %w", err)
	}

	clientConfig := sshssh.ClientConfig{
		Hostname: server.Hostname,
		Port:     server.Port,
		Username: server.Username,
		Timeout:  5 * 60, // 5 minute timeout for testing
	}

	// Try each auth method
	var lastErr error
	for i, authMethod := range authMethods {
		err := sshssh.TestConnection(clientConfig, authMethod)
		if err == nil {
			return nil // Success
		}
		lastErr = err
		
		// Log which method failed (for debugging)
		methodName := "unknown"
		if i == 0 {
			methodName = server.AuthType
		} else {
			methodName = "ssh-agent"
		}
		
		// Continue to next method
		_ = methodName // Avoid unused variable
	}

	return fmt.Errorf("all authentication methods failed, last error: %w", lastErr)
}

// StoreCredential stores a credential in the keyring for a server
func (a *AuthManager) StoreCredential(server config.Server, credential string) error {
	if a.keyringManager == nil {
		return fmt.Errorf("keyring not available")
	}

	if !server.UseKeyring || server.KeyringID == "" {
		return fmt.Errorf("server not configured to use keyring")
	}

	return a.keyringManager.Store(server.KeyringID, credential)
}

// RetrieveCredential retrieves a credential from the keyring for a server
func (a *AuthManager) RetrieveCredential(server config.Server) (string, error) {
	if a.keyringManager == nil {
		return "", fmt.Errorf("keyring not available")
	}

	if !server.UseKeyring || server.KeyringID == "" {
		return "", fmt.Errorf("server not configured to use keyring")
	}

	return a.keyringManager.Retrieve(server.KeyringID)
}