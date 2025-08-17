package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"sshm/internal/config"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage server profiles",
	Long: `Manage server profiles for organizing servers by environment or purpose.
	
Profiles allow you to group servers together for easier management and batch operations.
You can assign servers to profiles and then operate on entire profiles at once.

Examples:
  sshm profile create development    # Create a new profile
  sshm profile list                  # List all profiles
  sshm profile delete staging        # Delete a profile
  sshm profile assign web-dev dev    # Assign server to profile
  sshm profile unassign web-dev dev  # Remove server from profile`,
}

var profileCreateCmd = &cobra.Command{
	Use:   "create [profile-name]",
	Short: "Create a new profile",
	Long: `Create a new profile with the specified name.
	
You will be prompted to enter an optional description for the profile.
Once created, you can assign servers to this profile using the assign command.

Examples:
  sshm profile create development
  sshm profile create production`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Prompt for description
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Enter description for profile '%s' (optional): ", profileName)
		description, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read description: %w", err)
		}
		description = strings.TrimSpace(description)

		// Create profile
		profile := config.Profile{
			Name:        profileName,
			Description: description,
			Servers:     []string{},
		}

		// Add profile to configuration
		if err := cfg.AddProfile(profile); err != nil {
			return fmt.Errorf("failed to add profile: %w", err)
		}

		// Save configuration
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		cmd.Printf("Profile '%s' created successfully\n", profileName)
		return nil
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long: `List all configured profiles with their descriptions and assigned servers.
	
Shows profile name, description, and the number of servers assigned to each profile.

Examples:
  sshm profile list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		profiles := cfg.GetProfiles()
		if len(profiles) == 0 {
			cmd.Println("No profiles configured")
			return nil
		}

		// Create tabwriter for formatted output
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tSERVERS")
		fmt.Fprintln(w, "----\t-----------\t-------")

		for _, profile := range profiles {
			description := profile.Description
			if description == "" {
				description = "-"
			}
			fmt.Fprintf(w, "%s\t%s\t%d\n", profile.Name, description, len(profile.Servers))
		}

		w.Flush()
		return nil
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete [profile-name]",
	Short: "Delete a profile",
	Long: `Delete the specified profile.
	
This will remove the profile from the configuration but will not delete the servers
that were assigned to it. The servers will remain in the configuration.

You will be prompted to confirm the deletion.

Examples:
  sshm profile delete staging
  sshm profile delete old-environment`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Check if profile exists
		profile, err := cfg.GetProfile(profileName)
		if err != nil {
			return fmt.Errorf("profile '%s' not found", profileName)
		}

		// Show profile details and ask for confirmation
		fmt.Printf("Profile: %s\n", profile.Name)
		if profile.Description != "" {
			fmt.Printf("Description: %s\n", profile.Description)
		}
		fmt.Printf("Assigned servers: %d\n", len(profile.Servers))
		fmt.Printf("Are you sure you want to delete this profile? (y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			cmd.Println("Deletion cancelled")
			return nil
		}

		// Remove profile
		if err := cfg.RemoveProfile(profileName); err != nil {
			return fmt.Errorf("failed to remove profile: %w", err)
		}

		// Save configuration
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		cmd.Printf("Profile '%s' deleted successfully\n", profileName)
		return nil
	},
}

var profileAssignCmd = &cobra.Command{
	Use:   "assign [server-name] [profile-name]",
	Short: "Assign a server to a profile",
	Long: `Assign the specified server to the specified profile.
	
The server must already exist in the configuration. If the server is already
assigned to the profile, this command will succeed without changes.

Examples:
  sshm profile assign web-server-1 production
  sshm profile assign db-dev development`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		profileName := args[1]

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Assign server to profile
		if err := cfg.AssignServerToProfile(serverName, profileName); err != nil {
			return fmt.Errorf("failed to assign server to profile: %w", err)
		}

		// Save configuration
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		cmd.Printf("Server '%s' assigned to profile '%s'\n", serverName, profileName)
		return nil
	},
}

var profileUnassignCmd = &cobra.Command{
	Use:   "unassign [server-name] [profile-name]",
	Short: "Remove a server from a profile",
	Long: `Remove the specified server from the specified profile.
	
The server will remain in the configuration but will no longer be associated
with the specified profile.

Examples:
  sshm profile unassign web-server-1 production
  sshm profile unassign db-dev development`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		profileName := args[1]

		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Unassign server from profile
		if err := cfg.UnassignServerFromProfile(serverName, profileName); err != nil {
			return fmt.Errorf("failed to unassign server from profile: %w", err)
		}

		// Save configuration
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		cmd.Printf("Server '%s' unassigned from profile '%s'\n", serverName, profileName)
		return nil
	},
}

func init() {
	// Add subcommands to profile command
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	profileCmd.AddCommand(profileAssignCmd)
	profileCmd.AddCommand(profileUnassignCmd)
}