package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// HelpSystem manages the enhanced help system with context-sensitive content
type HelpSystem struct {
	app *TUIApp
}

// NewHelpSystem creates a new help system instance
func NewHelpSystem(app *TUIApp) *HelpSystem {
	return &HelpSystem{
		app: app,
	}
}

// ShowHelp displays context-sensitive help based on the current focused panel
func (h *HelpSystem) ShowHelp() {
	var helpContent string
	
	switch h.app.focusedPanel {
	case "servers":
		helpContent = h.getServersHelpContent()
	case "sessions":
		helpContent = h.getSessionsHelpContent()
	default:
		helpContent = h.getGeneralHelpContent()
	}
	
	h.displayHelpModal(helpContent)
}

// getServersHelpContent returns help content specific to the servers panel
func (h *HelpSystem) getServersHelpContent() string {
	return fmt.Sprintf(`[aqua::b]SSHM TUI Help - Servers Panel[::-]

[yellow::b]🖥️  Server Management:[white::-]
  [lime]a[white]           Add new server with form
  [lime]e[white]           Edit selected server configuration  
  [lime]d[white]           Delete selected server (with confirmation)
  [lime]Enter[white]       Connect to selected server via SSH/tmux

[yellow::b]📁 Profile Navigation:[white::-]
  [lime]Tab[white]         Switch to next profile tab
  [lime]Shift+Tab[white]   Switch to previous profile tab
  [lime]p[white]           Cycle through profile tabs
  [lime]b[white]           Batch connect to all servers in current profile

[yellow::b]📋 Profile Management:[white::-]
  [lime]c[white]           Create new profile
  [lime]o[white]           Edit current profile name/description
  [lime]x[white]           Delete current profile (with confirmation)
  [lime]i[white]           Assign selected server to current profile
  [lime]u[white]           Unassign selected server from current profile

[yellow::b]⚡ Navigation:[white::-]
  [lime]↑/↓, j/k[white]    Move selection up/down in server list
  [lime]s[white]           Switch focus to Sessions panel
  [lime]Tab[white]         Navigate between profile tabs

[yellow::b]💾 Configuration:[white::-]
  [lime]m[white]           Import configuration (YAML/JSON/SSH config)
  [lime]w[white]           Export configuration (YAML/JSON)
  [lime]r[white]           Refresh data from configuration files

[yellow::b]🎯 Current Context:[white::-]
  Active Panel:   [aqua]Servers[white]
  Profile Filter: [aqua]%s[white]
  Server Count:   [aqua]%d[white]

[green::b]💡 Tips:[white::-]
[green]•[white] [yellow]Yellow border[white] indicates the active panel
[green]•[white] Use [lime]Tab[white] to cycle through profiles when in server panel
[green]•[white] [lime]Enter[white] creates tmux session and stays in TUI - switch to Sessions to attach
[green]•[white] Profile filtering shows only servers assigned to the selected profile

[gray]Press [lime]?[white] [lime]Enter[white] [lime]Escape[white] to close this help[gray]`,
		h.getCurrentProfileName(),
		h.getVisibleServerCount())
}

// getSessionsHelpContent returns help content specific to the sessions panel
func (h *HelpSystem) getSessionsHelpContent() string {
	return fmt.Sprintf(`[aqua::b]SSHM TUI Help - Sessions Panel[::-]

[yellow::b]🔗 Session Management:[white::-]
  [lime]Enter[white]       Attach to selected tmux session
  [lime]y[white]           Kill selected session (with confirmation)
  [lime]z[white]           Cleanup orphaned/inactive sessions

[yellow::b]⚡ Navigation:[white::-]
  [lime]↑/↓, j/k[white]    Move selection up/down in session list
  [lime]s[white]           Switch focus back to Servers panel

[yellow::b]📊 Session Information:[white::-]
  [aqua]Session[white]     tmux session name
  [aqua]Status[white]      attached/detached/multi-attached
  [aqua]Windows[white]     number of tmux windows in session
  [aqua]Last Activity[white] when session was last used

[yellow::b]🎯 Current Context:[white::-]
  Active Panel:    [aqua]Sessions[white]
  Active Sessions: [aqua]%d[white]
  tmux Available:  [aqua]%s[white]

[yellow::b]🔄 Session States:[white::-]
  [green]detached[white]      Session running, ready to attach
  [yellow]attached[white]     Session has one client attached
  [orange]multi-attached[white] Session has multiple clients
  [red]inactive[white]        Session may have issues

[green::b]💡 Tips:[white::-]
[green]•[white] Sessions auto-refresh every 5 seconds
[green]•[white] [lime]Enter[white] on a session suspends TUI and attaches to tmux
[green]•[white] Detach from tmux ([lime]Ctrl+B, d[white]) returns to TUI automatically
[green]•[white] Use [lime]y[white] to kill stuck sessions, [lime]z[white] for bulk cleanup
[green]•[white] Group sessions (created with [lime]b[white]) have multiple windows

[gray]Press [lime]?[white] [lime]Enter[white] [lime]Escape[white] to close this help[gray]`,
		h.getActiveSessionCount(),
		h.getTmuxAvailabilityStatus())
}

