package color

import (
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	// Color output control
	colorOutput = true
	
	// Color functions for different text types
	headerColor    = color.New(color.Bold, color.FgBlue)
	commandColor   = color.New(color.FgCyan)
	exampleColor   = color.New(color.FgGreen)
	flagColor      = color.New(color.FgYellow)
	requiredColor  = color.New(color.Bold, color.FgRed)
	optionalColor  = color.New(color.FgMagenta)
	successColor   = color.New(color.FgGreen)
	errorColor     = color.New(color.FgRed)
	warningColor   = color.New(color.FgYellow)
	infoColor      = color.New(color.FgBlue)
)

func init() {
	// Check for NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		SetColorOutput(false)
	}
}

// SetColorOutput enables or disables color output
func SetColorOutput(enabled bool) {
	colorOutput = enabled
	color.NoColor = !enabled
}

// IsColorEnabled returns true if color output is enabled
func IsColorEnabled() bool {
	return colorOutput && !color.NoColor
}

// Header formats text as a header (bold blue)
func Header(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return headerColor.Sprint(text)
}

// Command formats text as a command name (cyan)
func Command(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return commandColor.Sprint(text)
}

// Example formats text as an example command (green)
func Example(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return exampleColor.Sprint(text)
}

// Flag formats text as a CLI flag (yellow)
func Flag(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return flagColor.Sprint(text)
}

// Required formats text as a required parameter (bold red)
func Required(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return requiredColor.Sprint(text)
}

// Optional formats text as an optional parameter (magenta)
func Optional(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return optionalColor.Sprint(text)
}

// Success formats text as a success message (green)
func Success(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return successColor.Sprint(text)
}

// Error formats text as an error message (red)
func Error(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return errorColor.Sprint(text)
}

// Warning formats text as a warning message (yellow)
func Warning(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return warningColor.Sprint(text)
}

// Info formats text as an info message (blue)
func Info(text string) string {
	if !IsColorEnabled() {
		return text
	}
	return infoColor.Sprint(text)
}

// FormatHelp enhances help text with color formatting
func FormatHelp(helpText string) string {
	if !IsColorEnabled() {
		return helpText
	}
	
	lines := strings.Split(helpText, "\n")
	var formattedLines []string
	
	for _, line := range lines {
		// Format section headers (lines ending with ':' that aren't indented)
		if strings.HasSuffix(strings.TrimSpace(line), ":") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			formattedLines = append(formattedLines, Header(line))
			continue
		}
		
		// Format command examples (lines that start with spaces and contain 'sshm')
		if (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) && strings.Contains(line, "sshm") {
			formattedLines = append(formattedLines, Example(line))
			continue
		}
		
		// Format CLI flags (lines containing --)
		if strings.Contains(line, "--") {
			// Replace flag patterns with colored versions
			formatted := line
			words := strings.Fields(line)
			for _, word := range words {
				if strings.HasPrefix(word, "--") || strings.HasPrefix(word, "-") {
					formatted = strings.ReplaceAll(formatted, word, Flag(word))
				}
			}
			formattedLines = append(formattedLines, formatted)
			continue
		}
		
		// Keep other lines as-is
		formattedLines = append(formattedLines, line)
	}
	
	return strings.Join(formattedLines, "\n")
}

// FormatCommandList formats a list of commands with colors
func FormatCommandList(commands []string) []string {
	if !IsColorEnabled() {
		return commands
	}
	
	var formatted []string
	for _, cmd := range commands {
		formatted = append(formatted, Command(cmd))
	}
	return formatted
}