package cmd

import (
  "fmt"
  "io"
  "text/tabwriter"

  "github.com/spf13/cobra"
  "sshm/internal/tmux"
)

var sessionsCmd = &cobra.Command{
  Use:   "sessions",
  Short: "Manage tmux sessions",
  Long: `Manage active tmux sessions created by sshm.
  
View, list, and clean up tmux sessions created for SSH connections.
This command helps manage session resources and clean up orphaned sessions.

Examples:
  sshm sessions list               # List all active tmux sessions
  sshm sessions kill <session>    # Kill a specific session
  sshm sessions cleanup           # Remove orphaned sshm sessions`,
}

var sessionsListCmd = &cobra.Command{
  Use:   "list",
  Short: "List all active tmux sessions",
  Long: `List all active tmux sessions with their names and status.
  
Shows both individual server sessions and group profile sessions
created by sshm, helping you identify active connections and their status.`,
  RunE: func(cmd *cobra.Command, args []string) error {
    return runSessionsListCommand(cmd.OutOrStdout())
  },
}

var sessionsKillCmd = &cobra.Command{
  Use:   "kill <session-name>",
  Short: "Kill a specific tmux session",
  Long: `Kill a specific tmux session by name.
  
This will terminate the specified session and all windows within it,
closing any active SSH connections in that session.

Examples:
  sshm sessions kill production-web    # Kill session for production-web server
  sshm sessions kill development       # Kill group session for development profile`,
  Args: cobra.ExactArgs(1),
  RunE: func(cmd *cobra.Command, args []string) error {
    sessionName := args[0]
    return runSessionsKillCommand(sessionName, cmd.OutOrStdout())
  },
}

var sessionsCleanupCmd = &cobra.Command{
  Use:   "cleanup",
  Short: "Clean up orphaned tmux sessions",
  Long: `Clean up orphaned tmux sessions that may be left running.
  
This command will identify and optionally remove tmux sessions that
appear to be orphaned or no longer needed. Use with caution as this
will terminate active SSH connections.`,
  RunE: func(cmd *cobra.Command, args []string) error {
    force, _ := cmd.Flags().GetBool("force")
    return runSessionsCleanupCommand(force, cmd.OutOrStdout())
  },
}

func init() {
  sessionsCleanupCmd.Flags().BoolP("force", "f", false, "Force cleanup without confirmation")
  
  sessionsCmd.AddCommand(sessionsListCmd)
  sessionsCmd.AddCommand(sessionsKillCmd)
  sessionsCmd.AddCommand(sessionsCleanupCmd)
}

func runSessionsListCommand(output io.Writer) error {
  // Initialize tmux manager
  tmuxManager := tmux.NewManager()

  // Check if tmux is available
  if !tmuxManager.IsAvailable() {
    return fmt.Errorf("❌ tmux is not available on this system. Please install tmux to use session management")
  }

  // Get list of sessions
  sessions, err := tmuxManager.ListSessions()
  if err != nil {
    return fmt.Errorf("❌ Failed to list tmux sessions: %w", err)
  }

  if len(sessions) == 0 {
    fmt.Fprintln(output, "📋 No active tmux sessions found.")
    fmt.Fprintln(output, "💡 Use 'sshm connect <server>' or 'sshm batch --profile <profile>' to create sessions.")
    return nil
  }

  // Create formatted table output
  w := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
  fmt.Fprintln(w, "SESSION NAME\tTYPE\tSTATUS")
  fmt.Fprintln(w, "------------\t----\t------")

  for _, sessionName := range sessions {
    sessionType := "Individual"
    if isGroupSession(sessionName) {
      sessionType = "Group"
    }
    
    fmt.Fprintf(w, "%s\t%s\tActive\n", sessionName, sessionType)
  }

  w.Flush()
  
  fmt.Fprintf(output, "\n📊 Active sessions: %d\n", len(sessions))
  fmt.Fprintln(output, "💡 Use 'sshm sessions kill <session-name>' to terminate a session")
  fmt.Fprintln(output, "💡 Use 'tmux attach-session -t <session-name>' to attach to a session")
  return nil
}

