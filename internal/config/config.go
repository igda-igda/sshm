package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Server represents a server configuration
type Server struct {
	Name                string `yaml:"name" json:"name"`
	Hostname            string `yaml:"hostname" json:"hostname"`
	Port                int    `yaml:"port" json:"port"`
	Username            string `yaml:"username" json:"username"`
	AuthType            string `yaml:"auth_type" json:"auth_type"` // "key" or "password"
	KeyPath             string `yaml:"key_path,omitempty" json:"key_path,omitempty"`
	PassphraseProtected bool   `yaml:"passphrase_protected,omitempty" json:"passphrase_protected,omitempty"`
}

// Config represents the main configuration structure
type Config struct {
	Servers    []Server `yaml:"servers" json:"servers"`
	configPath string   // internal field to track config file path
}

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() (string, error) {
	// Check for test environment
	if testConfigDir := os.Getenv("SSHM_CONFIG_DIR"); testConfigDir != "" {
		return filepath.Join(testConfigDir, "config.yaml"), nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	configDir := filepath.Join(homeDir, ".sshm")
	return filepath.Join(configDir, "config.yaml"), nil
}

// Load loads configuration from the default path
func Load() (*Config, error) {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadFromPath(configPath)
}

// LoadFromPath loads configuration from the specified path
// If the file doesn't exist, it returns a default empty configuration
func LoadFromPath(configPath string) (*Config, error) {
	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// If file doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{Servers: []Server{}, configPath: configPath}, nil
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	config.configPath = configPath
	return &config, nil
}

// Save saves the configuration to the stored path with proper permissions
func (c *Config) Save() error {
	return c.SaveToPath(c.configPath)
}

// SaveToPath saves the configuration to the specified path with proper permissions
func (c *Config) SaveToPath(configPath string) error {
	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file with proper permissions (600 - owner read/write only)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddServer adds a new server to the configuration
func (c *Config) AddServer(server Server) error {
	// Validate server configuration
	if err := server.Validate(); err != nil {
		return fmt.Errorf("invalid server configuration: %w", err)
	}

	// Check for duplicate names
	for _, existing := range c.Servers {
		if existing.Name == server.Name {
			return fmt.Errorf("server with name '%s' already exists", server.Name)
		}
	}

	// Set default port if not specified
	if server.Port == 0 {
		server.Port = 22
	}

	c.Servers = append(c.Servers, server)
	return nil
}

// RemoveServer removes a server from the configuration by name
func (c *Config) RemoveServer(name string) error {
	for i, server := range c.Servers {
		if server.Name == name {
			// Remove server from slice
			c.Servers = append(c.Servers[:i], c.Servers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("server '%s' not found", name)
}

// GetServer retrieves a server by name
func (c *Config) GetServer(name string) (*Server, error) {
	for _, server := range c.Servers {
		if server.Name == name {
			return &server, nil
		}
	}
	return nil, fmt.Errorf("server '%s' not found", name)
}

// GetServers returns all servers
func (c *Config) GetServers() []Server {
	return c.Servers
}

// Validate validates a server configuration
func (s *Server) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("server name is required")
	}

	if strings.TrimSpace(s.Hostname) == "" {
		return fmt.Errorf("hostname is required")
	}

	if strings.TrimSpace(s.Username) == "" {
		return fmt.Errorf("username is required")
	}

	if s.Port <= 0 || s.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	// Validate auth type
	if s.AuthType != "key" && s.AuthType != "password" {
		return fmt.Errorf("auth_type must be 'key' or 'password'")
	}

	// If using key authentication, key path is required
	if s.AuthType == "key" && strings.TrimSpace(s.KeyPath) == "" {
		return fmt.Errorf("key_path is required when auth_type is 'key'")
	}

	return nil
}

// ExpandPath expands ~ to the user's home directory in file paths
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}

	return path, nil
}