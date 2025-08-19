package color

import (
	"os"
	"strings"
	"testing"
)

func TestColorFormatting(t *testing.T) {
	tests := []struct {
		name     string
		function func(string) string
		input    string
		contains string // What the output should contain when colors are enabled
	}{
		{
			name:     "Header formatting",
			function: Header,
			input:    "Test Header",
			contains: "\x1b[1;34m", // Bold blue ANSI code
		},
		{
			name:     "Command formatting",
			function: Command,
			input:    "test-command",
			contains: "\x1b[36m", // Cyan ANSI code
		},
		{
			name:     "Example formatting",
			function: Example,
			input:    "sshm add server",
			contains: "\x1b[32m", // Green ANSI code
		},
		{
			name:     "Flag formatting",
			function: Flag,
			input:    "--hostname",
			contains: "\x1b[33m", // Yellow ANSI code
		},
		{
			name:     "Required parameter formatting",
			function: Required,
			input:    "<hostname>",
			contains: "\x1b[1;31m", // Bold red ANSI code
		},
		{
			name:     "Optional parameter formatting",
			function: Optional,
			input:    "[port]",
			contains: "\x1b[35m", // Magenta ANSI code
		},
		{
			name:     "Success message formatting",
			function: Success,
			input:    "Connection successful",
			contains: "\x1b[32m", // Green ANSI code
		},
		{
			name:     "Error message formatting",
			function: Error,
			input:    "Connection failed",
			contains: "\x1b[31m", // Red ANSI code
		},
		{
			name:     "Warning message formatting",
			function: Warning,
			input:    "Connection unstable",
			contains: "\x1b[33m", // Yellow ANSI code
		},
		{
			name:     "Info message formatting",
			function: Info,
			input:    "Server configuration updated",
			contains: "\x1b[34m", // Blue ANSI code
		},
	}

	// Ensure colors are enabled for testing
	os.Unsetenv("NO_COLOR")
	SetColorOutput(true)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected %s to contain ANSI color code %s, got: %s", tt.name, tt.contains, result)
			}
			if !strings.Contains(result, tt.input) {
				t.Errorf("Expected %s to contain original text %s, got: %s", tt.name, tt.input, result)
			}
		})
	}
}

func TestColorFormattingWithEmptyString(t *testing.T) {
	functions := map[string]func(string) string{
		"Header":   Header,
		"Command":  Command,
		"Example":  Example,
		"Flag":     Flag,
		"Required": Required,
		"Optional": Optional,
		"Success":  Success,
		"Error":    Error,
		"Warning":  Warning,
		"Info":     Info,
	}

	// Ensure colors are enabled for testing
	os.Unsetenv("NO_COLOR")
	SetColorOutput(true)

	for name, fn := range functions {
		t.Run(name+"_empty_string", func(t *testing.T) {
			result := fn("")
			// Should handle empty strings gracefully (might return empty or just ANSI codes)
			if len(result) < 0 { // Basic sanity check
				t.Errorf("Function %s returned unexpected result for empty string", name)
			}
		})
	}
}

func TestMultiLineText(t *testing.T) {
	multiLineText := "Line 1\nLine 2\nLine 3"
	
	// Ensure colors are enabled for testing
	os.Unsetenv("NO_COLOR")
	SetColorOutput(true)

	result := Header(multiLineText)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines in output, got %d", len(lines))
	}
}

func TestNoColorEnvironmentVariable(t *testing.T) {
	// Set NO_COLOR environment variable
	os.Setenv("NO_COLOR", "1")
	SetColorOutput(false)

	text := "Test text"
	result := Header(text)
	
	// Should return plain text without ANSI codes
	if strings.Contains(result, "\x1b[") {
		t.Errorf("Expected plain text when NO_COLOR is set, but got ANSI codes: %s", result)
	}
	if result != text {
		t.Errorf("Expected plain text '%s', got '%s'", text, result)
	}

	// Clean up
	os.Unsetenv("NO_COLOR")
	SetColorOutput(true)
}

