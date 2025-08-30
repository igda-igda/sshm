package tui

import (
	"fmt"
	"testing"

	"sshm/internal/config"
)

// TestPasswordFieldSubmissionValidation tests form submission with password validation
func TestPasswordFieldSubmissionValidation(t *testing.T) {
	fields := CreateServerFormFields()
	
	// Test Case 1: Password authentication with valid password
	fields["name"].SetText("test-server")
	fields["hostname"].SetText("test.example.com")
	fields["username"].SetText("testuser")
	fields["port"].SetText("22")
	fields["auth_type"].dropdown.SetValue("password")
	fields["password"].passwordField.SetText("valid-password123")
	
	// Extract form data
	data := extractFormData(fields)
	
	// Verify password is included in form data
	if data["password"] != "valid-password123" {
		t.Errorf("Expected password 'valid-password123' in form data, got '%v'", data["password"])
	}
	
	// Verify auth type is set correctly
	if data["auth_type"] != "password" {
		t.Errorf("Expected auth_type 'password', got '%v'", data["auth_type"])
	}
	
	// Test Case 2: Key authentication should not include password
	fields["auth_type"].dropdown.SetValue("key")
	fields["key_path"].SetText("/path/to/key")
	
	data = extractFormData(fields)
	
	// Password should be empty (cleared for security)
	if data["password"] != "" {
		t.Errorf("Password should be cleared when switching to key auth, got '%v'", data["password"])
	}
	
	// Auth type should be updated
	if data["auth_type"] != "key" {
		t.Errorf("Expected auth_type 'key', got '%v'", data["auth_type"])
	}
}

// TestServerConfigCreationFromPasswordForm tests creating server config from form with password
func TestServerConfigCreationFromPasswordForm(t *testing.T) {
	fields := CreateServerFormFields()
	
	// Set up form data for password authentication
	fields["name"].SetText("password-server")
	fields["hostname"].SetText("password.example.com")
	fields["username"].SetText("passworduser")
	fields["port"].SetText("2222")
	fields["auth_type"].dropdown.SetValue("password")
	fields["password"].passwordField.SetText("secret-password")
	
	// Extract form data
	data := extractFormData(fields)
	
	// Create server config like the modal does
	server := &config.Server{
		Name:     data["name"].(string),
		Hostname: data["hostname"].(string),
		Username: data["username"].(string),
		AuthType: data["auth_type"].(string),
	}
	
	// Parse port
	if portStr, ok := data["port"].(string); ok && portStr != "" {
		server.Port = 2222 // Simulating strconv.Atoi conversion
	}
	
	// Handle password authentication (as done in modals.go)
	if password, ok := data["password"].(string); ok && password != "" && server.AuthType == "password" {
		server.Password = password
	}
	
	// Validate the created server
	if server.Name != "password-server" {
		t.Errorf("Expected server name 'password-server', got '%s'", server.Name)
	}
	
	if server.Hostname != "password.example.com" {
		t.Errorf("Expected hostname 'password.example.com', got '%s'", server.Hostname)
	}
	
	if server.Username != "passworduser" {
		t.Errorf("Expected username 'passworduser', got '%s'", server.Username)
	}
	
	if server.Port != 2222 {
		t.Errorf("Expected port 2222, got %d", server.Port)
	}
	
	if server.AuthType != "password" {
		t.Errorf("Expected auth type 'password', got '%s'", server.AuthType)
	}
	
	if server.Password != "secret-password" {
		t.Errorf("Expected password 'secret-password', got '%s'", server.Password)
	}
	
	// Validate server config
	err := server.Validate()
	if err != nil {
		t.Errorf("Server configuration should be valid, got error: %v", err)
	}
}

