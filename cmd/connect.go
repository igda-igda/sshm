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
  
Example:
  sshm connect production-api`,
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
    return fmt.Errorf("failed to load configuration: %w", err)
  }

  // Get server configuration
  server, err := cfg.GetServer(serverName)
  if err != nil {
    return fmt.Errorf("server '%s' not found", serverName)
  }

  // Initialize tmux manager
  tmuxManager := tmux.NewManager()
  
  // Check if tmux is available
  if !tmuxManager.IsAvailable() {
    return fmt.Errorf("tmux is not available on this system. Please install tmux to use sshm")
  }

  // Build SSH command based on server configuration
  sshCommand, err := buildSSHCommand(*server)
  if err != nil {
    return fmt.Errorf("failed to build SSH command: %w", err)
  }

  fmt.Fprintf(output, "Connecting to %s (%s@%s:%d)...\n", 
    server.Name, server.Username, server.Hostname, server.Port)

  // Create tmux session and connect
  sessionName, err := tmuxManager.ConnectToServer(server.Name, sshCommand)
  if err != nil {
    return fmt.Errorf("failed to create tmux session: %w", err)
  }

  fmt.Fprintf(output, "Created tmux session: %s\n", sessionName)
  fmt.Fprintf(output, "SSH command sent to session\n")

  // Attach to the session
  fmt.Fprintf(output, "Attaching to session...\n")
  err = tmuxManager.AttachSession(sessionName)
  if err != nil {
    return fmt.Errorf("failed to attach to session: %w", err)
  }

  fmt.Fprintf(output, "Connected to %s successfully!\n", server.Name)
  return nil
}

func buildSSHCommand(server config.Server) (string, error) {
  // Validate server configuration
  if err := server.Validate(); err != nil {
    return "", fmt.Errorf("invalid server configuration: %w", err)
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