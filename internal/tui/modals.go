package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sshm/internal/config"
)

// CreateAddServerForm creates a form for adding new servers
func (t *TUIApp) CreateAddServerForm() *TUIForm {
	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().
				SetLabel("Server Name: ").
				SetFieldWidth(30),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "name", Message: "Server name is required"}
				}
				// Check if server already exists
				if _, err := t.config.GetServer(value); err == nil {
					return &ValidationError{Field: "name", Message: "Server name already exists"}
				}
				return nil
			},
			required: true,
		},
		"hostname": {
			inputField: tview.NewInputField().
				SetLabel("Hostname: ").
				SetFieldWidth(30),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "hostname", Message: "Hostname is required"}
				}
				return nil
			},
			required: true,
		},
		"port": {
			inputField: tview.NewInputField().
				SetLabel("Port: ").
				SetText("22").
				SetFieldWidth(10),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "port", Message: "Port is required"}
				}
				// Could add numeric validation here
				return nil
			},
			required: true,
		},
		"username": {
			inputField: tview.NewInputField().
				SetLabel("Username: ").
				SetFieldWidth(20),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "username", Message: "Username is required"}
				}
				return nil
			},
			required: true,
		},
		"auth_type": {
			inputField: tview.NewInputField().
				SetLabel("Auth Type (key/password): ").
				SetText("key").
				SetFieldWidth(15),
			validator: func(value string) error {
				if value != "key" && value != "password" {
					return &ValidationError{Field: "auth_type", Message: "Auth type must be 'key' or 'password'"}
				}
				return nil
			},
			required: true,
		},
		"key_path": {
			inputField: tview.NewInputField().
				SetLabel("Key Path (optional): ").
				SetFieldWidth(40),
			validator: func(value string) error {
				// Key path is optional but could validate file existence
				return nil
			},
			required: false,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		// Create new server configuration
		server := config.Server{
			Name:     data["name"].(string),
			Hostname: data["hostname"].(string),
			Port:     22, // Default port, could parse from data["port"]
			Username: data["username"].(string),
			AuthType: data["auth_type"].(string),
		}
		
		if keyPath, ok := data["key_path"].(string); ok && keyPath != "" {
			server.KeyPath = keyPath
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

	return NewTUIForm(fields, onSubmit, onCancel)
}

// CreateEditServerForm creates a form for editing existing servers
func (t *TUIApp) CreateEditServerForm(serverName string) *TUIForm {
	// Get existing server configuration
	server, err := t.config.GetServer(serverName)
	if err != nil {
		// Return empty form if server not found
		return t.CreateAddServerForm()
	}

	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().
				SetLabel("Server Name: ").
				SetText(server.Name).
				SetFieldWidth(30),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "name", Message: "Server name is required"}
				}
				// Allow same name (editing) but check for conflicts with other servers
				if value != serverName {
					if _, err := t.config.GetServer(value); err == nil {
						return &ValidationError{Field: "name", Message: "Server name already exists"}
					}
				}
				return nil
			},
			required: true,
		},
		"hostname": {
			inputField: tview.NewInputField().
				SetLabel("Hostname: ").
				SetText(server.Hostname).
				SetFieldWidth(30),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "hostname", Message: "Hostname is required"}
				}
				return nil
			},
			required: true,
		},
		"port": {
			inputField: tview.NewInputField().
				SetLabel("Port: ").
				SetText(fmt.Sprintf("%d", server.Port)).
				SetFieldWidth(10),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "port", Message: "Port is required"}
				}
				return nil
			},
			required: true,
		},
		"username": {
			inputField: tview.NewInputField().
				SetLabel("Username: ").
				SetText(server.Username).
				SetFieldWidth(20),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "username", Message: "Username is required"}
				}
				return nil
			},
			required: true,
		},
		"auth_type": {
			inputField: tview.NewInputField().
				SetLabel("Auth Type (key/password): ").
				SetText(server.AuthType).
				SetFieldWidth(15),
			validator: func(value string) error {
				if value != "key" && value != "password" {
					return &ValidationError{Field: "auth_type", Message: "Auth type must be 'key' or 'password'"}
				}
				return nil
			},
			required: true,
		},
		"key_path": {
			inputField: tview.NewInputField().
				SetLabel("Key Path (optional): ").
				SetText(server.KeyPath).
				SetFieldWidth(40),
			validator: func(value string) error {
				return nil
			},
			required: false,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		// Update server configuration
		updatedServer := config.Server{
			Name:     data["name"].(string),
			Hostname: data["hostname"].(string),
			Port:     22, // Could parse from data["port"]
			Username: data["username"].(string),
			AuthType: data["auth_type"].(string),
		}
		
		if keyPath, ok := data["key_path"].(string); ok && keyPath != "" {
			updatedServer.KeyPath = keyPath
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

	return NewTUIForm(fields, onSubmit, onCancel)
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