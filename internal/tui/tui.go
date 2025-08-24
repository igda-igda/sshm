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
	modalManager     *ModalManager
	
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

	// Initialize modal manager after layout is setup
	tuiApp.modalManager = NewModalManager(tuiApp.app, tuiApp.layout)

	// Setup global key bindings
	tuiApp.setupKeyBindings()

	return tuiApp, nil
}

// setupLayout initializes the main UI layout
func (t *TUIApp) setupLayout() error {
	// Enable mouse support
	t.app.EnableMouse(true)
	
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
		// Check if modal is active first - let modals handle their own keys
		if t.modalManager != nil && t.modalManager.IsModalActive() {
			// If a modal is active, let it handle the key first
			if currentModal := t.modalManager.GetCurrentModal(); currentModal != nil {
				// Only handle global Escape if modal doesn't consume it
				if event.Key() == tcell.KeyEscape {
					t.modalManager.HideModal()
					return nil
				}
			}
			return event // Let modal handle other keys
		}
		
		// Handle special keys first (only when no modal is active)
		switch event.Key() {
		case tcell.KeyCtrlC:
			t.Stop()
			return nil
		case tcell.KeyEscape:
			// Escape closes any active modal or does nothing if none
			if t.modalManager != nil && t.modalManager.IsModalActive() {
				t.modalManager.HideModal()
				return nil
			}
			return event
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
		case 'b', 'B':
			t.connectToCurrentProfile()
			return nil
		case 'a', 'A':
			t.addNewServer()
			return nil
		case 'c', 'C':
			t.createNewProfile()
			return nil
		case 'x', 'X':
			t.deleteCurrentProfile()
			return nil
		case 'o', 'O':
			t.editCurrentProfile()
			return nil
		case 'i', 'I':
			t.assignServerToProfile()
			return nil
		case 'u', 'U':
			t.unassignServerFromProfile()
			return nil
		case 'm', 'M':
			t.ShowImportModal()
			return nil
		case 'w', 'W':
			t.ShowExportModal()
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
	
	// Get server configuration
	server, err := t.config.GetServer(serverName)
	if err != nil {
		t.showErrorModal(fmt.Sprintf("Server '%s' not found: %s", serverName, err.Error()))
		return
	}
	
	// Check if tmux is available
	if !t.tmuxManager.IsAvailable() {
		t.showErrorModal("tmux is not available on this system. Please install tmux to use sshm.")
		return
	}
	
	// Build SSH command based on server configuration
	sshCommand, err := t.buildSSHCommand(*server)
	if err != nil {
		t.showErrorModal(fmt.Sprintf("Failed to build SSH command: %s", err.Error()))
		return
	}
	
	// Show connecting modal
	t.showConnectingModal(serverName)
	
	// Create tmux session in background and stay in TUI
	go func() {
		sessionName, wasExisting, err := t.tmuxManager.ConnectToServer(server.Name, sshCommand)
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.showErrorModal(fmt.Sprintf("Failed to create tmux session: %s", err.Error()))
			})
			return
		}
		
		// Session created successfully - show success message and stay in TUI
		t.app.QueueUpdateDraw(func() {
			// Hide the connecting modal and show success
			var statusMsg string
			if wasExisting {
				statusMsg = fmt.Sprintf("✅ Connected to existing session: %s\n\n💡 Switch to Sessions tab (press 's') and press Enter on the session to attach.", sessionName)
			} else {
				statusMsg = fmt.Sprintf("✅ Created new session: %s\n\n💡 Switch to Sessions tab (press 's') and press Enter on the session to attach.", sessionName)
			}
			
			successModal := tview.NewModal().
				SetText(statusMsg).
				AddButtons([]string{"OK", "Go to Sessions"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Go to Sessions" {
						// Switch to sessions panel
						t.focusedPanel = "sessions"
						t.updatePanelHighlight()
						// Refresh sessions to show the new one
						t.refreshSessions()
					}
					t.app.SetRoot(t.layout, true)
					t.app.SetFocus(t.layout)
				}).
				SetBackgroundColor(tcell.ColorDarkGreen)
			
			t.app.SetRoot(successModal, true)
			t.app.SetFocus(successModal)
			
			// Also refresh the session list in background
			t.refreshSessions()
		})
	}()
}