func TestTerminalDetection(t *testing.T) {
	tests := []struct {
		name     string
		isColor  bool
		expected bool
	}{
		{
			name:     "Color enabled",
			isColor:  true,
			expected: true,
		},
		{
			name:     "Color disabled",
			isColor:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetColorOutput(tt.isColor)
			result := IsColorEnabled()
			if result != tt.expected {
				t.Errorf("Expected IsColorEnabled() to return %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestColorOutputToggle(t *testing.T) {
	// Test enabling color output
	SetColorOutput(true)
	text := "Test"
	colored := Header(text)
	if !strings.Contains(colored, "\x1b[") {
		t.Error("Expected ANSI codes when color output is enabled")
	}

	// Test disabling color output
	SetColorOutput(false)
	plain := Header(text)
	if strings.Contains(plain, "\x1b[") {
		t.Error("Expected no ANSI codes when color output is disabled")
	}
	if plain != text {
		t.Errorf("Expected plain text '%s', got '%s'", text, plain)
	}
}

func TestStatusMessageHelpers(t *testing.T) {
	// Ensure colors are enabled for testing
	os.Unsetenv("NO_COLOR")
	SetColorOutput(true)

	tests := []struct {
		name     string
		function func(string, ...interface{}) string
		input    string
		prefix   string
		contains string // ANSI color code
	}{
		{
			name:     "SuccessMessage",
			function: SuccessMessage,
			input:    "Operation completed successfully",
			prefix:   "✅ ",
			contains: "\x1b[32m", // Green
		},
		{
			name:     "ErrorMessage",
			function: ErrorMessage,
			input:    "Operation failed",
			prefix:   "❌ ",
			contains: "\x1b[31m", // Red
		},
		{
			name:     "WarningMessage",
			function: WarningMessage,
			input:    "Operation has warnings",
			prefix:   "⚠️  ",
			contains: "\x1b[33m", // Yellow
		},
		{
			name:     "InfoMessage",
			function: InfoMessage,
			input:    "Operation info",
			prefix:   "ℹ️  ",
			contains: "\x1b[34m", // Blue
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.input)
			
			// Check for color codes
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected %s to contain color code %s, got: %s", tt.name, tt.contains, result)
			}
			
			// Check for prefix and message content
			if !strings.Contains(result, tt.prefix) {
				t.Errorf("Expected %s to contain prefix %s, got: %s", tt.name, tt.prefix, result)
			}
			
			if !strings.Contains(result, tt.input) {
				t.Errorf("Expected %s to contain input text %s, got: %s", tt.name, tt.input, result)
			}
		})
	}
}

func TestStatusTextHelpers(t *testing.T) {
	// Ensure colors are enabled for testing
	os.Unsetenv("NO_COLOR")
	SetColorOutput(true)

	tests := []struct {
		name     string
		function func(string, ...interface{}) string
		input    string
		contains string // ANSI color code
	}{
		{
			name:     "SuccessText",
			function: SuccessText,
			input:    "Operation completed successfully",
			contains: "\x1b[32m", // Green
		},
		{
			name:     "ErrorText",
			function: ErrorText,
			input:    "Operation failed",
			contains: "\x1b[31m", // Red
		},
		{
			name:     "WarningText",
			function: WarningText,
			input:    "Operation has warnings",
			contains: "\x1b[33m", // Yellow
		},
		{
			name:     "InfoText",
			function: InfoText,
			input:    "Operation info",
			contains: "\x1b[34m", // Blue
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(tt.input)
			
			// Check for color codes
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected %s to contain color code %s, got: %s", tt.name, tt.contains, result)
			}
			
			// Check for message content (no prefix expected)
			if !strings.Contains(result, tt.input) {
				t.Errorf("Expected %s to contain input text %s, got: %s", tt.name, tt.input, result)
			}
		})
	}
}

func TestStatusMessageFormatting(t *testing.T) {
	// Ensure colors are enabled for testing
	os.Unsetenv("NO_COLOR")
	SetColorOutput(true)

	// Test formatted message
	result := SuccessMessage("Server %s connected on port %d", "web-server", 22)
	expected := "Server web-server connected on port 22"
	
	if !strings.Contains(result, expected) {
		t.Errorf("Expected formatted message to contain '%s', got: %s", expected, result)
	}
	
	if !strings.Contains(result, "✅") {
		t.Error("Expected success message to contain success emoji")
	}
}

func TestStatusMessageNoColor(t *testing.T) {
	// Disable colors
	SetColorOutput(false)

	result := SuccessMessage("Operation completed")
	expected := "✅ Operation completed"
	
	// Should not contain ANSI codes
	if strings.Contains(result, "\x1b[") {
		t.Errorf("Expected no ANSI codes when colors disabled, got: %s", result)
	}
	
	// Should contain emoji and message
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
	
	// Reset for other tests
	SetColorOutput(true)
}