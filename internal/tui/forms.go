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
	form             *tview.Form
	fields           map[string]*FormField
	fieldOrder       []string                             // Maintains field order
	onSubmit         func(map[string]interface{}) error
	onCancel         func()
	realTimeValidate bool                                 // Enable real-time validation
	errorDisplay     *tview.TextView                      // Error display area
	validationErrors map[string]string                    // Current field errors
}

// NewTUIForm creates a new TUI form with the specified fields and callbacks
func NewTUIForm(fields map[string]*FormField, onSubmit func(map[string]interface{}) error, onCancel func()) *TUIForm {
	return NewTUIFormWithOptions(fields, onSubmit, onCancel, false)
}

// NewTUIFormWithOptions creates a new TUI form with enhanced options
func NewTUIFormWithOptions(fields map[string]*FormField, onSubmit func(map[string]interface{}) error, onCancel func(), realTimeValidate bool) *TUIForm {
	form := tview.NewForm()
	
	// Create error display area
	errorDisplay := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetMaxLines(3)
	
	tuiForm := &TUIForm{
		form:             form,
		fields:           fields,
		fieldOrder:       make([]string, 0, len(fields)),
		onSubmit:         onSubmit,
		onCancel:         onCancel,
		realTimeValidate: realTimeValidate,
		errorDisplay:     errorDisplay,
		validationErrors: make(map[string]string),
	}
	
	// Add fields to the form
	tuiForm.setupFormFields()
	
	// Setup keyboard navigation
	tuiForm.setupKeyboardNavigation()
	
	// Setup real-time validation if enabled
	if realTimeValidate {
		tuiForm.setupRealTimeValidation()
	}
	
	return tuiForm
}

// setupFormFields adds all fields to the tview.Form
func (tf *TUIForm) setupFormFields() {
	// Define a preferred field order for server forms
	preferredOrder := []string{"name", "hostname", "port", "username", "auth_type", "key_path", "passphrase_protected"}
	
	// Build field order: preferred fields first, then remaining fields
	usedFields := make(map[string]bool)
	
	// Add preferred fields in order if they exist
	for _, fieldName := range preferredOrder {
		if field, exists := tf.fields[fieldName]; exists {
			tf.fieldOrder = append(tf.fieldOrder, fieldName)
			tf.form.AddFormItem(field.inputField)
			usedFields[fieldName] = true
		}
	}
	
	// Add any remaining fields
	for fieldName, field := range tf.fields {
		if !usedFields[fieldName] {
			tf.fieldOrder = append(tf.fieldOrder, fieldName)
			tf.form.AddFormItem(field.inputField)
		}
	}
	
	// Add error display if real-time validation is enabled
	if tf.realTimeValidate {
		tf.form.AddTextView("Errors:", "", 0, 3, true, false)
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

// setupRealTimeValidation configures real-time validation for form fields
func (tf *TUIForm) setupRealTimeValidation() {
	for fieldName, field := range tf.fields {
		// Capture field name in closure
		currentFieldName := fieldName
		currentField := field
		
		// Set up changed function for real-time validation
		currentField.inputField.SetChangedFunc(func(text string) {
			// Perform validation
			tf.validateFieldRealTime(currentFieldName, text)
		})
	}
}

// validateFieldRealTime performs real-time validation and updates error display
func (tf *TUIForm) validateFieldRealTime(fieldName, value string) {
	err := tf.ValidateField(fieldName, value)
	
	if err != nil {
		tf.validationErrors[fieldName] = err.Error()
	} else {
		delete(tf.validationErrors, fieldName)
	}
	
	tf.updateErrorDisplay()
}

// updateErrorDisplay updates the error display area with current validation errors
func (tf *TUIForm) updateErrorDisplay() {
	if !tf.realTimeValidate {
		return
	}
	
	errorText := ""
	if len(tf.validationErrors) > 0 {
		errorText = "[red]Validation Errors:[white]\n"
		for fieldName, errorMsg := range tf.validationErrors {
			errorText += fmt.Sprintf("• %s: %s\n", fieldName, errorMsg)
		}
	} else {
		errorText = "[green]All fields valid[white]"
	}
	
	tf.errorDisplay.SetText(errorText)
}

// HasValidationErrors returns true if there are any current validation errors
func (tf *TUIForm) HasValidationErrors() bool {
	return len(tf.validationErrors) > 0
}

// GetValidationErrors returns the current validation errors
func (tf *TUIForm) GetValidationErrors() map[string]string {
	errorsCopy := make(map[string]string)
	for k, v := range tf.validationErrors {
		errorsCopy[k] = v
	}
	return errorsCopy
}

// GetErrorDisplay returns the error display TextView
func (tf *TUIForm) GetErrorDisplay() *tview.TextView {
	return tf.errorDisplay
}

// ValidateAllFields validates all fields and updates error display
func (tf *TUIForm) ValidateAllFields() error {
	tf.validationErrors = make(map[string]string) // Clear existing errors
	
	for fieldName, field := range tf.fields {
		value := field.inputField.GetText()
		if err := tf.ValidateField(fieldName, value); err != nil {
			tf.validationErrors[fieldName] = err.Error()
		}
	}
	
	tf.updateErrorDisplay()
	
	if len(tf.validationErrors) > 0 {
		// Return the first validation error encountered
		for fieldName, errorMsg := range tf.validationErrors {
			return &ValidationError{Field: fieldName, Message: errorMsg}
		}
	}
	
	return nil
}

// CreateServerFormFields creates the standard server form fields with enhanced validation
func CreateServerFormFields() map[string]*FormField {
	return map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().
				SetLabel("Server Name: ").
				SetFieldWidth(30).
				SetPlaceholder("e.g., production-api"),
			validator: ValidateServerName,
			required:  true,
		},
		"hostname": {
			inputField: tview.NewInputField().
				SetLabel("Hostname: ").
				SetFieldWidth(40).
				SetPlaceholder("e.g., example.com or 192.168.1.100"),
			validator: ValidateHostname,
			required:  true,
		},
		"port": {
			inputField: tview.NewInputField().
				SetLabel("Port: ").
				SetText("22").
				SetFieldWidth(10).
				SetPlaceholder("1-65535"),
			validator: ValidatePort,
			required:  true,
		},
		"username": {
			inputField: tview.NewInputField().
				SetLabel("Username: ").
				SetFieldWidth(25).
				SetPlaceholder("e.g., ubuntu, admin, root"),
			validator: ValidateUsername,
			required:  true,
		},
		"auth_type": {
			inputField: tview.NewInputField().
				SetLabel("Auth Type: ").
				SetText("key").
				SetFieldWidth(15).
				SetPlaceholder("key or password"),
			validator: ValidateAuthType,
			required:  true,
		},
		"key_path": {
			inputField: tview.NewInputField().
				SetLabel("Key Path (optional): ").
				SetFieldWidth(50).
				SetPlaceholder("e.g., ~/.ssh/id_rsa"),
			validator: ValidateKeyPath,
			required:  false,
		},
		"passphrase_protected": {
			inputField: tview.NewInputField().
				SetLabel("Passphrase Protected: ").
				SetText("false").
				SetFieldWidth(10).
				SetPlaceholder("true or false"),
			validator: ValidatePassphraseProtected,
			required:  false,
		},
	}
}

