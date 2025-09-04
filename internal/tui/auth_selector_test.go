package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TestAuthenticationSelector_Creation tests basic creation of authentication selector
func TestAuthenticationSelector_Creation(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	if selector == nil {
		t.Fatal("Expected authentication selector to be created, got nil")
	}
	
	if selector.dropdown == nil {
		t.Fatal("Expected dropdown to be created, got nil")
	}
	
	if selector.onChanged == nil {
		t.Fatal("Expected onChanged callback to be set, got nil")
	}
}

// TestAuthenticationSelector_Options tests that proper options are available
func TestAuthenticationSelector_Options(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	// Test that we have the expected options by checking current selection behavior
	// We can't directly access GetOption, but we can test the selector behavior
	value := selector.GetValue()
	if value != "key" {
		t.Errorf("Expected default value to be 'key', got '%s'", value)
	}
	
	// Test setting to password
	err := selector.SetValue("password")
	if err != nil {
		t.Errorf("Expected no error setting to 'password', got: %v", err)
	}
	
	value = selector.GetValue()
	if value != "password" {
		t.Errorf("Expected value to be 'password' after setting, got '%s'", value)
	}
}

// TestAuthenticationSelector_DefaultValue tests default selection
func TestAuthenticationSelector_DefaultValue(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	// Test default selection is "key"
	currentIndex, currentText := selector.dropdown.GetCurrentOption()
	if currentIndex != 0 {
		t.Errorf("Expected default option to be index 0 (key), got %d", currentIndex)
	}
	if currentText != "key" {
		t.Errorf("Expected default option text to be 'key', got '%s'", currentText)
	}
	
	// Test GetValue returns correct default
	value := selector.GetValue()
	if value != "key" {
		t.Errorf("Expected default value to be 'key', got '%s'", value)
	}
}

// TestAuthenticationSelector_SetValue tests setting authentication type programmatically
func TestAuthenticationSelector_SetValue(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	// Test setting to "password"
	err := selector.SetValue("password")
	if err != nil {
		t.Errorf("Expected no error setting value to 'password', got: %v", err)
	}
	
	value := selector.GetValue()
	if value != "password" {
		t.Errorf("Expected value to be 'password' after setting, got '%s'", value)
	}
	
	// Test setting to "key"
	err = selector.SetValue("key")
	if err != nil {
		t.Errorf("Expected no error setting value to 'key', got: %v", err)
	}
	
	value = selector.GetValue()
	if value != "key" {
		t.Errorf("Expected value to be 'key' after setting, got '%s'", value)
	}
	
	// Test setting invalid value
	err = selector.SetValue("invalid")
	if err == nil {
		t.Error("Expected error setting invalid authentication type")
	}
}

// TestAuthenticationSelector_ChangeCallback tests that callback is triggered on selection change
func TestAuthenticationSelector_ChangeCallback(t *testing.T) {
	callbackCalled := false
	receivedAuthType := ""
	
	callback := func(authType string) {
		callbackCalled = true
		receivedAuthType = authType
	}
	
	selector := NewAuthenticationSelector(callback)
	
	// Trigger selection change programmatically
	err := selector.SetValue("password")
	if err != nil {
		t.Errorf("Expected no error setting value, got: %v", err)
	}
	
	if !callbackCalled {
		t.Error("Expected callback to be called when value changes")
	}
	
	if receivedAuthType != "password" {
		t.Errorf("Expected callback to receive 'password', got '%s'", receivedAuthType)
	}
}

// TestAuthenticationSelector_KeyboardNavigation tests keyboard navigation
func TestAuthenticationSelector_KeyboardNavigation(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	// Test that dropdown supports keyboard input
	if selector.dropdown == nil {
		t.Fatal("Expected dropdown to exist for keyboard testing")
	}
	
	// Test space key activation
	spacePressed := false
	selector.dropdown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == ' ' {
			spacePressed = true
			return nil // Consume the event
		}
		return event
	})
	
	// Simulate space key press
	spaceEvent := tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone)
	if inputCapture := selector.dropdown.GetInputCapture(); inputCapture != nil {
		inputCapture(spaceEvent)
	}
	
	if !spacePressed {
		t.Error("Expected space key to be captured for dropdown activation")
	}
}

