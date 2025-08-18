package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"sshm/internal/config"
)

var (
	importType    string
	importProfile string
)

var importCmd = &cobra.Command{
	Use:   "import [flags] <file>",
	Short: "Import server configurations from various file formats",
	Long: `Import server configurations from SSH config files, YAML, or JSON files.

Supported formats:
  • SSH config files (~/.ssh/config format)
  • YAML configuration files
  • JSON configuration files

The file type is automatically detected based on the file extension, but can be
explicitly specified using the --type flag.

Examples:
  sshm import ~/.ssh/config              # Import from SSH config
  sshm import servers.yaml               # Import from YAML file
  sshm import --type json servers.txt    # Force JSON parsing
  sshm import --profile imported servers.yaml  # Import to specific profile`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	importCmd.Flags().StringVarP(&importType, "type", "t", "", "File type (ssh, yaml, json) - auto-detected if not specified")
	importCmd.Flags().StringVarP(&importProfile, "profile", "p", "", "Import servers into specified profile")
}

func runImport(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}
	
	// Detect file type if not specified
	fileType := importType
	if fileType == "" {
		fileType = detectFileType(filePath)
	}
	
	// Validate file type
	if fileType != "ssh" && fileType != "yaml" && fileType != "json" {
		return fmt.Errorf("unsupported file type: %s (supported: ssh, yaml, json)", fileType)
	}
	
	// Load current configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	var servers []config.Server
	var profiles []config.Profile
	
	// Parse file based on type
	switch fileType {
	case "ssh":
		servers, err = config.ParseSSHConfig(filePath)
		if err != nil {
			return fmt.Errorf("failed to parse SSH config: %w", err)
		}
		
	case "yaml", "yml":
		servers, profiles, err = parseYAMLConfig(filePath)
		if err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
		
	case "json":
		servers, profiles, err = parseJSONConfig(filePath)
		if err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	}
	
	if len(servers) == 0 {
		return fmt.Errorf("no valid server configurations found in file")
	}
	
	// Import servers
	imported := 0
	updated := 0
	
	for _, server := range servers {
		// Check if server already exists
		existing, err := cfg.GetServer(server.Name)
		if err == nil {
			// Server exists - update it
			if err := cfg.RemoveServer(existing.Name); err != nil {
				fmt.Printf("Warning: failed to remove existing server %s: %v\n", existing.Name, err)
				continue
			}
			updated++
		} else {
			imported++
		}
		
		// Add the server
		if err := cfg.AddServer(server); err != nil {
			fmt.Printf("Warning: failed to import server %s: %v\n", server.Name, err)
			continue
		}
	}
	
	// Import profiles if any were found
	for _, profile := range profiles {
		// Check if profile already exists
		existing, err := cfg.GetProfile(profile.Name)
		if err == nil {
			// Profile exists - update it
			if err := cfg.RemoveProfile(existing.Name); err != nil {
				fmt.Printf("Warning: failed to remove existing profile %s: %v\n", existing.Name, err)
				continue
			}
		}
		
		if err := cfg.AddProfile(profile); err != nil {
			fmt.Printf("Warning: failed to import profile %s: %v\n", profile.Name, err)
			continue
		}
	}
	
	// If profile flag is specified, create/update profile with imported servers
	if importProfile != "" {
		var serverNames []string
		for _, server := range servers {
			serverNames = append(serverNames, server.Name)
		}
		
		profile := config.Profile{
			Name:        importProfile,
			Description: fmt.Sprintf("Servers imported from %s", filepath.Base(filePath)),
			Servers:     serverNames,
		}
		
		// Remove existing profile if it exists
		if existing, err := cfg.GetProfile(importProfile); err == nil {
			if err := cfg.RemoveProfile(existing.Name); err != nil {
				fmt.Printf("Warning: failed to remove existing profile %s: %v\n", existing.Name, err)
			}
		}
		
		if err := cfg.AddProfile(profile); err != nil {
			fmt.Printf("Warning: failed to create profile %s: %v\n", importProfile, err)
		} else {
			fmt.Printf("Created profile '%s' with %d servers\n", importProfile, len(serverNames))
		}
	}
	
	// Save configuration
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	// Print summary
	fmt.Printf("Import completed:\n")
	fmt.Printf("  • %d servers imported\n", imported)
	if updated > 0 {
		fmt.Printf("  • %d servers updated\n", updated)
	}
	if len(profiles) > 0 {
		fmt.Printf("  • %d profiles imported\n", len(profiles))
	}
	
	return nil
}

// detectFileType determines the file type based on extension
func detectFileType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	base := strings.ToLower(filepath.Base(filePath))
	
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	default:
		// Check for common SSH config file names
		if base == "config" || base == "ssh_config" || strings.Contains(base, "ssh") {
			return "ssh"
		}
		// Default to YAML for unknown extensions
		return "yaml"
	}
}

// parseYAMLConfig parses a YAML configuration file
func parseYAMLConfig(filePath string) ([]config.Server, []config.Profile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	// Validate servers
	var validServers []config.Server
	for _, server := range cfg.Servers {
		if err := server.Validate(); err != nil {
			fmt.Printf("Warning: skipping invalid server %s: %v\n", server.Name, err)
			continue
		}
		validServers = append(validServers, server)
	}
	
	return validServers, cfg.Profiles, nil
}

// parseJSONConfig parses a JSON configuration file
func parseJSONConfig(filePath string) ([]config.Server, []config.Profile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	// Validate servers
	var validServers []config.Server
	for _, server := range cfg.Servers {
		if err := server.Validate(); err != nil {
			fmt.Printf("Warning: skipping invalid server %s: %v\n", server.Name, err)
			continue
		}
		validServers = append(validServers, server)
	}
	
	return validServers, cfg.Profiles, nil
}