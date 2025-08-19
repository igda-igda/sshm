package cmd

import (
  "fmt"
  "io"

  "github.com/spf13/cobra"
  "sshm/internal/color"
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

  fmt.Fprintf(output, "%s\n", color.InfoMessage("Connecting to %s (%s@%s:%d)...", 
    server.Name, server.Username, server.Hostname, server.Port))

  // Create tmux session and connect (or reattach to existing)
  sessionName, wasExisting, err := tmuxManager.ConnectToServer(server.Name, sshCommand)
  if err != nil {
    return fmt.Errorf("❌ Failed to create tmux session: %w", err)
  }

  if wasExisting {
    fmt.Fprintf(output, "%s\n", color.InfoMessage("Found existing tmux session: %s", sessionName))
    fmt.Fprintf(output, "%s\n", color.InfoMessage("Reattaching to existing session"))
  } else {
    fmt.Fprintf(output, "%s\n", color.InfoMessage("Created tmux session: %s", sessionName))
    fmt.Fprintf(output, "%s\n", color.InfoMessage("SSH command sent to session"))
  }

  // Attach to the session
  fmt.Fprintf(output, "%s\n", color.InfoMessage("Attaching to session..."))
  err = tmuxManager.AttachSession(sessionName)
  if err != nil {
    // Don't fail the entire command if attach fails - provide manual instructions
    fmt.Fprintf(output, "%s\n", color.WarningMessage("Automatic attach failed (this can happen in non-TTY environments)"))
    fmt.Fprintf(output, "%s\n", color.InfoText("To manually attach to your session, run:"))
    fmt.Fprintf(output, "   tmux attach-session -t %s\n", sessionName)
    fmt.Fprintf(output, "%s\n", color.SuccessMessage("Session %s is ready for connection!", sessionName))
    return nil
  }

  fmt.Fprintf(output, "%s\n", color.SuccessMessage("Connected to %s successfully!", server.Name))
  return nil
}

func buildSSHCommand(server config.Server) (string, error) {
  // Validate server configuration
  if err := server.Validate(); err != nil {
    return "", fmt.Errorf("❌ Invalid server configuration: %w", err)
  }

  // Build base SSH command with pseudo-terminal allocation
  sshCmd := fmt.Sprintf("ssh -t %s@%s", server.Username, server.Hostname)
  
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