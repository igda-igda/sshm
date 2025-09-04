package tui

import (
	"testing"

	"github.com/rivo/tview"
	"sshm/internal/config"
)

// TestPasswordFieldIntegration tests that password field integrates correctly with forms
func TestPasswordFieldIntegration(t *testing.T) {
	// Create form fields with our new password field integration
	fields := CreateServerFormFields()

	// Verify password field exists
	passwordField, exists := fields["password"]
	if !exists {
		t.Fatal("Password field not found in CreateServerFormFields")
	}

	// Verify password field has PasswordField component
	if passwordField.passwordField == nil {
		t.Fatal("Password field should have PasswordField component")
	}

	// Verify auth type field exists
	authField, exists := fields["auth_type"]
	if !exists {
		t.Fatal("Auth type field not found in CreateServerFormFields")
	}

	// Verify auth type field has AuthenticationSelector component
	if authField.dropdown == nil {
		t.Fatal("Auth type field should have AuthenticationSelector component")
	}

	// Test that password field can be controlled
	passwordField.passwordField.SetText("test-password")
	if passwordField.passwordField.GetText() != "test-password" {
		t.Error("Password field SetText/GetText not working correctly")
	}

	// Test that password field can be cleared
	passwordField.passwordField.Clear()
	if passwordField.passwordField.GetText() != "" {
		t.Error("Password field Clear() not working correctly")
	}

	// Test form item interface
	formItem := passwordField.passwordField.GetFormItem()
	if formItem == nil {
		t.Error("Password field should return valid form item")
	}

	// Verify the form item is an InputField (required by tview forms)
	if _, ok := formItem.(*tview.InputField); !ok {
		t.Error("Password field form item should be InputField for tview compatibility")
	}
}

// TestFormSubmissionWithPasswordField tests form submission handles password correctly
func TestFormSubmissionWithPasswordField(t *testing.T) {
	fields := CreateServerFormFields()

	// Set up server data with password authentication
	fields["name"].inputField.SetText("test-server")
	fields["hostname"].inputField.SetText("test.example.com")
	fields["username"].inputField.SetText("testuser")
	fields["port"].inputField.SetText("22")
	fields["auth_type"].dropdown.SetValue("password")
	fields["password"].passwordField.SetText("secret123")

	// Extract form data like extractFormData would
	data := make(map[string]interface{})
	for key, field := range fields {
		var value string
		if field.inputField != nil {
			value = field.inputField.GetText()
		} else if field.passwordField != nil {
			value = field.passwordField.GetText()
		} else if field.dropdown != nil {
			value = field.dropdown.GetValue()
		}
		data[key] = value
	}

	// Verify password is captured
	if data["password"] != "secret123" {
		t.Errorf("Expected password 'secret123', got '%v'", data["password"])
	}

	// Convert to server config (simulating what the modal does)
	server := &config.Server{
		Name:     data["name"].(string),
		Hostname: data["hostname"].(string),
		Username: data["username"].(string),
		AuthType: data["auth_type"].(string),
	}

	// Handle password authentication (as done in modals.go)
	if password, ok := data["password"].(string); ok && password != "" && server.AuthType == "password" {
		server.Password = password
	}

	// Verify server has password
	if server.Password != "secret123" {
		t.Errorf("Expected server password 'secret123', got '%s'", server.Password)
	}

	// Verify auth type is set correctly
	if server.AuthType != "password" {
		t.Errorf("Expected auth type 'password', got '%s'", server.AuthType)
	}
}

// TestPasswordFieldValidation tests password field validation
func TestPasswordFieldValidation(t *testing.T) {
	fields := CreateServerFormFields()
	
	passwordField := fields["password"]
	
	// Test validation function exists
	if passwordField.validator == nil {
		t.Fatal("Password field should have validator function")
	}
	
	// Test empty password validation (should be okay when not required)
	err := passwordField.validator("")
	if err != nil {
		t.Errorf("Empty password should be valid when not required, got error: %v", err)
	}
	
	// Test valid password
	err = passwordField.validator("valid-password")
	if err != nil {
		t.Errorf("Valid password should pass validation, got error: %v", err)
	}
	
	// Test very long password (should fail)
	longPassword := make([]byte, 200)
	for i := range longPassword {
		longPassword[i] = 'a'
	}
	err = passwordField.validator(string(longPassword))
	if err == nil {
		t.Error("Very long password should fail validation")
	}
}

// TestPasswordFieldSecurity tests security aspects of password field
func TestPasswordFieldSecurity(t *testing.T) {
	fields := CreateServerFormFields()
	passwordField := fields["password"]
	
	// Test password masking
	passwordField.passwordField.SetText("secret123")
	maskedText := passwordField.passwordField.GetMaskedText()
	
	// Should be masked (not the actual password)
	if maskedText == "secret123" {
		t.Error("Password should be masked in display")
	}
	
	// Should have same length as original password
	if len(maskedText) != len("secret123") {
		t.Errorf("Masked password should have same length as original, got %d expected %d", 
			len(maskedText), len("secret123"))
	}
	
	// But GetText should still return original
	if passwordField.passwordField.GetText() != "secret123" {
		t.Error("GetText() should return original password")
	}
}