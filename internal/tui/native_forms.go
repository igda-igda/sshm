package tui

import (
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sshm/internal/auth"
	"sshm/internal/config"
)

// CreateNativeAddServerForm creates a form using tview's native form with proper password masking
func (t *TUIApp) CreateNativeAddServerForm() *tview.Form {
	form := tview.NewForm().
		AddInputField("Server Name", "", 30, nil, nil).
		AddInputField("Hostname", "", 40, nil, nil).
		AddInputField("Port", "22", 10, nil, nil).
		AddInputField("Username", "", 25, nil, nil).
		AddDropDown("Auth Type", []string{"key", "password"}, 0, nil).
		AddPasswordField("Password", "", 30, '*', nil).
		AddInputField("Key Path (optional)", "", 50, nil, nil).
		AddCheckbox("Passphrase Protected", false, nil).
		AddButton("Submit", nil).
		AddButton("Cancel", nil)

	form.SetBorder(true).
		SetTitle(" Add New Server ").
		SetTitleAlign(tview.AlignCenter)

	// Get form items for easy access
	nameField := form.GetFormItem(0).(*tview.InputField)
	hostnameField := form.GetFormItem(1).(*tview.InputField)
	portField := form.GetFormItem(2).(*tview.InputField)
	usernameField := form.GetFormItem(3).(*tview.InputField)
	authDropdown := form.GetFormItem(4).(*tview.DropDown)
	passwordField := form.GetFormItem(5).(*tview.InputField) // This is the masked password field
	keyPathField := form.GetFormItem(6).(*tview.InputField)
	passphraseCheckbox := form.GetFormItem(7).(*tview.Checkbox)

	// Track current auth type
	currentAuthType := "key"

	// Handle auth type changes to update form validation
	authDropdown.SetSelectedFunc(func(text string, index int) {
		currentAuthType = text
	})

	// Set up form submission
	form.GetButton(0).SetSelectedFunc(func() {
		// Validate required fields
		name := nameField.GetText()
		hostname := hostnameField.GetText()
		portStr := portField.GetText()
		username := usernameField.GetText()

		if name == "" || hostname == "" || username == "" {
			t.showErrorModal("All required fields must be filled")
			return
		}

		// Check if server already exists
		if _, err := t.config.GetServer(name); err == nil {
			t.showErrorModal("Server name already exists")
			return
		}

		// Parse port
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			t.showErrorModal("Port must be a valid number between 1 and 65535")
			return
		}

		// Get auth type
		authType := currentAuthType

		// Create server
		server := config.Server{
			Name:     name,
			Hostname: hostname,
			Port:     port,
			Username: username,
			AuthType: authType,
		}

		// Handle key path
		keyPath := keyPathField.GetText()
		if keyPath != "" {
			server.KeyPath = keyPath
		}

		// Handle passphrase protected
		server.PassphraseProtected = passphraseCheckbox.IsChecked()

		// Handle password authentication with keyring storage
		if authType == "password" {
			password := passwordField.GetText()
			if password == "" {
				t.showErrorModal("Password is required for password authentication")
				return
			}

			// Store password securely in keyring
			passwordManager, err := auth.NewPasswordManager("auto")
			if err != nil {
				t.showErrorModal(fmt.Sprintf("Failed to initialize password manager: %s", err.Error()))
				return
			}

			// Store password in keyring and configure server to use it
			if err := passwordManager.StoreServerPassword(&server, password); err != nil {
				t.showErrorModal(fmt.Sprintf("Failed to store password: %s", err.Error()))
				return
			}
		}

		// Validate server configuration
		if err := server.Validate(); err != nil {
			t.showErrorModal(fmt.Sprintf("Server validation failed: %s", err.Error()))
			return
		}

		// Add server to configuration
		t.config.Servers = append(t.config.Servers, server)

		// Save configuration
		if err := t.config.Save(); err != nil {
			t.showErrorModal(fmt.Sprintf("Failed to save configuration: %s", err.Error()))
			return
		}

		// Refresh UI
		t.initializeProfileTabs()
		t.updateProfileDisplay()
		t.refreshServerList()

		// Hide modal and return to main interface
		if t.modalManager != nil {
			t.modalManager.HideModal()
		}
	})

	// Set up cancel button
	form.GetButton(1).SetSelectedFunc(func() {
		if t.modalManager != nil {
			t.modalManager.HideModal()
		}
	})

	// Set up keyboard navigation
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
			return nil
		}
		return event
	})

	return form
}

