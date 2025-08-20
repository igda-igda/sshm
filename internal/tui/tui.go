package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sshm/internal/config"
	"sshm/internal/tmux"
)

// SessionInfo represents tmux session information
type SessionInfo struct {
	Name         string
	Status       string
	Windows      int
	LastActivity string
}

// TUIApp represents the main TUI application
type TUIApp struct {
	app              *tview.Application
	layout           *tview.Flex
	serverList       *tview.Table
	profileNavigator *tview.TextView
	sessionPanel     *tview.Table
	statusBar        *tview.TextView
	config           *config.Config
	tmuxManager      *tmux.Manager
	
	// Application state
	running              bool
	mu                   sync.RWMutex
	stopChan             chan struct{}
	currentFilter        string   // Current profile filter, empty means all servers
	selectedRow          int      // Currently selected row (0 = header, 1+ = data rows)
	profileTabs          []string // List of profile tab names including "All"
	selectedProfileIndex int      // Currently selected profile tab index
	sessions             []SessionInfo // Current session list
	selectedSession      int      // Currently selected session (0 = header, 1+ = data rows)
	focusedPanel         string   // Currently focused panel: "servers" or "sessions"
}

// NewTUIApp creates a new TUI application instance
func NewTUIApp() (*TUIApp, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	tuiApp := &TUIApp{
		app:          tview.NewApplication(),
		config:       cfg,
		stopChan:     make(chan struct{}),
		tmuxManager:  tmux.NewManager(),
		focusedPanel: "servers", // Default focus on servers panel
	}

	// Setup the UI layout
	if err := tuiApp.setupLayout(); err != nil {
		return nil, fmt.Errorf("failed to setup layout: %w", err)
	}

	// Setup global key bindings
	tuiApp.setupKeyBindings()

	return tuiApp, nil
}

// setupLayout initializes the main UI layout
func (t *TUIApp) setupLayout() error {
	// Create status bar
	t.statusBar = tview.NewTextView().
		SetDynamicColors(true)

	// Create server list table
	t.serverList = tview.NewTable()
	t.serverList.SetBorder(true).SetTitle(" Servers ")
	t.serverList.SetBorders(false)
	t.serverList.SetSelectable(true, false)
	t.serverList.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite))

	// Setup server list headers
	t.serverList.SetCell(0, 0, tview.NewTableCell("Name").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))
	t.serverList.SetCell(0, 1, tview.NewTableCell("Host").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))
	t.serverList.SetCell(0, 2, tview.NewTableCell("Port").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))
	t.serverList.SetCell(0, 3, tview.NewTableCell("User").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))
	t.serverList.SetCell(0, 4, tview.NewTableCell("Auth").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))
	t.serverList.SetCell(0, 5, tview.NewTableCell("Status").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))
	t.serverList.SetCell(0, 6, tview.NewTableCell("Profile").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))

	// Create profile navigator
	t.profileNavigator = tview.NewTextView()
	t.profileNavigator.SetDynamicColors(true).SetBorder(true).SetTitle(" Profiles ")
	
	// Initialize profile tabs
	t.initializeProfileTabs()
	
	// Create session panel
	t.setupSessionPanel()
	
	// Create right pane with profile navigator and session manager
	rightPane := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.profileNavigator, 3, 0, false). // Fixed height for profile tabs
		AddItem(t.sessionPanel, 0, 1, false)     // Session manager takes remaining space

	// Create main horizontal layout: left pane (60%) server list, right pane (40%) profiles/sessions
	mainLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(t.serverList, 0, 6, true).  // 60% width, focusable
		AddItem(rightPane, 0, 4, false)    // 40% width, not focusable initially

	// Create overall layout with status bar at bottom
	t.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainLayout, 0, 1, true).
		AddItem(t.statusBar, 1, 0, false)

	// Set the main layout as root
	t.app.SetRoot(t.layout, true)

	// Load server data and update profile display
	t.refreshServerList()
	t.updateProfileDisplay()
	t.refreshSessions()
	t.updatePanelHighlight()

	return nil
}

