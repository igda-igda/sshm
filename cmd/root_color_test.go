package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"sshm/internal/color"
)

func TestRootCommandColorOutput(t *testing.T) {
	tests := []struct {
		name      string
		colorMode bool
		checkFunc func(output string) bool
		desc      string
	}{
		{
			name:      "Color enabled - should contain ANSI codes",
			colorMode: true,
			checkFunc: func(output string) bool {
				return strings.Contains(output, "\x1b[")
			},
			desc: "Output should contain ANSI color codes when colors are enabled",
		},
		{
			name:      "Color disabled - should be plain text",
			colorMode: false,
			checkFunc: func(output string) bool {
				return !strings.Contains(output, "\x1b[")
			},
			desc: "Output should not contain ANSI color codes when colors are disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set color mode
			color.SetColorOutput(tt.colorMode)
			if tt.colorMode {
				os.Unsetenv("NO_COLOR")
			} else {
				os.Setenv("NO_COLOR", "1")
			}

			// Create a new root command with colored help
			cmd := CreateRootCommand()
			
			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Execute help command
			cmd.SetArgs([]string{"--help"})
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Command execution failed: %v", err)
			}

			output := buf.String()
			if !tt.checkFunc(output) {
				t.Errorf("%s\nGot output: %s", tt.desc, output)
			}

			// Cleanup
			os.Unsetenv("NO_COLOR")
		})
	}
}

func TestRootCommandHelpSections(t *testing.T) {
	// Enable colors for this test
	color.SetColorOutput(true)
	os.Unsetenv("NO_COLOR")

	cmd := CreateRootCommand()
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	output := buf.String()

	// Check that help contains expected sections
	expectedSections := []string{
		"Usage:",
		"Examples:",
		"Available Commands:",
		"Flags:",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Expected help output to contain section '%s', but it was missing", section)
		}
	}
}

func TestRootCommandExamplesFormatting(t *testing.T) {
	// Enable colors
	color.SetColorOutput(true)
	os.Unsetenv("NO_COLOR")

	cmd := CreateRootCommand()
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	output := buf.String()

	// Check that example commands are present (we'll enhance them with colors later)
	expectedExamples := []string{
		"sshm add production-web",
		"sshm list",
		"sshm connect production-web",
		"sshm batch --profile staging",
	}

	for _, example := range expectedExamples {
		if !strings.Contains(output, example) {
			t.Errorf("Expected help output to contain example '%s', but it was missing", example)
		}
	}
}

func TestRootCommandWithColorFormatting(t *testing.T) {
	// This test will verify that our color formatting is applied correctly
	// once we implement the custom help template
	color.SetColorOutput(true)
	os.Unsetenv("NO_COLOR")

	cmd := CreateRootCommand()
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Get the long description and apply color formatting
	longDesc := cmd.Long
	coloredDesc := color.FormatHelp(longDesc)

	// Verify that FormatHelp function adds colors to the description
	if !strings.Contains(coloredDesc, "\x1b[") && color.IsColorEnabled() {
		t.Error("FormatHelp should add color codes to help text when colors are enabled")
	}

	// Verify specific formatting patterns
	if color.IsColorEnabled() {
		// Check that the formatted output contains ANSI codes
		if !strings.Contains(coloredDesc, "\x1b[") {
			t.Error("Expected colored description to contain ANSI codes")
		}

		// Check that "Features:" gets header formatting (look for the pattern)
		if strings.Contains(coloredDesc, "Features:") && !strings.Contains(coloredDesc, color.Header("Features:")) {
			t.Error("Expected 'Features:' to be formatted as a header")
		}

		// Check that "Examples:" gets header formatting (look for the pattern)
		if strings.Contains(coloredDesc, "Examples:") && !strings.Contains(coloredDesc, color.Header("Examples:")) {
			t.Error("Expected 'Examples:' to be formatted as a header")
		}

		// Check that the output contains green color codes (for examples)
		if strings.Contains(coloredDesc, "sshm add") && !strings.Contains(coloredDesc, "\x1b[32m") {
			t.Error("Expected command examples to be formatted with green color codes")
		}
	}
}

func TestRootCommandPlainTextFallback(t *testing.T) {
	// Test that when colors are disabled, we get plain text
	color.SetColorOutput(false)
	os.Setenv("NO_COLOR", "1")

	cmd := CreateRootCommand()
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	output := buf.String()

	// Verify no ANSI codes present
	if strings.Contains(output, "\x1b[") {
		t.Error("Expected plain text output when colors are disabled, but found ANSI codes")
	}

	// Verify content is still present
	if !strings.Contains(output, "SSHM is a CLI SSH connection manager") {
		t.Errorf("Expected help text content to be present even when colors are disabled. Got output: %s", output)
	}

	// Cleanup
	os.Unsetenv("NO_COLOR")
}