package keyring

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/99designs/keyring"
)

const (
	// DefaultNamespace is the default namespace for SSHM credentials
	DefaultNamespace = "sshm"
)

// KeyringManager interface defines the operations for secure credential storage
type KeyringManager interface {
	Store(key string, value string) error
	Retrieve(key string) (string, error)
	Delete(key string) error
	List() ([]string, error)
	IsAvailable() bool
	ServiceName() string
	
	// Password-specific operations
	StoreServerPassword(serverName, password string) error
	RetrieveServerPassword(serverName string) (string, error)
	DeleteServerPassword(serverName string) error
	HasServerPassword(serverName string) bool
}

// manager implements KeyringManager using the 99designs/keyring library
type manager struct {
	keyring   keyring.Keyring
	service   string
	namespace string
}

// NewKeyringManager creates a new KeyringManager with the specified service type
func NewKeyringManager(service string) KeyringManager {
	return NewKeyringManagerWithNamespace(service, DefaultNamespace)
}

// NewKeyringManagerWithNamespace creates a new KeyringManager with custom namespace
func NewKeyringManagerWithNamespace(service, namespace string) KeyringManager {
	m := &manager{
		service:   service,
		namespace: namespace,
	}

	// Determine the backend to use
	var backend keyring.BackendType
	switch strings.ToLower(service) {
	case "auto":
		backend = m.detectBestBackend()
	case "keychain":
		backend = keyring.KeychainBackend
	case "wincred":
		backend = keyring.WinCredBackend
	case "secret-service":
		backend = keyring.SecretServiceBackend
	case "file":
		backend = keyring.FileBackend
	case "pass":
		backend = keyring.PassBackend
	default:
		// Invalid service type
		return nil
	}

	// Create keyring config
	config := keyring.Config{
		ServiceName: namespace,
		
		// For file backend, store in ~/.sshm/keyring
		FileDir: "~/.sshm/keyring",
		
		// Prompt functions for password-protected backends
		FilePasswordFunc: keyring.FixedStringPrompt("sshm-keyring"),
		
		// LibSecret collection (Linux)
		LibSecretCollectionName: namespace,
	}

	// Create the keyring
	kr, err := keyring.Open(config)
	if err != nil {
		// If the preferred backend fails, try file backend as fallback
		if backend != keyring.FileBackend {
			config.AllowedBackends = []keyring.BackendType{keyring.FileBackend}
			kr, err = keyring.Open(config)
		}
		
		if err != nil {
			return nil
		}
	}

	m.keyring = kr
	return m
}

// detectBestBackend detects the best available backend for the current platform
func (m *manager) detectBestBackend() keyring.BackendType {
	switch runtime.GOOS {
	case "darwin":
		return keyring.KeychainBackend
	case "windows":
		return keyring.WinCredBackend
	case "linux":
		// Try secret service first, fall back to file
		return keyring.SecretServiceBackend
	default:
		return keyring.FileBackend
	}
}

// Store stores a credential in the keyring
func (m *manager) Store(key string, value string) error {
	if m.keyring == nil {
		return fmt.Errorf("keyring not initialized")
	}

	item := keyring.Item{
		Key:   m.prefixKey(key),
		Data:  []byte(value),
		Label: fmt.Sprintf("SSHM credential: %s", key),
		Description: fmt.Sprintf("SSH credential for %s managed by SSHM", key),
	}

	return m.keyring.Set(item)
}

// Retrieve retrieves a credential from the keyring
func (m *manager) Retrieve(key string) (string, error) {
	if m.keyring == nil {
		return "", fmt.Errorf("keyring not initialized")
	}

	item, err := m.keyring.Get(m.prefixKey(key))
	if err != nil {
		return "", fmt.Errorf("failed to retrieve credential: %w", err)
	}

	return string(item.Data), nil
}

// Delete removes a credential from the keyring
func (m *manager) Delete(key string) error {
	if m.keyring == nil {
		return fmt.Errorf("keyring not initialized")
	}

	err := m.keyring.Remove(m.prefixKey(key))
	if err != nil {
		// Don't treat "not found" as an error for delete operations
		if strings.Contains(strings.ToLower(err.Error()), "not found") ||
		   strings.Contains(strings.ToLower(err.Error()), "no such") {
			return nil
		}
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	return nil
}

// List returns all keys stored in the keyring for this namespace
func (m *manager) List() ([]string, error) {
	if m.keyring == nil {
		return nil, fmt.Errorf("keyring not initialized")
	}

	keys, err := m.keyring.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	// Filter keys that belong to our namespace and remove the prefix
	var filteredKeys []string
	prefix := m.prefixKey("")
	
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			// Remove the prefix to get the original key name
			originalKey := strings.TrimPrefix(key, prefix)
			if originalKey != "" { // Skip empty keys
				filteredKeys = append(filteredKeys, originalKey)
			}
		}
	}

	return filteredKeys, nil
}

// IsAvailable returns true if the keyring service is available
func (m *manager) IsAvailable() bool {
	if m.keyring == nil {
		return false
	}
	
	// Try a simple operation to test availability
	testKey := m.prefixKey("availability-test")
	
	// Try to set and immediately remove a test item
	testItem := keyring.Item{
		Key:  testKey,
		Data: []byte("test"),
	}
	
	err := m.keyring.Set(testItem)
	if err != nil {
		return false
	}
	
	// Clean up the test item
	_ = m.keyring.Remove(testKey)
	
	return true
}

// ServiceName returns the name of the keyring service being used
func (m *manager) ServiceName() string {
	if m.keyring == nil {
		return "unavailable"
	}
	
	// Try to determine the backend type by testing the keyring
	// This is a bit of a hack since the library doesn't expose the backend type directly
	switch runtime.GOOS {
	case "darwin":
		if m.service == "auto" || m.service == "keychain" {
			return "keychain"
		}
	case "windows":
		if m.service == "auto" || m.service == "wincred" {
			return "wincred"
		}
	case "linux":
		if m.service == "auto" || m.service == "secret-service" {
			// Test if secret service is actually working
			if m.IsAvailable() {
				return "secret-service"
			}
			return "file"
		}
	}
	
	// Fallback to file backend or the specified service
	if m.service == "auto" {
		return "file"
	}
	
	return m.service
}

// prefixKey adds the namespace prefix to a key
func (m *manager) prefixKey(key string) string {
	if key == "" {
		return m.namespace + ":"
	}
	return m.namespace + ":" + key
}

// Password-specific keyring operations

// GeneratePasswordKeyringID generates a unique keyring ID for server password storage
func GeneratePasswordKeyringID(serverName string) string {
	return "password-" + serverName
}

// StoreServerPassword stores a password for a server in the keyring
func (m *manager) StoreServerPassword(serverName, password string) error {
	keyringID := GeneratePasswordKeyringID(serverName)
	return m.Store(keyringID, password)
}

// RetrieveServerPassword retrieves a password for a server from the keyring
func (m *manager) RetrieveServerPassword(serverName string) (string, error) {
	keyringID := GeneratePasswordKeyringID(serverName)
	return m.Retrieve(keyringID)
}

// DeleteServerPassword deletes a password for a server from the keyring
func (m *manager) DeleteServerPassword(serverName string) error {
	keyringID := GeneratePasswordKeyringID(serverName)
	return m.Delete(keyringID)
}

// HasServerPassword checks if a password is stored for a server
func (m *manager) HasServerPassword(serverName string) bool {
	keyringID := GeneratePasswordKeyringID(serverName)
	_, err := m.Retrieve(keyringID)
	return err == nil
}