// setupSessionPanel initializes the session manager panel
func (t *TUIApp) setupSessionPanel() {
	// Only create session panel if tmux is available
	if !t.tmuxManager.IsAvailable() {
		// Create a simple text view indicating tmux is not available
		t.sessionPanel = tview.NewTable()
		t.sessionPanel.SetBorder(true).SetTitle(" Sessions (tmux not available) ")
		return
	}

	// Create session table
	t.sessionPanel = tview.NewTable()
	t.sessionPanel.SetBorder(true).SetTitle(" Sessions ")
	t.sessionPanel.SetBorders(false)
	t.sessionPanel.SetSelectable(true, false)
	t.sessionPanel.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite))

	// Setup session table headers
	t.sessionPanel.SetCell(0, 0, tview.NewTableCell("Session").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))
	t.sessionPanel.SetCell(0, 1, tview.NewTableCell("Status").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))
	t.sessionPanel.SetCell(0, 2, tview.NewTableCell("Windows").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignCenter))
	t.sessionPanel.SetCell(0, 3, tview.NewTableCell("Last Activity").SetTextColor(tcell.ColorYellow).SetSelectable(false).SetAlign(tview.AlignLeft))

	// Set initial selection to first data row if it exists
	t.selectedSession = 1
	t.sessionPanel.Select(1, 0)
}

// initializeProfileTabs initializes the profile tabs list
func (t *TUIApp) initializeProfileTabs() {
	profiles := t.config.GetProfiles()
	
	// Always start with "All" tab
	t.profileTabs = []string{"All"}
	
	// Add profile names
	for _, profile := range profiles {
		t.profileTabs = append(t.profileTabs, profile.Name)
	}
	
	// Initialize selected index to 0 (All tab)
	t.selectedProfileIndex = 0
	t.currentFilter = "" // Empty filter means show all servers
}

// updateProfileDisplay updates the profile navigator display
func (t *TUIApp) updateProfileDisplay() {
	tabText := t.renderProfileTabs()
	t.profileNavigator.SetText(tabText)
}

// renderProfileTabs generates the tab display text with highlighting
func (t *TUIApp) renderProfileTabs() string {
	if len(t.profileTabs) == 0 {
		return "[white]No profiles configured"
	}
	
	var tabStrings []string
	for i, tab := range t.profileTabs {
		if i == t.selectedProfileIndex {
			// Highlight selected tab
			tabStrings = append(tabStrings, fmt.Sprintf("[aqua][%s][white]", tab))
		} else {
			// Normal tab
			tabStrings = append(tabStrings, tab)
		}
	}
	
	// Join tabs with separators
	return strings.Join(tabStrings, " | ")
}

// setupKeyBindings configures global key bindings
func (t *TUIApp) setupKeyBindings() {
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle special keys first
		switch event.Key() {
		case tcell.KeyCtrlC:
			t.Stop()
			return nil
		case tcell.KeyUp:
			t.handleNavigationUp()
			return nil
		case tcell.KeyDown:
			t.handleNavigationDown()
			return nil
		case tcell.KeyEnter:
			t.handleEnterKey()
			return nil
		case tcell.KeyTab:
			// Tab switches profiles when on servers panel, or switches focus
			if t.focusedPanel == "servers" {
				t.switchToNextProfile()
			} else {
				t.switchFocus()
			}
			return nil
		case tcell.KeyBacktab: // Shift+Tab
			if t.focusedPanel == "servers" {
				t.switchToPreviousProfile()
			} else {
				t.switchFocus()
			}
			return nil
		}
		
		// Handle character keys
		switch event.Rune() {
		case 'q', 'Q':
			t.Stop()
			return nil
		case '?':
			t.showHelp()
			return nil
		case 'j', 'J':
			t.handleNavigationDown()
			return nil
		case 'k', 'K':
			t.handleNavigationUp()
			return nil
		case 'r', 'R':
			t.refreshData()
			return nil
		case 'p', 'P':
			t.switchToNextProfile()
			return nil
		case 's', 'S':
			t.switchFocus()
			return nil
		case 'e', 'E':
			t.editSelectedServer()
			return nil
		case 'd', 'D':
			t.deleteSelectedServer()
			return nil
		}
		
		return event
	})
}