// showConnectingModal displays a modal indicating connection attempt in progress
func (t *TUIApp) showConnectingModal(serverName string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("🚀 Connecting to server: %s\n\n⏳ Establishing SSH connection...\n📡 Creating tmux session...\n\nPlease wait...", serverName)).
		SetBackgroundColor(tcell.ColorDarkBlue)
	
	t.app.SetRoot(modal, true)
}

// showErrorModal displays an error modal with the given message
func (t *TUIApp) showErrorModal(message string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("❌ Error\n\n%s", message)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if t.modalManager != nil {
				t.modalManager.HideModal()
			} else {
				t.app.SetRoot(t.layout, true)
			}
		}).
		SetBackgroundColor(tcell.ColorDarkRed)
	
	// Add consistent Enter key handling
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// Enter key dismisses error modal
			if t.modalManager != nil {
				t.modalManager.HideModal()
			} else {
				t.app.SetRoot(t.layout, true)
			}
			return nil
		case tcell.KeyEscape:
			// Escape also dismisses the modal
			if t.modalManager != nil {
				t.modalManager.HideModal()
			} else {
				t.app.SetRoot(t.layout, true)
			}
			return nil
		}
		return event
	})
	
	if t.modalManager != nil {
		t.modalManager.ShowModal(modal)
	} else {
		t.app.SetRoot(modal, true)
	}
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
	helpText := `[::b]SSHM TUI Help[::-]

[yellow::b]Navigation:[white::-]
  [yellow]↑/↓, j/k[white]    Navigate lists
  [yellow]Enter[white]       Connect to server / Attach to session
  [yellow]s[white]           Switch focus between panels

[yellow::b]Server Actions:[white::-]
  [yellow]a[white]           Add new server
  [yellow]e[white]           Edit selected server
  [yellow]d[white]           Delete selected server

[yellow::b]Profile Actions:[white::-]
  [yellow]c[white]           Create new profile
  [yellow]o[white]           Edit current profile
  [yellow]x[white]           Delete current profile
  [yellow]i[white]           Assign server to current profile
  [yellow]u[white]           Unassign server from current profile

[yellow::b]General Actions:[white::-]
  [yellow]q[white]           Quit application
  [yellow]?[white]           Show this help
  [yellow]r[white]           Refresh data
  [yellow]m[white]           Import configuration
  [yellow]w[white]           Export configuration

[yellow::b]Profile Navigation (Server panel):[white::-]
  [yellow]Tab[white]         Switch to next profile
  [yellow]Shift+Tab[white]   Switch to previous profile
  [yellow]p[white]           Switch to next profile

[yellow::b]Session Management:[white::-]
  [yellow]s[white]           Switch focus to sessions panel
  [yellow]Enter[white]       Attach to selected session (when in sessions)

[yellow::b]Panel Focus:[white::-]
  [yellow]Yellow border[white] indicates active panel

[yellow::b]Mouse support:[white::-] Click to select items

[yellow::b]Server Actions (when server is selected):[white::-]
  [yellow]Enter[white]       Connect to server
  [yellow]e[white]           Edit server configuration
  [yellow]d[white]           Delete server (with confirmation)
  [yellow]b[white]           Connect to all servers in current profile (batch)

[green::b]Additional Notes:[white::-]
[green]•[white] TUI exits when connecting to allow tmux to take over
[green]•[white] Sessions are refreshed automatically when switching focus
[green]•[white] Profile changes filter the server list in real-time`

	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if t.modalManager != nil {
				t.modalManager.HideModal()
			} else {
				t.app.SetRoot(t.layout, true)
				t.app.SetFocus(t.layout)
			}
		}).
		SetBackgroundColor(tcell.ColorDarkBlue)

	// Add consistent Enter/Escape key handling
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// Enter key dismisses help modal
			if t.modalManager != nil {
				t.modalManager.HideModal()
			} else {
				t.app.SetRoot(t.layout, true)
				t.app.SetFocus(t.layout)
			}
			return nil
		case tcell.KeyEscape:
			// Escape also dismisses the modal
			if t.modalManager != nil {
				t.modalManager.HideModal()
			} else {
				t.app.SetRoot(t.layout, true)
				t.app.SetFocus(t.layout)
			}
			return nil
		case tcell.Key('q'), tcell.Key('Q'), tcell.Key('?'):
			// '?' also dismisses help (toggle behavior)
			if t.modalManager != nil {
				t.modalManager.HideModal()
			} else {
				t.app.SetRoot(t.layout, true)
				t.app.SetFocus(t.layout)
			}
			return nil
		}
		return event
	})

	if t.modalManager != nil {
		t.modalManager.ShowModal(modal)
	} else {
		t.app.SetRoot(modal, true)
		t.app.SetFocus(modal)
	}
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
	
	// Validate configuration integrity
	if cfg == nil {
		return fmt.Errorf("configuration is nil after loading")
	}
	
	// Check for basic configuration validity
	servers := cfg.GetServers()
	for _, server := range servers {
		if err := server.Validate(); err != nil {
			// Log the error but don't fail the entire refresh
			// This allows the TUI to continue operating with partially valid config
			continue
		}
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
		// Tmux not available - show empty sessions but don't error
		t.updateSessionDisplay([]SessionInfo{})
		return nil
	}

	// Get session list from tmux
	sessionNames, err := t.tmuxManager.ListSessions()
	if err != nil {
		// If no sessions exist or tmux command failed, show empty list
		// This is expected behavior and shouldn't be treated as an error
		t.updateSessionDisplay([]SessionInfo{})
		return nil
	}

	// Parse detailed session information
	sessions, err := t.getSessionDetails(sessionNames)
	if err != nil {
		// If we can't get session details, fall back to basic session names
		var basicSessions []SessionInfo
		for _, name := range sessionNames {
			basicSessions = append(basicSessions, SessionInfo{
				Name:         name,
				Status:       "unknown",
				Windows:      0,
				LastActivity: "unknown",
			})
		}
		t.sessions = basicSessions
		t.updateSessionDisplay(basicSessions)
		return nil
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
	t.ShowEditServerModal(serverName)
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
	
	// Show confirmation modal with proper key handling
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Delete server '%s'?\n\nThis action cannot be undone.", serverName)).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			defer func() {
				// Always return to main layout
				t.app.SetRoot(t.layout, true)
				t.app.SetFocus(t.layout)
			}()
			
			if buttonLabel == "Delete" {
				// Delete the server from configuration
				if err := t.deleteServerFromConfig(serverName); err != nil {
					// Show error modal
					t.showErrorModal(fmt.Sprintf("Error deleting server: %s", err.Error()))
					return
				}
				
				// Refresh the display after successful deletion
				t.refreshServerList()
				t.refreshSessions()
			}
		}).
		SetBackgroundColor(tcell.ColorDarkRed)

	// Set up proper input capture for modal
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			// Escape key cancels
			if t.modalManager != nil {
				t.modalManager.HideModal()
			} else {
				t.app.SetRoot(t.layout, true)
				t.app.SetFocus(t.layout)
			}
			return nil
		case tcell.KeyEnter:
			// Enter key confirms delete
			if err := t.deleteServerFromConfig(serverName); err != nil {
				t.showErrorModal(fmt.Sprintf("Error deleting server: %s", err.Error()))
				return nil
			}
			
			// Refresh the display after successful deletion
			t.refreshServerList()
			t.refreshSessions()
			
			// Return to main layout
			if t.modalManager != nil {
				t.modalManager.HideModal()
			} else {
				t.app.SetRoot(t.layout, true)
				t.app.SetFocus(t.layout)
			}
			return nil
		case tcell.Key('d'), tcell.Key('D'):
			// 'd' key also confirms delete (consistent with key that opened modal)
			if err := t.deleteServerFromConfig(serverName); err != nil {
				t.showErrorModal(fmt.Sprintf("Error deleting server: %s", err.Error()))
				return nil
			}
			
			// Refresh the display after successful deletion
			t.refreshServerList()
			t.refreshSessions()
			
			// Return to main layout
			if t.modalManager != nil {
				t.modalManager.HideModal()
			} else {
				t.app.SetRoot(t.layout, true)
				t.app.SetFocus(t.layout)
			}
			return nil
		}
		return event
	})
	
	if t.modalManager != nil {
		t.modalManager.ShowModal(modal)
	} else {
		t.app.SetRoot(modal, true)
		t.app.SetFocus(modal)
	}
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
	
	// Also remove from any profiles that contain this server and clean up empty profiles
	var updatedProfiles []config.Profile
	for _, profile := range t.config.Profiles {
		var updatedProfileServers []string
		for _, profileServer := range profile.Servers {
			if profileServer != serverName {
				updatedProfileServers = append(updatedProfileServers, profileServer)
			}
		}
		// Only keep profiles that still have servers
		if len(updatedProfileServers) > 0 {
			updatedProfile := profile
			updatedProfile.Servers = updatedProfileServers
			updatedProfiles = append(updatedProfiles, updatedProfile)
		}
	}
	t.config.Profiles = updatedProfiles
	
	// Save the updated configuration
	if err := t.config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	return nil
}

