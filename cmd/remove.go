package cmd

import (
  "bufio"
  "fmt"
  "io"
  "os"
  "strings"

  "github.com/spf13/cobra"
  "sshm/internal/config"
)

var removeCmd = &cobra.Command{
  Use:   "remove <server-name>",
  Short: "Remove a server configuration",
  Long: `Remove a server configuration with optional confirmation prompt.

This command will:
  • Display the server details to be removed
  • Ask for confirmation before deletion (unless --yes is used)
  • Remove the server from ~/.sshm/config.yaml
  • Preserve other server configurations

By default, you will be prompted to confirm the deletion. Use --yes to skip confirmation.

Examples:
  sshm remove production-api      # Interactive confirmation
  sshm remove old-server --yes    # Non-interactive deletion
  sshm remove test-server -y      # Short flag version`,
  Args: cobra.ExactArgs(1),
  RunE: func(cmd *cobra.Command, args []string) error {
    return runRemoveCommand(cmd, args, cmd.OutOrStdout())
  },
}

func runRemoveCommand(cmd *cobra.Command, args []string, output io.Writer) error {
  serverName := args[0]
  
  // Load existing configuration
  cfg, err := config.Load()
  if err != nil {
    return fmt.Errorf("❌ Failed to load configuration: %w", err)
  }

  // Check if server exists
  server, err := cfg.GetServer(serverName)
  if err != nil {
    return fmt.Errorf("❌ Server '%s' not found. Use 'sshm list' to see available servers", serverName)
  }

  // Check if --yes flag is provided for non-interactive mode
  skipConfirmation, _ := cmd.Flags().GetBool("yes")
  
  if !skipConfirmation {
    // Display server details and confirmation prompt
    fmt.Fprintf(output, "🗑️  Server to remove:\n")
    fmt.Fprintf(output, "   Name: %s\n", server.Name)
    fmt.Fprintf(output, "   Hostname: %s:%d\n", server.Hostname, server.Port)
    fmt.Fprintf(output, "   Username: %s\n", server.Username)
    fmt.Fprintf(output, "   Auth Type: %s\n", server.AuthType)
    if server.KeyPath != "" {
      fmt.Fprintf(output, "   Key Path: %s\n", server.KeyPath)
    }
    fmt.Fprintf(output, "\n")

    // Confirmation prompt
    fmt.Fprint(output, "Are you sure you want to remove this server? (y/n): ")
    
    scanner := bufio.NewScanner(os.Stdin)
    if !scanner.Scan() {
      return fmt.Errorf("failed to read confirmation")
    }
    
    confirmation := strings.TrimSpace(strings.ToLower(scanner.Text()))
    if confirmation != "y" && confirmation != "yes" {
      fmt.Fprintln(output, "❌ Removal cancelled.")
      return nil
    }
  }

  // Remove server from configuration
  if err := cfg.RemoveServer(serverName); err != nil {
    return fmt.Errorf("❌ Failed to remove server: %w", err)
  }

  // Save configuration
  if err := cfg.Save(); err != nil {
    return fmt.Errorf("❌ Failed to save configuration: %w", err)
  }

  fmt.Fprintf(output, "✅ Server '%s' removed successfully!\n", serverName)
  return nil
}

func init() {
  // Add flags for remove command
  removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt and remove server")
}