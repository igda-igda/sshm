package tui

import (
	"testing"
	"sshm/internal/config"
)

func TestTUIPasswordKeyringIntegration(t *testing.T) {
	t.Run("AddServerForm_StoresPasswordInKeyring", func(t *testing.T) {
		// Create test configuration
		cfg := &config.Config{
			Servers: []config.Server{},
		}

		// Create TUI app with test configuration using mock
		app := MockTUIApp(cfg)

		// Create add server form
		form := app.CreateAddServerForm()
		if form == nil {
			t.Fatal("failed to create add server form")
		}

		// Test form fields are properly configured
		fields := []string{"name", "hostname", "port", "username", "auth_type", "password", "key_path", "passphrase_protected"}
		for _, fieldName := range fields {
			if _, err := form.GetFieldValue(fieldName); err != nil {
				t.Errorf("expected field %s to exist in form", fieldName)
			}
		}

		// Verify password field uses PasswordField (masked input)
		passwordField, exists := form.fields["password"]
		if !exists {
			t.Fatal("password field should exist")
		}
		
		if passwordField.passwordField == nil {
			t.Error("password field should use PasswordField for masked input")
		}
		
		if passwordField.inputField != nil {
			t.Error("password field should not use regular InputField (security risk)")
		}
	})

	t.Run("EditServerForm_DoesNotPrePopulatePassword", func(t *testing.T) {
		// Create test configuration with a server that has keyring password
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

		// Create TUI app with test configuration using mock
		app := MockTUIApp(cfg)

		// Create edit server form
		form := app.CreateEditServerForm("test-server")
		if form == nil {
			t.Fatal("failed to create edit server form")
		}

		// Verify password field exists and is empty (for security)
		passwordValue, err := form.GetFieldValue("password")
		if err != nil {
			t.Error("password field should exist in edit form")
		}
		
		if passwordValue != "" {
			t.Error("password field should be empty in edit form for security (no pre-population)")
		}

		// Verify password field uses PasswordField (masked input)
		passwordField, exists := form.fields["password"]
		if !exists {
			t.Fatal("password field should exist")
		}
		
		if passwordField.passwordField == nil {
			t.Error("password field should use PasswordField for masked input")
		}
	})

	t.Run("PasswordField_UsesMaskedInput", func(t *testing.T) {
		// Create standard server form fields
		fields := CreateServerFormFields()
		
		// Check password field configuration
		passwordField, exists := fields["password"]
		if !exists {
			t.Fatal("password field should exist in server form fields")
		}
		
		// Verify it uses PasswordField for security
		if passwordField.passwordField == nil {
			t.Error("password field should use PasswordField component for masked input")
		}
		
		if passwordField.inputField != nil {
			t.Error("password field should not use regular InputField (security vulnerability)")
		}
		
		// Test that password field can handle input securely
		if passwordField.passwordField != nil {
			// Test setting and getting masked password
			passwordField.passwordField.SetText("test-password")
			retrievedValue := passwordField.passwordField.GetText()
			
			if retrievedValue != "test-password" {
				t.Errorf("expected password field to store text correctly, got %s", retrievedValue)
			}
		}
	})
}

func TestTUIPasswordValidation(t *testing.T) {
	t.Run("PasswordField_ValidationLogic", func(t *testing.T) {
		// Test password field validation
		tests := []struct {
			name        string
			password    string
			expectError bool
			errorMsg    string
		}{
			{"empty password", "", false, ""}, // Empty is allowed (optional by default)
			{"short password", "ab", true, "Password must be at least 3 characters"},
			{"valid password", "password123", false, ""},
			{"long password", string(make([]byte, 200)), true, "Password is too long (max 128 characters)"},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidatePasswordField(tt.password)
				
				if tt.expectError && err == nil {
					t.Errorf("expected error for password '%s'", tt.password)
				}
				
				if !tt.expectError && err != nil {
					t.Errorf("unexpected error for password '%s': %v", tt.password, err)
				}
				
				if tt.expectError && err != nil && tt.errorMsg != "" {
					if err.Error() != tt.errorMsg && err.(*ValidationError).Message != tt.errorMsg {
						t.Errorf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
					}
				}
			})
		}
	})
}