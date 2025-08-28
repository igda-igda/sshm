package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"sshm/internal/tui"
)

// tuiCmd represents the tui command
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch interactive TUI interface",
	Long: `Launch the interactive Terminal User Interface (TUI) for SSHM.

The TUI provides a k9s-inspired interface for managing SSH connections,
browsing servers, and managing tmux sessions visually.

Key features:
  • Visual server browser with profile organization
  • Active session management
  • Keyboard-driven navigation
  • Context-aware help system

Usage:
  sshm tui    # Launch the TUI interface

Navigation:
  • Use arrow keys or j/k to navigate
  • Press Enter to connect to a server
  • Press q to quit
  • Press ? for help`,
	RunE: runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Create TUI application
	app, err := tui.NewTUIApp()
	if err != nil {
		return fmt.Errorf("failed to create TUI application: %w", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		app.Stop()
		cancel()
	}()

	// Run the application
	if err := app.Run(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("TUI application failed: %w", err)
	}

	return nil
}