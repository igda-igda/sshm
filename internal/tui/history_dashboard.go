package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sshm/internal/connection"
	"sshm/internal/history"
)

// HistoryDashboard represents the TUI history dashboard
type HistoryDashboard struct {
	app              *tview.Application
	layout           *tview.Flex
	historyTable     *tview.Table
	statsPanel       *tview.TextView
	filterPanel      *tview.TextView
	statusBar        *tview.TextView
	manager          *connection.Manager
	
	// State management
	historyEntries   []history.ConnectionHistoryEntry
	currentStats     *history.ConnectionStats
	recentActivity   map[string]int
	
	// Filter state
	serverFilter     string
	profileFilter    string
	statusFilter     string
	daysFilter       int
	limitFilter      int
	selectedRow      int
	focusedPanel     string // "history", "stats"
	
	// UI state
	showingStats     bool // Toggle between history list and stats view
}

// NewHistoryDashboard creates a new history dashboard
func NewHistoryDashboard(app *tview.Application) (*HistoryDashboard, error) {
	// Create connection manager to access history
	manager, err := connection.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize connection manager: %w", err)
	}

	dashboard := &HistoryDashboard{
		app:          app,
		manager:      manager,
		limitFilter:  20,          // Default limit
		selectedRow:  1,           // Start at first data row
		focusedPanel: "history",   // Default focus on history
		showingStats: false,       // Start with history view
	}

	if err := dashboard.setupLayout(); err != nil {
		manager.Close()
		return nil, fmt.Errorf("failed to setup dashboard layout: %w", err)
	}

	// Load initial data
	if err := dashboard.refreshData(); err != nil {
		manager.Close()
		return nil, fmt.Errorf("failed to load initial data: %w", err)
	}

	return dashboard, nil
}

// setupLayout initializes the dashboard UI layout
func (hd *HistoryDashboard) setupLayout() error {
	// Create history table
	hd.historyTable = tview.NewTable()
	hd.historyTable.SetBorder(true).SetTitle(" Connection History ")
	hd.historyTable.SetBorders(false)
	hd.historyTable.SetSelectable(true, false)
	hd.historyTable.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite))

	// Setup history table headers
	hd.historyTable.SetCell(0, 0, tview.NewTableCell("Server").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))
	hd.historyTable.SetCell(0, 1, tview.NewTableCell("Status").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))
	hd.historyTable.SetCell(0, 2, tview.NewTableCell("Host").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))
	hd.historyTable.SetCell(0, 3, tview.NewTableCell("Time").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))
	hd.historyTable.SetCell(0, 4, tview.NewTableCell("Duration").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignRight))
	hd.historyTable.SetCell(0, 5, tview.NewTableCell("Profile").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))

	// Create stats panel
	hd.statsPanel = tview.NewTextView()
	hd.statsPanel.SetBorder(true).SetTitle(" Statistics ")
	hd.statsPanel.SetDynamicColors(true).SetWrap(true)

	// Create filter panel
	hd.filterPanel = tview.NewTextView()
	hd.filterPanel.SetBorder(true).SetTitle(" Filters ")
	hd.filterPanel.SetDynamicColors(true)

	// Create status bar
	hd.statusBar = tview.NewTextView()
	hd.statusBar.SetDynamicColors(true)

	// Create right pane with stats and filters
	rightPane := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(hd.statsPanel, 0, 6, false).    // Stats panel takes 60%
		AddItem(hd.filterPanel, 0, 4, false)    // Filter panel takes 40%

	// Create main layout: history table (70%) and right pane (30%)
	mainLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(hd.historyTable, 0, 7, true).   // History takes 70%
		AddItem(rightPane, 0, 3, false)         // Right pane takes 30%

	// Create overall layout with status bar at bottom
	hd.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainLayout, 0, 1, true).
		AddItem(hd.statusBar, 1, 0, false)

	// Setup key bindings
	hd.setupKeyBindings()

	// Update UI state
	hd.updatePanelHighlight()
	hd.updateStatusBar()

	return nil
}

