package cmd

import (
  "bufio"
  "fmt"
  "io"
  "os"
  "strconv"
  "strings"

  "github.com/spf13/cobra"
  "sshm/internal/config"
)

var addCmd = &cobra.Command{
  Use:   "add <server-name>",
  Short: "Add a new server configuration",
  Long: `Add a new server configuration with interactive prompts for connection details.

This command will prompt you for:
  • Hostname/IP address of the server
  • SSH port (default: 22)
  • Username for authentication
  • Authentication method (SSH key or password)
  • SSH key path (if using key authentication)
  • Passphrase protection status (for SSH keys)

The server configuration will be stored securely in ~/.sshm/config.yaml
  
Examples:
  sshm add production-api        # Add production API server
  sshm add staging-db           # Add staging database server
  sshm add jump-host            # Add bastion/jump host`,
  Args: cobra.ExactArgs(1),
  RunE: func(cmd *cobra.Command, args []string) error {
    return runAddCommand(args, cmd.OutOrStdout())
  },
}

func runAddCommand(args []string, output io.Writer) error {
  serverName := args[0]
  
  // Load existing configuration
  cfg, err := config.Load()
  if err != nil {
    return fmt.Errorf("❌ Failed to load configuration: %w", err)
  }

  // Check if server already exists
  if _, err := cfg.GetServer(serverName); err == nil {
    return fmt.Errorf("❌ Server '%s' already exists. Use 'sshm remove %s' to remove it first", serverName, serverName)
  }

  // Interactive prompts for server configuration
  scanner := bufio.NewScanner(os.Stdin)
  
  fmt.Fprintf(output, "Adding server '%s'\n\n", serverName)
  
  // Hostname
  fmt.Fprint(output, "Hostname: ")
  if !scanner.Scan() {
    return fmt.Errorf("failed to read hostname")
  }
  hostname := strings.TrimSpace(scanner.Text())
  if hostname == "" {
    return fmt.Errorf("❌ Hostname is required")
  }

  // Port
  fmt.Fprint(output, "Port (default: 22): ")
  if !scanner.Scan() {
    return fmt.Errorf("failed to read port")
  }
  portStr := strings.TrimSpace(scanner.Text())
  if portStr == "" {
    portStr = "22"
  }
  port, err := strconv.Atoi(portStr)
  if err != nil || port <= 0 || port > 65535 {
    return fmt.Errorf("❌ Invalid port: %s. Port must be between 1 and 65535", portStr)
  }

  // Username
  fmt.Fprint(output, "Username: ")
  if !scanner.Scan() {
    return fmt.Errorf("failed to read username")
  }
  username := strings.TrimSpace(scanner.Text())
  if username == "" {
    return fmt.Errorf("❌ Username is required")
  }

  // Authentication type
  fmt.Fprint(output, "Authentication type (key/password): ")
  if !scanner.Scan() {
    return fmt.Errorf("failed to read auth type")
  }
  authType := strings.TrimSpace(strings.ToLower(scanner.Text()))
  if authType != "key" && authType != "password" {
    return fmt.Errorf("❌ Authentication type must be 'key' or 'password', got: %s", authType)
  }

  // Create server configuration
  server := config.Server{
    Name:     serverName,
    Hostname: hostname,
    Port:     port,
    Username: username,
    AuthType: authType,
  }

  // Additional prompts for key authentication
  if authType == "key" {
    fmt.Fprint(output, "SSH key path: ")
    if !scanner.Scan() {
      return fmt.Errorf("failed to read key path")
    }
    keyPath := strings.TrimSpace(scanner.Text())
    if keyPath == "" {
      return fmt.Errorf("❌ SSH key path is required for key authentication")
    }
    server.KeyPath = keyPath

    fmt.Fprint(output, "Is the key passphrase protected? (y/n): ")
    if !scanner.Scan() {
      return fmt.Errorf("failed to read passphrase protection")
    }
    passphraseResp := strings.TrimSpace(strings.ToLower(scanner.Text()))
    server.PassphraseProtected = passphraseResp == "y" || passphraseResp == "yes"
  }

  // Validate the server configuration
  if err := server.Validate(); err != nil {
    return fmt.Errorf("❌ Invalid server configuration: %w", err)
  }

  // Add server to configuration
  if err := cfg.AddServer(server); err != nil {
    return fmt.Errorf("❌ Failed to add server: %w", err)
  }

  // Save configuration
  if err := cfg.Save(); err != nil {
    return fmt.Errorf("❌ Failed to save configuration: %w", err)
  }

  fmt.Fprintf(output, "\n✅ Server '%s' added successfully!\n", serverName)
  fmt.Fprintf(output, "💡 Use 'sshm connect %s' to connect to this server\n", serverName)
  return nil
}