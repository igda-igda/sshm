package cmd

import (
  "bufio"
  "fmt"
  "io"
  "os"
  "strconv"
  "strings"

  "github.com/spf13/cobra"
  "sshm/internal/color"
  "sshm/internal/config"
)

var addCmd = &cobra.Command{
  Use:   "add <server-name>",
  Short: "Add a new server configuration",
  Long: `Add a new server configuration with CLI flags or interactive prompts.

You can provide all connection details using flags for non-interactive usage,
or use interactive mode by omitting flags (you will be prompted for details).

CLI Flags:
  • --hostname: Hostname/IP address of the server (required for non-interactive)
  • --port: SSH port (default: 22)
  • --username: Username for authentication (required for non-interactive)
  • --auth-type: Authentication method - 'key' or 'password' (required for non-interactive)
  • --key-path: Path to SSH key file (required if auth-type is 'key')
  • --passphrase-protected: Whether the SSH key is passphrase protected (default: false)

The server configuration will be stored securely in ~/.sshm/config.yaml
  
Examples:
  # Interactive mode
  sshm add production-api
  
  # Non-interactive with key authentication
  sshm add web-server --hostname web.example.com --username webuser --auth-type key --key-path ~/.ssh/web_key
  
  # Non-interactive with password authentication  
  sshm add db-server --hostname db.example.com --username dbuser --auth-type password --port 3306`,
  Args: cobra.ExactArgs(1),
  RunE: func(cmd *cobra.Command, args []string) error {
    return runAddCommand(cmd, args, cmd.OutOrStdout())
  },
}

