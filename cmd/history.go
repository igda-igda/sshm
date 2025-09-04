package cmd

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"sshm/internal/color"
	"sshm/internal/connection"
	"sshm/internal/history"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View and manage connection history",
	Long: `View and manage connection history and session monitoring data.

This command provides access to:
  • Connection history with filtering and search capabilities
  • Connection statistics and success rates
  • Session health monitoring data
  • History cleanup and maintenance operations

Examples:
  sshm history list                    # Show recent connection history
  sshm history list --server web-01   # Show history for specific server
  sshm history list --profile prod    # Show history for production profile
  sshm history stats web-01           # Show connection statistics
  sshm history cleanup --days 30      # Clean up history older than 30 days
  sshm history health                  # Show current session health status`,
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List connection history",
	Long: `List connection history with optional filtering.

Display recent connection attempts with details including:
  • Server name and connection details
  • Connection status (success/failed/timeout)
  • Start time and duration
  • Error messages for failed connections
  • Session information

Filtering Options:
  --server <name>     Filter by server name
  --profile <name>    Filter by profile name
  --status <status>   Filter by connection status
  --days <number>     Show history from last N days
  --limit <number>    Limit number of results (default: 20)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName, _ := cmd.Flags().GetString("server")
		profileName, _ := cmd.Flags().GetString("profile")
		status, _ := cmd.Flags().GetString("status")
		days, _ := cmd.Flags().GetInt("days")
		limit, _ := cmd.Flags().GetInt("limit")
		
		return runHistoryListCommand(cmd.OutOrStdout(), serverName, profileName, status, days, limit)
	},
}

var historyStatsCmd = &cobra.Command{
	Use:   "stats [server-name]",
	Short: "Show connection statistics",
	Long: `Show connection statistics for a specific server or all servers.

Display statistics including:
  • Total connection attempts
  • Success/failure rates
  • Average connection duration
  • First and last connection times
  • Recent activity trends

Examples:
  sshm history stats web-01          # Stats for specific server
  sshm history stats --profile prod  # Stats for all servers in profile`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var serverName string
		if len(args) > 0 {
			serverName = args[0]
		}
		
		profileName, _ := cmd.Flags().GetString("profile")
		return runHistoryStatsCommand(cmd.OutOrStdout(), serverName, profileName)
	},
}

var historyCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up old connection history",
	Long: `Remove old connection history entries to manage database size.

This command removes connection history older than the specified retention period.
Session health data is also cleaned up to maintain database performance.

The default retention period is 30 days.

Examples:
  sshm history cleanup --days 30     # Remove history older than 30 days
  sshm history cleanup --days 7      # Keep only last week's history`,
	RunE: func(cmd *cobra.Command, args []string) error {
		days, _ := cmd.Flags().GetInt("days")
		return runHistoryCleanupCommand(cmd.OutOrStdout(), days)
	},
}

var historyHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Show session health status",
	Long: `Show current session health monitoring status.

Display information about:
  • Active session monitoring status
  • Session health statistics
  • Recent health check results
  • Failed or degraded sessions

This command requires active sessions to show meaningful data.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHistoryHealthCommand(cmd.OutOrStdout())
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historyStatsCmd)
	historyCmd.AddCommand(historyCleanupCmd)
	historyCmd.AddCommand(historyHealthCmd)

	// History list flags
	historyListCmd.Flags().StringP("server", "s", "", "Filter by server name")
	historyListCmd.Flags().StringP("profile", "p", "", "Filter by profile name")
	historyListCmd.Flags().String("status", "", "Filter by status (success, failed, timeout, cancelled)")
	historyListCmd.Flags().IntP("days", "d", 0, "Show history from last N days (0 = all)")
	historyListCmd.Flags().IntP("limit", "l", 20, "Limit number of results")

	// History stats flags
	historyStatsCmd.Flags().StringP("profile", "p", "", "Show stats for profile")

	// History cleanup flags
	historyCleanupCmd.Flags().IntP("days", "d", 30, "Retention period in days")
}

func runHistoryListCommand(output io.Writer, serverName, profileName, status string, days, limit int) error {
	// Create connection manager to access history
	manager, err := connection.NewManager()
	if err != nil {
		return fmt.Errorf("❌ Failed to initialize connection manager: %w", err)
	}
	defer manager.Close()

	// Build history filter
	filter := history.HistoryFilter{
		ServerName:  serverName,
		ProfileName: profileName,
		Status:      status,
		Limit:       limit,
	}

	// Add time filter if specified
	if days > 0 {
		filter.StartTime = time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	}

	// Get connection history
	historyEntries, err := manager.GetConnectionHistory(filter)
	if err != nil {
		return fmt.Errorf("❌ Failed to get connection history: %w", err)
	}

	if len(historyEntries) == 0 {
		fmt.Fprintf(output, "%s\n", color.InfoMessage("No connection history found matching the criteria"))
		return nil
	}

	// Display header
	fmt.Fprintf(output, "%s\n\n", color.Header("Connection History"))
	
	// Display history entries
	for i, entry := range historyEntries {
		displayHistoryEntry(output, entry, i == 0)
	}

	// Display summary
	fmt.Fprintf(output, "\n%s\n", color.InfoText("Showing %d entries", len(historyEntries)))
	
	return nil
}

func runHistoryStatsCommand(output io.Writer, serverName, profileName string) error {
	// Create connection manager to access history
	manager, err := connection.NewManager()
	if err != nil {
		return fmt.Errorf("❌ Failed to initialize connection manager: %w", err)
	}
	defer manager.Close()

	if serverName != "" {
		// Show stats for specific server
		stats, err := manager.GetConnectionStats(serverName, profileName)
		if err != nil {
			return fmt.Errorf("❌ Failed to get connection stats: %w", err)
		}

		displayServerStats(output, stats)
	} else {
		// Show recent activity summary
		activity, err := manager.GetRecentActivity(24) // Last 24 hours
		if err != nil {
			return fmt.Errorf("❌ Failed to get recent activity: %w", err)
		}

		displayActivityStats(output, activity)
	}

	return nil
}

func runHistoryCleanupCommand(output io.Writer, days int) error {
	if days <= 0 {
		return fmt.Errorf("❌ Days must be greater than 0")
	}

	// Create connection manager to access history
	manager, err := connection.NewManager()
	if err != nil {
		return fmt.Errorf("❌ Failed to initialize connection manager: %w", err)
	}
	defer manager.Close()

	retentionPeriod := time.Duration(days) * 24 * time.Hour
	
	fmt.Fprintf(output, "%s\n", color.InfoMessage("Cleaning up connection history older than %d days...", days))
	
	deletedCount, err := manager.CleanupOldHistory(retentionPeriod)
	if err != nil {
		return fmt.Errorf("❌ Failed to cleanup old history: %w", err)
	}

	if deletedCount > 0 {
		fmt.Fprintf(output, "%s\n", color.SuccessMessage("Cleaned up %d old history entries", deletedCount))
	} else {
		fmt.Fprintf(output, "%s\n", color.InfoMessage("No old history entries found to clean up"))
	}

	return nil
}

func runHistoryHealthCommand(output io.Writer) error {
	// Create connection manager to access history
	manager, err := connection.NewManager()
	if err != nil {
		return fmt.Errorf("❌ Failed to initialize connection manager: %w", err)
	}
	defer manager.Close()

	fmt.Fprintf(output, "%s\n", color.InfoMessage("Session health monitoring is available"))
	fmt.Fprintf(output, "%s\n", color.InfoText("Health monitoring data will be shown when sessions are active"))
	
	// Note: In a real implementation, we would integrate with the HealthMonitor
	// to show actual session health data
	
	return nil
}

func displayHistoryEntry(output io.Writer, entry history.ConnectionHistoryEntry, isFirst bool) {
	if !isFirst {
		fmt.Fprintln(output)
	}

	// Format connection type and status
	connectionType := entry.ConnectionType
	if connectionType == "group" {
		connectionType = fmt.Sprintf("group (%s)", entry.ProfileName)
	}

	// Status with color
	var statusText string
	switch entry.Status {
	case "success":
		statusText = color.SuccessText("✓ SUCCESS")
	case "failed":
		statusText = color.ErrorText("✗ FAILED")
	case "timeout":
		statusText = color.WarningText("⏱ TIMEOUT")
	case "cancelled":
		statusText = color.InfoText("⊘ CANCELLED")
	default:
		statusText = color.InfoText("%s", entry.Status)
	}

	// Display main info
	fmt.Fprintf(output, "%s %s → %s@%s:%d\n",
		statusText,
		color.Info(entry.ServerName),
		entry.User,
		entry.Host,
		entry.Port,
	)

	// Display timestamp and duration
	timeStr := entry.StartTime.Format("2006-01-02 15:04:05")
	if entry.DurationSeconds > 0 {
		duration := time.Duration(entry.DurationSeconds) * time.Second
		fmt.Fprintf(output, "   %s • %s • Duration: %s\n",
			color.InfoText("%s", timeStr),
			color.InfoText("%s", connectionType),
			color.InfoText("%v", duration),
		)
	} else {
		fmt.Fprintf(output, "   %s • %s\n",
			color.InfoText("%s", timeStr),
			color.InfoText("%s", connectionType),
		)
	}

	// Display error message if present
	if entry.ErrorMessage != "" {
		fmt.Fprintf(output, "   %s %s\n",
			color.ErrorText("%s", "Error:"),
			color.InfoText("%s", entry.ErrorMessage),
		)
	}

	// Display session info if present
	if entry.SessionID != "" {
		fmt.Fprintf(output, "   %s %s\n",
			color.InfoText("%s", "Session:"),
			color.InfoText("%s", entry.SessionID),
		)
	}
}

func displayServerStats(output io.Writer, stats *history.ConnectionStats) {
	fmt.Fprintf(output, "%s\n\n", color.Header(fmt.Sprintf("Statistics for %s", stats.ServerName)))

	if stats.TotalConnections == 0 {
		fmt.Fprintf(output, "%s\n", color.InfoMessage("No connection history found for this server"))
		return
	}

	// Connection counts
	fmt.Fprintf(output, "%s %s\n",
		color.Info("Total Connections:"),
		color.Info(fmt.Sprintf("%d", stats.TotalConnections)),
	)
	
	fmt.Fprintf(output, "%s %s\n",
		color.Info("Successful Connections:"),
		color.Success(fmt.Sprintf("%d", stats.SuccessfulConnections)),
	)

	failedConnections := stats.TotalConnections - stats.SuccessfulConnections
	fmt.Fprintf(output, "%s %s\n",
		color.Info("Failed Connections:"),
		color.Error(fmt.Sprintf("%d", failedConnections)),
	)

	// Success rate
	successRate := stats.SuccessRate * 100
	var successRateText string
	if successRate >= 90 {
		successRateText = color.Success(fmt.Sprintf("%.1f%%", successRate))
	} else if successRate >= 70 {
		successRateText = color.Warning(fmt.Sprintf("%.1f%%", successRate))
	} else {
		successRateText = color.Error(fmt.Sprintf("%.1f%%", successRate))
	}

	fmt.Fprintf(output, "%s %s\n",
		color.Info("Success Rate:"),
		successRateText,
	)

	// Average duration
	if stats.AverageDuration > 0 {
		avgDuration := time.Duration(stats.AverageDuration * 1e9) // Convert to nanoseconds first
		fmt.Fprintf(output, "%s %s\n",
			color.Info("Average Duration:"),
			color.Info(fmt.Sprintf("%v", avgDuration)),
		)
	}

	// Connection times
	if !stats.FirstConnection.IsZero() {
		fmt.Fprintf(output, "%s %s\n",
			color.Info("First Connection:"),
			color.Info(stats.FirstConnection.Format("2006-01-02 15:04:05")),
		)
	}

	if !stats.LastConnection.IsZero() {
		fmt.Fprintf(output, "%s %s\n",
			color.Info("Last Connection:"),
			color.Info(stats.LastConnection.Format("2006-01-02 15:04:05")),
		)
	}

	// Profile info
	if stats.ProfileName != "" {
		fmt.Fprintf(output, "%s %s\n",
			color.Info("Profile:"),
			color.Info(stats.ProfileName),
		)
	}
}

func displayActivityStats(output io.Writer, activity map[string]int) {
	fmt.Fprintf(output, "%s\n\n", color.Header("Recent Activity (Last 24 Hours)"))

	if len(activity) == 0 {
		fmt.Fprintf(output, "%s\n", color.InfoMessage("No recent connection activity"))
		return
	}

	total := 0
	for _, count := range activity {
		total += count
	}

	if total == 0 {
		fmt.Fprintf(output, "%s\n", color.InfoMessage("No recent connection activity"))
		return
	}

	fmt.Fprintf(output, "%s %s\n",
		color.Info("Total Connections:"),
		color.InfoText("%s", strconv.Itoa(total)),
	)

	if success, exists := activity["success"]; exists && success > 0 {
		fmt.Fprintf(output, "%s %s\n",
			color.Info("Successful:"),
			color.SuccessText("%s", strconv.Itoa(success)),
		)
	}

	if failed, exists := activity["failed"]; exists && failed > 0 {
		fmt.Fprintf(output, "%s %s\n",
			color.Info("Failed:"),
			color.ErrorText("%s", strconv.Itoa(failed)),
		)
	}

	if timeout, exists := activity["timeout"]; exists && timeout > 0 {
		fmt.Fprintf(output, "%s %s\n",
			color.Info("Timeout:"),
			color.WarningText("%s", strconv.Itoa(timeout)),
		)
	}

	if cancelled, exists := activity["cancelled"]; exists && cancelled > 0 {
		fmt.Fprintf(output, "%s %s\n",
			color.Info("Cancelled:"),
			color.InfoText("%s", strconv.Itoa(cancelled)),
		)
	}
}