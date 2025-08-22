package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sshm/internal/config"
)

// TestProfileCRUDOperations tests profile management CRUD operations through TUI
func TestProfileCRUDOperations(t *testing.T) {
	// Setup test configuration directory
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test configuration with some servers
	cfg := &config.Config{
		Servers: []config.Server{
			{Name: "test-server-1", Hostname: "host1.example.com", Port: 22, Username: "user1", AuthType: "key", KeyPath: "/path/to/key1"},
			{Name: "test-server-2", Hostname: "host2.example.com", Port: 22, Username: "user2", AuthType: "key", KeyPath: "/path/to/key2"},
			{Name: "test-server-3", Hostname: "host3.example.com", Port: 22, Username: "user3", AuthType: "key", KeyPath: "/path/to/key3"},
		},
		Profiles: []config.Profile{
			{Name: "existing-profile", Description: "Existing profile", Servers: []string{"test-server-1"}},
		},
	}
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := cfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	t.Run("CreateProfile", func(t *testing.T) {
		tuiApp, err := NewTUIApp()
		if err != nil {
			t.Fatalf("Failed to create TUI app: %v", err)
		}

		// Test profile creation form
		form := tuiApp.CreateProfileForm()
		if form == nil {
			t.Fatal("Expected profile form to be created")
		}

		// Set form field values
		if err := form.SetFieldValue("name", "new-profile"); err != nil {
			t.Errorf("Failed to set profile name: %v", err)
		}
		if err := form.SetFieldValue("description", "Test profile description"); err != nil {
			t.Errorf("Failed to set profile description: %v", err)
		}

		// Validate form data
		data, err := form.CollectFormData()
		if err != nil {
			t.Errorf("Form validation failed: %v", err)
		}

		if data["name"] != "new-profile" {
			t.Errorf("Expected profile name 'new-profile', got %v", data["name"])
		}
		if data["description"] != "Test profile description" {
			t.Errorf("Expected description 'Test profile description', got %v", data["description"])
		}
	})

	t.Run("CreateProfileValidation", func(t *testing.T) {
		tuiApp, err := NewTUIApp()
		if err != nil {
			t.Fatalf("Failed to create TUI app: %v", err)
		}

		form := tuiApp.CreateProfileForm()

		// Test empty name validation
		if err := form.SetFieldValue("name", ""); err != nil {
			t.Errorf("Failed to set empty profile name: %v", err)
		}

		_, err = form.CollectFormData()
		if err == nil {
			t.Error("Expected validation error for empty profile name")
		}

		// Test duplicate name validation
		if err := form.SetFieldValue("name", "existing-profile"); err != nil {
			t.Errorf("Failed to set profile name: %v", err)
		}

		_, err = form.CollectFormData()
		if err == nil {
			t.Error("Expected validation error for duplicate profile name")
		}
	})

	t.Run("DeleteProfileConfirmation", func(t *testing.T) {
		tuiApp, err := NewTUIApp()
		if err != nil {
			t.Fatalf("Failed to create TUI app: %v", err)
		}

		// Test delete confirmation dialog
		modal := tuiApp.CreateDeleteProfileModal("existing-profile")
		if modal == nil {
			t.Fatal("Expected delete confirmation modal to be created")
		}

		// Check modal content includes profile information
		// This would be tested by examining the modal text in a real UI test
	})

	t.Run("ServerAssignmentInterface", func(t *testing.T) {
		tuiApp, err := NewTUIApp()
		if err != nil {
			t.Fatalf("Failed to create TUI app: %v", err)
		}

		// Test server assignment form
		form := tuiApp.CreateServerAssignmentForm("existing-profile")
		if form == nil {
			t.Fatal("Expected server assignment form to be created")
		}

		// Test that form contains available servers
		// This would check that unassigned servers are presented as options
	})

	t.Run("ProfileModification", func(t *testing.T) {
		tuiApp, err := NewTUIApp()
		if err != nil {
			t.Fatalf("Failed to create TUI app: %v", err)
		}

		// Test profile editing form
		form := tuiApp.CreateEditProfileForm("existing-profile")
		if form == nil {
			t.Fatal("Expected profile edit form to be created")
		}

		// Check that form is pre-populated with existing values
		nameValue, err := form.GetFieldValue("name")
		if err != nil {
			t.Errorf("Failed to get profile name value: %v", err)
		}
		if nameValue != "existing-profile" {
			t.Errorf("Expected profile name 'existing-profile', got %s", nameValue)
		}

		descriptionValue, err := form.GetFieldValue("description")
		if err != nil {
			t.Errorf("Failed to get profile description value: %v", err)
		}
		if descriptionValue != "Existing profile" {
			t.Errorf("Expected description 'Existing profile', got %s", descriptionValue)
		}
	})

	t.Run("ProfileListing", func(t *testing.T) {
		tuiApp, err := NewTUIApp()
		if err != nil {
			t.Fatalf("Failed to create TUI app: %v", err)
		}

		// Test that profiles are listed in the TUI
		profiles := tuiApp.config.GetProfiles()
		if len(profiles) != 1 {
			t.Errorf("Expected 1 profile, got %d", len(profiles))
		}

		if profiles[0].Name != "existing-profile" {
			t.Errorf("Expected profile name 'existing-profile', got %s", profiles[0].Name)
		}
	})
}