// setupKeyBindings configures key bindings for the dashboard
func (hd *HistoryDashboard) setupKeyBindings() {
	hd.layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle special keys
		switch event.Key() {
		case tcell.KeyUp:
			hd.navigateUp()
			return nil
		case tcell.KeyDown:
			hd.navigateDown()
			return nil
		case tcell.KeyEscape:
			// Return nil to let parent handle (close dashboard)
			return event
		}

		// Handle character keys
		switch event.Rune() {
		case 'q', 'Q':
			// Return event to let parent handle (close dashboard)
			return event
		case 'r', 'R':
			// Refresh data
			hd.refreshData()
			return nil
		case 't', 'T':
			// Toggle between history and stats view
			hd.toggleView()
			return nil
		case 'f', 'F':
			// Show filter modal
			hd.showFilterModal()
			return nil
		case 'c', 'C':
			// Clear filters
			hd.clearFilters()
			return nil
		case 'h', 'H':
			// Show help
			hd.showHelp()
			return nil
		case '1':
			// Filter by success status
			hd.statusFilter = "success"
			hd.refreshData()
			return nil
		case '2':
			// Filter by failed status
			hd.statusFilter = "failed"
			hd.refreshData()
			return nil
		case '3':
			// Filter by timeout status
			hd.statusFilter = "timeout"
			hd.refreshData()
			return nil
		case '0':
			// Clear status filter
			hd.statusFilter = ""
			hd.refreshData()
			return nil
		}

		return event
	})
}

// updatePanelHighlight updates the visual highlighting of focused panel
func (hd *HistoryDashboard) updatePanelHighlight() {
	if hd.focusedPanel == "history" {
		hd.historyTable.SetBorderColor(tcell.ColorYellow)
		hd.statsPanel.SetBorderColor(tcell.ColorWhite)
	} else {
		hd.historyTable.SetBorderColor(tcell.ColorWhite)
		hd.statsPanel.SetBorderColor(tcell.ColorYellow)
	}
}

// navigateUp moves selection up in the history list
func (hd *HistoryDashboard) navigateUp() {
	if hd.historyTable.GetRowCount() <= 1 {
		return // Only header row exists
	}
	
	if hd.selectedRow > 1 {
		hd.selectedRow--
		hd.historyTable.Select(hd.selectedRow, 0)
	}
}

// navigateDown moves selection down in the history list
func (hd *HistoryDashboard) navigateDown() {
	rowCount := hd.historyTable.GetRowCount()
	if rowCount <= 1 {
		return // Only header row exists
	}
	
	if hd.selectedRow < rowCount-1 {
		hd.selectedRow++
		hd.historyTable.Select(hd.selectedRow, 0)
	}
}

// toggleView toggles between history list and statistics view
func (hd *HistoryDashboard) toggleView() {
	hd.showingStats = !hd.showingStats
	
	if hd.showingStats {
		// Show stats view - update title and content
		hd.historyTable.SetTitle(" Connection Statistics ")
		hd.displayStatsInTable()
	} else {
		// Show history view - restore normal display
		hd.historyTable.SetTitle(" Connection History ")
		hd.displayHistoryInTable()
	}
	
	hd.updateStatusBar()
}

// refreshData refreshes all dashboard data
func (hd *HistoryDashboard) refreshData() error {
	// Load connection history with current filters
	filter := history.HistoryFilter{
		ServerName:  hd.serverFilter,
		ProfileName: hd.profileFilter,
		Status:      hd.statusFilter,
		Limit:       hd.limitFilter,
	}

	// Add time filter if specified
	if hd.daysFilter > 0 {
		filter.StartTime = time.Now().Add(-time.Duration(hd.daysFilter) * 24 * time.Hour)
	}

	// Get connection history
	entries, err := hd.manager.GetConnectionHistory(filter)
	if err != nil {
		return fmt.Errorf("failed to get connection history: %w", err)
	}
	hd.historyEntries = entries

	// Get recent activity stats
	activity, err := hd.manager.GetRecentActivity(24) // Last 24 hours
	if err != nil {
		return fmt.Errorf("failed to get recent activity: %w", err)
	}
	hd.recentActivity = activity

	// Update displays
	if hd.showingStats {
		hd.displayStatsInTable()
	} else {
		hd.displayHistoryInTable()
	}
	
	hd.updateStatsPanel()
	hd.updateFilterPanel()
	hd.updateStatusBar()

	return nil
}