// handleNavigationUp handles up navigation based on focused panel
func (t *TUIApp) handleNavigationUp() {
	switch t.focusedPanel {
	case "servers":
		t.navigateUp()
	case "sessions":
		t.navigateSessionUp()
	}
}

// handleNavigationDown handles down navigation based on focused panel
func (t *TUIApp) handleNavigationDown() {
	switch t.focusedPanel {
	case "servers":
		t.navigateDown()
	case "sessions":
		t.navigateSessionDown()
	}
}

// handleEnterKey handles Enter key based on focused panel
func (t *TUIApp) handleEnterKey() {
	switch t.focusedPanel {
	case "servers":
		t.connectToSelectedServer()
	case "sessions":
		t.attachToSelectedSession()
	}
}

// switchFocus switches focus between server list and session panel
func (t *TUIApp) switchFocus() {
	if t.sessionPanel == nil {
		return // Can't switch to sessions if panel doesn't exist
	}
	
	if t.focusedPanel == "servers" {
		t.focusedPanel = "sessions"
		t.updatePanelHighlight()
	} else {
		t.focusedPanel = "servers" 
		t.updatePanelHighlight()
	}
}

// updatePanelHighlight updates the visual highlighting of focused panel
func (t *TUIApp) updatePanelHighlight() {
	if t.focusedPanel == "servers" {
		t.serverList.SetBorderColor(tcell.ColorYellow)
		if t.sessionPanel != nil {
			t.sessionPanel.SetBorderColor(tcell.ColorWhite)
		}
	} else {
		t.serverList.SetBorderColor(tcell.ColorWhite)
		if t.sessionPanel != nil {
			t.sessionPanel.SetBorderColor(tcell.ColorYellow)
		}
	}
}

// navigateUp moves selection up in the server list
func (t *TUIApp) navigateUp() {
	if t.serverList.GetRowCount() <= 1 {
		return // Only header row exists
	}
	
	currentRow, _ := t.serverList.GetSelection()
	if currentRow > 1 {
		newRow := currentRow - 1
		t.serverList.Select(newRow, 0)
		t.selectedRow = newRow
	}
}

// navigateDown moves selection down in the server list
func (t *TUIApp) navigateDown() {
	rowCount := t.serverList.GetRowCount()
	if rowCount <= 1 {
		return // Only header row exists
	}
	
	currentRow, _ := t.serverList.GetSelection()
	if currentRow < rowCount-1 {
		newRow := currentRow + 1
		t.serverList.Select(newRow, 0)
		t.selectedRow = newRow
	}
}

// connectToSelectedServer attempts to connect to the currently selected server
func (t *TUIApp) connectToSelectedServer() {
	currentRow, _ := t.serverList.GetSelection()
	if currentRow <= 0 {
		return // Header row selected or invalid selection
	}
	
	// Get server name from the selected row
	nameCell := t.serverList.GetCell(currentRow, 0)
	if nameCell == nil {
		return
	}
	
	serverName := nameCell.Text
	// TODO: Implement connection logic here
	// For now, show a placeholder modal
	t.showConnectionModal(serverName)
}

// showConnectionModal displays a modal indicating connection attempt
func (t *TUIApp) showConnectionModal(serverName string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Connecting to server: %s\n\n(Connection logic will be implemented in integration phase)", serverName)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.app.SetRoot(t.layout, true)
		})
	
	t.app.SetRoot(modal, true)
}

