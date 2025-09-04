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
	return fmt.Sprintf(`[yellow::b]🖥️  SSHM Help - Servers Panel  🖥️[::-]

[white::b]🚀 Server Management:[white::-]
[yellow]a[white]: Add new server with connection details
[yellow]e[white]: Edit selected server configuration
[yellow]d[white]: Delete selected server (with confirmation)
[yellow]Enter[white]: Connect to server via SSH/tmux

[white::b]📁 Profile Navigation:[white::-]
[yellow]Tab[white]: Switch to next profile tab
[yellow]Shift+Tab[white]: Switch to previous profile tab
[yellow]p[white]: Cycle through all profiles
[yellow]b[white]: Batch connect to entire profile

[white::b]⚙️  Profile Management:[white::-]
[yellow]c[white]: Create new profile
[yellow]o[white]: Edit current profile name/description
[yellow]x[white]: Delete current profile (with confirmation)
[yellow]i[white]: Assign server to current profile
[yellow]u[white]: Unassign server from current profile

[white::b]🧭 Navigation:[white::-]
[yellow]↑/↓, j/k[white]: Move selection up/down in server list
[yellow]s[white]: Switch focus to Sessions panel
[yellow]v[white]: View connection history dashboard
[yellow]Home/End[white]: Jump to first/last server

[white::b]💾 Configuration:[white::-]
[yellow]m[white]: Import config (YAML/JSON/SSH)
[yellow]w[white]: Export configuration to file
[yellow]r[white]: Refresh data from disk

[white::b]📊 Current Context:[white::-]
Profile: [aqua]%s[white] 📋
Server Count: [aqua]%d[white] 🖥️

[green::b]💡 Pro Tips:[white::-]
[green]•[white] [yellow]Yellow border[white] indicates the currently active panel
[green]•[white] [yellow]Enter[white] creates persistent tmux sessions that survive disconnects
[green]•[white] Profile filtering shows only matching servers
[green]•[white] [yellow]b[white] connects all servers in profile as group session

[lime]Press [white]?[lime] or [white]Enter[lime] or [white]Escape[white] to close • [lime]g[white] General • [lime]s[white] Shortcuts`,
		h.getCurrentProfileName(),
		h.getVisibleServerCount())
}

// getSessionsHelpContent returns help content specific to the sessions panel
func (h *HelpSystem) getSessionsHelpContent() string {
	return fmt.Sprintf(`[yellow::b]🔗 SSHM Help - Sessions Panel  🔗[::-]

[white::b]⚡ Session Management:[white::-]
[yellow]Enter[white]: Attach to session (suspend TUI)
[yellow]y[white]: Kill selected session
[yellow]z[white]: Cleanup orphaned sessions
[yellow]r[white]: Refresh session list manually

[white::b]🧭 Navigation:[white::-]
[yellow]↑/↓, j/k[white]: Move up/down in session list
[yellow]s[white]: Switch focus to Servers panel
[yellow]v[white]: View connection history dashboard
[yellow]Home/End[white]: Jump to first/last session

[white::b]🚦 Session Status Indicators:[white::-]
[green]🟢 detached[white]: Ready to attach
[yellow]🟡 attached[white]: One client connected
[orange]🟠 multi-attached[white]: Multiple clients
[red]🔴 inactive[white]: Connection issues

[white::b]📊 Current Context:[white::-]
Active Sessions: [aqua]%d[white] 🔗
tmux Available: [aqua]%s[white] ⚙️
Auto-refresh: [aqua]Every 5 seconds[white] 🔄

[green::b]💡 Pro Tips:[white::-]
[green]•[white] [yellow]Enter[white] suspends TUI and attaches to tmux session
[green]•[white] [yellow]Ctrl+B, d[white] in tmux returns to TUI automatically
[green]•[white] [yellow]y[white] kills stuck sessions, [yellow]z[white] for bulk cleanup
[green]•[white] Group sessions have multiple windows for easy management

[lime]Press [white]?[lime] or [white]Enter[lime] or [white]Escape[white] to close • [lime]g[white] General • [lime]s[white] Shortcuts`,
		h.getActiveSessionCount(),
		h.getTmuxAvailabilityStatus())
}