// displayHistoryInTable displays connection history in the table
func (hd *HistoryDashboard) displayHistoryInTable() {
	// Clear existing data (except headers)
	for row := hd.historyTable.GetRowCount() - 1; row > 0; row-- {
		hd.historyTable.RemoveRow(row)
	}

	// Add history entries
	for i, entry := range hd.historyEntries {
		row := i + 1 // Skip header row

		// Format status with color
		var statusText string
		var statusColor tcell.Color
		switch entry.Status {
		case "success":
			statusText = "✓ SUCCESS"
			statusColor = tcell.ColorGreen
		case "failed":
			statusText = "✗ FAILED"
			statusColor = tcell.ColorRed
		case "timeout":
			statusText = "⏱ TIMEOUT"
			statusColor = tcell.ColorOrange
		case "cancelled":
			statusText = "⊘ CANCEL"
			statusColor = tcell.ColorYellow
		default:
			statusText = entry.Status
			statusColor = tcell.ColorGray
		}

		// Format time
		timeStr := entry.StartTime.Format("01-02 15:04")

		// Format duration
		durationStr := "N/A"
		if entry.DurationSeconds > 0 {
			duration := time.Duration(entry.DurationSeconds) * time.Second
			if duration < time.Minute {
				durationStr = fmt.Sprintf("%.1fs", duration.Seconds())
			} else {
				durationStr = fmt.Sprintf("%.1fm", duration.Minutes())
			}
		}

		// Format profile
		profileStr := entry.ProfileName
		if profileStr == "" {
			profileStr = "-"
		}

		hd.historyTable.SetCell(row, 0, tview.NewTableCell(entry.ServerName).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		hd.historyTable.SetCell(row, 1, tview.NewTableCell(statusText).SetTextColor(statusColor).SetAlign(tview.AlignCenter))
		hd.historyTable.SetCell(row, 2, tview.NewTableCell(entry.Host).SetTextColor(tcell.ColorLightBlue).SetAlign(tview.AlignLeft))
		hd.historyTable.SetCell(row, 3, tview.NewTableCell(timeStr).SetTextColor(tcell.ColorLightGray).SetAlign(tview.AlignLeft))
		hd.historyTable.SetCell(row, 4, tview.NewTableCell(durationStr).SetTextColor(tcell.ColorLightGreen).SetAlign(tview.AlignRight))
		hd.historyTable.SetCell(row, 5, tview.NewTableCell(profileStr).SetTextColor(tcell.ColorAqua).SetAlign(tview.AlignLeft))
	}

	// Update selection
	if len(hd.historyEntries) > 0 {
		if hd.selectedRow > len(hd.historyEntries) {
			hd.selectedRow = len(hd.historyEntries)
		}
		hd.historyTable.Select(hd.selectedRow, 0)
	}
}

// displayStatsInTable displays connection statistics in the table
func (hd *HistoryDashboard) displayStatsInTable() {
	// Clear existing data (except headers)
	for row := hd.historyTable.GetRowCount() - 1; row > 0; row-- {
		hd.historyTable.RemoveRow(row)
	}

	// Clear headers and set up stats headers
	for col := 0; col < 6; col++ {
		hd.historyTable.SetCell(0, col, tview.NewTableCell(""))
	}
	
	hd.historyTable.SetCell(0, 0, tview.NewTableCell("Metric").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))
	hd.historyTable.SetCell(0, 1, tview.NewTableCell("Value").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))
	hd.historyTable.SetCell(0, 2, tview.NewTableCell("").SetTextColor(tcell.ColorYellow).SetSelectable(false))
	hd.historyTable.SetCell(0, 3, tview.NewTableCell("").SetTextColor(tcell.ColorYellow).SetSelectable(false))
	hd.historyTable.SetCell(0, 4, tview.NewTableCell("").SetTextColor(tcell.ColorYellow).SetSelectable(false))
	hd.historyTable.SetCell(0, 5, tview.NewTableCell("").SetTextColor(tcell.ColorYellow).SetSelectable(false))

	row := 1

	// Display recent activity summary
	if len(hd.recentActivity) > 0 {
		hd.historyTable.SetCell(row, 0, tview.NewTableCell("[yellow::b]Recent Activity (24h)").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignLeft))
		hd.historyTable.SetCell(row, 1, tview.NewTableCell("").SetAlign(tview.AlignLeft))
		row++

		total := 0
		for _, count := range hd.recentActivity {
			total += count
		}

		hd.historyTable.SetCell(row, 0, tview.NewTableCell("Total Connections").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		hd.historyTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", total)).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		row++

		if success, exists := hd.recentActivity["success"]; exists && success > 0 {
			hd.historyTable.SetCell(row, 0, tview.NewTableCell("Successful").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
			hd.historyTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", success)).SetTextColor(tcell.ColorGreen).SetAlign(tview.AlignLeft))
			row++
		}

		if failed, exists := hd.recentActivity["failed"]; exists && failed > 0 {
			hd.historyTable.SetCell(row, 0, tview.NewTableCell("Failed").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
			hd.historyTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", failed)).SetTextColor(tcell.ColorRed).SetAlign(tview.AlignLeft))
			row++
		}

		if timeout, exists := hd.recentActivity["timeout"]; exists && timeout > 0 {
			hd.historyTable.SetCell(row, 0, tview.NewTableCell("Timeout").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
			hd.historyTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", timeout)).SetTextColor(tcell.ColorOrange).SetAlign(tview.AlignLeft))
			row++
		}
	} else {
		hd.historyTable.SetCell(row, 0, tview.NewTableCell("No recent activity").SetTextColor(tcell.ColorGray).SetAlign(tview.AlignLeft))
		hd.historyTable.SetCell(row, 1, tview.NewTableCell("").SetAlign(tview.AlignLeft))
		row++
	}

	// Add spacing
	hd.historyTable.SetCell(row, 0, tview.NewTableCell("").SetAlign(tview.AlignLeft))
	hd.historyTable.SetCell(row, 1, tview.NewTableCell("").SetAlign(tview.AlignLeft))
	row++

	// Display connection history summary
	hd.historyTable.SetCell(row, 0, tview.NewTableCell("[yellow::b]History Summary").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignLeft))
	hd.historyTable.SetCell(row, 1, tview.NewTableCell("").SetAlign(tview.AlignLeft))
	row++

	hd.historyTable.SetCell(row, 0, tview.NewTableCell("Total Entries").SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
	hd.historyTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", len(hd.historyEntries))).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
	row++

	// Count by status
	statusCounts := make(map[string]int)
	for _, entry := range hd.historyEntries {
		statusCounts[entry.Status]++
	}

	for status, count := range statusCounts {
		var color tcell.Color
		switch status {
		case "success":
			color = tcell.ColorGreen
		case "failed":
			color = tcell.ColorRed
		case "timeout":
			color = tcell.ColorOrange
		default:
			color = tcell.ColorYellow
		}

		hd.historyTable.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%s Connections", strings.Title(status))).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		hd.historyTable.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%d", count)).SetTextColor(color).SetAlign(tview.AlignLeft))
		row++
	}
}