// refreshData reloads configuration and refreshes the display
func (t *TUIApp) refreshData() {
	if err := t.RefreshConfig(); err != nil {
		// Show error modal
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Error refreshing data: %s", err.Error())).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				t.app.SetRoot(t.layout, true)
			})
		t.app.SetRoot(modal, true)
		return
	}
	
	// Also refresh session data
	if err := t.refreshSessions(); err != nil {
		// Sessions refresh failed, but don't show modal - just log/ignore
		// since sessions may not always be available
	}
}

// switchToNextProfile switches to the next profile tab
func (t *TUIApp) switchToNextProfile() {
	if len(t.profileTabs) <= 1 {
		return // No switching needed with only one tab
	}
	
	t.selectedProfileIndex = (t.selectedProfileIndex + 1) % len(t.profileTabs)
	t.updateFilterFromProfile()
	t.updateProfileDisplay()
	t.refreshServerList()
}

// switchToPreviousProfile switches to the previous profile tab
func (t *TUIApp) switchToPreviousProfile() {
	if len(t.profileTabs) <= 1 {
		return // No switching needed with only one tab
	}
	
	t.selectedProfileIndex = (t.selectedProfileIndex - 1 + len(t.profileTabs)) % len(t.profileTabs)
	t.updateFilterFromProfile()
	t.updateProfileDisplay()
	t.refreshServerList()
}

// switchToProfile switches to a specific profile by index
func (t *TUIApp) switchToProfile(index int) {
	if index < 0 || index >= len(t.profileTabs) {
		return // Invalid index
	}
	
	t.selectedProfileIndex = index
	t.updateFilterFromProfile()
	t.updateProfileDisplay()
	t.refreshServerList()
}

// updateFilterFromProfile updates the currentFilter based on selected profile
func (t *TUIApp) updateFilterFromProfile() {
	if t.selectedProfileIndex >= len(t.profileTabs) {
		t.currentFilter = ""
		return
	}
	
	selectedTab := t.profileTabs[t.selectedProfileIndex]
	if selectedTab == "All" {
		t.currentFilter = ""
	} else {
		t.currentFilter = selectedTab
	}
}

// refreshServerList loads server data into the table with optional profile filtering
func (t *TUIApp) refreshServerList() {
	var servers []config.Server
	
	// Apply profile filter if set
	if t.currentFilter != "" && t.currentFilter != "all" {
		filteredServers, err := t.config.GetServersByProfile(t.currentFilter)
		if err != nil {
			// If profile doesn't exist, show all servers
			servers = t.config.GetServers()
		} else {
			servers = filteredServers
		}
	} else {
		servers = t.config.GetServers()
	}
	
	// Clear existing data (except headers)
	for row := t.serverList.GetRowCount() - 1; row > 0; row-- {
		t.serverList.RemoveRow(row)
	}

	// Add server data
	for i, server := range servers {
		row := i + 1 // Skip header row
		
		// Determine which profiles this server belongs to
		profileNames := t.getServerProfiles(server.Name)
		profileDisplay := "none"
		if len(profileNames) > 0 {
			profileDisplay = profileNames[0] // Show first profile for now
			if len(profileNames) > 1 {
				profileDisplay += "+" // Indicate multiple profiles
			}
		}
		
		// Determine status (placeholder for now, will be enhanced later)
		status := "unknown"
		statusColor := tcell.ColorGray
		
		t.serverList.SetCell(row, 0, tview.NewTableCell(server.Name).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		t.serverList.SetCell(row, 1, tview.NewTableCell(server.Hostname).SetTextColor(tcell.ColorLightBlue).SetAlign(tview.AlignLeft))
		t.serverList.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", server.Port)).SetTextColor(tcell.ColorLightGray).SetAlign(tview.AlignCenter))
		t.serverList.SetCell(row, 3, tview.NewTableCell(server.Username).SetTextColor(tcell.ColorLightGreen).SetAlign(tview.AlignLeft))
		t.serverList.SetCell(row, 4, tview.NewTableCell(server.AuthType).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))
		t.serverList.SetCell(row, 5, tview.NewTableCell(status).SetTextColor(statusColor).SetAlign(tview.AlignCenter))
		t.serverList.SetCell(row, 6, tview.NewTableCell(profileDisplay).SetTextColor(tcell.ColorAqua).SetAlign(tview.AlignLeft))
	}

	// Update selected row if needed
	if len(servers) > 0 {
		if t.selectedRow <= 0 || t.selectedRow > len(servers) {
			t.selectedRow = 1 // First data row
		}
		t.serverList.Select(t.selectedRow, 0)
	} else {
		t.selectedRow = 0
	}

	// Update status bar with server count and filter info
	t.updateStatusBar(len(servers))
}

