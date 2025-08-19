package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"sshm/internal/color"
)

// setColorHelpFunc applies color formatting to command help output
func setColorHelpFunc(cmd *cobra.Command) {
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
}

// applyColorFormattingToAllCommands applies color help formatting to all commands
// This should be called after all commands are initialized
func applyColorFormattingToAllCommands() {
	// Get commands directly from the rootCmd to ensure they're initialized
	for _, cmd := range rootCmd.Commands() {
		setColorHelpFunc(cmd)
		
		// Also apply to any subcommands recursively
		applyColorFormattingRecursively(cmd)
	}
}

// applyColorFormattingRecursively applies color formatting to a command and all its subcommands
func applyColorFormattingRecursively(cmd *cobra.Command) {
	for _, subCmd := range cmd.Commands() {
		setColorHelpFunc(subCmd)
		applyColorFormattingRecursively(subCmd)
	}
}