// updateStatsPanel updates the statistics panel content
func (hd *HistoryDashboard) updateStatsPanel() {
	if len(hd.recentActivity) == 0 {
		hd.statsPanel.SetText("[yellow::b]Recent Activity[white::-]\n\nNo recent connection activity")
		return
	}

	total := 0
	for _, count := range hd.recentActivity {
		total += count
	}

	var statsText strings.Builder
	statsText.WriteString("[yellow::b]Recent Activity (24h)[white::-]\n\n")
	statsText.WriteString(fmt.Sprintf("Total: [white::b]%d[white::-] connections\n", total))

	if success, exists := hd.recentActivity["success"]; exists && success > 0 {
		percentage := float64(success) / float64(total) * 100
		statsText.WriteString(fmt.Sprintf("✓ Success: [green::b]%d[white::-] (%.1f%%)\n", success, percentage))
	}

	if failed, exists := hd.recentActivity["failed"]; exists && failed > 0 {
		percentage := float64(failed) / float64(total) * 100
		statsText.WriteString(fmt.Sprintf("✗ Failed: [red::b]%d[white::-] (%.1f%%)\n", failed, percentage))
	}

	if timeout, exists := hd.recentActivity["timeout"]; exists && timeout > 0 {
		percentage := float64(timeout) / float64(total) * 100
		statsText.WriteString(fmt.Sprintf("⏱ Timeout: [orange::b]%d[white::-] (%.1f%%)\n", timeout, percentage))
	}

	if cancelled, exists := hd.recentActivity["cancelled"]; exists && cancelled > 0 {
		percentage := float64(cancelled) / float64(total) * 100
		statsText.WriteString(fmt.Sprintf("⊘ Cancelled: [yellow::b]%d[white::-] (%.1f%%)\n", cancelled, percentage))
	}

	// Add history count info
	statsText.WriteString("\n[yellow::b]History[white::-]\n")
	statsText.WriteString(fmt.Sprintf("Entries shown: %d", len(hd.historyEntries)))

	hd.statsPanel.SetText(statsText.String())
}