// getServerProfiles returns the list of profile names that contain the given server
func (t *TUIApp) getServerProfiles(serverName string) []string {
	var profiles []string
	for _, profile := range t.config.GetProfiles() {
		for _, profileServer := range profile.Servers {
			if profileServer == serverName {
				profiles = append(profiles, profile.Name)
				break
			}
		}
	}
	return profiles
}

// updateStatusBar updates the status bar with current information
func (t *TUIApp) updateStatusBar(serverCount int) {
	filterText := ""
	if t.currentFilter != "" && t.currentFilter != "all" {
		filterText = fmt.Sprintf(" | Filter: [aqua]%s[white]", t.currentFilter)
	}
	
	statusText := fmt.Sprintf("[white]SSHM TUI - [yellow]%d[white] servers%s | Press [yellow]q[white] to quit, [yellow]?[white] for help", 
		serverCount, filterText)
	t.statusBar.SetText(statusText)
}

// showHelp displays a help modal
func (t *TUIApp) showHelp() {
	helpText := `SSHM TUI Help

Navigation:
  ↑/↓, j/k    Navigate lists
  Enter       Connect to server / Attach to session
  s           Switch focus between panels
  
Actions:
  q           Quit application
  ?           Show this help
  r           Refresh data
  e           Edit selected server
  d           Delete selected server
  
Profile Navigation (Server panel):
  Tab         Switch to next profile
  Shift+Tab   Switch to previous profile
  p           Switch to next profile

Session Management:
  s           Switch focus to sessions panel
  Enter       Attach to selected session (when in sessions)
  
Panel Focus:
  Yellow border indicates active panel
  
Mouse support: Click to select items

Server Actions (when server is selected):
  Enter       Connect to server
  e           Edit server configuration
  d           Delete server (with confirmation)`

	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.app.SetRoot(t.layout, true)
		})

	t.app.SetRoot(modal, true)
}

// Run starts the TUI application
func (t *TUIApp) Run(ctx context.Context) error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return fmt.Errorf("application is already running")
	}
	t.running = true
	t.mu.Unlock()

	// Handle context cancellation
	go func() {
		select {
		case <-ctx.Done():
			t.Stop()
		case <-t.stopChan:
			// Normal stop
		}
	}()

	// Run the application
	err := t.app.Run()
	
	t.mu.Lock()
	t.running = false
	t.mu.Unlock()

	if err != nil {
		return fmt.Errorf("TUI application error: %w", err)
	}

	return ctx.Err()
}

// Stop stops the TUI application gracefully
func (t *TUIApp) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return
	}

	// Stop the application
	if t.app != nil {
		t.app.Stop()
	}

	// Signal stop
	select {
	case t.stopChan <- struct{}{}:
	default:
		// Channel might be full or already closed
	}
}

// GetConfig returns the current configuration
func (t *TUIApp) GetConfig() *config.Config {
	return t.config
}

// RefreshConfig reloads the configuration and updates the UI
func (t *TUIApp) RefreshConfig() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}
	
	t.config = cfg
	t.initializeProfileTabs()
	t.updateProfileDisplay()
	t.refreshServerList()
	
	return nil
}

