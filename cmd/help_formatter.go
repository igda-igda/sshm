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
func applyColorFormattingToAllCommands() {
	commands := []*cobra.Command{
		addCmd,
		connectCmd,
		listCmd,
		removeCmd,
		batchCmd,
		profileCmd,
		sessionsCmd,
		importCmd,
		exportCmd,
	}

	for _, cmd := range commands {
		setColorHelpFunc(cmd)
		
		// Also apply to any subcommands
		for _, subCmd := range cmd.Commands() {
			setColorHelpFunc(subCmd)
		}
	}
}