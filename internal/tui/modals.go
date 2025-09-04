package tui

import (
	"fmt"

	"sshm/internal/auth"
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
		
		// Handle password authentication with keyring storage
		if password, ok := data["password"].(string); ok && password != "" && server.AuthType == "password" {
			// Store password securely in keyring
			passwordManager, err := auth.NewPasswordManager("auto")
			if err != nil {
				return &ValidationError{Field: "password", Message: fmt.Sprintf("Failed to initialize password manager: %s", err.Error())}
			}
			
			// Store password in keyring and configure server to use it
			if err := passwordManager.StoreServerPassword(&server, password); err != nil {
				return &ValidationError{Field: "password", Message: fmt.Sprintf("Failed to store password: %s", err.Error())}
			}
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
	// Handle password field for password authentication
	// For security, we don't pre-populate the password field when editing
	// Users must re-enter the password if they want to update it
	if server.AuthType == "password" {
		// Leave password field empty for security - user can enter new password if needed
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
		
		// Handle password authentication with keyring storage
		if password, ok := data["password"].(string); ok && password != "" && updatedServer.AuthType == "password" {
			// Store password securely in keyring
			passwordManager, err := auth.NewPasswordManager("auto")
			if err != nil {
				return &ValidationError{Field: "password", Message: fmt.Sprintf("Failed to initialize password manager: %s", err.Error())}
			}
			
			// Update password in keyring and configure server to use it
			if err := passwordManager.StoreServerPassword(&updatedServer, password); err != nil {
				return &ValidationError{Field: "password", Message: fmt.Sprintf("Failed to store password: %s", err.Error())}
			}
		} else if updatedServer.AuthType == "password" {
			// If password auth but no new password provided, preserve existing keyring settings
			if server.UseKeyring && server.KeyringID != "" {
				updatedServer.UseKeyring = server.UseKeyring
				updatedServer.KeyringID = server.KeyringID
			}
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

// ShowAddServerModal displays the add server modal using native tview form
func (t *TUIApp) ShowAddServerModal() {
	form := t.CreateNativeAddServerForm()
	
	// Show the form directly as modal
	if t.modalManager != nil {
		t.modalManager.ShowModal(form)
	}
}

// ShowEditServerModal displays the edit server modal using native tview form
func (t *TUIApp) ShowEditServerModal(serverName string) {
	form := t.CreateNativeEditServerForm(serverName)
	
	// Show the form directly as modal
	if t.modalManager != nil {
		t.modalManager.ShowModal(form)
	}
}