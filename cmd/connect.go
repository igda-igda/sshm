package cmd

import (
  "fmt"
  "io"

  "github.com/spf13/cobra"
  "sshm/internal/config"
  "sshm/internal/tmux"
)

var connectCmd = &cobra.Command{
  Use:   "connect <server-name>",
  Short: "Connect to a server via SSH in a tmux session",
  Long: `Connect to a configured server via SSH within a dedicated tmux session.

This command will:
  • Load the server configuration
  • Build the appropriate SSH command with authentication
  • Create a tmux session named after the server
  • Execute the SSH connection within the tmux session
  • Attach to the session for interactive use

Requirements:
  • tmux must be installed and available in PATH
  • SSH key must be accessible (if using key authentication)
  • Network connectivity to the target server
  
Examples:
  sshm connect production-api   # Connect to production API server
  sshm connect staging-db       # Connect to staging database
  sshm connect jump-host        # Connect to bastion/jump host`,
  Args: cobra.ExactArgs(1),
  RunE: func(cmd *cobra.Command, args []string) error {
    return runConnectCommand(args, cmd.OutOrStdout())
  },
}

func runConnectCommand(args []string, output io.Writer) error {
  serverName := args[0]
  
  // Load configuration
  cfg, err := config.Load()
  if err != nil {
    return fmt.Errorf("❌ Failed to load configuration: %w", err)
  }

  // Get server configuration
  server, err := cfg.GetServer(serverName)
  if err != nil {
    return fmt.Errorf("❌ Server '%s' not found. Use 'sshm list' to see available servers", serverName)
  }

  // Initialize tmux manager
  tmuxManager := tmux.NewManager()
  
  // Check if tmux is available
  if !tmuxManager.IsAvailable() {
    return fmt.Errorf("❌ tmux is not available on this system. Please install tmux to use sshm")
  }

  // Build SSH command based on server configuration
  sshCommand, err := buildSSHCommand(*server)
  if err != nil {
    return fmt.Errorf("❌ Failed to build SSH command: %w", err)
  }

  fmt.Fprintf(output, "🔌 Connecting to %s (%s@%s:%d)...\n", 
    server.Name, server.Username, server.Hostname, server.Port)

  // Create tmux session and connect
  sessionName, err := tmuxManager.ConnectToServer(server.Name, sshCommand)
  if err != nil {
    return fmt.Errorf("❌ Failed to create tmux session: %w", err)
  }

  fmt.Fprintf(output, "📺 Created tmux session: %s\n", sessionName)
  fmt.Fprintf(output, "⚡ SSH command sent to session\n")

  // Attach to the session
  fmt.Fprintf(output, "🔗 Attaching to session...\n")
  err = tmuxManager.AttachSession(sessionName)
  if err != nil {
    return fmt.Errorf("❌ Failed to attach to session: %w", err)
  }

  fmt.Fprintf(output, "✅ Connected to %s successfully!\n", server.Name)
  return nil
}

func buildSSHCommand(server config.Server) (string, error) {
  // Validate server configuration
  if err := server.Validate(); err != nil {
    return "", fmt.Errorf("❌ Invalid server configuration: %w", err)
  }

  // Build base SSH command
  sshCmd := fmt.Sprintf("ssh %s@%s", server.Username, server.Hostname)
  
  // Add port if not default
  if server.Port != 22 {
    sshCmd += fmt.Sprintf(" -p %d", server.Port)
  }

  // Add key-specific options
  if server.AuthType == "key" && server.KeyPath != "" {
    sshCmd += fmt.Sprintf(" -i %s", server.KeyPath)
  }

  // Add common SSH options
  sshCmd += " -o ServerAliveInterval=60 -o ServerAliveCountMax=3"

  return sshCmd, nil
}