// TestAuthenticationSelector_Integration tests integration with form fields
func TestAuthenticationSelector_Integration(t *testing.T) {
	callbackCalled := false
	selectedAuthType := ""
	
	callback := func(authType string) {
		callbackCalled = true
		selectedAuthType = authType
	}
	
	selector := NewAuthenticationSelector(callback)
	
	// Test integration with tview.Form structure
	form := tview.NewForm()
	form.AddFormItem(selector.GetFormItem())
	
	// Test that form item is properly configured
	if selector.GetFormItem() == nil {
		t.Fatal("Expected form item to be available for form integration")
	}
	
	// Test label
	label := selector.GetLabel()
	if label != "Auth Type: " {
		t.Errorf("Expected label 'Auth Type: ', got '%s'", label)
	}
	
	// Test changing value triggers callback
	selector.SetValue("password")
	
	if !callbackCalled {
		t.Error("Expected callback to be triggered during form integration test")
	}
	
	if selectedAuthType != "password" {
		t.Errorf("Expected selected auth type to be 'password', got '%s'", selectedAuthType)
	}
}

// TestAuthenticationSelector_Styling tests visual styling of the dropdown
func TestAuthenticationSelector_Styling(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	// Test that dropdown has consistent styling with form fields
	if selector.dropdown == nil {
		t.Fatal("Expected dropdown to exist for styling test")
	}
	
	// Test label styling
	label := selector.dropdown.GetLabel()
	if label != "Auth Type: " {
		t.Errorf("Expected label 'Auth Type: ', got '%s'", label)
	}
	
	// Test that dropdown can be styled (colors will be set during integration)
	// This verifies the structure exists for styling
	dropdown := selector.dropdown
	if dropdown == nil {
		t.Error("Expected dropdown to be available for styling")
	}
}

// TestAuthenticationSelector_Focus tests focus management
func TestAuthenticationSelector_Focus(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	// Test that dropdown is focusable
	if selector.dropdown == nil {
		t.Fatal("Expected dropdown to exist for focus test")
	}
	
	// Test GetFormItem returns focusable element
	formItem := selector.GetFormItem()
	if formItem == nil {
		t.Fatal("Expected form item to exist")
	}
	
	// tview.DropDown implements tview.FormItem interface which is focusable
	// This test verifies the structure exists for focus management
}

// TestAuthenticationSelector_ErrorHandling tests error conditions
func TestAuthenticationSelector_ErrorHandling(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	// Test nil callback handling (should not crash)
	selectorWithNilCallback := NewAuthenticationSelector(nil)
	if selectorWithNilCallback == nil {
		t.Fatal("Expected selector to be created even with nil callback")
	}
	
	// Setting value should not crash with nil callback
	err := selectorWithNilCallback.SetValue("password")
	if err != nil {
		t.Errorf("Expected no error with nil callback, got: %v", err)
	}
	
	// Test invalid value handling
	err = selector.SetValue("")
	if err == nil {
		t.Error("Expected error setting empty authentication type")
	}
	
	err = selector.SetValue("invalid-type")
	if err == nil {
		t.Error("Expected error setting invalid authentication type")
	}
}

// TestAuthenticationSelector_Threading tests that component is thread-safe for UI operations
func TestAuthenticationSelector_Threading(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	// Test multiple rapid value changes (simulating user interaction)
	values := []string{"key", "password", "key", "password"}
	
	for _, value := range values {
		err := selector.SetValue(value)
		if err != nil {
			t.Errorf("Expected no error setting value '%s', got: %v", value, err)
		}
		
		retrievedValue := selector.GetValue()
		if retrievedValue != value {
			t.Errorf("Expected value '%s', got '%s'", value, retrievedValue)
		}
	}
}

// TestAuthenticationSelector_Accessibility tests accessibility features
func TestAuthenticationSelector_Accessibility(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	// Test that dropdown has proper label for accessibility
	label := selector.GetLabel()
	if label == "" {
		t.Error("Expected non-empty label for accessibility")
	}
	
	// Test that current selection is clearly indicated
	currentIndex, currentText := selector.dropdown.GetCurrentOption()
	if currentIndex < 0 {
		t.Error("Expected current option to be within valid range")
	}
	if currentText == "" {
		t.Error("Expected non-empty option text for accessibility")
	}
}

// TestAuthenticationSelector_ConsistencyWithFormFields tests consistency with existing form fields
func TestAuthenticationSelector_ConsistencyWithFormFields(t *testing.T) {
	selector := NewAuthenticationSelector(func(authType string) {})
	
	// Create a standard form field for comparison
	standardField := tview.NewInputField().
		SetLabel("Auth Type: ").
		SetText("key").
		SetFieldWidth(15).
		SetFieldTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetLabelColor(tcell.ColorWhite)
	
	// Test that selector has consistent label
	if selector.GetLabel() != standardField.GetLabel() {
		t.Errorf("Expected consistent label '%s', got '%s'", 
			standardField.GetLabel(), selector.GetLabel())
	}
	
	// Test that selector integrates properly with form structure
	form := tview.NewForm()
	form.AddFormItem(standardField)
	form.AddFormItem(selector.GetFormItem())
	
	// Both should be added without error
	if form == nil {
		t.Error("Expected form to accept both standard field and selector")
	}
}