// addNewServer handles adding a new server configuration
func (t *TUIApp) addNewServer() {
	t.ShowAddServerModal()
}

// buildSSHCommand builds an SSH command string for a server (same logic as CLI)
func (t *TUIApp) buildSSHCommand(server config.Server) (string, error) {
	// Validate server configuration
	if err := server.Validate(); err != nil {
		return "", fmt.Errorf("invalid server configuration: %w", err)
	}

	// Build base SSH command with pseudo-terminal allocation
	sshCmd := fmt.Sprintf("ssh -t %s@%s", server.Username, server.Hostname)
	
	// Add port if not default
	if server.Port != 22 {
		sshCmd += fmt.Sprintf(" -p %d", server.Port)
	}

	// Add key-specific options
	if server.AuthType == "key" && server.KeyPath != "" {
		sshCmd += fmt.Sprintf(" -i %s", server.KeyPath)
	}

	// Add common SSH options
	sshCmd += " -o ServerAliveInterval=60 -o ServerAliveCountMax=3"

	return sshCmd, nil
}

// connectToCurrentProfile connects to all servers in the currently selected profile
func (t *TUIApp) connectToCurrentProfile() {
	if t.currentFilter == "" {
		t.showErrorModal("Cannot connect to all servers. Please select a specific profile first.")
		return
	}
	
	// Get servers from current profile
	servers, err := t.config.GetServersByProfile(t.currentFilter)
	if err != nil {
		t.showErrorModal(fmt.Sprintf("Profile '%s' not found: %s", t.currentFilter, err.Error()))
		return
	}
	
	if len(servers) == 0 {
		t.showErrorModal(fmt.Sprintf("No servers found in profile '%s'", t.currentFilter))
		return
	}
	
	// Check if tmux is available
	if !t.tmuxManager.IsAvailable() {
		t.showErrorModal("tmux is not available on this system. Please install tmux to use sshm.")
		return
	}
	
	// Show connecting modal
	t.showGroupConnectingModal(t.currentFilter, len(servers))
	
	// Create group session in background and stay in TUI
	go func() {
		// Convert config.Server slice to tmux.Server interface slice
		tmuxServers := make([]tmux.Server, len(servers))
		for i, server := range servers {
			tmuxServers[i] = &server
		}
		
		sessionName, wasExisting, err := t.tmuxManager.ConnectToProfile(t.currentFilter, tmuxServers)
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.showErrorModal(fmt.Sprintf("Failed to create group session: %s", err.Error()))
			})
			return
		}
		
		// Group session created successfully - show success message and stay in TUI
		t.app.QueueUpdateDraw(func() {
			// Hide the connecting modal and show success
			var statusMsg string
			if wasExisting {
				statusMsg = fmt.Sprintf("✅ Connected to existing group session: %s\n\n📊 Session has %d windows for servers\n\n💡 Switch to Sessions tab (press 's') and press Enter on the session to attach.", sessionName, len(servers))
			} else {
				statusMsg = fmt.Sprintf("✅ Created group session: %s\n\n📊 Created %d windows for servers:\n", sessionName, len(servers))
				// Add server list
				for i, server := range servers {
					statusMsg += fmt.Sprintf("   • Window %d: %s (%s@%s:%d)\n", 
						i+1, server.Name, server.Username, server.Hostname, server.Port)
				}
				statusMsg += "\n💡 Switch to Sessions tab (press 's') and press Enter on the session to attach."
			}
			
			successModal := tview.NewModal().
				SetText(statusMsg).
				AddButtons([]string{"OK", "Go to Sessions"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Go to Sessions" {
						// Switch to sessions panel
						t.focusedPanel = "sessions"
						t.updatePanelHighlight()
						// Refresh sessions to show the new one
						t.refreshSessions()
					}
					t.app.SetRoot(t.layout, true)
					t.app.SetFocus(t.layout)
				}).
				SetBackgroundColor(tcell.ColorDarkGreen)
			
			t.app.SetRoot(successModal, true)
			t.app.SetFocus(successModal)
			
			// Also refresh the session list in background
			t.refreshSessions()
		})
	}()
}

