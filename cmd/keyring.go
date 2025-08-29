package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"sshm/internal/color"
	"sshm/internal/config"
	"sshm/internal/keyring"
)

var keyringCmd = &cobra.Command{
	Use:   "keyring",
	Short: "Manage encrypted credential storage",
	Long: `Manage encrypted credential storage using system keyring services.

The keyring command provides tools for:
  • Checking keyring service status and availability
  • Migrating plaintext credentials to encrypted storage
  • Managing keyring configuration settings

Supported keyring services:
  • macOS: Keychain
  • Windows: Credential Manager
  • Linux: Secret Service (GNOME Keyring, KDE KWallet)
  • Fallback: Encrypted file storage

Examples:
  sshm keyring status              # Check keyring availability
  sshm keyring migrate             # Migrate plaintext credentials
  sshm keyring migrate --server web01  # Migrate specific server`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeyringStatusCommand(cmd.OutOrStdout())
	},
}

var keyringStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show keyring service status and configuration",
	Long: `Display information about keyring service availability and configuration.

This command shows:
  • Current keyring service (keychain, wincred, secret-service, file)
  • Service availability status
  • Configuration settings
  • Migration status for each server

The output helps determine if credentials can be securely stored
and whether migration from plaintext storage is needed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeyringStatusCommand(cmd.OutOrStdout())
	},
}

var keyringMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate plaintext credentials to encrypted keyring storage",
	Long: `Migrate plaintext credentials to encrypted keyring storage.

This command will:
  • Identify servers that need credential migration
  • Prompt for passwords and passphrases
  • Store credentials securely in the keyring
  • Update server configurations to use keyring storage

Migration is safe and reversible. Original configurations are preserved
until credentials are successfully stored in the keyring.

The migration process only affects servers that require stored credentials:
  • Password authentication servers
  • SSH key servers with passphrases

Servers using keys without passphrases don't need migration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName, _ := cmd.Flags().GetString("server")
		return runKeyringMigrateCommand(cmd.OutOrStdout(), serverName)
	},
}

func init() {
	rootCmd.AddCommand(keyringCmd)
	keyringCmd.AddCommand(keyringStatusCmd)
	keyringCmd.AddCommand(keyringMigrateCmd)

	// Add flags for migrate command
	keyringMigrateCmd.Flags().StringP("server", "s", "", "Migrate credentials for a specific server only")
}

func runKeyringStatusCommand(output io.Writer) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("❌ Failed to load configuration: %w", err)
	}

	fmt.Fprintf(output, "%s\n", color.Header("🔐 Keyring Status"))
	fmt.Fprintf(output, "\n")

	// Create keyring manager
	var keyringManager keyring.KeyringManager
	if cfg.Keyring.Enabled {
		service := cfg.Keyring.Service
		if service == "" {
			service = "auto"
		}
		namespace := cfg.Keyring.Namespace
		if namespace == "" {
			namespace = "sshm"
		}
		keyringManager = keyring.NewKeyringManagerWithNamespace(service, namespace)
	}

	// Display keyring configuration
	fmt.Fprintf(output, "%s\n", color.Header("Configuration"))
	fmt.Fprintf(output, "  Enabled: %s\n", formatBoolStatus(cfg.Keyring.Enabled))
	fmt.Fprintf(output, "  Service: %s\n", cfg.Keyring.Service)
	fmt.Fprintf(output, "  Namespace: %s\n", cfg.Keyring.Namespace)

	if keyringManager != nil {
		fmt.Fprintf(output, "  Detected Service: %s\n", keyringManager.ServiceName())
		fmt.Fprintf(output, "  Available: %s\n", formatBoolStatus(keyringManager.IsAvailable()))
	} else {
		fmt.Fprintf(output, "  Detected Service: %s\n", color.ErrorText("none"))
		fmt.Fprintf(output, "  Available: %s\n", color.ErrorText("false"))
	}

	fmt.Fprintf(output, "\n")

	// Display migration status
	fmt.Fprintf(output, "%s\n", color.Header("Migration Status"))
	
	migrationStatus := keyring.GetMigrationStatus(cfg)
	if len(migrationStatus) == 0 {
		fmt.Fprintf(output, "  No servers configured\n")
		return nil
	}

	needsMigrationCount := 0
	usingKeyringCount := 0

	for _, status := range migrationStatus {
		icon := "✅"
		statusText := "No migration needed"
		
		if status.NeedsMigration {
			icon = "⚠️ "
			statusText = fmt.Sprintf("Needs migration (%s)", status.CredentialType)
			needsMigrationCount++
		} else if status.UsingKeyring {
			icon = "🔐"
			statusText = fmt.Sprintf("Using keyring (%s)", status.KeyringID)
			usingKeyringCount++
		}

		fmt.Fprintf(output, "  %s %s: %s\n", icon, status.ServerName, statusText)
	}

	fmt.Fprintf(output, "\n")
	fmt.Fprintf(output, "%s\n", color.Header("Summary"))
	fmt.Fprintf(output, "  Total servers: %d\n", len(migrationStatus))
	fmt.Fprintf(output, "  Using keyring: %s\n", formatCountStatus(usingKeyringCount))
	fmt.Fprintf(output, "  Need migration: %s\n", formatCountStatus(needsMigrationCount))

	if needsMigrationCount > 0 {
		fmt.Fprintf(output, "\n")
		fmt.Fprintf(output, "%s\n", color.InfoMessage("To migrate credentials, run: sshm keyring migrate"))
	}

	return nil
}

