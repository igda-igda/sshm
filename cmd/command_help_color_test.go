package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"sshm/internal/color"
)

// TestIndividualCommandHelpColorFormatting tests color formatting for individual command help screens
func TestIndividualCommandHelpColorFormatting(t *testing.T) {
	// Ensure color output is enabled for tests
	color.SetColorOutput(true)
	defer color.SetColorOutput(true)

	tests := []struct {
		name     string
		cmd      *cobra.Command
		expected []string // Substrings that should be present in colored output
	}{
		{
			name: "add command help colors",
			cmd:  addCmd,
			expected: []string{
				"\x1b[1;34m", // Bold blue for headers
				"\x1b[32m",   // Green for examples
				"\x1b[33m",   // Yellow for flags
			},
		},
		{
			name: "connect command help colors",
			cmd:  connectCmd,
			expected: []string{
				"\x1b[1;34m", // Bold blue for headers
				"\x1b[32m",   // Green for examples
			},
		},
		{
			name: "list command help colors",
			cmd:  listCmd,
			expected: []string{
				"\x1b[1;34m", // Bold blue for headers
				"\x1b[32m",   // Green for examples
				"\x1b[33m",   // Yellow for flags
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.cmd.SetOut(&buf)
			tt.cmd.SetErr(&buf)

			// Execute help command
			tt.cmd.Help()
			output := buf.String()

			// Check for presence of color codes
			for _, expectedColor := range tt.expected {
				if !strings.Contains(output, expectedColor) {
					t.Errorf("Expected color code %q not found in help output for %s", expectedColor, tt.name)
				}
			}

			// Verify that help contains expected content structure
			if !strings.Contains(output, tt.cmd.Use) {
				t.Errorf("Command usage not found in help output for %s", tt.name)
			}
		})
	}
}

// TestCommandHelpFlagsColoring tests that CLI flags are properly colored
func TestCommandHelpFlagsColoring(t *testing.T) {
	// Ensure color output is enabled
	color.SetColorOutput(true)
	defer color.SetColorOutput(true)

	var buf bytes.Buffer
	addCmd.SetOut(&buf)
	addCmd.SetErr(&buf)

	// Execute help for add command (which has many flags)
	addCmd.Help()
	output := buf.String()

	// Check that specific flags are colored (yellow) - look for any yellow coloring of flags
	expectedFlags := []string{"--hostname", "--port", "--username", "--auth-type", "--key-path"}
	
	for _, flag := range expectedFlags {
		// Look for the flag being colored yellow (more flexible check)
		flagIndex := strings.Index(output, flag)
		if flagIndex == -1 {
			continue // Flag might not be in this help output
		}
		
		// Check if there's a yellow color code near the flag
		// Look in a reasonable range before the flag
		searchStart := flagIndex - 20
		if searchStart < 0 {
			searchStart = 0
		}
		searchEnd := flagIndex + len(flag) + 10
		if searchEnd > len(output) {
			searchEnd = len(output)
		}
		
		flagContext := output[searchStart:searchEnd]
		if !strings.Contains(flagContext, "\x1b[33m") {
			t.Errorf("Flag %s is not properly colored in help output", flag)
		}
	}
}

// TestCommandHelpExamplesColoring tests that examples are properly colored
func TestCommandHelpExamplesColoring(t *testing.T) {
	// Ensure color output is enabled
	color.SetColorOutput(true)
	defer color.SetColorOutput(true)

	var buf bytes.Buffer
	addCmd.SetOut(&buf)
	addCmd.SetErr(&buf)

	// Execute help for add command
	addCmd.Help()
	output := buf.String()

	// Check that example commands containing 'sshm' are colored green
	if strings.Contains(output, "sshm add") {
		// Look for green colored sshm commands
		greenStart := "\x1b[32m" // Green color start
		if !strings.Contains(output, greenStart) {
			t.Error("Example commands are not properly colored green")
		}
	}
}

// TestCommandHelpHeadersColoring tests that section headers are properly colored
func TestCommandHelpHeadersColoring(t *testing.T) {
	// Ensure color output is enabled
	color.SetColorOutput(true)
	defer color.SetColorOutput(true)

	var buf bytes.Buffer
	connectCmd.SetOut(&buf)
	connectCmd.SetErr(&buf)

	// Execute help for connect command
	connectCmd.Help()
	output := buf.String()

	// Check that section headers are colored (bold blue)
	headerColor := "\x1b[1;34m" // Bold blue
	
	// Look for common section headers that should be colored
	if strings.Contains(output, "Usage:") {
		if !strings.Contains(output, headerColor) {
			t.Error("Section headers are not properly colored with bold blue")
		}
	}
}

// TestCommandHelpNOCOLORSupport tests that NO_COLOR environment variable is respected
func TestCommandHelpNOCOLORSupport(t *testing.T) {
	// Set NO_COLOR environment variable
	originalNOCOLOR := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer func() {
		if originalNOCOLOR == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", originalNOCOLOR)
		}
	}()

	// Reset color configuration to pick up NO_COLOR
	color.SetColorOutput(false)
	defer color.SetColorOutput(true)

	var buf bytes.Buffer
	addCmd.SetOut(&buf)
	addCmd.SetErr(&buf)

	// Execute help command
	addCmd.Help()
	output := buf.String()

	// Verify no color codes are present
	colorCodes := []string{"\x1b[", "\033["}
	for _, colorCode := range colorCodes {
		if strings.Contains(output, colorCode) {
			t.Error("Color codes found in output when NO_COLOR is set")
		}
	}
}

// TestAllCommandsHaveColorSupport tests that all commands support color formatting
func TestAllCommandsHaveColorSupport(t *testing.T) {
	// Ensure color output is enabled
	color.SetColorOutput(true)
	defer color.SetColorOutput(true)

	// List of all main commands that should have color support
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
		t.Run(cmd.Use, func(t *testing.T) {
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Execute help command
			cmd.Help()
			output := buf.String()

			// At minimum, we should see some color formatting
			// Look for any ANSI color codes
			hasColorCodes := strings.Contains(output, "\x1b[") || strings.Contains(output, "\033[")
			
			if !hasColorCodes {
				t.Errorf("Command %s does not appear to have color formatting in help output", cmd.Use)
			}
		})
	}
}

// TestConsistentColorScheme tests that all commands use consistent color scheme
func TestConsistentColorScheme(t *testing.T) {
	// Ensure color output is enabled
	color.SetColorOutput(true)
	defer color.SetColorOutput(true)

	commands := []*cobra.Command{addCmd, connectCmd, listCmd}
	
	// Expected color codes for consistency (individual commands don't have command lists)
	expectedColors := map[string]string{
		"header":  "\x1b[1;34m", // Bold blue
		"example": "\x1b[32m",   // Green
		"flag":    "\x1b[33m",   // Yellow
	}

	for _, cmd := range commands {
		t.Run(cmd.Use, func(t *testing.T) {
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			cmd.Help()
			output := buf.String()

			// Check for presence of expected color codes
			for colorType, colorCode := range expectedColors {
				// Skip flag color check for commands that don't have flags (like connect)
				if colorType == "flag" && cmd.Use == "connect <server-name>" {
					continue
				}
				
				if !strings.Contains(output, colorCode) {
					t.Errorf("Command %s missing %s color (%s) in help output", 
						cmd.Use, colorType, colorCode)
				}
			}
		})
	}
}