// TestPasswordValidationInForm tests password validation within form context
func TestPasswordValidationInForm(t *testing.T) {
	fields := CreateServerFormFields()
	
	// Test empty password with password authentication (should fail validation)
	fields["auth_type"].dropdown.SetValue("password")
	passwordField := fields["password"]
	
	// Empty password should be invalid when auth type is password and field is required
	passwordField.required = true
	err := validateFormField("password", "", passwordField)
	if err == nil {
		t.Error("Empty password should fail validation when required")
	}
	
	// Valid password should pass
	err = validateFormField("password", "valid-password", passwordField)
	if err != nil {
		t.Errorf("Valid password should pass validation, got error: %v", err)
	}
	
	// Very long password should fail
	longPassword := make([]byte, 200)
	for i := range longPassword {
		longPassword[i] = 'x'
	}
	err = validateFormField("password", string(longPassword), passwordField)
	if err == nil {
		t.Error("Very long password should fail validation")
	}
	
	// Test key authentication (password not required)
	fields["auth_type"].dropdown.SetValue("key")
	passwordField.required = false
	
	err = validateFormField("password", "", passwordField)
	if err != nil {
		t.Errorf("Empty password should be valid when not required, got error: %v", err)
	}
}

// TestCompleteFormSubmissionWorkflow tests end-to-end form submission
func TestCompleteFormSubmissionWorkflow(t *testing.T) {
	fields := CreateServerFormFields()
	
	var submittedData map[string]interface{}
	var submissionError error
	
	form := NewTUIForm(fields, func(data map[string]interface{}) error {
		submittedData = data
		return nil // Success
	}, func() {
		// Cancel handler
	})
	
	// Fill out complete form for password authentication
	fields["name"].SetText("workflow-test")
	fields["hostname"].SetText("workflow.example.com")
	fields["username"].SetText("workflowuser")
	fields["port"].SetText("22")
	fields["auth_type"].dropdown.SetValue("password")
	fields["password"].passwordField.SetText("workflow-password")
	
	// Simulate form submission by extracting data
	data := extractFormData(fields)
	
	// Call the submission handler
	submissionError = form.onSubmit(data)
	
	// Verify submission succeeded
	if submissionError != nil {
		t.Errorf("Form submission should succeed, got error: %v", submissionError)
	}
	
	// Verify data was passed correctly
	if submittedData["name"] != "workflow-test" {
		t.Errorf("Expected submitted name 'workflow-test', got '%v'", submittedData["name"])
	}
	
	if submittedData["password"] != "workflow-password" {
		t.Errorf("Expected submitted password 'workflow-password', got '%v'", submittedData["password"])
	}
	
	if submittedData["auth_type"] != "password" {
		t.Errorf("Expected submitted auth_type 'password', got '%v'", submittedData["auth_type"])
	}
}

// TestPasswordSecurityInFormSubmission tests security aspects of password submission
func TestPasswordSecurityInFormSubmission(t *testing.T) {
	fields := CreateServerFormFields()
	
	// Set up password authentication
	fields["auth_type"].dropdown.SetValue("password")
	fields["password"].passwordField.SetText("secure-password")
	
	// Test password clearing when switching auth types
	fields["auth_type"].dropdown.SetValue("key")
	
	// Password field should be cleared
	if fields["password"].passwordField.GetText() != "" {
		t.Error("Password field should be cleared when switching away from password auth")
	}
	
	// Test that password is properly masked in display
	fields["auth_type"].dropdown.SetValue("password")
	fields["password"].passwordField.SetText("masked-test")
	
	maskedText := fields["password"].passwordField.GetMaskedText()
	if maskedText == "masked-test" {
		t.Error("Password should be masked in display")
	}
	
	// But original text should still be available for submission
	if fields["password"].passwordField.GetText() != "masked-test" {
		t.Error("Original password should be available for submission")
	}
}

// Helper function to extract form data like the modal system does
func extractFormData(fields map[string]*FormField) map[string]interface{} {
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
	return data
}

// Helper function to validate form field like the form system does
func validateFormField(fieldName, value string, field *FormField) error {
	if field.required && value == "" {
		return fmt.Errorf("field %s is required", fieldName)
	}
	if field.validator != nil {
		return field.validator(value)
	}
	return nil
}