// refreshSessions refreshes the session display with current tmux sessions
func (t *TUIApp) refreshSessions() error {
	if !t.tmuxManager.IsAvailable() {
		return fmt.Errorf("tmux is not available")
	}

	// Get session list from tmux
	sessionNames, err := t.tmuxManager.ListSessions()
	if err != nil {
		// If no sessions exist, show empty list
		t.updateSessionDisplay([]SessionInfo{})
		return nil
	}

	// Parse detailed session information
	sessions, err := t.getSessionDetails(sessionNames)
	if err != nil {
		return err
	}

	// Update the display
	t.sessions = sessions
	t.updateSessionDisplay(sessions)
	return nil
}

// getSessionDetails retrieves detailed information for each session
func (t *TUIApp) getSessionDetails(sessionNames []string) ([]SessionInfo, error) {
	var sessions []SessionInfo

	for _, name := range sessionNames {
		sessionInfo, err := t.getDetailedSessionInfo(name)
		if err != nil {
			// Fall back to basic info if detailed query fails
			sessions = append(sessions, SessionInfo{
				Name:         name,
				Status:       "active",
				Windows:      1,
				LastActivity: "unknown",
			})
		} else {
			sessions = append(sessions, sessionInfo)
		}
	}

	return sessions, nil
}

// getDetailedSessionInfo gets detailed information for a specific session
func (t *TUIApp) getDetailedSessionInfo(sessionName string) (SessionInfo, error) {
	sessionInfo := SessionInfo{
		Name:         sessionName,
		Status:       "active", // Default
		Windows:      1,        // Default
		LastActivity: "unknown", // Default
	}

	// Try to get window count using tmux list-windows
	if windowCount, err := t.getSessionWindowCount(sessionName); err == nil {
		sessionInfo.Windows = windowCount
	}

	// Try to get session status (attached/detached)
	if status, err := t.getSessionStatus(sessionName); err == nil {
		sessionInfo.Status = status
	}

	// Try to get last activity time
	if activity, err := t.getSessionActivity(sessionName); err == nil {
		sessionInfo.LastActivity = activity
	}

	return sessionInfo, nil
}

// getSessionWindowCount returns the number of windows for a session
func (t *TUIApp) getSessionWindowCount(sessionName string) (int, error) {
	// Use tmux list-windows to count windows in the session
	cmd := fmt.Sprintf("tmux list-windows -t %s -F '#{window_index}' 2>/dev/null | wc -l", sessionName)
	output, err := t.executeCommand(cmd)
	if err != nil {
		return 1, err
	}
	
	count, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 1, err
	}
	
	return count, nil
}

// getSessionStatus returns whether the session is attached or not
func (t *TUIApp) getSessionStatus(sessionName string) (string, error) {
	// Use tmux list-sessions to check if session is attached
	cmd := fmt.Sprintf("tmux list-sessions -F '#{session_name} #{session_attached}' 2>/dev/null | grep '^%s '", sessionName)
	output, err := t.executeCommand(cmd)
	if err != nil {
		return "active", err
	}
	
	fields := strings.Fields(output)
	if len(fields) >= 2 {
		if fields[1] == "1" {
			return "attached", nil
		}
	}
	
	return "active", nil
}

// getSessionActivity returns the last activity time for a session
func (t *TUIApp) getSessionActivity(sessionName string) (string, error) {
	// Use tmux list-sessions to get activity time
	cmd := fmt.Sprintf("tmux list-sessions -F '#{session_name} #{session_activity}' 2>/dev/null | grep '^%s '", sessionName)
	output, err := t.executeCommand(cmd)
	if err != nil {
		return "unknown", err
	}
	
	fields := strings.Fields(output)
	if len(fields) >= 2 {
		// Convert unix timestamp to readable format
		if timestamp, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
			t := time.Unix(timestamp, 0)
			return t.Format("15:04"), nil
		}
	}
	
	return "unknown", nil
}