// CreateNativeEditServerForm creates an edit form using tview's native form with proper password masking
func (t *TUIApp) CreateNativeEditServerForm(serverName string) *tview.Form {
	// Get existing server configuration
	server, err := t.config.GetServer(serverName)
	if err != nil {
		// Return empty form if server not found
		return t.CreateNativeAddServerForm()
	}

	form := tview.NewForm().
		AddInputField("Server Name", server.Name, 30, nil, nil).
		AddInputField("Hostname", server.Hostname, 40, nil, nil).
		AddInputField("Port", fmt.Sprintf("%d", server.Port), 10, nil, nil).
		AddInputField("Username", server.Username, 25, nil, nil).
		AddDropDown("Auth Type", []string{"key", "password"}, 0, nil).
		AddPasswordField("Password", "", 30, '*', nil). // Always empty for security
		AddInputField("Key Path (optional)", server.KeyPath, 50, nil, nil).
		AddCheckbox("Passphrase Protected", server.PassphraseProtected, nil).
		AddButton("Update", nil).
		AddButton("Cancel", nil)

	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" Edit Server: %s ", serverName)).
		SetTitleAlign(tview.AlignCenter)

	// Get form items for easy access
	nameField := form.GetFormItem(0).(*tview.InputField)
	hostnameField := form.GetFormItem(1).(*tview.InputField)
	portField := form.GetFormItem(2).(*tview.InputField)
	usernameField := form.GetFormItem(3).(*tview.InputField)
	authDropdown := form.GetFormItem(4).(*tview.DropDown)
	passwordField := form.GetFormItem(5).(*tview.InputField) // This is the masked password field
	keyPathField := form.GetFormItem(6).(*tview.InputField)
	passphraseCheckbox := form.GetFormItem(7).(*tview.Checkbox)

	// Set current auth type in dropdown
	if server.AuthType == "password" {
		authDropdown.SetCurrentOption(1)
	} else {
		authDropdown.SetCurrentOption(0)
	}

	// Track current auth type
	currentAuthType := server.AuthType

	// Handle auth type changes to update form validation
	authDropdown.SetSelectedFunc(func(text string, index int) {
		currentAuthType = text
	})

	// Set up form submission
	form.GetButton(0).SetSelectedFunc(func() {
		// Validate required fields
		name := nameField.GetText()
		hostname := hostnameField.GetText()
		portStr := portField.GetText()
		username := usernameField.GetText()

		if name == "" || hostname == "" || username == "" {
			t.showErrorModal("All required fields must be filled")
			return
		}

		// Check if server name conflicts with other servers (but allow same name)
		if name != serverName {
			if _, err := t.config.GetServer(name); err == nil {
				t.showErrorModal("Server name already exists")
				return
			}
		}

		// Parse port
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			t.showErrorModal("Port must be a valid number between 1 and 65535")
			return
		}

		// Get auth type
		authType := currentAuthType

		// Update server
		updatedServer := config.Server{
			Name:     name,
			Hostname: hostname,
			Port:     port,
			Username: username,
			AuthType: authType,
		}

		// Handle key path
		keyPath := keyPathField.GetText()
		if keyPath != "" {
			updatedServer.KeyPath = keyPath
		}

		// Handle passphrase protected
		updatedServer.PassphraseProtected = passphraseCheckbox.IsChecked()

		// Handle password authentication with keyring storage
		if authType == "password" {
			password := passwordField.GetText()
			if password != "" {
				// New password provided - store it in keyring
				passwordManager, err := auth.NewPasswordManager("auto")
				if err != nil {
					t.showErrorModal(fmt.Sprintf("Failed to initialize password manager: %s", err.Error()))
					return
				}

				// Store password in keyring and configure server to use it
				if err := passwordManager.StoreServerPassword(&updatedServer, password); err != nil {
					t.showErrorModal(fmt.Sprintf("Failed to store password: %s", err.Error()))
					return
				}
			} else {
				// No new password provided - preserve existing keyring settings
				if server.UseKeyring && server.KeyringID != "" {
					updatedServer.UseKeyring = server.UseKeyring
					updatedServer.KeyringID = server.KeyringID
				}
			}
		}

		// Validate server configuration
		if err := updatedServer.Validate(); err != nil {
			t.showErrorModal(fmt.Sprintf("Server validation failed: %s", err.Error()))
			return
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
			t.showErrorModal(fmt.Sprintf("Failed to save configuration: %s", err.Error()))
			return
		}

		// Refresh UI
		t.initializeProfileTabs()
		t.updateProfileDisplay()
		t.refreshServerList()

		// Hide modal and return to main interface
		if t.modalManager != nil {
			t.modalManager.HideModal()
		}
	})

	// Set up cancel button
	form.GetButton(1).SetSelectedFunc(func() {
		if t.modalManager != nil {
			t.modalManager.HideModal()
		}
	})

	// Set up keyboard navigation
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
			return nil
		}
		return event
	})

	return form
}