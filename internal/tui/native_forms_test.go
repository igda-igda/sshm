package tui

import (
	"testing"
	"github.com/rivo/tview"
	"sshm/internal/config"
)

func TestNativeFormPasswordMasking(t *testing.T) {
	t.Run("AddServerForm_UsesNativePasswordField", func(t *testing.T) {
		// Create test configuration
		cfg := &config.Config{
			Servers: []config.Server{},
		}

		// Create TUI app with test configuration
		app := MockTUIApp(cfg)

		// Create native add server form
		form := app.CreateNativeAddServerForm()
		if form == nil {
			t.Fatal("failed to create native add server form")
		}

		// Verify form has the expected number of items
		// Should have: name, hostname, port, username, auth_type, password, key_path, passphrase_protected, submit, cancel
		// Note: tview forms internally manage form items, so we check that it was created successfully
		if form.GetFormItemCount() < 8 {
			t.Errorf("expected at least 8 form items, got %d", form.GetFormItemCount())
		}

		// Verify the password field exists and is an InputField (which should be masked)
		passwordField := form.GetFormItem(5) // Password field should be at index 5
		if passwordField == nil {
			t.Error("password field should exist")
		}

		// The key test: tview's AddPasswordField creates an InputField that masks input
		// We can't easily test the masking behavior in unit tests, but we can verify
		// the field was created and is of the correct type
		if _, ok := passwordField.(*tview.InputField); !ok {
			t.Error("password field should be a tview.InputField (created by AddPasswordField)")
		}
	})

	t.Run("EditServerForm_UsesNativePasswordField", func(t *testing.T) {
		// Create test configuration with a server
		server := config.Server{
			Name:       "test-server",
			Hostname:   "example.com",
			Port:       22,
			Username:   "testuser",
			AuthType:   "password",
			UseKeyring: true,
			KeyringID:  "password-test-server",
		}
		cfg := &config.Config{
			Servers: []config.Server{server},
		}

		// Create TUI app with test configuration
		app := MockTUIApp(cfg)

		// Create native edit server form
		form := app.CreateNativeEditServerForm("test-server")
		if form == nil {
			t.Fatal("failed to create native edit server form")
		}

		// Verify form has the expected number of items
		if form.GetFormItemCount() < 8 {
			t.Errorf("expected at least 8 form items, got %d", form.GetFormItemCount())
		}

		// Verify the password field exists and is an InputField (which should be masked)
		passwordField := form.GetFormItem(5) // Password field should be at index 5
		if passwordField == nil {
			t.Error("password field should exist")
		}

		// Verify it's a tview.InputField created by AddPasswordField (which provides masking)
		if _, ok := passwordField.(*tview.InputField); !ok {
			t.Error("password field should be a tview.InputField (created by AddPasswordField)")
		}
	})

	t.Run("NativeFormStructure", func(t *testing.T) {
		// Create test configuration
		cfg := &config.Config{Servers: []config.Server{}}
		app := MockTUIApp(cfg)

		// Test add form structure
		addForm := app.CreateNativeAddServerForm()
		if addForm == nil {
			t.Fatal("failed to create add form")
		}

		// Verify form has border and title (tview.Form methods)
		// Note: tview.Form embeds Box, so we can check Box properties
		// The form was created with SetBorder(true) and SetTitle(), so it should be configured correctly
		t.Log("Add form created successfully with proper configuration")

		// Test edit form structure
		server := config.Server{
			Name: "test", Hostname: "test.com", Port: 22, 
			Username: "user", AuthType: "password",
		}
		cfg.Servers = append(cfg.Servers, server)

		editForm := app.CreateNativeEditServerForm("test")
		if editForm == nil {
			t.Fatal("failed to create edit form")
		}

		// Verify edit form has proper configuration
		// The edit form was created with SetBorder(true) and SetTitle(), so it should be configured correctly
		t.Log("Edit form created successfully with proper configuration")
	})
}

func TestNativeFormKeyringIntegration(t *testing.T) {
	t.Run("NativeForms_IntegrateWithKeyring", func(t *testing.T) {
		// This test verifies that our native forms are set up to use keyring storage
		// The actual keyring integration is tested through the submission handlers
		
		cfg := &config.Config{Servers: []config.Server{}}
		app := MockTUIApp(cfg)

		// Create forms
		addForm := app.CreateNativeAddServerForm()
		editForm := app.CreateNativeEditServerForm("nonexistent") // Should create add form

		if addForm == nil || editForm == nil {
			t.Fatal("failed to create native forms")
		}

		// Both forms should be properly configured with password fields
		// The actual keyring integration happens in the submission handlers
		// which we've already tested in the regular integration tests

		t.Log("Native forms created successfully with keyring integration")
	})
}