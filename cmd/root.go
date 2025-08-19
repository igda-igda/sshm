package cmd

import (
  "fmt"
  "io"
  "os"

  "github.com/spf13/cobra"
  "sshm/internal/color"
)

var rootCmd = &cobra.Command{
  Use:   "sshm",
  Short: "SSH Connection Manager with tmux integration",
  Long: `SSHM is a CLI SSH connection manager that helps DevOps engineers, 
system administrators, and developers connect to multiple remote servers 
simultaneously through organized tmux sessions.

Features:
  • Manage server configurations with profiles
  • Connect via SSH with automatic tmux session creation
  • Support for SSH keys and password authentication
  • Secure credential storage and management
  • Group connections via profiles with individual tmux windows
  • Profile-based server organization and filtering

Examples:
  sshm add production-web          # Add a new server configuration
  sshm list                        # List all configured servers
  sshm list --profile dev          # List servers in 'dev' profile
  sshm connect production-web      # Connect to server in tmux session
  sshm batch --profile staging     # Connect to all staging servers
  sshm profile create development  # Create a new profile
  sshm sessions list               # List active tmux sessions
  sshm import ~/.ssh/config        # Import servers from SSH config
  sshm export servers.yaml         # Export configuration to file
  sshm remove production-web       # Remove server configuration`,
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}

// SetOutput allows tests to capture command output
func SetOutput(w io.Writer) {
  rootCmd.SetOut(w)
  rootCmd.SetErr(w)
}

// CreateRootCommand creates a new root command for testing
func CreateRootCommand() *cobra.Command {
  cmd := &cobra.Command{
    Use:   "sshm",
    Short: "SSH Connection Manager with tmux integration",
    Long: `SSHM is a CLI SSH connection manager that helps DevOps engineers, 
system administrators, and developers connect to multiple remote servers 
simultaneously through organized tmux sessions.

Features:
  • Manage server configurations with profiles
  • Connect via SSH with automatic tmux session creation
  • Support for SSH keys and password authentication
  • Secure credential storage and management
  • Group connections via profiles with individual tmux windows
  • Profile-based server organization and filtering

Examples:
  sshm add production-web          # Add a new server configuration
  sshm list                        # List all configured servers
  sshm list --profile dev          # List servers in 'dev' profile
  sshm connect production-web      # Connect to server in tmux session
  sshm batch --profile staging     # Connect to all staging servers
  sshm profile create development  # Create a new profile
  sshm sessions list               # List active tmux sessions
  sshm import ~/.ssh/config        # Import servers from SSH config
  sshm export servers.yaml         # Export configuration to file
  sshm remove production-web       # Remove server configuration`,
  }
  
  cmd.AddCommand(addCmd)
  cmd.AddCommand(listCmd)
  cmd.AddCommand(removeCmd)
  cmd.AddCommand(connectCmd)
  cmd.AddCommand(batchCmd)
  cmd.AddCommand(profileCmd)
  cmd.AddCommand(sessionsCmd)
  cmd.AddCommand(importCmd)
  cmd.AddCommand(exportCmd)
  
  // Set custom help template with color formatting
  cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
    // Create comprehensive help text including Long description
    helpText := ""
    if len(cmd.Long) > 0 {
      helpText += cmd.Long + "\n\n"
    }
    helpText += cmd.UsageString()
    
    coloredHelp := color.FormatHelp(helpText)
    fmt.Fprint(cmd.OutOrStdout(), coloredHelp)
  })
  
  // Apply color formatting to all individual commands for testing
  applyColorFormattingToAllCommands()
  
  return cmd
}

func init() {
  rootCmd.AddCommand(addCmd)
  rootCmd.AddCommand(listCmd)
  rootCmd.AddCommand(removeCmd)
  rootCmd.AddCommand(connectCmd)
  rootCmd.AddCommand(batchCmd)
  rootCmd.AddCommand(profileCmd)
  rootCmd.AddCommand(sessionsCmd)
  rootCmd.AddCommand(importCmd)
  rootCmd.AddCommand(exportCmd)
  
  // Set custom help template with color formatting for root command
  rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
    // Create comprehensive help text including Long description
    helpText := ""
    if len(cmd.Long) > 0 {
      helpText += cmd.Long + "\n\n"
    }
    helpText += cmd.UsageString()
    
    coloredHelp := color.FormatHelp(helpText)
    fmt.Fprint(cmd.OutOrStdout(), coloredHelp)
  })
  
  // Apply color formatting to all individual commands AFTER they're all added
  applyColorFormattingToAllCommands()
}