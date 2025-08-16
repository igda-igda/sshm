package cmd

import (
  "fmt"
  "io"
  "text/tabwriter"

  "github.com/spf13/cobra"
  "sshm/internal/config"
)

var listCmd = &cobra.Command{
  Use:   "list",
  Short: "List all configured servers",
  Long: `List all configured servers with their connection details.
  
Example:
  sshm list`,
  RunE: func(cmd *cobra.Command, args []string) error {
    return runListCommand(cmd.OutOrStdout())
  },
}

func runListCommand(output io.Writer) error {
  // Load configuration
  cfg, err := config.Load()
  if err != nil {
    return fmt.Errorf("failed to load configuration: %w", err)
  }

  servers := cfg.GetServers()
  if len(servers) == 0 {
    fmt.Fprintln(output, "No servers configured.")
    fmt.Fprintln(output, "Use 'sshm add <server-name>' to add a server.")
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
  
  fmt.Fprintf(output, "\nTotal: %d server(s)\n", len(servers))
  return nil
}