func runSessionsKillCommand(sessionName string, output io.Writer) error {
  // Initialize tmux manager
  tmuxManager := tmux.NewManager()

  // Check if tmux is available
  if !tmuxManager.IsAvailable() {
    return fmt.Errorf("❌ tmux is not available on this system")
  }

  // Check if session exists
  sessions, err := tmuxManager.ListSessions()
  if err != nil {
    return fmt.Errorf("❌ Failed to list tmux sessions: %w", err)
  }

  sessionExists := false
  for _, session := range sessions {
    if session == sessionName {
      sessionExists = true
      break
    }
  }

  if !sessionExists {
    return fmt.Errorf("❌ Session '%s' not found", sessionName)
  }

  // Kill the session
  fmt.Fprintf(output, "🔪 Killing tmux session '%s'...\n", sessionName)
  err = tmuxManager.KillSession(sessionName)
  if err != nil {
    return fmt.Errorf("❌ Failed to kill session '%s': %w", sessionName, err)
  }

  fmt.Fprintf(output, "✅ Session '%s' terminated successfully\n", sessionName)
  return nil
}

func runSessionsCleanupCommand(force bool, output io.Writer) error {
  // Initialize tmux manager
  tmuxManager := tmux.NewManager()

  // Check if tmux is available
  if !tmuxManager.IsAvailable() {
    return fmt.Errorf("❌ tmux is not available on this system")
  }

  // Get list of sessions
  sessions, err := tmuxManager.ListSessions()
  if err != nil {
    return fmt.Errorf("❌ Failed to list tmux sessions: %w", err)
  }

  if len(sessions) == 0 {
    fmt.Fprintln(output, "📋 No active tmux sessions found.")
    return nil
  }

  fmt.Fprintf(output, "🔍 Found %d active tmux session(s):\n", len(sessions))
  for _, sessionName := range sessions {
    sessionType := "Individual"
    if isGroupSession(sessionName) {
      sessionType = "Group"
    }
    fmt.Fprintf(output, "   • %s (%s)\n", sessionName, sessionType)
  }

  if !force {
    fmt.Fprintln(output, "\n⚠️  Session cleanup will terminate all SSH connections in these sessions.")
    fmt.Fprintln(output, "💡 Use 'sshm sessions cleanup --force' to proceed with cleanup")
    fmt.Fprintln(output, "💡 Use 'sshm sessions kill <session-name>' to terminate specific sessions")
    return nil
  }

  // Force cleanup - kill all sessions
  fmt.Fprintf(output, "\n🔪 Force cleanup enabled. Terminating %d session(s)...\n", len(sessions))
  
  successCount := 0
  for _, sessionName := range sessions {
    err := tmuxManager.KillSession(sessionName)
    if err != nil {
      fmt.Fprintf(output, "❌ Failed to kill session '%s': %v\n", sessionName, err)
    } else {
      fmt.Fprintf(output, "✅ Terminated session '%s'\n", sessionName)
      successCount++
    }
  }

  fmt.Fprintf(output, "\n📊 Cleanup complete: %d/%d sessions terminated\n", successCount, len(sessions))
  return nil
}

// isGroupSession determines if a session name represents a group session
// Group sessions typically don't contain port numbers or specific server indicators
func isGroupSession(sessionName string) bool {
  // Simple heuristic: if the session name doesn't contain common server naming patterns,
  // it's likely a group/profile session
  // This is a basic implementation and could be enhanced with more sophisticated detection
  
  // Check for patterns that indicate individual server sessions
  // Individual servers often have patterns like:
  // - Contains dots (server.domain.com style)
  // - Contains port numbers
  // - Contains underscores from normalization of dots
  
  // If it contains underscores (from normalized dots), it's likely an individual server
  if containsAnyChar(sessionName, "_") {
    // Contains underscores (from normalized dots), likely an individual server
    return false
  }
  
  // If it contains numeric suffixes like -1, -2, it might be a conflict-resolved session
  if containsAnyChar(sessionName, "-") {
    // Could be either type, default to individual for safety
    return false
  }
  
  // Simple name without separators is likely a profile
  return true
}

// containsAnyChar checks if a string contains any of the specified characters
func containsAnyChar(str string, chars ...string) bool {
  for _, char := range chars {
    for _, c := range str {
      if string(c) == char {
        return true
      }
    }
  }
  return false
}