// getGeneralHelpContent returns general help content
func (h *HelpSystem) getGeneralHelpContent() string {
	return `[yellow::b]🌟 SSHM Help - General Guide  🌟[::-]

[white::b]🚀 Quick Start:[white::-]
[yellow]a[white]: Add your first server
[yellow]c[white]: Create a profile to organize servers
[yellow]Enter[white]: Connect to server (creates tmux session)
[yellow]s[white]: Switch to sessions panel
[yellow]?[white]: Show context-sensitive help

[white::b]⌨️  Global Shortcuts:[white::-]
[yellow]q / Ctrl+C[white]: Quit application safely
[yellow]?[white]: Show/hide help system
[yellow]r[white]: Refresh all data
[yellow]s[white]: Switch between panels
[yellow]v[white]: View connection history dashboard
[yellow]Escape[white]: Cancel/close modals

[white::b]💾 Configuration:[white::-]
[yellow]m[white]: Import servers (YAML/JSON/SSH config)
[yellow]w[white]: Export configuration to file

Config: [aqua]~/.sshm/config.yaml[white] 📄
Profiles: [aqua]~/.sshm/profiles/[white] 📁

[green::b]💡 Best Practices:[white::-]
[green]•[white] Create profiles for environments (dev/staging/prod)
[green]•[white] Use [yellow]b[white] to connect to entire profile as group
[green]•[white] tmux sessions persist - detach/reattach anytime
[green]•[white] Import existing SSH configs to migrate easily

[orange::b]🆘 Troubleshooting:[white::-]
[orange]•[white] No tmux: [yellow]brew install tmux[white] (macOS)
[orange]•[white] Connection issues: check with [yellow]e[white] (edit)
[orange]•[white] Stuck sessions: use [yellow]z[white] for cleanup
[orange]•[white] Reset config: delete [yellow]~/.sshm/[white] directory

[lime]Press [white]?[lime] or [white]Enter[lime] or [white]Escape[white] to close • [lime]s[white] for Shortcuts • Context help available!`
}

// displayHelpModal creates and shows the enhanced help modal with proper sizing and scrolling
func (h *HelpSystem) displayHelpModal(content string) {
	// Use tview.Modal with enhanced content and sizing
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

	// Set modal title based on context
	modal.SetTitle(fmt.Sprintf(" Help - %s ", h.getContextTitle()))

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
	return `[yellow::b]⌨️  SSHM TUI - Keyboard Shortcuts Reference  ⌨️[::-]

[white::b]🌐 Global Shortcuts (work anywhere):[white::-]
[yellow]q / Ctrl+C[white]: Quit application safely
[yellow]?[white]: Show context-sensitive help
[yellow]r[white]: Refresh all data from disk
[yellow]s[white]: Switch focus between panels
[yellow]v[white]: View connection history dashboard
[yellow]Escape[white]: Cancel/close modals and dialogs

[white::b]🖥️  Servers Panel Navigation:[white::-]
[yellow]↑/↓ or j/k[white]: Navigate up/down in server list
[yellow]Enter[white]: Connect to selected server
[yellow]Home/End[white]: Jump to first/last server
[yellow]Tab/Shift+Tab[white]: Switch between profile tabs
[yellow]p[white]: Cycle to next profile

[white::b]🔧 Server Management:[white::-]
[yellow]a[white]: Add new server configuration
[yellow]e[white]: Edit selected server details
[yellow]d[white]: Delete server (with confirmation)
[yellow]i[white]: Assign server to current profile
[yellow]u[white]: Unassign server from profile

[white::b]📋 Profile Operations:[white::-]
[yellow]c[white]: Create new profile
[yellow]o[white]: Edit current profile name/description
[yellow]x[white]: Delete current profile (with confirmation)
[yellow]b[white]: Batch connect to entire profile

[white::b]🔗 Sessions Panel:[white::-]
[yellow]↑/↓ or j/k[white]: Navigate session list
[yellow]Enter[white]: Attach to session (suspend TUI)
[yellow]y[white]: Kill selected session
[yellow]z[white]: Cleanup orphaned sessions
[yellow]Home/End[white]: Jump to first/last session

[white::b]📁 Configuration Management:[white::-]
[yellow]m[white]: Import config (YAML/JSON/SSH)
[yellow]w[white]: Export configuration to file

[white::b]📝 Forms & Modal Navigation:[white::-]
[yellow]Tab/Shift+Tab[white]: Navigate between form fields
[yellow]Enter[white]: Submit form/confirm action
[yellow]Escape[white]: Cancel form/close modal
[yellow]Ctrl+A[white]: Select all text in field
[yellow]Ctrl+E[white]: Move cursor to end of line

[green::b]💡 Pro Tips & Tricks:[white::-]
[green]•[white] Hold [yellow]Shift[white] with arrow keys for extended text selection
[green]•[white] Press [yellow]Tab[white] in help to switch between panel contexts
[green]•[white] Most destructive operations have confirmation dialogs
[green]•[white] Use [yellow]?[white] in different panels for context-specific help
[green]•[white] [yellow]Enter[white] in tmux creates persistent sessions that survive disconnects

[lime]Press [white]?[lime] or [white]Enter[lime] or [white]Escape[white] to close • [lime]g[white] General • [lime]Tab[white] Switch contexts`
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