func runAddCommand(cmd *cobra.Command, args []string, output io.Writer) error {
  serverName := strings.TrimSpace(args[0])
  
  // Validate server name
  if serverName == "" {
    return fmt.Errorf("❌ Server name cannot be empty")
  }
  
  // Load existing configuration
  cfg, err := config.Load()
  if err != nil {
    return fmt.Errorf("❌ Failed to load configuration: %w", err)
  }

  // Check if server already exists
  if _, err := cfg.GetServer(serverName); err == nil {
    return fmt.Errorf("❌ Server '%s' already exists. Use 'sshm remove %s' to remove it first", serverName, serverName)
  }

  // Check if we're using CLI flags or interactive mode
  usingFlags := cmd.Flags().Changed("hostname") || cmd.Flags().Changed("username") || cmd.Flags().Changed("auth-type")
  
  var hostname, username, authType, keyPath string
  var port int
  var passphraseProtected bool

  if usingFlags {
    // CLI flag mode - validate all required flags are provided
    if !cmd.Flags().Changed("hostname") {
      return fmt.Errorf("❌ --hostname is required for non-interactive mode")
    }
    if !cmd.Flags().Changed("username") {
      return fmt.Errorf("❌ --username is required for non-interactive mode")
    }
    if !cmd.Flags().Changed("auth-type") {
      return fmt.Errorf("❌ --auth-type is required for non-interactive mode")
    }

    // Get flag values
    hostname, _ = cmd.Flags().GetString("hostname")
    username, _ = cmd.Flags().GetString("username")
    authType, _ = cmd.Flags().GetString("auth-type")
    port, _ = cmd.Flags().GetInt("port")
    keyPath, _ = cmd.Flags().GetString("key-path")
    passphraseProtected, _ = cmd.Flags().GetBool("passphrase-protected")

    // Validate flag values
    if hostname == "" {
      return fmt.Errorf("❌ Hostname cannot be empty")
    }
    if username == "" {
      return fmt.Errorf("❌ Username cannot be empty")
    }
    if authType != "key" && authType != "password" {
      return fmt.Errorf("❌ Authentication type must be 'key' or 'password', got: %s", authType)
    }
    if authType == "key" && keyPath == "" {
      return fmt.Errorf("❌ --key-path is required when auth-type is 'key'")
    }
    if port <= 0 || port > 65535 {
      return fmt.Errorf("❌ Invalid port: %d. Port must be between 1 and 65535", port)
    }

  } else {
    // Interactive mode (existing logic)
    scanner := bufio.NewScanner(os.Stdin)
    
    fmt.Fprintf(output, "Adding server '%s'\n\n", serverName)
    
    // Hostname
    fmt.Fprint(output, "Hostname: ")
    if !scanner.Scan() {
      return fmt.Errorf("failed to read hostname")
    }
    hostname = strings.TrimSpace(scanner.Text())
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
    port, err = strconv.Atoi(portStr)
    if err != nil || port <= 0 || port > 65535 {
      return fmt.Errorf("❌ Invalid port: %s. Port must be between 1 and 65535", portStr)
    }

    // Username
    fmt.Fprint(output, "Username: ")
    if !scanner.Scan() {
      return fmt.Errorf("failed to read username")
    }
    username = strings.TrimSpace(scanner.Text())
    if username == "" {
      return fmt.Errorf("❌ Username is required")
    }

    // Authentication type
    fmt.Fprint(output, "Authentication type (key/password): ")
    if !scanner.Scan() {
      return fmt.Errorf("failed to read auth type")
    }
    authType = strings.TrimSpace(strings.ToLower(scanner.Text()))
    if authType != "key" && authType != "password" {
      return fmt.Errorf("❌ Authentication type must be 'key' or 'password', got: %s", authType)
    }

    // Additional prompts for key authentication
    if authType == "key" {
      fmt.Fprint(output, "SSH key path: ")
      if !scanner.Scan() {
        return fmt.Errorf("failed to read key path")
      }
      keyPath = strings.TrimSpace(scanner.Text())
      if keyPath == "" {
        return fmt.Errorf("❌ SSH key path is required for key authentication")
      }

      fmt.Fprint(output, "Is the key passphrase protected? (y/n): ")
      if !scanner.Scan() {
        return fmt.Errorf("failed to read passphrase protection")
      }
      passphraseResp := strings.TrimSpace(strings.ToLower(scanner.Text()))
      passphraseProtected = passphraseResp == "y" || passphraseResp == "yes"
    }
  }

  // Create server configuration
  server := config.Server{
    Name:     serverName,
    Hostname: hostname,
    Port:     port,
    Username: username,
    AuthType: authType,
  }

  // Set optional fields for key authentication
  if authType == "key" {
    server.KeyPath = keyPath
    server.PassphraseProtected = passphraseProtected
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

  fmt.Fprintf(output, "\n%s\n", color.SuccessMessage("Server '%s' added successfully!", serverName))
  fmt.Fprintf(output, "%s\n", color.InfoMessage("Use 'sshm connect %s' to connect to this server", serverName))
  return nil
}

func init() {
  // Add CLI flags for non-interactive usage
  addCmd.Flags().StringP("hostname", "H", "", "Hostname/IP address of the server (required for non-interactive)")
  addCmd.Flags().IntP("port", "p", 22, "SSH port (default: 22)")
  addCmd.Flags().StringP("username", "u", "", "Username for authentication (required for non-interactive)")
  addCmd.Flags().StringP("auth-type", "a", "", "Authentication method - 'key' or 'password' (required for non-interactive)")
  addCmd.Flags().StringP("key-path", "k", "", "Path to SSH key file (required if auth-type is 'key')")
  addCmd.Flags().BoolP("passphrase-protected", "P", false, "Whether the SSH key is passphrase protected (default: false)")
  
  // Set color help function directly on this command
  addCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
    // Create comprehensive help text including Long description
    helpText := ""
    if len(cmd.Long) > 0 {
      helpText += cmd.Long + "\n\n"
    }
    helpText += cmd.UsageString()
    
    coloredHelp := color.FormatHelp(helpText)
    fmt.Fprint(cmd.OutOrStdout(), coloredHelp)
  })
}