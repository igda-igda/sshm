package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sshm/internal/config"
)

// CreateAddServerForm creates a form for adding new servers with enhanced validation
func (t *TUIApp) CreateAddServerForm() *TUIForm {
	// Start with the standard server form fields
	fields := CreateServerFormFields()
	
	// Override name validator to check for existing servers
	fields["name"].validator = func(value string) error {
		// First run the standard validation
		if err := ValidateServerName(value); err != nil {
			return err
		}
		
		// Check if server already exists
		if _, err := t.config.GetServer(value); err == nil {
			return &ValidationError{Field: "name", Message: "Server name already exists"}
		}
		return nil
	}

	onSubmit := func(data map[string]interface{}) error {
		// Parse port as integer
		portStr := data["port"].(string)
		port := 22 // Default
		parsedPort := 0
		for _, r := range portStr {
			if r >= '0' && r <= '9' {
				parsedPort = parsedPort*10 + int(r-'0')
			}
		}
		if parsedPort > 0 {
			port = parsedPort
		}
		
		// Create new server configuration
		server := config.Server{
			Name:     data["name"].(string),
			Hostname: data["hostname"].(string),
			Port:     port,
			Username: data["username"].(string),
			AuthType: data["auth_type"].(string),
		}
		
		if keyPath, ok := data["key_path"].(string); ok && keyPath != "" {
			server.KeyPath = keyPath
		}
		
		// Handle passphrase protected flag
		if passphraseStr, ok := data["passphrase_protected"].(string); ok {
			server.PassphraseProtected = (passphraseStr == "true")
		}
		
		// Validate server configuration
		if err := server.Validate(); err != nil {
			return &ValidationError{Field: "general", Message: fmt.Sprintf("Server validation failed: %s", err.Error())}
		}
		
		// Add server to configuration
		t.config.Servers = append(t.config.Servers, server)
		
		// Save configuration
		if err := t.config.Save(); err != nil {
			return &ValidationError{Field: "general", Message: fmt.Sprintf("Failed to save configuration: %s", err.Error())}
		}
		
		// Refresh UI
		t.initializeProfileTabs()
		t.updateProfileDisplay()
		t.refreshServerList()
		
		// Hide modal and return to main interface
		if t.modalManager != nil {
			t.modalManager.HideModal()
		}
		
		return nil
	}
	
	onCancel := func() {
		// Hide modal and return to main interface
		if t.modalManager != nil {
			t.modalManager.HideModal()
		}
	}

	// Create form with real-time validation enabled
	return NewTUIFormWithOptions(fields, onSubmit, onCancel, true)
}

