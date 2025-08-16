package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"
)

// ClientConfig holds the configuration for SSH connections
type ClientConfig struct {
	Hostname string
	Port     int
	Username string
	Timeout  time.Duration
}

// Client represents an SSH client wrapper
type Client struct {
	config ClientConfig
	client *ssh.Client
}

// NewClient creates a new SSH client with the given configuration
func NewClient(config ClientConfig) *Client {
	return &Client{
		config: config,
	}
}

// Validate validates the client configuration
func (c *ClientConfig) Validate() error {
	if strings.TrimSpace(c.Hostname) == "" {
		return fmt.Errorf("hostname is required")
	}

	if strings.TrimSpace(c.Username) == "" {
		return fmt.Errorf("username is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	return nil
}

// Connect establishes an SSH connection using the provided authentication method
func (c *Client) Connect(auth ssh.AuthMethod) error {
	if err := c.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	config := &ssh.ClientConfig{
		User: c.config.Username,
		Auth: []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key verification
		Timeout:         c.config.Timeout,
	}

	address := fmt.Sprintf("%s:%d", c.config.Hostname, c.config.Port)
	
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	c.client = client
	return nil
}

// Disconnect closes the SSH connection
func (c *Client) Disconnect() error {
	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	return c.client != nil
}

// ExecuteCommand executes a command on the remote server and returns the output
func (c *Client) ExecuteCommand(command string) (string, error) {
	if !c.IsConnected() {
		return "", fmt.Errorf("not connected to server")
	}

	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("command execution failed: %w", err)
	}

	return string(output), nil
}

// NewKeyAuth creates an SSH authentication method using a private key
func NewKeyAuth(keyPath, passphrase string) (ssh.AuthMethod, error) {
	if strings.TrimSpace(keyPath) == "" {
		return nil, fmt.Errorf("key path is required")
	}

	// Expand ~ to home directory
	expandedPath, err := expandPath(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to expand key path: %w", err)
	}

	// Read private key
	keyBytes, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	var signer ssh.Signer
	if passphrase != "" {
		// Parse encrypted private key
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("failed to parse encrypted private key: %w", err)
		}
	} else {
		// Try to parse without passphrase first
		signer, err = ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			// If it fails, it might be encrypted - prompt for passphrase
			if strings.Contains(err.Error(), "encrypted") {
				fmt.Print("Enter passphrase for key: ")
				passphraseBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println() // New line after password input
				if err != nil {
					return nil, fmt.Errorf("failed to read passphrase: %w", err)
				}

				signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, passphraseBytes)
				if err != nil {
					return nil, fmt.Errorf("failed to parse encrypted private key: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
		}
	}

	return ssh.PublicKeys(signer), nil
}

// NewPasswordAuth creates an SSH authentication method using a password
func NewPasswordAuth(password string) ssh.AuthMethod {
	return ssh.Password(password)
}

// NewAgentAuth creates an SSH authentication method using the SSH agent
func NewAgentAuth() (ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, fmt.Errorf("SSH agent not available (SSH_AUTH_SOCK not set)")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers), nil
}

// PromptPassword prompts the user for a password with no echo
func PromptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // New line after password input
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return string(passwordBytes), nil
}

// expandPath expands ~ to the user's home directory in file paths
func expandPath(path string) (string, error) {
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

// TestConnection tests if a connection can be established with the given configuration and auth
func TestConnection(config ClientConfig, auth ssh.AuthMethod) error {
	client := NewClient(config)
	
	if err := client.Connect(auth); err != nil {
		return err
	}
	
	defer client.Disconnect()
	
	// Try to execute a simple command to verify the connection works
	_, err := client.ExecuteCommand("echo 'connection test'")
	return err
}