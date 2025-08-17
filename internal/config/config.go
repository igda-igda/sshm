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

// Profile represents a profile configuration for organizing servers
type Profile struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Servers     []string `yaml:"servers" json:"servers"`
}

// Config represents the main configuration structure
type Config struct {
	Servers    []Server  `yaml:"servers" json:"servers"`
	Profiles   []Profile `yaml:"profiles,omitempty" json:"profiles,omitempty"`
	configPath string    // internal field to track config file path
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
		return &Config{Servers: []Server{}, Profiles: []Profile{}, configPath: configPath}, nil
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

	// Ensure backward compatibility: initialize Profiles if nil
	if config.Profiles == nil {
		config.Profiles = []Profile{}
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

// Profile validation and management methods

// Validate validates a profile configuration
func (p *Profile) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("profile name is required")
	}
	return nil
}

// AddProfile adds a new profile to the configuration
func (c *Config) AddProfile(profile Profile) error {
	// Validate profile configuration
	if err := profile.Validate(); err != nil {
		return fmt.Errorf("invalid profile configuration: %w", err)
	}

	// Check for duplicate names
	for _, existing := range c.Profiles {
		if existing.Name == profile.Name {
			return fmt.Errorf("profile with name '%s' already exists", profile.Name)
		}
	}

	// Initialize servers slice if nil
	if profile.Servers == nil {
		profile.Servers = []string{}
	}

	c.Profiles = append(c.Profiles, profile)
	return nil
}

// RemoveProfile removes a profile from the configuration by name
func (c *Config) RemoveProfile(name string) error {
	for i, profile := range c.Profiles {
		if profile.Name == name {
			// Remove profile from slice
			c.Profiles = append(c.Profiles[:i], c.Profiles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("profile '%s' not found", name)
}

// GetProfile retrieves a profile by name
func (c *Config) GetProfile(name string) (*Profile, error) {
	for i := range c.Profiles {
		if c.Profiles[i].Name == name {
			return &c.Profiles[i], nil
		}
	}
	return nil, fmt.Errorf("profile '%s' not found", name)
}

// GetProfiles returns all profiles
func (c *Config) GetProfiles() []Profile {
	return c.Profiles
}

// GetServersByProfile retrieves all servers belonging to a specific profile
func (c *Config) GetServersByProfile(profileName string) ([]Server, error) {
	profile, err := c.GetProfile(profileName)
	if err != nil {
		return nil, err
	}

	var servers []Server
	for _, serverName := range profile.Servers {
		// Find the server in the config
		for _, server := range c.Servers {
			if server.Name == serverName {
				servers = append(servers, server)
				break
			}
		}
		// Note: We skip servers that don't exist rather than returning an error
		// This allows for more flexible configuration management
	}

	return servers, nil
}

// AssignServerToProfile assigns a server to a profile
func (c *Config) AssignServerToProfile(serverName, profileName string) error {
	// Verify server exists
	if _, err := c.GetServer(serverName); err != nil {
		return fmt.Errorf("server '%s' not found", serverName)
	}

	// Get profile (this will error if profile doesn't exist)
	profile, err := c.GetProfile(profileName)
	if err != nil {
		return err
	}

	// Check if server is already assigned to this profile
	for _, assignedServer := range profile.Servers {
		if assignedServer == serverName {
			return nil // Server already assigned, no error
		}
	}

	// Add server to profile
	profile.Servers = append(profile.Servers, serverName)

	// Update the profile in the config
	for i := range c.Profiles {
		if c.Profiles[i].Name == profileName {
			c.Profiles[i] = *profile
			break
		}
	}

	return nil
}

// UnassignServerFromProfile removes a server from a profile
func (c *Config) UnassignServerFromProfile(serverName, profileName string) error {
	// Get profile (this will error if profile doesn't exist)
	profile, err := c.GetProfile(profileName)
	if err != nil {
		return err
	}

	// Find and remove server from profile
	for i, assignedServer := range profile.Servers {
		if assignedServer == serverName {
			// Remove server from slice
			profile.Servers = append(profile.Servers[:i], profile.Servers[i+1:]...)
			
			// Update the profile in the config
			for j := range c.Profiles {
				if c.Profiles[j].Name == profileName {
					c.Profiles[j] = *profile
					break
				}
			}
			return nil
		}
	}

	return fmt.Errorf("server '%s' is not assigned to profile '%s'", serverName, profileName)
}