// Enhanced validation functions

// ValidateServerName validates server name field
func ValidateServerName(value string) error {
	if value == "" {
		return &ValidationError{Field: "name", Message: "Server name is required"}
	}
	if len(value) < 2 {
		return &ValidationError{Field: "name", Message: "Server name must be at least 2 characters"}
	}
	if len(value) > 50 {
		return &ValidationError{Field: "name", Message: "Server name must be less than 50 characters"}
	}
	// Check for valid characters (alphanumeric, dash, underscore)
	for _, r := range value {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return &ValidationError{Field: "name", Message: "Server name can only contain letters, numbers, dashes, and underscores"}
		}
	}
	return nil
}

// ValidateHostname validates hostname field
func ValidateHostname(value string) error {
	if value == "" {
		return &ValidationError{Field: "hostname", Message: "Hostname is required"}
	}
	if len(value) > 253 {
		return &ValidationError{Field: "hostname", Message: "Hostname is too long (max 253 characters)"}
	}
	// Basic hostname/IP validation - could be enhanced with regex
	return nil
}

// ValidatePort validates port field with proper range checking
func ValidatePort(value string) error {
	if value == "" {
		return &ValidationError{Field: "port", Message: "Port is required"}
	}
	
	// Parse port as integer
	port := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return &ValidationError{Field: "port", Message: "Port must be a number"}
		}
		port = port*10 + int(r-'0')
		if port > 65535 {
			return &ValidationError{Field: "port", Message: "Port must be between 1 and 65535"}
		}
	}
	
	if port <= 0 {
		return &ValidationError{Field: "port", Message: "Port must be between 1 and 65535"}
	}
	
	return nil
}

// ValidateUsername validates username field
func ValidateUsername(value string) error {
	if value == "" {
		return &ValidationError{Field: "username", Message: "Username is required"}
	}
	if len(value) > 32 {
		return &ValidationError{Field: "username", Message: "Username is too long (max 32 characters)"}
	}
	return nil
}

// ValidateAuthType validates authentication type
func ValidateAuthType(value string) error {
	if value != "key" && value != "password" {
		return &ValidationError{Field: "auth_type", Message: "Auth type must be 'key' or 'password'"}
	}
	return nil
}

// ValidateKeyPath validates SSH key path (optional field)
func ValidateKeyPath(value string) error {
	// Key path is optional, so empty is allowed
	if value == "" {
		return nil
	}
	
	if len(value) > 500 {
		return &ValidationError{Field: "key_path", Message: "Key path is too long (max 500 characters)"}
	}
	
	// Could add file existence validation here in the future
	return nil
}

// ValidatePassphraseProtected validates passphrase protected field
func ValidatePassphraseProtected(value string) error {
	if value != "true" && value != "false" && value != "" {
		return &ValidationError{Field: "passphrase_protected", Message: "Must be 'true' or 'false'"}
	}
	return nil
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