// updateFilterPanel updates the filter panel content
func (hd *HistoryDashboard) updateFilterPanel() {
	var filterText strings.Builder
	filterText.WriteString("[yellow::b]Active Filters[white::-]\n\n")

	hasFilters := false

	if hd.serverFilter != "" {
		filterText.WriteString(fmt.Sprintf("Server: [aqua]%s[white]\n", hd.serverFilter))
		hasFilters = true
	}

	if hd.profileFilter != "" {
		filterText.WriteString(fmt.Sprintf("Profile: [aqua]%s[white]\n", hd.profileFilter))
		hasFilters = true
	}

	if hd.statusFilter != "" {
		filterText.WriteString(fmt.Sprintf("Status: [aqua]%s[white]\n", hd.statusFilter))
		hasFilters = true
	}

	if hd.daysFilter > 0 {
		filterText.WriteString(fmt.Sprintf("Days: [aqua]%d[white]\n", hd.daysFilter))
		hasFilters = true
	}

	if hd.limitFilter != 20 {
		filterText.WriteString(fmt.Sprintf("Limit: [aqua]%d[white]\n", hd.limitFilter))
		hasFilters = true
	}

	if !hasFilters {
		filterText.WriteString("[gray]No filters active[white]\n")
	}

	filterText.WriteString("\n[yellow::b]Quick Filters[white::-]\n")
	filterText.WriteString("1: Success only\n")
	filterText.WriteString("2: Failed only\n") 
	filterText.WriteString("3: Timeout only\n")
	filterText.WriteString("0: Clear status filter\n")

	hd.filterPanel.SetText(filterText.String())
}

// updateStatusBar updates the status bar content
func (hd *HistoryDashboard) updateStatusBar() {
	var mode string
	if hd.showingStats {
		mode = "Statistics"
	} else {
		mode = "History"
	}

	statusText := fmt.Sprintf("[white]History Dashboard - [yellow]%s[white] mode | [yellow]t[white]:toggle [yellow]f[white]:filter [yellow]c[white]:clear [yellow]r[white]:refresh [yellow]q[white]:quit [yellow]h[white]:help",
		mode)

	hd.statusBar.SetText(statusText)
}

// clearFilters clears all active filters
func (hd *HistoryDashboard) clearFilters() {
	hd.serverFilter = ""
	hd.profileFilter = ""
	hd.statusFilter = ""
	hd.daysFilter = 0
	hd.limitFilter = 20
	hd.refreshData()
}

