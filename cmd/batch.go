package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"sshm/internal/color"
	"sshm/internal/config"
	"sshm/internal/tmux"
)

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Connect to multiple servers simultaneously in a group tmux session",
	Long: `Connect to multiple servers from a profile simultaneously within a dedicated tmux session.

This command will:
  • Load servers from the specified profile
  • Create a tmux session named after the profile
  • Create individual windows for each server in the profile
  • Execute SSH connections to each server in their respective windows
  • Attach to the group session for management

Requirements:
  • tmux must be installed and available in PATH
  • At least one server must be assigned to the specified profile
  • All servers must have valid configurations
  • Network connectivity to all target servers

Session Management:
  • Session name: Based on profile name (e.g., "development")
  • Window names: Named after individual server names
  • Window layout: Each server gets its own window

Examples:
  sshm batch --profile development   # Connect to all servers in development profile
  sshm batch -p staging             # Connect to all servers in staging profile
  sshm batch --profile production   # Connect to all servers in production profile`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, _ := cmd.Flags().GetString("profile")
		if profile == "" {
			return fmt.Errorf("❌ Profile name is required. Use --profile <profile-name>")
		}
		return runBatchCommand(profile, cmd.OutOrStdout())
	},
}

func init() {
	batchCmd.Flags().StringP("profile", "p", "", "Profile name for group connection (required)")
	batchCmd.MarkFlagRequired("profile")
}

func runBatchCommand(profileName string, output io.Writer) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("❌ Failed to load configuration: %w", err)
	}

	// Get servers from profile
	servers, err := cfg.GetServersByProfile(profileName)
	if err != nil {
		return fmt.Errorf("❌ Profile '%s' not found", profileName)
	}

	if len(servers) == 0 {
		return fmt.Errorf("❌ No servers found in profile '%s'. Use 'sshm profile assign <server-name> %s' to add servers", profileName, profileName)
	}

	// Initialize tmux manager
	tmuxManager := tmux.NewManager()

	// Check if tmux is available
	if !tmuxManager.IsAvailable() {
		return fmt.Errorf("❌ tmux is not available on this system. Please install tmux to use sshm")
	}

	fmt.Fprintf(output, "%s\n", color.InfoMessage("Creating group session for profile '%s' with %d server(s)...", profileName, len(servers)))

	// Convert config.Server slice to tmux.Server interface slice
	tmuxServers := make([]tmux.Server, len(servers))
	for i, server := range servers {
		tmuxServers[i] = &server
	}

	// Create group session and connect to all servers
	sessionName, wasExisting, err := tmuxManager.ConnectToProfile(profileName, tmuxServers)
	if err != nil {
		return fmt.Errorf("❌ Failed to create group session: %w", err)
	}

	if wasExisting {
		fmt.Fprintf(output, "%s\n", color.InfoMessage("Found existing group session: %s", sessionName))
		fmt.Fprintf(output, "%s\n", color.InfoMessage("Reattaching to existing session"))
	} else {
		fmt.Fprintf(output, "%s\n", color.InfoMessage("Created group session: %s", sessionName))
		fmt.Fprintf(output, "%s\n", color.InfoMessage("Created %d windows for servers", len(servers)))
		
		// List the windows created
		for i, server := range servers {
			fmt.Fprintf(output, "   • Window %d: %s (%s@%s:%d)\n", 
				i+1, server.Name, server.Username, server.Hostname, server.Port)
		}
	}

	// Attach to the session
	fmt.Fprintf(output, "%s\n", color.InfoMessage("Attaching to group session..."))
	err = tmuxManager.AttachSession(sessionName)
	if err != nil {
		// Don't fail the entire command if attach fails - provide manual instructions
		fmt.Fprintf(output, "%s\n", color.WarningMessage("Automatic attach failed (this can happen in non-TTY environments)"))
		fmt.Fprintf(output, "%s\n", color.InfoText("To manually attach to your group session, run:"))
		fmt.Fprintf(output, "   tmux attach-session -t %s\n", sessionName)
		fmt.Fprintf(output, "%s\n", color.InfoText("To switch between windows, use:"))
		fmt.Fprintf(output, "   Ctrl+b, then number key (1, 2, 3, etc.)\n")
		fmt.Fprintf(output, "   Ctrl+b, then 'n' for next window\n")
		fmt.Fprintf(output, "   Ctrl+b, then 'p' for previous window\n")
		fmt.Fprintf(output, "%s\n", color.SuccessMessage("Group session %s is ready!", sessionName))
		return nil
	}

	fmt.Fprintf(output, "%s\n", color.SuccessMessage("Connected to profile '%s' group session successfully!", profileName))
	return nil
}

func buildSSHCommandForServer(server config.Server) (string, error) {
	// Validate server configuration
	if err := server.Validate(); err != nil {
		return "", fmt.Errorf("❌ Invalid server configuration for %s: %w", server.Name, err)
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