// CreateEditServerForm creates a form for editing existing servers with enhanced validation
func (t *TUIApp) CreateEditServerForm(serverName string) *TUIForm {
	// Get existing server configuration
	server, err := t.config.GetServer(serverName)
	if err != nil {
		// Return empty form if server not found
		return t.CreateAddServerForm()
	}

	// Start with the standard server form fields
	fields := CreateServerFormFields()
	
	// Pre-populate fields with existing server data
	fields["name"].SetText(server.Name)
	fields["hostname"].SetText(server.Hostname)
	fields["port"].SetText(fmt.Sprintf("%d", server.Port))
	fields["username"].SetText(server.Username)
	fields["auth_type"].SetText(server.AuthType)
	fields["key_path"].SetText(server.KeyPath)
	if server.PassphraseProtected {
		fields["passphrase_protected"].SetText("true")
	} else {
		fields["passphrase_protected"].SetText("false")
	}
	
	// Override name validator to allow same name but check for conflicts with other servers
	fields["name"].validator = func(value string) error {
		// First run the standard validation
		if err := ValidateServerName(value); err != nil {
			return err
		}
		
		// Allow same name (editing) but check for conflicts with other servers
		if value != serverName {
			if _, err := t.config.GetServer(value); err == nil {
				return &ValidationError{Field: "name", Message: "Server name already exists"}
			}
		}
		return nil
	}

	onSubmit := func(data map[string]interface{}) error {
		// Parse port as integer
		portStr := data["port"].(string)
		port := 22 // Default
		parsedPort := 0
		for _, r := range portStr {
			if r >= '0' && r <= '9' {
				parsedPort = parsedPort*10 + int(r-'0')
			}
		}
		if parsedPort > 0 {
			port = parsedPort
		}
		
		// Update server configuration
		updatedServer := config.Server{
			Name:     data["name"].(string),
			Hostname: data["hostname"].(string),
			Port:     port,
			Username: data["username"].(string),
			AuthType: data["auth_type"].(string),
		}
		
		if keyPath, ok := data["key_path"].(string); ok && keyPath != "" {
			updatedServer.KeyPath = keyPath
		}
		
		// Handle passphrase protected flag
		if passphraseStr, ok := data["passphrase_protected"].(string); ok {
			updatedServer.PassphraseProtected = (passphraseStr == "true")
		}
		
		// Validate server configuration
		if err := updatedServer.Validate(); err != nil {
			return &ValidationError{Field: "general", Message: fmt.Sprintf("Server validation failed: %s", err.Error())}
		}
		
		// Find and replace the server in configuration
		for i, s := range t.config.Servers {
			if s.Name == serverName {
				t.config.Servers[i] = updatedServer
				break
			}
		}
		
		// Save configuration
		if err := t.config.Save(); err != nil {
			return &ValidationError{Field: "general", Message: fmt.Sprintf("Failed to save configuration: %s", err.Error())}
		}
		
		// Refresh UI
		t.initializeProfileTabs()
		t.updateProfileDisplay()
		t.refreshServerList()
		
		// Hide modal and return to main interface
		if t.modalManager != nil {
			t.modalManager.HideModal()
		}
		
		return nil
	}
	
	onCancel := func() {
		// Hide modal and return to main interface
		if t.modalManager != nil {
			t.modalManager.HideModal()
		}
	}

	// Create form with real-time validation enabled
	return NewTUIFormWithOptions(fields, onSubmit, onCancel, true)
}

// ShowAddServerModal displays the add server modal
func (t *TUIApp) ShowAddServerModal() {
	form := t.CreateAddServerForm()
	
	// Create flex container for the form with title and border
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText(" Add New Server ").SetTextAlign(tview.AlignCenter), 1, 0, false).
		AddItem(form.GetForm(), 0, 1, true)
	
	container.SetBorder(true).SetTitle(" Add Server ").SetBorderColor(tcell.ColorYellow)
	
	// Create modal wrapper with the form
	modal := tview.NewModal().
		SetBackgroundColor(tcell.ColorDarkBlue).
		SetText("").
		AddButtons([]string{}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			// This won't be called since we have no buttons
		})
	
	// Replace modal content with our form
	modal.SetText("")
	
	// Setup enhanced keyboard navigation for the form directly
	form.GetForm().SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
			return nil
		case tcell.KeyTab:
			// Let tview handle Tab navigation between form fields
			return event
		case tcell.KeyBacktab:
			// Let tview handle Shift+Tab navigation
			return event
		case tcell.KeyEnter:
			// Let form handle Enter for submission
			return event
		}
		return event
	})
	
	// Show the form directly as modal
	if t.modalManager != nil {
		t.modalManager.ShowModal(form.GetForm())
	}
}

// ShowEditServerModal displays the edit server modal
func (t *TUIApp) ShowEditServerModal(serverName string) {
	form := t.CreateEditServerForm(serverName)
	
	// Setup enhanced keyboard navigation for the form directly
	form.GetForm().SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
			return nil
		case tcell.KeyTab:
			// Let tview handle Tab navigation between form fields
			return event
		case tcell.KeyBacktab:
			// Let tview handle Shift+Tab navigation
			return event
		case tcell.KeyEnter:
			// Let form handle Enter for submission
			return event
		}
		return event
	})
	
	// Set title and border for the form
	form.GetForm().SetBorder(true).SetTitle(fmt.Sprintf(" Edit Server: %s ", serverName)).SetBorderColor(tcell.ColorYellow)
	
	// Show the form directly as modal
	if t.modalManager != nil {
		t.modalManager.ShowModal(form.GetForm())
	}
}