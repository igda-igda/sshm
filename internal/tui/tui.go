package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sshm/internal/config"
)

// TUIApp represents the main TUI application
type TUIApp struct {
	app              *tview.Application
	layout           *tview.Flex
	serverList       *tview.Table
	profileNavigator *tview.TextView
	statusBar        *tview.TextView
	config           *config.Config
	
	// Application state
	running              bool
	mu                   sync.RWMutex
	stopChan             chan struct{}
	currentFilter        string   // Current profile filter, empty means all servers
	selectedRow          int      // Currently selected row (0 = header, 1+ = data rows)
	profileTabs          []string // List of profile tab names including "All"
	selectedProfileIndex int      // Currently selected profile tab index
}

// NewTUIApp creates a new TUI application instance
func NewTUIApp() (*TUIApp, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	tuiApp := &TUIApp{
		app:      tview.NewApplication(),
		config:   cfg,
		stopChan: make(chan struct{}),
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
	
	// Create right pane with profile navigator (placeholder for future session manager)
	rightPane := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.profileNavigator, 3, 0, false) // Fixed height for profile tabs
		// AddItem(sessionManager, 0, 1, false) // Future session manager component

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

	return nil
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
			t.navigateUp()
			return nil
		case tcell.KeyDown:
			t.navigateDown()
			return nil
		case tcell.KeyEnter:
			t.connectToSelectedServer()
			return nil
		case tcell.KeyTab:
			t.switchToNextProfile()
			return nil
		case tcell.KeyBacktab: // Shift+Tab
			t.switchToPreviousProfile()
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
			t.navigateDown()
			return nil
		case 'k', 'K':
			t.navigateUp()
			return nil
		case 'r', 'R':
			t.refreshData()
			return nil
		case 'p', 'P':
			t.switchToNextProfile()
			return nil
		}
		
		return event
	})
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
  ↑/↓, j/k    Navigate server list
  Enter       Connect to selected server
  
Actions:
  q           Quit application
  ?           Show this help
  r           Refresh data
  
Profile Navigation:
  Tab         Switch to next profile
  Shift+Tab   Switch to previous profile
  p           Switch to next profile
  
Mouse support: Click to select servers`

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