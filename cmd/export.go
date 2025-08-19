package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"sshm/internal/color"
	"sshm/internal/config"
)

var (
	exportFormat  string
	exportProfile string
)

var exportCmd = &cobra.Command{
	Use:   "export [flags] <file>",
	Short: "Export server configurations to various file formats",
	Long: `Export server configurations to YAML or JSON files.

The export includes all servers and profiles unless a specific profile is selected
using the --profile flag.

Supported formats:
  • YAML (default)
  • JSON

The file format is automatically detected based on the file extension, but can be
explicitly specified using the --format flag.

Examples:
  sshm export servers.yaml                    # Export all to YAML
  sshm export servers.json                    # Export all to JSON
  sshm export --format json servers.txt       # Force JSON format
  sshm export --profile production prod.yaml  # Export specific profile`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "", "Output format (yaml, json) - auto-detected if not specified")
	exportCmd.Flags().StringVarP(&exportProfile, "profile", "p", "", "Export servers from specified profile only")
}

func runExport(cmd *cobra.Command, args []string) error {
	outputPath := args[0]
	
	// Load current configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Determine output format
	format := exportFormat
	if format == "" {
		format = detectExportFormat(outputPath)
	}
	
	// Validate format
	if format != "yaml" && format != "json" {
		return fmt.Errorf("unsupported export format: %s (supported: yaml, json)", format)
	}
	
	// Prepare export configuration
	var exportConfig config.Config
	
	if exportProfile != "" {
		// Export specific profile
		profile, err := cfg.GetProfile(exportProfile)
		if err != nil {
			return fmt.Errorf("profile '%s' not found", exportProfile)
		}
		
		// Get servers belonging to this profile
		servers, err := cfg.GetServersByProfile(exportProfile)
		if err != nil {
			return fmt.Errorf("failed to get servers for profile '%s': %w", exportProfile, err)
		}
		
		exportConfig = config.Config{
			Servers:  servers,
			Profiles: []config.Profile{*profile},
		}
		
		fmt.Printf("%s\n", color.InfoMessage("Exporting profile '%s' with %d servers", exportProfile, len(servers)))
		
	} else {
		// Export all servers and profiles
		exportConfig = config.Config{
			Servers:  cfg.GetServers(),
			Profiles: cfg.GetProfiles(),
		}
		
		fmt.Printf("%s\n", color.InfoMessage("Exporting %d servers and %d profiles", len(exportConfig.Servers), len(exportConfig.Profiles)))
	}
	
	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Marshal configuration based on format
	var data []byte
	switch format {
	case "yaml":
		data, err = yaml.Marshal(exportConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
		
	case "json":
		data, err = json.MarshalIndent(exportConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	}
	
	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}
	
	fmt.Printf("%s\n", color.SuccessMessage("Configuration exported to %s (%s format)", outputPath, format))
	
	return nil
}

// detectExportFormat determines the export format based on file extension
func detectExportFormat(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		// Default to YAML
		return "yaml"
	}
}