// getGeneralHelpContent returns general help content
func (h *HelpSystem) getGeneralHelpContent() string {
	return `[aqua::b]SSHM TUI Help - General Commands[::-]

[yellow::b]🚀 Quick Start:[white::-]
  [lime]a[white]           Add your first server
  [lime]c[white]           Create a profile to organize servers  
  [lime]Enter[white]       Connect to server (creates tmux session)
  [lime]s[white]           Switch to sessions panel to attach

[yellow::b]⌨️  Global Shortcuts:[white::-]
  [lime]q[white]           Quit application safely
  [lime]?[white]           Show/hide this help (context-sensitive)
  [lime]r[white]           Refresh all data and connections
  [lime]s[white]           Switch focus between Servers and Sessions

[yellow::b]🖱️  Mouse Support:[white::-]
  [lime]Click[white]       Select servers, sessions, and buttons
  [lime]Scroll[white]      Navigate through long lists
  [lime]Right-click[white] Context menu (where applicable)

[yellow::b]📁 File Operations:[white::-]
  [lime]m[white]           Import servers from files:
                   • YAML/JSON configuration files
                   • SSH config files (~/.ssh/config format)
                   • Automatic format detection
  [lime]w[white]           Export current configuration:
                   • YAML format (default)
                   • JSON format option
                   • Profile-specific exports

[yellow::b]🔧 Configuration:[white::-]
  Config Location: [aqua]~/.sshm/config.yaml[white]
  Profile Storage: [aqua]~/.sshm/profiles/[white]
  Session Logs:    [aqua]~/.sshm/logs/[white]

[green::b]💡 Workflow Tips:[white::-]
[green]•[white] Create profiles for different environments (dev/staging/prod)
[green]•[white] Use [lime]b[white] to connect to entire profile as group session
[green]•[white] tmux sessions persist - you can detach/reattach anytime
[green]•[white] Sessions panel shows real-time status of all connections
[green]•[white] Import existing SSH configs to migrate from other tools

[green::b]🆘 Troubleshooting:[white::-]
[green]•[white] If tmux unavailable: install with [lime]brew install tmux[white] (macOS)
[green]•[white] Connection issues: check server details with [lime]e[white]
[green]•[white] Stuck sessions: use [lime]z[white] in Sessions panel for cleanup
[green]•[white] Config problems: delete [lime]~/.sshm/[white] to reset

[gray]Press [lime]?[white] [lime]Enter[white] [lime]Escape[white] to close this help[gray]`
}

// displayHelpModal creates and shows the help modal with enhanced styling
func (h *HelpSystem) displayHelpModal(content string) {
	modal := tview.NewModal().
		SetText(content).
		AddButtons([]string{"Close", "General Help", "Shortcuts Reference"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case "General Help":
				// Show general help regardless of context
				h.displayHelpModal(h.getGeneralHelpContent())
				return
			case "Shortcuts Reference":
				// Show keyboard shortcuts reference
				h.displayHelpModal(h.getShortcutsReference())
				return
			default:
				// Close help
				h.closeHelpModal()
			}
		}).
		SetBackgroundColor(tcell.ColorDarkBlue).
		SetButtonBackgroundColor(tcell.ColorDarkGreen).
		SetButtonTextColor(tcell.ColorWhite)

	// Enhanced input capture for better keyboard handling
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// Enter key closes help
			h.closeHelpModal()
			return nil
		case tcell.KeyEscape:
			// Escape also closes help
			h.closeHelpModal()
			return nil
		case tcell.KeyTab:
			// Tab switches between different help sections
			if h.app.focusedPanel == "servers" {
				h.displayHelpModal(h.getSessionsHelpContent())
			} else {
				h.displayHelpModal(h.getServersHelpContent())
			}
			return nil
		}
		
		// Handle character keys
		switch event.Rune() {
		case '?':
			// Toggle help (close current)
			h.closeHelpModal()
			return nil
		case 'g', 'G':
			// Show general help
			h.displayHelpModal(h.getGeneralHelpContent())
			return nil
		case 's', 'S':
			// Show shortcuts reference
			h.displayHelpModal(h.getShortcutsReference())
			return nil
		case 'q', 'Q':
			// Close help
			h.closeHelpModal()
			return nil
		}
		
		return event
	})

	// Set modal title based on context
	modal.SetTitle(fmt.Sprintf(" Help - %s ", h.getContextTitle()))
	
	// Show the modal
	if h.app.modalManager != nil {
		h.app.modalManager.ShowModal(modal)
	} else {
		h.app.app.SetRoot(modal, true)
		h.app.app.SetFocus(modal)
	}
}

