package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ValidationError represents a form validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// FormField represents a single form field with validation
type FormField struct {
	inputField *tview.InputField
	validator  func(string) error
	required   bool
}

// TUIForm represents a modal form with validation and keyboard navigation
type TUIForm struct {
	form     *tview.Form
	fields   map[string]*FormField
	onSubmit func(map[string]interface{}) error
	onCancel func()
}

// NewTUIForm creates a new TUI form with the specified fields and callbacks
func NewTUIForm(fields map[string]*FormField, onSubmit func(map[string]interface{}) error, onCancel func()) *TUIForm {
	form := tview.NewForm()
	
	tuiForm := &TUIForm{
		form:     form,
		fields:   fields,
		onSubmit: onSubmit,
		onCancel: onCancel,
	}
	
	// Add fields to the form
	tuiForm.setupFormFields()
	
	// Setup keyboard navigation
	tuiForm.setupKeyboardNavigation()
	
	return tuiForm
}

// setupFormFields adds all fields to the tview.Form
func (tf *TUIForm) setupFormFields() {
	// Add input fields in a consistent order
	fieldNames := make([]string, 0, len(tf.fields))
	for name := range tf.fields {
		fieldNames = append(fieldNames, name)
	}
	
	// Add fields to form
	for _, name := range fieldNames {
		field := tf.fields[name]
		tf.form.AddFormItem(field.inputField)
	}
	
	// Add Submit and Cancel buttons
	tf.form.AddButton("Submit", func() {
		tf.handleSubmit()
	})
	
	tf.form.AddButton("Cancel", func() {
		tf.handleCancel()
	})
}

// setupKeyboardNavigation configures keyboard navigation for the form
func (tf *TUIForm) setupKeyboardNavigation() {
	tf.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// Enter key submits the form
			tf.handleSubmit()
			return nil
		case tcell.KeyEscape:
			// Escape key cancels the form
			tf.handleCancel()
			return nil
		case tcell.KeyTab:
			// Tab moves to next field
			return event // Let tview handle Tab navigation
		case tcell.KeyBacktab:
			// Shift+Tab moves to previous field
			return event // Let tview handle Shift+Tab navigation
		}
		return event
	})
}

// handleSubmit processes form submission
func (tf *TUIForm) handleSubmit() {
	// Collect and validate form data
	data, err := tf.CollectFormData()
	if err != nil {
		// Handle validation error
		// In a real implementation, this would show an error message in the form
		return
	}
	
	// Call the submission callback
	if tf.onSubmit != nil {
		if err := tf.onSubmit(data); err != nil {
			// Handle submission error
			// In a real implementation, this would show an error message
			return
		}
	}
}

// handleCancel processes form cancellation
func (tf *TUIForm) handleCancel() {
	if tf.onCancel != nil {
		tf.onCancel()
	}
}

// ValidateField validates a single field
func (tf *TUIForm) ValidateField(fieldName, value string) error {
	field, exists := tf.fields[fieldName]
	if !exists {
		return fmt.Errorf("field %s does not exist", fieldName)
	}
	
	// Run custom validator first if provided
	if field.validator != nil {
		return field.validator(value)
	}
	
	// Check required fields if no custom validator handled it
	if field.required && value == "" {
		return &ValidationError{
			Field:   fieldName,
			Message: "This field is required",
		}
	}
	
	return nil
}

// CollectFormData collects all form field data and validates it
func (tf *TUIForm) CollectFormData() (map[string]interface{}, error) {
	data := make(map[string]interface{})
	
	for fieldName, field := range tf.fields {
		value := field.inputField.GetText()
		
		// Validate field
		if err := tf.ValidateField(fieldName, value); err != nil {
			return nil, err
		}
		
		data[fieldName] = value
	}
	
	return data, nil
}

// GetForm returns the underlying tview.Form for display
func (tf *TUIForm) GetForm() *tview.Form {
	return tf.form
}

// SetFieldValue sets the value of a specific field
func (tf *TUIForm) SetFieldValue(fieldName, value string) error {
	field, exists := tf.fields[fieldName]
	if !exists {
		return fmt.Errorf("field %s does not exist", fieldName)
	}
	
	field.inputField.SetText(value)
	return nil
}

// GetFieldValue gets the current value of a specific field
func (tf *TUIForm) GetFieldValue(fieldName string) (string, error) {
	field, exists := tf.fields[fieldName]
	if !exists {
		return "", fmt.Errorf("field %s does not exist", fieldName)
	}
	
	return field.inputField.GetText(), nil
}

// ModalManager manages modal display and keyboard routing
type ModalManager struct {
	app        *tview.Application
	layout     *tview.Flex
	modalStack []tview.Primitive
}

// NewModalManager creates a new modal manager
func NewModalManager(app *tview.Application, layout *tview.Flex) *ModalManager {
	return &ModalManager{
		app:        app,
		layout:     layout,
		modalStack: make([]tview.Primitive, 0),
	}
}

// ShowModal displays a modal on top of the current interface
func (mm *ModalManager) ShowModal(modal tview.Primitive) {
	mm.modalStack = append(mm.modalStack, modal)
	mm.app.SetRoot(modal, true)
	mm.app.SetFocus(modal)
}

// HideModal hides the current modal and returns to the previous one or main interface
func (mm *ModalManager) HideModal() {
	if len(mm.modalStack) == 0 {
		return
	}
	
	// Remove the current modal
	mm.modalStack = mm.modalStack[:len(mm.modalStack)-1]
	
	if len(mm.modalStack) > 0 {
		// Show the previous modal
		previousModal := mm.modalStack[len(mm.modalStack)-1]
		mm.app.SetRoot(previousModal, true)
		mm.app.SetFocus(previousModal)
	} else {
		// Return to main layout
		mm.app.SetRoot(mm.layout, true)
		mm.app.SetFocus(mm.layout)
	}
}

// IsModalActive returns whether any modal is currently active
func (mm *ModalManager) IsModalActive() bool {
	return len(mm.modalStack) > 0
}

// GetCurrentModal returns the currently active modal, or nil if none
func (mm *ModalManager) GetCurrentModal() tview.Primitive {
	if len(mm.modalStack) == 0 {
		return nil
	}
	return mm.modalStack[len(mm.modalStack)-1]
}

// ClearAllModals closes all modals and returns to the main interface
func (mm *ModalManager) ClearAllModals() {
	mm.modalStack = make([]tview.Primitive, 0)
	mm.app.SetRoot(mm.layout, true)
	mm.app.SetFocus(mm.layout)
}