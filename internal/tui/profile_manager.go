package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sshm/internal/config"
)

// CreateProfileForm creates a form for creating new profiles
func (t *TUIApp) CreateProfileForm() *TUIForm {
	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().
				SetLabel("Profile Name: ").
				SetFieldWidth(30).
				SetPlaceholder("e.g., development, production"),
			validator: func(value string) error {
				return t.validateProfileName(value, "")
			},
			required: true,
		},
		"description": {
			inputField: tview.NewInputField().
				SetLabel("Description (optional): ").
				SetFieldWidth(50).
				SetPlaceholder("e.g., Development environment servers"),
			validator: ValidateProfileDescription,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		// Create new profile
		profile := config.Profile{
			Name:        data["name"].(string),
			Description: data["description"].(string),
			Servers:     []string{},
		}

		// Add profile to configuration
		if err := t.config.AddProfile(profile); err != nil {
			return &ValidationError{Field: "general", Message: fmt.Sprintf("Failed to add profile: %s", err.Error())}
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

// CreateEditProfileForm creates a form for editing existing profiles
func (t *TUIApp) CreateEditProfileForm(profileName string) *TUIForm {
	// Get existing profile configuration
	profile, err := t.config.GetProfile(profileName)
	if err != nil {
		// Return empty form if profile not found
		return t.CreateProfileForm()
	}

	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().
				SetLabel("Profile Name: ").
				SetText(profile.Name).
				SetFieldWidth(30).
				SetPlaceholder("e.g., development, production"),
			validator: func(value string) error {
				return t.validateProfileName(value, profileName)
			},
			required: true,
		},
		"description": {
			inputField: tview.NewInputField().
				SetLabel("Description (optional): ").
				SetText(profile.Description).
				SetFieldWidth(50).
				SetPlaceholder("e.g., Development environment servers"),
			validator: ValidateProfileDescription,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		// Update profile configuration
		updatedProfile := config.Profile{
			Name:        data["name"].(string),
			Description: data["description"].(string),
			Servers:     profile.Servers, // Keep existing server assignments
		}

		// Find and replace the profile in configuration
		for i, p := range t.config.Profiles {
			if p.Name == profileName {
				t.config.Profiles[i] = updatedProfile
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

// CreateDeleteProfileModal creates a confirmation modal for profile deletion
func (t *TUIApp) CreateDeleteProfileModal(profileName string) *tview.Modal {
	// Get profile information
	profile, err := t.config.GetProfile(profileName)
	if err != nil {
		// If profile doesn't exist, show error
		return tview.NewModal().
			SetText(fmt.Sprintf("Profile '%s' not found.", profileName)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				if t.modalManager != nil {
					t.modalManager.HideModal()
				}
			}).
			SetBackgroundColor(tcell.ColorDarkRed)
	}

	// Create confirmation dialog with profile details
	modalText := fmt.Sprintf("Delete profile '%s'?\n\nDescription: %s\nAssigned servers: %d\n\nThis action cannot be undone.\nServers will not be deleted, only removed from this profile.",
		profile.Name,
		profile.Description,
		len(profile.Servers))

	modal := tview.NewModal().
		SetText(modalText).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			defer func() {
				// Always return to main layout
				if t.modalManager != nil {
					t.modalManager.HideModal()
				}
			}()

			if buttonLabel == "Delete" {
				// Delete the profile from configuration
				if err := t.deleteProfileFromConfig(profileName); err != nil {
					// Show error modal
					t.showErrorModal(fmt.Sprintf("Error deleting profile: %s", err.Error()))
					return
				}

				// Refresh the display after successful deletion
				t.initializeProfileTabs()
				t.updateProfileDisplay()
				t.refreshServerList()
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
			}
			return nil
		case tcell.KeyEnter:
			// Enter key confirms delete
			if err := t.deleteProfileFromConfig(profileName); err != nil {
				t.showErrorModal(fmt.Sprintf("Error deleting profile: %s", err.Error()))
				return nil
			}

			// Refresh the display after successful deletion
			t.initializeProfileTabs()
			t.updateProfileDisplay()
			t.refreshServerList()

			// Return to main layout
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
			return nil
		case tcell.Key('d'), tcell.Key('D'):
			// 'd' key also confirms delete
			if err := t.deleteProfileFromConfig(profileName); err != nil {
				t.showErrorModal(fmt.Sprintf("Error deleting profile: %s", err.Error()))
				return nil
			}

			// Refresh the display after successful deletion
			t.initializeProfileTabs()
			t.updateProfileDisplay()
			t.refreshServerList()

			// Return to main layout
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
			return nil
		}
		return event
	})

	return modal
}

// CreateServerAssignmentForm creates a form for assigning servers to profiles
func (t *TUIApp) CreateServerAssignmentForm(profileName string) *TUIForm {
	// Get unassigned servers
	unassignedServers := t.getUnassignedServers(profileName)
	
	if len(unassignedServers) == 0 {
		// No servers available to assign, return a simple message form
		fields := map[string]*FormField{
			"message": {
				inputField: tview.NewInputField().
					SetLabel("").
					SetText("No servers available to assign to this profile"),
				validator: nil,
				required:  false,
			},
		}

		onSubmit := func(data map[string]interface{}) error {
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
			return nil
		}

		onCancel := func() {
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
		}

		return NewTUIForm(fields, onSubmit, onCancel)
	}

	// Create server selection field
	serverOptions := make([]string, len(unassignedServers))
	for i, server := range unassignedServers {
		serverOptions[i] = server.Name
	}

	fields := map[string]*FormField{
		"server": {
			inputField: tview.NewInputField().
				SetLabel("Select Server: ").
				SetFieldWidth(30).
				SetPlaceholder("Enter server name"),
			validator: func(value string) error {
				return t.validateServerSelection(value, unassignedServers)
			},
			required: true,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		serverName := data["server"].(string)

		// Assign server to profile
		if err := t.config.AssignServerToProfile(serverName, profileName); err != nil {
			return &ValidationError{Field: "general", Message: fmt.Sprintf("Failed to assign server: %s", err.Error())}
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

// CreateServerUnassignmentForm creates a form for unassigning servers from profiles
func (t *TUIApp) CreateServerUnassignmentForm(profileName string) *TUIForm {
	// Get profile and its assigned servers
	profile, err := t.config.GetProfile(profileName)
	if err != nil {
		// Return empty form if profile not found
		fields := map[string]*FormField{
			"message": {
				inputField: tview.NewInputField().
					SetLabel("").
					SetText("Profile not found"),
				validator: nil,
				required:  false,
			},
		}

		onSubmit := func(data map[string]interface{}) error {
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
			return nil
		}

		onCancel := func() {
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
		}

		return NewTUIForm(fields, onSubmit, onCancel)
	}

	if len(profile.Servers) == 0 {
		// No servers assigned to profile
		fields := map[string]*FormField{
			"message": {
				inputField: tview.NewInputField().
					SetLabel("").
					SetText("No servers assigned to this profile"),
				validator: nil,
				required:  false,
			},
		}

		onSubmit := func(data map[string]interface{}) error {
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
			return nil
		}

		onCancel := func() {
			if t.modalManager != nil {
				t.modalManager.HideModal()
			}
		}

		return NewTUIForm(fields, onSubmit, onCancel)
	}

	fields := map[string]*FormField{
		"server": {
			inputField: tview.NewInputField().
				SetLabel("Select Server to Remove: ").
				SetFieldWidth(30).
				SetPlaceholder("Enter server name"),
			validator: func(value string) error {
				return t.validateAssignedServerSelection(value, profile.Servers)
			},
			required: true,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		serverName := data["server"].(string)

		// Unassign server from profile
		if err := t.config.UnassignServerFromProfile(serverName, profileName); err != nil {
			return &ValidationError{Field: "general", Message: fmt.Sprintf("Failed to unassign server: %s", err.Error())}
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

// Validation functions for profile management

// validateProfileName validates profile name and checks for duplicates
func (t *TUIApp) validateProfileName(value, currentProfileName string) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: "name", Message: "Profile name is required"}
	}

	if len(value) < 2 {
		return &ValidationError{Field: "name", Message: "Profile name must be at least 2 characters"}
	}

	if len(value) > 50 {
		return &ValidationError{Field: "name", Message: "Profile name must be less than 50 characters"}
	}

	// Check for valid characters (alphanumeric, dash, underscore, space)
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == ' ') {
			return &ValidationError{Field: "name", Message: "Profile name can only contain letters, numbers, dashes, underscores, and spaces"}
		}
	}

	// Check for duplicates (allow same name when editing)
	if value != currentProfileName {
		if _, err := t.config.GetProfile(value); err == nil {
			return &ValidationError{Field: "name", Message: "Profile name already exists"}
		}
	}

	return nil
}

// ValidateProfileDescription validates profile description
func ValidateProfileDescription(value string) error {
	if len(value) > 200 {
		return &ValidationError{Field: "description", Message: "Description must be less than 200 characters"}
	}
	return nil
}

// validateServerSelection validates server selection for assignment
func (t *TUIApp) validateServerSelection(value string, availableServers []config.Server) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: "server", Message: "Server selection is required"}
	}

	// Check if server exists in available servers
	for _, server := range availableServers {
		if server.Name == value {
			return nil
		}
	}

	return &ValidationError{Field: "server", Message: "Selected server is not available for assignment"}
}

// validateAssignedServerSelection validates server selection for unassignment
func (t *TUIApp) validateAssignedServerSelection(value string, assignedServers []string) error {
	if strings.TrimSpace(value) == "" {
		return &ValidationError{Field: "server", Message: "Server selection is required"}
	}

	// Check if server exists in assigned servers
	for _, serverName := range assignedServers {
		if serverName == value {
			return nil
		}
	}

	return &ValidationError{Field: "server", Message: "Selected server is not assigned to this profile"}
}

// Helper functions

// getUnassignedServers returns servers that are not assigned to the specified profile
func (t *TUIApp) getUnassignedServers(profileName string) []config.Server {
	profile, err := t.config.GetProfile(profileName)
	if err != nil {
		// If profile doesn't exist, return all servers
		return t.config.GetServers()
	}

	allServers := t.config.GetServers()
	var unassignedServers []config.Server

	// Create a map of assigned server names for quick lookup
	assignedMap := make(map[string]bool)
	for _, serverName := range profile.Servers {
		assignedMap[serverName] = true
	}

	// Filter out assigned servers
	for _, server := range allServers {
		if !assignedMap[server.Name] {
			unassignedServers = append(unassignedServers, server)
		}
	}

	return unassignedServers
}

// deleteProfileFromConfig removes a profile from the configuration
func (t *TUIApp) deleteProfileFromConfig(profileName string) error {
	// Remove profile from configuration
	if err := t.config.RemoveProfile(profileName); err != nil {
		return fmt.Errorf("failed to remove profile: %w", err)
	}

	// Save the updated configuration
	if err := t.config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// Profile management modal display functions

// ShowCreateProfileModal displays the create profile modal
func (t *TUIApp) ShowCreateProfileModal() {
	form := t.CreateProfileForm()

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
	form.GetForm().SetBorder(true).SetTitle(" Create Profile ").SetBorderColor(tcell.ColorYellow)

	// Show the form directly as modal
	if t.modalManager != nil {
		t.modalManager.ShowModal(form.GetForm())
	}
}

// ShowEditProfileModal displays the edit profile modal
func (t *TUIApp) ShowEditProfileModal(profileName string) {
	form := t.CreateEditProfileForm(profileName)

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
	form.GetForm().SetBorder(true).SetTitle(fmt.Sprintf(" Edit Profile: %s ", profileName)).SetBorderColor(tcell.ColorYellow)

	// Show the form directly as modal
	if t.modalManager != nil {
		t.modalManager.ShowModal(form.GetForm())
	}
}

// ShowDeleteProfileModal displays the delete profile confirmation modal
func (t *TUIApp) ShowDeleteProfileModal(profileName string) {
	modal := t.CreateDeleteProfileModal(profileName)

	if t.modalManager != nil {
		t.modalManager.ShowModal(modal)
	}
}

// ShowServerAssignmentModal displays the server assignment modal
func (t *TUIApp) ShowServerAssignmentModal(profileName string) {
	form := t.CreateServerAssignmentForm(profileName)

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
	form.GetForm().SetBorder(true).SetTitle(fmt.Sprintf(" Assign Server to %s ", profileName)).SetBorderColor(tcell.ColorYellow)

	// Show the form directly as modal
	if t.modalManager != nil {
		t.modalManager.ShowModal(form.GetForm())
	}
}

// ShowServerUnassignmentModal displays the server unassignment modal
func (t *TUIApp) ShowServerUnassignmentModal(profileName string) {
	form := t.CreateServerUnassignmentForm(profileName)

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
	form.GetForm().SetBorder(true).SetTitle(fmt.Sprintf(" Remove Server from %s ", profileName)).SetBorderColor(tcell.ColorYellow)

	// Show the form directly as modal
	if t.modalManager != nil {
		t.modalManager.ShowModal(form.GetForm())
	}
}