// getShortcutsReference returns a quick reference of all keyboard shortcuts
func (h *HelpSystem) getShortcutsReference() string {
	return `[aqua::b]SSHM TUI - Keyboard Shortcuts Reference[::-]

[yellow::b]🌐 Global Shortcuts (work anywhere):[white::-]
  [lime]q / Ctrl+C[white]  Quit application
  [lime]?[white]           Show context help
  [lime]r[white]           Refresh data
  [lime]s[white]           Switch panel focus
  [lime]Escape[white]      Cancel/close modals

[yellow::b]🖥️  Servers Panel:[white::-]
  [lime]↑/↓ or j/k[white]  Navigate server list
  [lime]Enter[white]       Connect to server
  [lime]a[white]           Add server
  [lime]e[white]           Edit server
  [lime]d[white]           Delete server
  [lime]Tab/Shift+Tab[white] Switch profiles
  [lime]p[white]           Next profile
  [lime]b[white]           Batch connect profile
  [lime]c[white]           Create profile
  [lime]o[white]           Edit profile
  [lime]x[white]           Delete profile
  [lime]i[white]           Assign server to profile
  [lime]u[white]           Unassign server from profile

[yellow::b]🔗 Sessions Panel:[white::-]
  [lime]↑/↓ or j/k[white]  Navigate session list
  [lime]Enter[white]       Attach to session
  [lime]y[white]           Kill session
  [lime]z[white]           Cleanup orphaned sessions

[yellow::b]📁 Configuration:[white::-]
  [lime]m[white]           Import configuration
  [lime]w[white]           Export configuration

[yellow::b]📝 Forms & Modals:[white::-]
  [lime]Tab/Shift+Tab[white] Navigate form fields
  [lime]Enter[white]       Submit/confirm
  [lime]Escape[white]      Cancel/close
  [lime]Ctrl+A[white]      Select all text
  [lime]Ctrl+E[white]      Move to end of line

[green::b]💡 Pro Tips:[white::-]
[green]•[white] Hold [lime]Shift[white] with arrow keys for extended selection
[green]•[white] [lime]Tab[white] in help switches between panel contexts  
[green]•[white] Most operations have confirmation dialogs for safety
[green]•[white] Use [lime]?[white] in different panels for context-specific help

[gray]Press [lime]?[white] [lime]Enter[white] [lime]Escape[white] to close[gray]`
}

// Helper methods for dynamic content

// getCurrentProfileName returns the name of the currently selected profile
func (h *HelpSystem) getCurrentProfileName() string {
	if h.app.currentFilter == "" {
		return "All Servers"
	}
	return h.app.currentFilter
}

// getVisibleServerCount returns the number of servers visible in current filter
func (h *HelpSystem) getVisibleServerCount() int {
	if h.app.currentFilter == "" {
		return len(h.app.config.GetServers())
	}
	servers, err := h.app.config.GetServersByProfile(h.app.currentFilter)
	if err != nil {
		return 0
	}
	return len(servers)
}

// getActiveSessionCount returns the number of active tmux sessions
func (h *HelpSystem) getActiveSessionCount() int {
	return len(h.app.sessions)
}

// getTmuxAvailabilityStatus returns tmux availability status
func (h *HelpSystem) getTmuxAvailabilityStatus() string {
	if h.app.tmuxManager.IsAvailable() {
		return "✅ Available"
	}
	return "❌ Not Available"
}

// getContextTitle returns the appropriate title for the current context
func (h *HelpSystem) getContextTitle() string {
	switch h.app.focusedPanel {
	case "servers":
		return "Servers Panel"
	case "sessions":
		return "Sessions Panel"
	default:
		return "General"
	}
}

// closeHelpModal closes the help modal and returns to main interface
func (h *HelpSystem) closeHelpModal() {
	if h.app.modalManager != nil {
		h.app.modalManager.HideModal()
	} else {
		h.app.app.SetRoot(h.app.layout, true)
		h.app.app.SetFocus(h.app.layout)
	}
}

// ShowQuickHelp displays a compact help overlay for first-time users
func (h *HelpSystem) ShowQuickHelp() {
	quickHelp := `[yellow::b]🚀 Quick Start Guide[::-]

[lime]a[white] Add Server  [lime]c[white] Create Profile  [lime]Enter[white] Connect  [lime]s[white] Switch Panels

Press [lime]?[white] for detailed help`

	// Create a compact modal for quick help
	modal := tview.NewModal().
		SetText(quickHelp).
		AddButtons([]string{"Got it!", "Full Help"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Full Help" {
				h.ShowHelp()
			} else {
				h.closeHelpModal()
			}
		}).
		SetBackgroundColor(tcell.ColorDarkGreen)

	modal.SetTitle(" Quick Help ")
	
	if h.app.modalManager != nil {
		h.app.modalManager.ShowModal(modal)
	} else {
		h.app.app.SetRoot(modal, true)
		h.app.app.SetFocus(modal)
	}
}