// showFilterModal shows a modal for setting filters
func (hd *HistoryDashboard) showFilterModal() {
	// Create form for filter settings
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" Set Filters ").SetTitleAlign(tview.AlignCenter)

	// Add form fields with current values
	form.AddInputField("Server", hd.serverFilter, 20, nil, nil)
	form.AddInputField("Profile", hd.profileFilter, 20, nil, nil)
	form.AddDropDown("Status", []string{"", "success", "failed", "timeout", "cancelled"}, 0, nil)
	form.AddInputField("Days back", strconv.Itoa(hd.daysFilter), 10, nil, nil)
	form.AddInputField("Limit", strconv.Itoa(hd.limitFilter), 10, nil, nil)

	// Set current status filter selection
	statusOptions := []string{"", "success", "failed", "timeout", "cancelled"}
	for i, status := range statusOptions {
		if status == hd.statusFilter {
			form.GetFormItem(2).(*tview.DropDown).SetCurrentOption(i)
			break
		}
	}

	form.AddButton("Apply", func() {
		// Get form values
		hd.serverFilter = form.GetFormItem(0).(*tview.InputField).GetText()
		hd.profileFilter = form.GetFormItem(1).(*tview.InputField).GetText()
		_, hd.statusFilter = form.GetFormItem(2).(*tview.DropDown).GetCurrentOption()
		
		if days, err := strconv.Atoi(form.GetFormItem(3).(*tview.InputField).GetText()); err == nil {
			hd.daysFilter = days
		}
		
		if limit, err := strconv.Atoi(form.GetFormItem(4).(*tview.InputField).GetText()); err == nil && limit > 0 {
			hd.limitFilter = limit
		}

		// Refresh data and close modal
		hd.refreshData()
		hd.app.SetRoot(hd.layout, true)
	})

	form.AddButton("Cancel", func() {
		hd.app.SetRoot(hd.layout, true)
	})

	// Show the form
	hd.app.SetRoot(form, true)
}

// showHelp shows the help modal for the history dashboard
func (hd *HistoryDashboard) showHelp() {
	helpText := `[aqua::b]History Dashboard Help[::-]

[yellow::b]⚡ Navigation:[white::-]
  [lime]↑/↓[white]          Navigate history entries
  [lime]Enter[white]        (Reserved for future use)

[yellow::b]🔍 Filtering:[white::-]  
  [lime]f[white]            Set custom filters
  [lime]c[white]            Clear all filters
  [lime]1[white]            Show only successful connections
  [lime]2[white]            Show only failed connections
  [lime]3[white]            Show only timeout connections
  [lime]0[white]            Clear status filter

[yellow::b]📊 Views:[white::-]
  [lime]t[white]            Toggle between History and Statistics view
  [lime]r[white]            Refresh data

[yellow::b]🌐 General:[white::-]
  [lime]q/Esc[white]        Close dashboard
  [lime]h[white]            Show this help

[green::b]💡 Tips:[white::-]
• History shows recent connection attempts with status, timing, and details
• Statistics view shows aggregated data and success rates
• Use filters to focus on specific servers, profiles, or time periods
• Quick filters (1,2,3,0) provide rapid status-based filtering

[gray]Press [lime]Enter[white] or [lime]Escape[white] to close help[gray]`

	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			hd.app.SetRoot(hd.layout, true)
		}).
		SetBackgroundColor(tcell.ColorDarkBlue)

	// Add consistent key handling
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter, tcell.KeyEscape:
			hd.app.SetRoot(hd.layout, true)
			return nil
		}
		return event
	})

	hd.app.SetRoot(modal, true)
}

// GetLayout returns the dashboard layout for embedding
func (hd *HistoryDashboard) GetLayout() *tview.Flex {
	return hd.layout
}

// Close cleans up the dashboard resources
func (hd *HistoryDashboard) Close() {
	if hd.manager != nil {
		hd.manager.Close()
	}
}