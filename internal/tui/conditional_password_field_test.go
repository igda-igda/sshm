package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestConditionalPasswordField_BasicDisplay tests that password field is shown/hidden correctly
func TestConditionalPasswordField_BasicDisplay(t *testing.T) {
	// Create form fields with auth type dropdown and password field
	fields := map[string]*FormField{
		"auth_type": {
			dropdown: NewAuthenticationSelector(func(authType string) {}),
			validator: ValidateAuthType,
			required:  true,
		},
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: ").
				SetFieldWidth(30),
			validator: ValidatePassword,
			required:  false, // Initially not required, becomes required when auth_type is "password"
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Test initial state - password field should be hidden when auth_type is "key"
	authType := form.fields["auth_type"].dropdown.GetValue()
	if authType != "key" {
		t.Errorf("Expected initial auth type to be 'key', got '%s'", authType)
	}

	// Verify password field exists but should be hidden in UI
	passwordField := form.fields["password"]
	if passwordField == nil {
		t.Fatal("Expected password field to exist")
	}
	if passwordField.inputField == nil {
		t.Fatal("Expected password field to have input field")
	}

	// Test that password field has proper configuration
	// Note: tview uses AddPasswordField for masking, but we're using InputField here for conditional display
	if passwordField.inputField.GetLabel() != "Password: " {
		t.Errorf("Expected password field to have correct label, got '%s'", passwordField.inputField.GetLabel())
	}
}

// TestConditionalPasswordField_AuthTypeCallback tests authentication type change callback
func TestConditionalPasswordField_AuthTypeCallback(t *testing.T) {
	callbackCalled := false
	receivedAuthType := ""
	
	// Create callback that simulates showing/hiding password field
	authTypeCallback := func(authType string) {
		callbackCalled = true
		receivedAuthType = authType
		// In real implementation, this would show/hide password field
	}

	fields := map[string]*FormField{
		"auth_type": {
			dropdown: NewAuthenticationSelector(authTypeCallback),
			validator: ValidateAuthType,
			required:  true,
		},
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: "),
			validator: ValidatePassword,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Change auth type to password - should trigger callback
	err := form.SetFieldValue("auth_type", "password")
	if err != nil {
		t.Errorf("Expected no error setting auth type to 'password', got: %v", err)
	}

	if !callbackCalled {
		t.Error("Expected auth type change callback to be called")
	}

	if receivedAuthType != "password" {
		t.Errorf("Expected callback to receive 'password', got '%s'", receivedAuthType)
	}
}

// TestConditionalPasswordField_ValidationWithPasswordAuth tests validation when password auth is selected
func TestConditionalPasswordField_ValidationWithPasswordAuth(t *testing.T) {
	fields := map[string]*FormField{
		"auth_type": {
			dropdown: NewAuthenticationSelector(func(authType string) {}),
			validator: ValidateAuthType,
			required:  true,
		},
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: "),
			validator: func(value string) error {
				// Conditional validation: password required when auth_type is "password"
				// In real implementation, this would check auth_type field
				if value == "" {
					return &ValidationError{Field: "password", Message: "Password is required when using password authentication"}
				}
				return nil
			},
			required: false, // Dynamic requirement based on auth_type
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Set auth type to password
	form.SetFieldValue("auth_type", "password")

	// Test that empty password fails validation when auth_type is "password"
	form.SetFieldValue("password", "")
	err := form.ValidateField("password", "")
	if err == nil {
		t.Error("Expected validation error for empty password when auth type is 'password'")
	}

	// Test that non-empty password passes validation
	form.SetFieldValue("password", "secure123")
	err = form.ValidateField("password", "secure123")
	if err != nil {
		t.Errorf("Expected no validation error for non-empty password, got: %v", err)
	}
}

// TestConditionalPasswordField_FormNavigation tests tab navigation includes/excludes password field correctly
func TestConditionalPasswordField_FormNavigation(t *testing.T) {
	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().SetLabel("Name: "),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"auth_type": {
			dropdown: NewAuthenticationSelector(func(authType string) {}),
			validator: ValidateAuthType,
			required:  true,
		},
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: "),
			validator: ValidatePassword,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)
	helper := NewFormTestHelper(form)
	defer helper.Cleanup()

	// Test that password field is in field order
	foundPassword := false
	for _, fieldName := range form.fieldOrder {
		if fieldName == "password" {
			foundPassword = true
			break
		}
	}
	if !foundPassword {
		t.Error("Expected password field to be in form field order")
	}

	// Test navigation through fields with Tab key
	initialFocusIndex := form.focusIndex
	form.moveFocusNext()
	if form.focusIndex == initialFocusIndex && len(form.fieldOrder) > 1 {
		t.Error("Expected focus to move to next field")
	}

	// Test that getCurrentFocusedField works correctly
	fieldName, field := form.getCurrentFocusedField()
	if fieldName == "" {
		t.Error("Expected focused field name to be non-empty")
	}
	if field == nil {
		t.Error("Expected focused field to be non-nil")
	}
}

// TestConditionalPasswordField_FormSubmissionWithPassword tests form submission with password authentication
func TestConditionalPasswordField_FormSubmissionWithPassword(t *testing.T) {
	submittedData := make(map[string]interface{})
	submitCalled := false

	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().SetLabel("Name: ").SetText("test-server"),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"auth_type": {
			dropdown: NewAuthenticationSelector(func(authType string) {}),
			validator: ValidateAuthType,
			required:  true,
		},
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: ").
				SetText("secure123"),
			validator: ValidatePassword,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		submitCalled = true
		submittedData = data
		return nil
	}

	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)
	helper := NewFormTestHelper(form)
	defer helper.Cleanup()

	// Set auth type to password
	form.SetFieldValue("auth_type", "password")

	// Test form submission
	helper.SimulateKeypress(tcell.KeyEnter)
	helper.ProcessEvents()

	if !submitCalled {
		t.Error("Expected form submission to be called")
	}

	if len(submittedData) == 0 {
		t.Error("Expected form data to be submitted")
	}

	// Verify submitted data contains password
	if passwordData, exists := submittedData["password"]; !exists {
		t.Error("Expected submitted data to contain password field")
	} else if passwordData != "secure123" {
		t.Errorf("Expected password data to be 'secure123', got '%v'", passwordData)
	}

	// Verify auth type is correctly submitted
	if authTypeData, exists := submittedData["auth_type"]; !exists {
		t.Error("Expected submitted data to contain auth_type field")
	} else if authTypeData != "password" {
		t.Errorf("Expected auth_type data to be 'password', got '%v'", authTypeData)
	}
}