func runKeyringMigrateCommand(output io.Writer, serverName string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("❌ Failed to load configuration: %w", err)
	}

	// Check if keyring is enabled
	if !cfg.Keyring.Enabled {
		return fmt.Errorf("❌ Keyring is not enabled. Enable it in your configuration first")
	}

	// Create keyring manager
	service := cfg.Keyring.Service
	if service == "" {
		service = "auto"
	}
	namespace := cfg.Keyring.Namespace
	if namespace == "" {
		namespace = "sshm"
	}

	keyringManager := keyring.NewKeyringManagerWithNamespace(service, namespace)
	if keyringManager == nil {
		return fmt.Errorf("❌ Failed to initialize keyring service: %s", service)
	}

	if !keyringManager.IsAvailable() {
		return fmt.Errorf("❌ Keyring service is not available. Service: %s", keyringManager.ServiceName())
	}

	fmt.Fprintf(output, "%s\n", color.Header("🔐 Credential Migration"))
	fmt.Fprintf(output, "Using keyring service: %s\n", keyringManager.ServiceName())
	fmt.Fprintf(output, "\n")

	// Create a separate config for migration processing
	// This preserves the original config while allowing filtered migration
	migrationCfg := &config.Config{
		Servers: cfg.Servers,
		Keyring: cfg.Keyring,
	}

	// Filter servers if specific server requested
	// IMPORTANT: Filter the migration config, not the original config
	if serverName != "" {
		server, err := cfg.GetServer(serverName)
		if err != nil {
			return fmt.Errorf("❌ Server '%s' not found", serverName)
		}
		migrationCfg.Servers = []config.Server{*server}
	}

	// Get migration status based on the filtered config
	migrationStatus := keyring.GetMigrationStatus(migrationCfg)
	needsMigration := []keyring.MigrationStatus{}

	for _, status := range migrationStatus {
		if status.NeedsMigration {
			needsMigration = append(needsMigration, status)
		}
	}

	if len(needsMigration) == 0 {
		fmt.Fprintf(output, "%s\n", color.SuccessMessage("✅ All servers are already using keyring or don't need migration"))
		return nil
	}

	fmt.Fprintf(output, "%s\n", color.InfoText("The following servers need credential migration:"))
	for _, status := range needsMigration {
		fmt.Fprintf(output, "  • %s (%s)\n", status.ServerName, status.CredentialType)
	}
	fmt.Fprintf(output, "\n")

	// Confirm migration
	if !confirmMigration(output) {
		fmt.Fprintf(output, "%s\n", color.InfoMessage("Migration cancelled"))
		return nil
	}

	// Perform migration on the filtered config
	fmt.Fprintf(output, "\n%s\n", color.InfoMessage("Starting migration..."))

	promptFunc := func(prompt string) (string, error) {
		return promptForCredential(prompt)
	}

	results, err := keyring.MigrateFromPlaintext(migrationCfg, keyringManager, promptFunc)
	if err != nil {
		return fmt.Errorf("❌ Migration failed: %w", err)
	}

	// Display results
	fmt.Fprintf(output, "\n%s\n", color.Header("Migration Results"))

	successCount := 0
	for _, result := range results {
		if result.Success {
			fmt.Fprintf(output, "  ✅ %s: %s migrated successfully\n", 
				result.ServerName, result.CredentialType)
			successCount++
		} else {
			fmt.Fprintf(output, "  ❌ %s: %s migration failed - %v\n", 
				result.ServerName, result.CredentialType, result.Error)
		}
	}

	if successCount > 0 {
		// Apply migration results to original config
		// We need to update the original config with keyring settings from migrated servers
		updatedServers := 0
		for _, result := range results {
			if result.Success {
				// Find the server in the original config and update it
				serverFound := false
				for i := range cfg.Servers {
					if cfg.Servers[i].Name == result.ServerName {
						cfg.Servers[i].UseKeyring = true
						cfg.Servers[i].KeyringID = result.KeyringID
						serverFound = true
						updatedServers++
						break
					}
				}
				
				// This should not happen if our logic is correct, but let's be safe
				if !serverFound {
					fmt.Fprintf(output, "  ⚠️  Warning: Could not find server '%s' in original config to update\n", result.ServerName)
				}
			}
		}

		// Validate that we updated the expected number of servers
		if updatedServers != successCount {
			fmt.Fprintf(output, "  ⚠️  Warning: Updated %d servers but expected %d\n", updatedServers, successCount)
		}

		// Save updated configuration (now preserves all servers)
		err = cfg.Save()
		if err != nil {
			// If save fails after successful migration, we need to clean up keyring entries
			fmt.Fprintf(output, "\n❌ Failed to save configuration after successful migration: %v\n", err)
			fmt.Fprintf(output, "%s\n", color.WarningText("Attempting to rollback keyring changes..."))
			
			rollbackErr := keyring.RollbackMigration(cfg, keyringManager, results)
			if rollbackErr != nil {
				return fmt.Errorf("❌ Critical error: Failed to save config AND failed to rollback keyring changes: save error: %w, rollback error: %v", err, rollbackErr)
			}
			
			fmt.Fprintf(output, "%s\n", color.InfoMessage("✅ Rollback completed. Keyring entries have been removed."))
			return fmt.Errorf("❌ Migration rolled back due to config save failure: %w", err)
		}

		fmt.Fprintf(output, "\n%s\n", 
			color.SuccessMessage("✅ Migration completed! %d credentials migrated successfully", successCount))
		fmt.Fprintf(output, "%s\n", 
			color.InfoText("Your credentials are now stored securely in the keyring"))
	}

	return nil
}

func formatBoolStatus(value bool) string {
	if value {
		return color.SuccessText("true")
	}
	return color.ErrorText("false")
}

func formatCountStatus(count int) string {
	if count == 0 {
		return color.SuccessText("0")
	}
	return color.WarningText("%d", count)
}

func confirmMigration(output io.Writer) bool {
	fmt.Fprintf(output, "%s ", color.InfoText("Proceed with migration? (y/N):"))
	
	var response string
	fmt.Scanln(&response)
	
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func promptForCredential(prompt string) (string, error) {
	fmt.Print(color.InfoText("%s ", prompt))
	
	credentialBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // New line after password input
	
	if err != nil {
		return "", fmt.Errorf("failed to read credential: %w", err)
	}
	
	return string(credentialBytes), nil
}