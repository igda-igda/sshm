package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestPasswordFieldNavigation tests password field navigation behavior
func TestPasswordFieldNavigation(t *testing.T) {
	fields := CreateServerFormFields()
	
	// Create a tview form like the modal system does
	form := tview.NewForm()
	
	// Add fields in the same order as setupFormFields
	preferredOrder := []string{"name", "hostname", "port", "username", "auth_type", "password", "key_path", "passphrase_protected"}
	
	var addedFields []string
	for _, fieldName := range preferredOrder {
		if field, exists := fields[fieldName]; exists {
			form.AddFormItem(field.GetFormItem())
			addedFields = append(addedFields, fieldName)
		}
	}
	
	// Verify password field is in the expected position
	expectedIndex := -1
	for i, name := range addedFields {
		if name == "password" {
			expectedIndex = i
			break
		}
	}
	
	if expectedIndex == -1 {
		t.Fatal("Password field not found in added fields")
	}
	
	// Verify password field comes after auth_type
	authTypeIndex := -1
	for i, name := range addedFields {
		if name == "auth_type" {
			authTypeIndex = i
			break
		}
	}
	
	if authTypeIndex == -1 {
		t.Fatal("Auth type field not found in added fields")
	}
	
	if expectedIndex <= authTypeIndex {
		t.Errorf("Password field (index %d) should come after auth_type field (index %d)", expectedIndex, authTypeIndex)
	}
}

// TestPasswordFieldFocusStates tests password field focus behavior
func TestPasswordFieldFocusStates(t *testing.T) {
	fields := CreateServerFormFields()
	passwordField := fields["password"]
	
	if passwordField.passwordField == nil {
		t.Fatal("Password field should have PasswordField component")
	}
	
	// Test focus styling application
	passwordField.passwordField.ApplyFocusStyling()
	
	// Verify the field can receive focus
	formItem := passwordField.passwordField.GetFormItem()
	if formItem == nil {
		t.Fatal("Password field should return valid form item for focus")
	}
	
	// Test unfocus styling
	passwordField.passwordField.ApplyUnfocusStyling()
	
	// Test that password field responds to basic events
	inputField := passwordField.passwordField.GetInputField()
	if inputField == nil {
		t.Fatal("Password field should have underlying InputField")
	}
}

// TestPasswordFieldConditionalNavigation tests navigation with conditional password field
func TestPasswordFieldConditionalNavigation(t *testing.T) {
	fields := CreateServerFormFields()
	authTypeField := fields["auth_type"]
	passwordField := fields["password"]
	
	// Initially password field should not be required (key auth is default)
	if passwordField.required {
		t.Error("Password field should not be required initially")
	}
	
	// Switch to password authentication
	authTypeField.dropdown.SetValue("password")
	
	// Password field should now be required
	if !passwordField.required {
		t.Error("Password field should be required when auth type is password")
	}
	
	// Switch back to key authentication
	authTypeField.dropdown.SetValue("key")
	
	// Password field should no longer be required
	if passwordField.required {
		t.Error("Password field should not be required when auth type is key")
	}
	
	// Password field should be cleared for security
	if passwordField.passwordField.GetText() != "" {
		t.Error("Password field should be cleared when switching to key auth")
	}
}

// TestPasswordFieldKeyboardHandling tests keyboard navigation integration
func TestPasswordFieldKeyboardHandling(t *testing.T) {
	fields := CreateServerFormFields()
	passwordField := fields["password"]
	
	inputField := passwordField.passwordField.GetInputField()
	
	// Test tab key handling (should pass through to tview)
	event := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	inputField.InputHandler()(event, func(p tview.Primitive) {})
	
	// Should not crash and field should still be functional after tab key
	
	// Test escape key handling (depends on implementation)
	escapeEvent := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	inputField.InputHandler()(escapeEvent, func(p tview.Primitive) {})
	
	// Should not crash and field should still be functional
	passwordField.passwordField.SetText("test")
	if passwordField.passwordField.GetText() != "test" {
		t.Error("Password field should remain functional after keyboard events")
	}
}

// TestPasswordFieldAccessibility tests accessibility features
func TestPasswordFieldAccessibility(t *testing.T) {
	fields := CreateServerFormFields()
	passwordField := fields["password"]
	
	// Test label accessibility
	label := passwordField.passwordField.GetLabel()
	if label == "" {
		t.Error("Password field should have accessible label")
	}
	
	// Test that field supports standard form operations
	passwordField.passwordField.SetText("accessible-test")
	if passwordField.passwordField.GetText() != "accessible-test" {
		t.Error("Password field should support standard text operations")
	}
	
	// Test masking for accessibility tools
	maskedText := passwordField.passwordField.GetMaskedText()
	if maskedText == "accessible-test" {
		t.Error("Password field should provide masked text for accessibility")
	}
	
	// Test clearing for accessibility
	passwordField.passwordField.Clear()
	if passwordField.passwordField.GetText() != "" {
		t.Error("Password field should support clear operation")
	}
}

// TestPasswordFieldFormIntegration tests integration with TUIForm system
func TestPasswordFieldFormIntegration(t *testing.T) {
	fields := CreateServerFormFields()
	
	// Create TUIForm like the modal system does
	form := NewTUIForm(fields, func(data map[string]interface{}) error {
		// Mock submit handler
		return nil
	}, func() {
		// Mock cancel handler
	})
	
	if form == nil {
		t.Fatal("Should be able to create TUIForm with password field")
	}
	
	// Test that password field is included in form
	formPrimitive := form.GetForm()
	if formPrimitive == nil {
		t.Fatal("Form should be created successfully with password field")
	}
	
	// Test field order includes password
	if len(form.fieldOrder) == 0 {
		t.Fatal("Form should have field order including password")
	}
	
	passwordFieldFound := false
	for _, fieldName := range form.fieldOrder {
		if fieldName == "password" {
			passwordFieldFound = true
			break
		}
	}
	
	if !passwordFieldFound {
		t.Error("Password field should be included in form field order")
	}
}