// TestProfileFormValidation tests profile form validation logic
func TestProfileFormValidation(t *testing.T) {
	// Setup test configuration directory
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	tests := []struct {
		name        string
		profileName string
		description string
		wantErr     bool
		errContains string
	}{
		{
			name:        "ValidProfile",
			profileName: "valid-profile",
			description: "A valid profile",
			wantErr:     false,
		},
		{
			name:        "EmptyName",
			profileName: "",
			description: "Profile with empty name",
			wantErr:     true,
			errContains: "required",
		},
		{
			name:        "ValidWithoutDescription",
			profileName: "minimal-profile",
			description: "",
			wantErr:     false,
		},
		{
			name:        "LongName",
			profileName: "this-is-a-very-long-profile-name-that-exceeds-reasonable-limits-and-should-be-rejected",
			description: "Profile with very long name",
			wantErr:     true,
			errContains: "must be less than 50 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := tuiApp.CreateProfileForm()
			
			if err := form.SetFieldValue("name", tt.profileName); err != nil {
				t.Errorf("Failed to set profile name: %v", err)
			}
			if err := form.SetFieldValue("description", tt.description); err != nil {
				t.Errorf("Failed to set profile description: %v", err)
			}

			_, err := form.CollectFormData()
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected validation error, got none")
				} else if tt.errContains != "" && !containsIgnoreCase(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestKeyboardNavigationInProfileForms tests keyboard navigation in profile forms
func TestKeyboardNavigationInProfileForms(t *testing.T) {
	// Setup test configuration directory
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	t.Run("TabNavigationInProfileForm", func(t *testing.T) {
		form := tuiApp.CreateProfileForm()
		
		// Simulate keyboard navigation
		formWidget := form.GetForm()
		
		// Test Tab key navigation
		event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		// Call input handler but don't expect a return value
		formWidget.InputHandler()(event, nil)
		
		// In a full implementation, we would check that focus moved to the next field
		// For now, we just ensure the event is processed without errors
	})

	t.Run("EscapeKeyInProfileForm", func(t *testing.T) {
		form := tuiApp.CreateProfileForm()
		formWidget := form.GetForm()
		
		// Test Escape key handling
		event := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
		formWidget.InputHandler()(event, nil)
	})

	t.Run("EnterKeySubmission", func(t *testing.T) {
		form := tuiApp.CreateProfileForm()
		
		// Set valid form data
		if err := form.SetFieldValue("name", "test-profile"); err != nil {
			t.Errorf("Failed to set profile name: %v", err)
		}
		if err := form.SetFieldValue("description", "Test description"); err != nil {
			t.Errorf("Failed to set profile description: %v", err)
		}
		
		// Test Enter key submission
		formWidget := form.GetForm()
		event := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		formWidget.InputHandler()(event, nil)
	})
}

// TestServerAssignmentOperations tests server assignment and unassignment
func TestServerAssignmentOperations(t *testing.T) {
	// Setup test configuration directory
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test configuration
	cfg := &config.Config{
		Servers: []config.Server{
			{Name: "server-1", Hostname: "host1.com", Port: 22, Username: "user1", AuthType: "key", KeyPath: "/key1"},
			{Name: "server-2", Hostname: "host2.com", Port: 22, Username: "user2", AuthType: "key", KeyPath: "/key2"},
			{Name: "server-3", Hostname: "host3.com", Port: 22, Username: "user3", AuthType: "key", KeyPath: "/key3"},
		},
		Profiles: []config.Profile{
			{Name: "test-profile", Description: "Test profile", Servers: []string{"server-1"}},
		},
	}
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := cfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	tuiApp, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	t.Run("AssignServerToProfile", func(t *testing.T) {
		// Test server assignment interface
		form := tuiApp.CreateServerAssignmentForm("test-profile")
		if form == nil {
			t.Fatal("Expected server assignment form to be created")
		}

		// The assignment form should show available servers (server-2, server-3)
		// and allow assignment to the profile
	})

	t.Run("UnassignServerFromProfile", func(t *testing.T) {
		// Test server unassignment interface
		form := tuiApp.CreateServerUnassignmentForm("test-profile")
		if form == nil {
			t.Fatal("Expected server unassignment form to be created")
		}

		// The unassignment form should show assigned servers (server-1)
		// and allow removal from the profile
	})
}

// Helper function for case-insensitive string contains check
func containsIgnoreCase(s, substr string) bool {
	s = strings.ToLower(s)
	substr = strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// MockTUIApp creates a mock TUI app for testing
func MockTUIApp(cfg *config.Config) *TUIApp {
	app := tview.NewApplication()
	layout := tview.NewFlex()
	
	return &TUIApp{
		app:          app,
		layout:       layout,
		config:       cfg,
		modalManager: NewModalManager(app, layout),
	}
}

// ProfileFormTestHelper provides utilities for testing profile forms
type ProfileFormTestHelper struct {
	form    *TUIForm
	testApp *tview.Application
	events  chan *tcell.EventKey
}

// NewProfileFormTestHelper creates a new test helper for profile forms
func NewProfileFormTestHelper(form *TUIForm) *ProfileFormTestHelper {
	return &ProfileFormTestHelper{
		form:    form,
		testApp: tview.NewApplication(),
		events:  make(chan *tcell.EventKey, 10),
	}
}

// SimulateKeypress simulates a keypress event
func (pth *ProfileFormTestHelper) SimulateKeypress(key tcell.Key) {
	pth.events <- tcell.NewEventKey(key, 0, tcell.ModNone)
}

// SimulateInput simulates text input
func (pth *ProfileFormTestHelper) SimulateInput(text string) {
	for _, r := range text {
		pth.events <- tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
	}
}

// WaitForFormSubmission waits for form submission with timeout
func (pth *ProfileFormTestHelper) WaitForFormSubmission(timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return false
	default:
		// In a real implementation, this would wait for form submission callback
		return true
	}
}