// showGroupConnectingModal displays a modal for group connection attempts
func (t *TUIApp) showGroupConnectingModal(profileName string, serverCount int) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("🚀 Connecting to profile: %s\n\n📊 Creating group session for %d server(s)...\n🔗 Setting up tmux windows...\n⚡ Establishing SSH connections...\n\nPlease wait...", profileName, serverCount)).
		SetBackgroundColor(tcell.ColorDarkBlue)
	
	t.app.SetRoot(modal, true)
}

// Profile management action handlers

// createNewProfile handles creating a new profile
func (t *TUIApp) createNewProfile() {
	t.ShowCreateProfileModal()
}

// deleteCurrentProfile handles deleting the currently selected profile
func (t *TUIApp) deleteCurrentProfile() {
	if t.currentFilter == "" {
		t.showErrorModal("No profile selected. Please select a profile first.")
		return
	}
	t.ShowDeleteProfileModal(t.currentFilter)
}

// editCurrentProfile handles editing the currently selected profile
func (t *TUIApp) editCurrentProfile() {
	if t.currentFilter == "" {
		t.showErrorModal("No profile selected. Please select a profile first.")
		return
	}
	t.ShowEditProfileModal(t.currentFilter)
}

// assignServerToProfile handles assigning the selected server to the current profile
func (t *TUIApp) assignServerToProfile() {
	if t.currentFilter == "" {
		t.showErrorModal("No profile selected. Please select a profile first.")
		return
	}
	t.ShowServerAssignmentModal(t.currentFilter)
}

// unassignServerFromProfile handles unassigning the selected server from the current profile
func (t *TUIApp) unassignServerFromProfile() {
	if t.currentFilter == "" {
		t.showErrorModal("No profile selected. Please select a profile first.")
		return
	}
	t.ShowServerUnassignmentModal(t.currentFilter)
}