// executeCommand executes a shell command and returns output (helper for tmux queries)
func (t *TUIApp) executeCommand(cmd string) (string, error) {
	// This is a simplified implementation
	// In a production system, you might want to use proper command execution
	// For now, return empty to avoid shell injection risks in tests
	return "", fmt.Errorf("command execution not implemented in test mode")
}

// parseTmuxSessions parses tmux session output format
func (t *TUIApp) parseTmuxSessions(output string) []SessionInfo {
	var sessions []SessionInfo
	
	if strings.TrimSpace(output) == "" {
		return sessions
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		// Parse format: "session_name windows status last_activity"
		// This is a simplified parser - real implementation would use tmux format strings
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			windows, _ := strconv.Atoi(fields[1])
			sessions = append(sessions, SessionInfo{
				Name:         fields[0],
				Windows:      windows,
				Status:       fields[2],
				LastActivity: strings.Join(fields[3:], " "),
			})
		}
	}
	
	return sessions
}

// updateSessionDisplay updates the session panel with given sessions
func (t *TUIApp) updateSessionDisplay(sessions []SessionInfo) {
	if t.sessionPanel == nil {
		return
	}

	// Clear existing data (except headers)
	for row := t.sessionPanel.GetRowCount() - 1; row > 0; row-- {
		t.sessionPanel.RemoveRow(row)
	}

	// Add session data
	for i, session := range sessions {
		row := i + 1 // Skip header row
		
		// Determine status color
		statusColor := tcell.ColorGray
		switch session.Status {
		case "active":
			statusColor = tcell.ColorGreen
		case "attached":
			statusColor = tcell.ColorYellow  
		case "inactive":
			statusColor = tcell.ColorRed
		}

		t.sessionPanel.SetCell(row, 0, tview.NewTableCell(session.Name).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
		t.sessionPanel.SetCell(row, 1, tview.NewTableCell(session.Status).SetTextColor(statusColor).SetAlign(tview.AlignCenter))
		t.sessionPanel.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("%d", session.Windows)).SetTextColor(tcell.ColorLightBlue).SetAlign(tview.AlignCenter))
		t.sessionPanel.SetCell(row, 3, tview.NewTableCell(session.LastActivity).SetTextColor(tcell.ColorLightGray).SetAlign(tview.AlignLeft))
	}

	// Update selected session if needed
	if len(sessions) > 0 {
		if t.selectedSession <= 0 || t.selectedSession > len(sessions) {
			t.selectedSession = 1 // First data row
		}
		t.sessionPanel.Select(t.selectedSession, 0)
	} else {
		t.selectedSession = 0
	}
}

// navigateSessionUp moves selection up in the session list
func (t *TUIApp) navigateSessionUp() {
	if t.sessionPanel == nil || t.sessionPanel.GetRowCount() <= 1 {
		return // Only header row exists or panel doesn't exist
	}
	
	currentRow, _ := t.sessionPanel.GetSelection()
	if currentRow > 1 {
		newRow := currentRow - 1
		t.sessionPanel.Select(newRow, 0)
		t.selectedSession = newRow
	}
}

// navigateSessionDown moves selection down in the session list
func (t *TUIApp) navigateSessionDown() {
	if t.sessionPanel == nil {
		return
	}
	
	rowCount := t.sessionPanel.GetRowCount()
	if rowCount <= 1 {
		return // Only header row exists
	}
	
	currentRow, _ := t.sessionPanel.GetSelection()
	if currentRow < rowCount-1 {
		newRow := currentRow + 1
		t.sessionPanel.Select(newRow, 0)
		t.selectedSession = newRow
	}
}

