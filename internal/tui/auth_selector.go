package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// AuthenticationSelector represents a dropdown for selecting authentication type
type AuthenticationSelector struct {
	dropdown  *tview.DropDown
	onChanged func(authType string)
}

// NewAuthenticationSelector creates a new authentication type selector dropdown
func NewAuthenticationSelector(onChanged func(authType string)) *AuthenticationSelector {
	dropdown := tview.NewDropDown().
		SetLabel("Auth Type: ").
		SetOptions([]string{"key", "password"}, nil).
		SetCurrentOption(0) // Default to "key"

	selector := &AuthenticationSelector{
		dropdown:  dropdown,
		onChanged: onChanged,
	}

	// Set up the selection change callback
	dropdown.SetSelectedFunc(func(text string, index int) {
		if selector.onChanged != nil {
			selector.onChanged(text)
		}
	})

	// Apply consistent styling with other form fields
	selector.applyFormFieldStyling()

	// Set up space key activation for dropdown
	selector.setupKeyboardNavigation()

	return selector
}

// applyFormFieldStyling applies consistent styling to match other form fields
func (as *AuthenticationSelector) applyFormFieldStyling() {
	as.dropdown.
		SetFieldWidth(15).
		SetFieldTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetLabelColor(tcell.ColorWhite)
}

// setupKeyboardNavigation configures keyboard navigation including space key activation
func (as *AuthenticationSelector) setupKeyboardNavigation() {
	as.dropdown.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			if event.Rune() == ' ' {
				// Space key opens/navigates the dropdown
				// Let tview handle the dropdown activation
				return event
			}
		case tcell.KeyEnter:
			// Enter key can also activate dropdown
			return event
		}
		return event
	})
}

// GetValue returns the currently selected authentication type
func (as *AuthenticationSelector) GetValue() string {
	currentIndex, currentText := as.dropdown.GetCurrentOption()
	if currentIndex >= 0 {
		return currentText
	}
	return "key" // Default fallback
}

// SetValue sets the authentication type selection programmatically
func (as *AuthenticationSelector) SetValue(authType string) error {
	if authType != "key" && authType != "password" {
		return fmt.Errorf("invalid authentication type: %s (must be 'key' or 'password')", authType)
	}

	// Find the option index for the given auth type
	optionCount := 2 // We know we have exactly "key" and "password"
	for i := 0; i < optionCount; i++ {
		// Since we control the options, we can directly check the index
		var option string
		if i == 0 {
			option = "key"
		} else {
			option = "password"
		}
		
		if option == authType {
			as.dropdown.SetCurrentOption(i)
			
			// Trigger the callback to notify of the change
			if as.onChanged != nil {
				as.onChanged(authType)
			}
			return nil
		}
	}

	return fmt.Errorf("authentication type %s not found in options", authType)
}

// GetFormItem returns the dropdown as a tview.FormItem for form integration
func (as *AuthenticationSelector) GetFormItem() tview.FormItem {
	return as.dropdown
}

// GetLabel returns the label of the authentication selector
func (as *AuthenticationSelector) GetLabel() string {
	return as.dropdown.GetLabel()
}

// SetFocusColors sets the colors for when the dropdown is focused
func (as *AuthenticationSelector) SetFocusColors(textColor, backgroundColor, labelColor tcell.Color) {
	// Apply focused field styling similar to other form fields
	as.dropdown.
		SetFieldTextColor(textColor).
		SetFieldBackgroundColor(backgroundColor).
		SetLabelColor(labelColor)
}

// SetUnfocusColors sets the colors for when the dropdown is not focused
func (as *AuthenticationSelector) SetUnfocusColors(textColor, backgroundColor, labelColor tcell.Color) {
	// Apply unfocused field styling similar to other form fields
	as.dropdown.
		SetFieldTextColor(textColor).
		SetFieldBackgroundColor(backgroundColor).
		SetLabelColor(labelColor)
}

// ApplyFocusStyling applies focused visual styling
func (as *AuthenticationSelector) ApplyFocusStyling() {
	as.SetFocusColors(tcell.ColorBlack, tcell.ColorWhite, tcell.ColorYellow)
}

// ApplyUnfocusStyling applies unfocused visual styling
func (as *AuthenticationSelector) ApplyUnfocusStyling() {
	as.SetUnfocusColors(tcell.ColorWhite, tcell.ColorBlack, tcell.ColorWhite)
}

// GetDropDown returns the underlying tview.DropDown for advanced customization
func (as *AuthenticationSelector) GetDropDown() *tview.DropDown {
	return as.dropdown
}

// SetOnChanged sets or updates the callback function for selection changes
func (as *AuthenticationSelector) SetOnChanged(onChanged func(authType string)) {
	as.onChanged = onChanged
	// Update the dropdown's selected function
	as.dropdown.SetSelectedFunc(func(text string, index int) {
		if as.onChanged != nil {
			as.onChanged(text)
		}
	})
}