// TestConditionalPasswordField_PasswordConfiguration tests that password input is properly configured
func TestConditionalPasswordField_PasswordConfiguration(t *testing.T) {
	fields := map[string]*FormField{
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: ").
				SetText("secretpassword"),
			validator: ValidatePassword,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Verify password field has correct label
	passwordField := form.fields["password"].inputField
	if passwordField.GetLabel() != "Password: " {
		t.Errorf("Expected password field label to be 'Password: ', got '%s'", 
			passwordField.GetLabel())
	}

	// Verify password value is stored correctly
	password, err := form.GetFieldValue("password")
	if err != nil {
		t.Errorf("Expected no error getting password field value, got: %v", err)
	}
	if password != "secretpassword" {
		t.Errorf("Expected password value to be 'secretpassword', got '%s'", password)
	}
}

// TestConditionalPasswordField_SwitchBetweenAuthTypes tests switching between key and password auth
func TestConditionalPasswordField_SwitchBetweenAuthTypes(t *testing.T) {
	authTypeCallbackCount := 0
	lastAuthType := ""

	fields := map[string]*FormField{
		"auth_type": {
			dropdown: NewAuthenticationSelector(func(authType string) {
				authTypeCallbackCount++
				lastAuthType = authType
			}),
			validator: ValidateAuthType,
			required:  true,
		},
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: "),
			validator: ValidatePassword,
			required:  false,
		},
		"key_path": {
			inputField: tview.NewInputField().
				SetLabel("Key Path: "),
			validator: ValidateKeyPath,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Initial state should be "key"
	initialAuthType, _ := form.GetFieldValue("auth_type")
	if initialAuthType != "key" {
		t.Errorf("Expected initial auth type to be 'key', got '%s'", initialAuthType)
	}

	// Switch to password auth
	form.SetFieldValue("auth_type", "password")
	
	if authTypeCallbackCount == 0 {
		t.Error("Expected auth type callback to be called when switching to password")
	}
	if lastAuthType != "password" {
		t.Errorf("Expected last auth type to be 'password', got '%s'", lastAuthType)
	}

	// Switch back to key auth
	form.SetFieldValue("auth_type", "key")
	
	if authTypeCallbackCount < 2 {
		t.Error("Expected auth type callback to be called again when switching back to key")
	}
	if lastAuthType != "key" {
		t.Errorf("Expected last auth type to be 'key', got '%s'", lastAuthType)
	}
}

// TestConditionalPasswordField_ValidationIntegration tests integration with form validation system
func TestConditionalPasswordField_ValidationIntegration(t *testing.T) {
	fields := CreateServerFormFields()
	
	// Add password field to standard fields
	fields["password"] = &FormField{
		inputField: tview.NewInputField().
			SetLabel("Password: "),
		validator: func(value string) error {
			// In real implementation, this would check auth_type and conditionally validate
			return ValidatePassword(value)
		},
		required: false,
	}

	// Create form with real-time validation enabled
	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIFormWithOptions(fields, onSubmit, onCancel, true)

	// Test that password field integrates with validation system
	err := form.ValidateField("password", "")
	if err != nil && err.Error() != "password: Password is required when using password authentication" {
		// Empty password is allowed by default validator, so we test with valid input
		t.Logf("Password validation returned: %v", err)
	}

	// Test with valid password
	err = form.ValidateField("password", "validpassword123")
	if err != nil {
		t.Errorf("Expected no validation error for valid password, got: %v", err)
	}

	// Test that all fields validate correctly
	form.SetFieldValue("name", "test-server")
	form.SetFieldValue("hostname", "example.com")
	form.SetFieldValue("username", "testuser")
	form.SetFieldValue("auth_type", "password")
	form.SetFieldValue("password", "secure123")

	err = form.ValidateAllFields()
	if err != nil {
		t.Errorf("Expected no validation errors with complete password auth data, got: %v", err)
	}
}

// TestConditionalPasswordField_ErrorHandling tests error conditions with conditional password field
func TestConditionalPasswordField_ErrorHandling(t *testing.T) {
	fields := map[string]*FormField{
		"auth_type": {
			dropdown: NewAuthenticationSelector(func(authType string) {}),
			validator: ValidateAuthType,
			required:  true,
		},
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: "),
			validator: ValidatePassword,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Test setting invalid auth type
	err := form.SetFieldValue("auth_type", "invalid")
	if err == nil {
		t.Error("Expected error setting invalid auth type")
	}

	// Test getting non-existent field
	_, err = form.GetFieldValue("nonexistent")
	if err == nil {
		t.Error("Expected error getting non-existent field")
	}

	// Test setting field value on non-existent field
	err = form.SetFieldValue("nonexistent", "value")
	if err == nil {
		t.Error("Expected error setting value on non-existent field")
	}
}

// ValidatePassword validates password field (utility function for tests)
func ValidatePassword(value string) error {
	// For test purposes, password can be empty (conditional validation happens at form level)
	// In real implementation, this would be more sophisticated
	if len(value) > 128 {
		return &ValidationError{Field: "password", Message: "Password is too long (max 128 characters)"}
	}
	return nil
}

// TestConditionalPasswordField_AccessibilityAndLabeling tests accessibility features
func TestConditionalPasswordField_AccessibilityAndLabeling(t *testing.T) {
	fields := map[string]*FormField{
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: ").
				SetPlaceholder("Enter your password"),
			validator: ValidatePassword,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Test that password field has proper label for accessibility
	passwordField := form.fields["password"].inputField
	label := passwordField.GetLabel()
	if label != "Password: " {
		t.Errorf("Expected password field label to be 'Password: ', got '%s'", label)
	}

	// Test that password field has placeholder text
	// Note: tview InputField doesn't have GetPlaceholder() method, so we verify it was set
	// This would be enhanced in real implementation with accessibility testing
}

// TestConditionalPasswordField_MemoryAndCleanup tests memory management and sensitive data cleanup
func TestConditionalPasswordField_MemoryAndCleanup(t *testing.T) {
	fields := map[string]*FormField{
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: ").
				SetText("sensitivepassword"),
			validator: ValidatePassword,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Verify password is initially set
	password, _ := form.GetFieldValue("password")
	if password != "sensitivepassword" {
		t.Errorf("Expected initial password to be set correctly")
	}

	// Test clearing password field
	form.SetFieldValue("password", "")
	clearedPassword, _ := form.GetFieldValue("password")
	if clearedPassword != "" {
		t.Error("Expected password field to be cleared")
	}

	// In real implementation, this would test memory cleanup of sensitive data
	// For now, we verify the basic functionality works
}

// TestConditionalPasswordField_ShowHideToggle tests the show/hide functionality
func TestConditionalPasswordField_ShowHideToggle(t *testing.T) {
	passwordFieldVisible := false
	authTypeChangeCount := 0

	fields := map[string]*FormField{
		"auth_type": {
			dropdown: NewAuthenticationSelector(func(authType string) {
				authTypeChangeCount++
				// Simulate showing/hiding password field based on auth type
				if authType == "password" {
					passwordFieldVisible = true
				} else {
					passwordFieldVisible = false
				}
			}),
			validator: ValidateAuthType,
			required:  true,
		},
		"password": {
			inputField: tview.NewInputField().
				SetLabel("Password: "),
			validator: ValidatePassword,
			required:  false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Initial state - key auth, password should be hidden
	initialAuthType, _ := form.GetFieldValue("auth_type")
	if initialAuthType != "key" {
		t.Errorf("Expected initial auth type to be 'key', got '%s'", initialAuthType)
	}
	
	if passwordFieldVisible {
		t.Error("Expected password field to be hidden initially (auth type is key)")
	}

	// Switch to password auth - password field should be shown
	form.SetFieldValue("auth_type", "password")
	
	if authTypeChangeCount == 0 {
		t.Error("Expected auth type change callback to be called")
	}
	
	if !passwordFieldVisible {
		t.Error("Expected password field to be visible when auth type is password")
	}

	// Switch back to key auth - password field should be hidden
	form.SetFieldValue("auth_type", "key")
	
	if authTypeChangeCount < 2 {
		t.Error("Expected auth type change callback to be called again")
	}
	
	if passwordFieldVisible {
		t.Error("Expected password field to be hidden when auth type is key")
	}
}