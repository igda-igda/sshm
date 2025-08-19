package cmd

import (
  "fmt"
  "io"
  "text/tabwriter"

  "github.com/spf13/cobra"
  "sshm/internal/color"
  "sshm/internal/config"
)

var listCmd = &cobra.Command{
  Use:   "list",
  Short: "List all configured servers",
  Long: `List all configured servers with their connection details in a formatted table.

The table shows:
  • Server name (for use with other commands)
  • Hostname and port combination
  • Username for authentication
  • Authentication method (key or password)
  • SSH key path (if using key authentication)
  
Examples:
  sshm list                     # List all servers
  sshm list --profile dev       # List servers in 'dev' profile
  sshm list | grep production   # Filter production servers`,
  RunE: func(cmd *cobra.Command, args []string) error {
    profile, _ := cmd.Flags().GetString("profile")
    return runListCommand(cmd.OutOrStdout(), profile)
  },
}

func init() {
  listCmd.Flags().StringP("profile", "p", "", "Filter servers by profile name")
}

func runListCommand(output io.Writer, profileName string) error {
  // Load configuration
  cfg, err := config.Load()
  if err != nil {
    return fmt.Errorf("❌ Failed to load configuration: %w", err)
  }

  var servers []config.Server
  var contextMessage string

  // Get servers based on profile filter
  if profileName != "" {
    servers, err = cfg.GetServersByProfile(profileName)
    if err != nil {
      return fmt.Errorf("❌ Profile '%s' not found", profileName)
    }
    contextMessage = fmt.Sprintf("Servers in profile '%s'", profileName)
  } else {
    servers = cfg.GetServers()
    contextMessage = "All configured servers"
  }

  if len(servers) == 0 {
    if profileName != "" {
      fmt.Fprintf(output, "%s\n", color.InfoMessage("No servers found in profile '%s'", profileName))
      fmt.Fprintln(output, color.InfoText("Use 'sshm profile assign <server-name> <profile-name>' to assign servers to this profile."))
    } else {
      fmt.Fprintln(output, color.InfoMessage("No servers configured."))
      fmt.Fprintln(output, color.InfoText("Use 'sshm add <server-name>' to add a server."))
    }
    return nil
  }

  // Create formatted table output
  w := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
  fmt.Fprintln(w, "NAME\tHOSTNAME:PORT\tUSERNAME\tAUTH TYPE\tKEY PATH")
  fmt.Fprintln(w, "----\t-------------\t--------\t---------\t--------")

  for _, server := range servers {
    hostPort := fmt.Sprintf("%s:%d", server.Hostname, server.Port)
    keyPath := server.KeyPath
    if keyPath == "" {
      keyPath = "-"
    }
    
    fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
      server.Name,
      hostPort,
      server.Username,
      server.AuthType,
      keyPath,
    )
  }

  w.Flush()
  
  fmt.Fprintf(output, "\n%s\n", color.InfoMessage("%s: %d server(s)", contextMessage, len(servers)))
  if profileName != "" {
    fmt.Fprintln(output, color.InfoText("Use 'sshm connect <server-name>' to connect to a server"))
    fmt.Fprintf(output, "%s\n", color.InfoText("Use 'sshm batch --profile %s' to connect to all servers in this profile", profileName))
  } else {
    fmt.Fprintln(output, color.InfoText("Use 'sshm connect <server-name>' to connect to a server"))
  }
  return nil
}