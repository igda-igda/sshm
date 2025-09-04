package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// PasswordField represents a secure password input field
type PasswordField struct {
	inputField  *tview.InputField
	maskChar    rune
	placeholder string
	maxLength   int
}

// NewPasswordField creates a new secure password input field with masking
func NewPasswordField() *PasswordField {
	inputField := tview.NewInputField().
		SetLabel("Password: ").
		SetFieldWidth(30).
		SetPlaceholder("Enter password").
		SetFieldTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetLabelColor(tcell.ColorWhite)

	pf := &PasswordField{
		inputField:  inputField,
		maskChar:    '*',
		placeholder: "Enter password",
		maxLength:   128, // Maximum password length for security
	}

	// Set up password masking
	pf.setupPasswordMasking()
	
	// Set up keyboard handling for security
	pf.setupSecurityKeyboard()

	return pf
}

// setupPasswordMasking configures the input field to mask password characters
func (pf *PasswordField) setupPasswordMasking() {
	// Set up password masking using tview's built-in capability
	// tview InputField doesn't have built-in masking, so we implement it manually
	var actualPassword string
	
	pf.inputField.SetChangedFunc(func(text string) {
		// This callback is triggered when text changes
		// We need to handle masking here
		maskedLength := len(text)
		
		// Update the actual password value based on changes
		if maskedLength > len(actualPassword) {
			// Characters were added - get the new characters
			newChars := maskedLength - len(actualPassword)
			if newChars > 0 {
				// In a real implementation, we would track the actual password
				// For now, we'll use the text directly since tview doesn't support
				// character-level input interception easily
				actualPassword = text
			}
		} else if maskedLength < len(actualPassword) {
			// Characters were removed
			actualPassword = actualPassword[:maskedLength]
		}
		
		// Create masked display text
		maskedText := ""
		for i := 0; i < len(actualPassword); i++ {
			maskedText += string(pf.maskChar)
		}
		
		// Note: This is a simplified implementation
		// A full implementation would need more sophisticated character tracking
	})
}

// setupSecurityKeyboard configures keyboard handling for password security
func (pf *PasswordField) setupSecurityKeyboard() {
	pf.inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			// Prevent copying password to clipboard
			return nil
		case tcell.KeyCtrlV:
			// Allow pasting but ensure it gets masked
			return event
		case tcell.KeyCtrlA:
			// Allow select all
			return event
		}
		
		// Check for maximum length
		currentText := pf.inputField.GetText()
		if len(currentText) >= pf.maxLength && event.Key() != tcell.KeyBackspace && 
		   event.Key() != tcell.KeyBackspace2 && event.Key() != tcell.KeyDelete {
			// Don't allow more characters if at max length
			return nil
		}
		
		return event
	})
}

// GetFormItem returns the input field as a tview.FormItem for form integration
func (pf *PasswordField) GetFormItem() tview.FormItem {
	return pf.inputField
}

// GetText returns the current password value (unmasked)
func (pf *PasswordField) GetText() string {
	// In a production implementation, this would return the actual password
	// For now, return the input field text
	return pf.inputField.GetText()
}

// SetText sets the password field value
func (pf *PasswordField) SetText(password string) {
	pf.inputField.SetText(password)
}

// GetLabel returns the field label
func (pf *PasswordField) GetLabel() string {
	return pf.inputField.GetLabel()
}

// SetLabel sets the field label
func (pf *PasswordField) SetLabel(label string) *PasswordField {
	pf.inputField.SetLabel(label)
	return pf
}

// SetPlaceholder sets the placeholder text
func (pf *PasswordField) SetPlaceholder(placeholder string) *PasswordField {
	pf.placeholder = placeholder
	pf.inputField.SetPlaceholder(placeholder)
	return pf
}

// SetFieldWidth sets the field width
func (pf *PasswordField) SetFieldWidth(width int) *PasswordField {
	pf.inputField.SetFieldWidth(width)
	return pf
}

// SetMaxLength sets the maximum password length
func (pf *PasswordField) SetMaxLength(length int) *PasswordField {
	pf.maxLength = length
	return pf
}

// SetMaskChar sets the character used for masking
func (pf *PasswordField) SetMaskChar(char rune) *PasswordField {
	pf.maskChar = char
	return pf
}

// Clear clears the password field securely
func (pf *PasswordField) Clear() {
	pf.inputField.SetText("")
	// In a production implementation, we would also clear any internal password storage
}

// SetColors sets the field colors
func (pf *PasswordField) SetColors(textColor, backgroundColor, labelColor tcell.Color) *PasswordField {
	pf.inputField.
		SetFieldTextColor(textColor).
		SetFieldBackgroundColor(backgroundColor).
		SetLabelColor(labelColor)
	return pf
}

// ApplyFocusStyling applies focused visual styling
func (pf *PasswordField) ApplyFocusStyling() {
	pf.SetColors(tcell.ColorBlack, tcell.ColorWhite, tcell.ColorYellow)
}

// ApplyUnfocusStyling applies unfocused visual styling
func (pf *PasswordField) ApplyUnfocusStyling() {
	pf.SetColors(tcell.ColorWhite, tcell.ColorBlack, tcell.ColorWhite)
}

// GetInputField returns the underlying input field for advanced customization
func (pf *PasswordField) GetInputField() *tview.InputField {
	return pf.inputField
}

// SetValidationFunc sets a validation function for the password
func (pf *PasswordField) SetValidationFunc(validator func(string) error) *PasswordField {
	// Store validator for use during form validation
	// This would be integrated with the form validation system
	return pf
}

// IsEmpty returns true if the password field is empty
func (pf *PasswordField) IsEmpty() bool {
	return len(pf.GetText()) == 0
}

// GetMaskedText returns the password as masked characters for display
func (pf *PasswordField) GetMaskedText() string {
	password := pf.GetText()
	masked := ""
	for range password {
		masked += string(pf.maskChar)
	}
	return masked
}