// attachToSelectedSession attempts to attach to the currently selected session
func (t *TUIApp) attachToSelectedSession() {
	if t.sessionPanel == nil {
		return
	}
	
	currentRow, _ := t.sessionPanel.GetSelection()
	if currentRow <= 0 || currentRow > len(t.sessions) {
		return // Header row selected or invalid selection
	}
	
	// Get session name from the selected row
	sessionIndex := currentRow - 1 // Convert to zero-based index
	sessionName := t.sessions[sessionIndex].Name
	
	// Stop the TUI application before attaching
	t.Stop()
	
	// Attach to the session
	err := t.tmuxManager.AttachSession(sessionName)
	if err != nil {
		// If attachment fails, show error modal and restart TUI
		t.showSessionErrorModal(fmt.Sprintf("Failed to attach to session '%s': %s", sessionName, err.Error()))
	}
}

// showSessionErrorModal displays an error modal for session operations
func (t *TUIApp) showSessionErrorModal(message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.app.SetRoot(t.layout, true)
		})
	
	t.app.SetRoot(modal, true)
}

// editSelectedServer handles editing the currently selected server
func (t *TUIApp) editSelectedServer() {
	if t.focusedPanel != "servers" {
		return // Only allow editing when focused on servers panel
	}
	
	currentRow, _ := t.serverList.GetSelection()
	if currentRow <= 0 {
		return // Header row selected or invalid selection
	}
	
	// Get server name from the selected row
	nameCell := t.serverList.GetCell(currentRow, 0)
	if nameCell == nil {
		return
	}
	
	serverName := nameCell.Text
	
	// Show edit modal
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Edit server: %s\n\n(Server editing functionality will be implemented in CLI integration phase)\n\nThis will open the server configuration for editing.", serverName)).
		AddButtons([]string{"OK", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "OK" {
				// TODO: Implement actual editing logic in integration phase
				// This would typically:
				// 1. Launch external editor or show edit form
				// 2. Update configuration
				// 3. Refresh display
			}
			t.app.SetRoot(t.layout, true)
		})
	
	t.app.SetRoot(modal, true)
}

// deleteSelectedServer handles deleting the currently selected server
func (t *TUIApp) deleteSelectedServer() {
	if t.focusedPanel != "servers" {
		return // Only allow deleting when focused on servers panel
	}
	
	currentRow, _ := t.serverList.GetSelection()
	if currentRow <= 0 {
		return // Header row selected or invalid selection
	}
	
	// Get server name from the selected row
	nameCell := t.serverList.GetCell(currentRow, 0)
	if nameCell == nil {
		return
	}
	
	serverName := nameCell.Text
	
	// Show confirmation modal
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Delete server '%s'?\n\nThis action cannot be undone.", serverName)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Delete" {
				// Delete the server from configuration
				if err := t.deleteServerFromConfig(serverName); err != nil {
					// Show error modal
					errorModal := tview.NewModal().
						SetText(fmt.Sprintf("Error deleting server: %s", err.Error())).
						AddButtons([]string{"OK"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							t.app.SetRoot(t.layout, true)
						})
					t.app.SetRoot(errorModal, true)
					return
				}
				
				// Refresh the display after successful deletion
				t.refreshServerList()
			}
			t.app.SetRoot(t.layout, true)
		})
	
	t.app.SetRoot(modal, true)
}

// deleteServerFromConfig removes a server from the configuration
func (t *TUIApp) deleteServerFromConfig(serverName string) error {
	// Find and remove the server
	servers := t.config.GetServers()
	var updatedServers []config.Server
	
	serverFound := false
	for _, server := range servers {
		if server.Name != serverName {
			updatedServers = append(updatedServers, server)
		} else {
			serverFound = true
		}
	}
	
	if !serverFound {
		return fmt.Errorf("server '%s' not found", serverName)
	}
	
	// Update configuration with the filtered servers
	t.config.Servers = updatedServers
	
	// Also remove from any profiles that contain this server
	for i, profile := range t.config.Profiles {
		var updatedProfileServers []string
		for _, profileServer := range profile.Servers {
			if profileServer != serverName {
				updatedProfileServers = append(updatedProfileServers, profileServer)
			}
		}
		t.config.Profiles[i].Servers = updatedProfileServers
	}
	
	// Save the updated configuration
	if err := t.config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	return nil
}