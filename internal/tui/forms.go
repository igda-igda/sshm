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
	inputField    *tview.InputField
	passwordField *PasswordField           // For secure password input
	dropdown      *AuthenticationSelector // For special dropdown fields
	validator     func(string) error
	required      bool
}

// GetFormItem returns the appropriate form item (InputField, PasswordField, or DropDown)
func (f *FormField) GetFormItem() tview.FormItem {
	if f.dropdown != nil {
		return f.dropdown.GetFormItem()
	}
	if f.passwordField != nil {
		return f.passwordField.GetFormItem()
	}
	return f.inputField
}

// GetText returns the current value from input field, password field, or dropdown
func (f *FormField) GetText() string {
	if f.dropdown != nil {
		return f.dropdown.GetValue()
	}
	if f.passwordField != nil {
		return f.passwordField.GetText()
	}
	return f.inputField.GetText()
}

// SetText sets the value in input field, password field, or dropdown
func (f *FormField) SetText(value string) error {
	if f.dropdown != nil {
		return f.dropdown.SetValue(value)
	}
	if f.passwordField != nil {
		f.passwordField.SetText(value)
		return nil
	}
	f.inputField.SetText(value)
	return nil
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
	focusIndex       int                                  // Current focused field index
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
		focusIndex:       0,
	}
	
	// Add fields to the form
	tuiForm.setupFormFields()
	
	// Setup keyboard navigation
	tuiForm.setupKeyboardNavigation()
	
	// Setup real-time validation if enabled
	if realTimeValidate {
		tuiForm.setupRealTimeValidation()
	}
	
	// Apply initial focus styling
	tuiForm.applyFocusStyling()
	
	return tuiForm
}

// setupFormFields adds all fields to the tview.Form
func (tf *TUIForm) setupFormFields() {
	// Define a preferred field order for server forms
	preferredOrder := []string{"name", "hostname", "port", "username", "auth_type", "password", "key_path", "passphrase_protected"}
	
	// Build field order: preferred fields first, then remaining fields
	usedFields := make(map[string]bool)
	
	// Add preferred fields in order if they exist
	for _, fieldName := range preferredOrder {
		if field, exists := tf.fields[fieldName]; exists {
			tf.fieldOrder = append(tf.fieldOrder, fieldName)
			tf.form.AddFormItem(field.GetFormItem())
			usedFields[fieldName] = true
		}
	}
	
	// Add any remaining fields
	for fieldName, field := range tf.fields {
		if !usedFields[fieldName] {
			tf.fieldOrder = append(tf.fieldOrder, fieldName)
			tf.form.AddFormItem(field.GetFormItem())
		}
	}
	
	// Add error display if real-time validation is enabled
	if tf.realTimeValidate {
		tf.form.AddTextView("Errors:", "", 0, 3, true, false)
	}
	
	// Add Submit and Cancel buttons with prominent styling
	tf.form.AddButton("Submit", func() {
		tf.handleSubmit()
	})
	
	tf.form.AddButton("Cancel", func() {
		tf.handleCancel()
	})
	
	// Apply button styling after buttons are added
	tf.setupButtonStyling()
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
			// Tab moves to next field with visual focus update
			tf.moveFocusNext()
			return event // Let tview handle Tab navigation
		case tcell.KeyBacktab:
			// Shift+Tab moves to previous field with visual focus update
			tf.moveFocusPrevious()
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
		value := field.GetText()
		
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
	
	return field.SetText(value)
}

// GetFieldValue gets the current value of a specific field
func (tf *TUIForm) GetFieldValue(fieldName string) (string, error) {
	field, exists := tf.fields[fieldName]
	if !exists {
		return "", fmt.Errorf("field %s does not exist", fieldName)
	}
	
	return field.GetText(), nil
}

// setupRealTimeValidation configures real-time validation for form fields
func (tf *TUIForm) setupRealTimeValidation() {
	for fieldName, field := range tf.fields {
		// Capture field name in closure
		currentFieldName := fieldName
		currentField := field
		
		// Set up changed function for real-time validation only for InputFields
		// Dropdowns handle changes through their callback mechanism
		if currentField.inputField != nil {
			currentField.inputField.SetChangedFunc(func(text string) {
				// Perform validation
				tf.validateFieldRealTime(currentFieldName, text)
			})
		}
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
		value := field.GetText()
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

// FormFieldWithSelector represents a form field that can be either an InputField or a custom selector
type FormFieldWithSelector struct {
	inputField *tview.InputField
	selector   *AuthenticationSelector
	validator  func(string) error
	required   bool
}

// GetFormItem returns the appropriate form item (InputField or DropDown)
func (f *FormFieldWithSelector) GetFormItem() tview.FormItem {
	if f.selector != nil {
		return f.selector.GetFormItem()
	}
	return f.inputField
}

// GetText returns the current value from either input field or selector
func (f *FormFieldWithSelector) GetText() string {
	if f.selector != nil {
		return f.selector.GetValue()
	}
	return f.inputField.GetText()
}

// SetText sets the value in either input field or selector
func (f *FormFieldWithSelector) SetText(value string) error {
	if f.selector != nil {
		return f.selector.SetValue(value)
	}
	f.inputField.SetText(value)
	return nil
}

// EnhancedFormField represents a form field that can contain either InputField or custom components
type EnhancedFormField struct {
	formItem  tview.FormItem
	getValue  func() string
	setValue  func(string) error
	validator func(string) error
	required  bool
}

// CreateServerFormFields creates the standard server form fields with enhanced validation
func CreateServerFormFields() map[string]*FormField {
	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().
				SetLabel("Server Name: ").
				SetFieldWidth(30).
				SetPlaceholder("e.g., production-api").
				SetFieldTextColor(tcell.ColorWhite).
				SetFieldBackgroundColor(tcell.ColorBlack).
				SetLabelColor(tcell.ColorWhite),
			validator: ValidateServerName,
			required:  true,
		},
		"hostname": {
			inputField: tview.NewInputField().
				SetLabel("Hostname: ").
				SetFieldWidth(40).
				SetPlaceholder("e.g., example.com or 192.168.1.100").
				SetFieldTextColor(tcell.ColorWhite).
				SetFieldBackgroundColor(tcell.ColorBlack).
				SetLabelColor(tcell.ColorWhite),
			validator: ValidateHostname,
			required:  true,
		},
		"port": {
			inputField: tview.NewInputField().
				SetLabel("Port: ").
				SetText("22").
				SetFieldWidth(10).
				SetPlaceholder("1-65535").
				SetFieldTextColor(tcell.ColorWhite).
				SetFieldBackgroundColor(tcell.ColorBlack).
				SetLabelColor(tcell.ColorWhite),
			validator: ValidatePort,
			required:  true,
		},
		"username": {
			inputField: tview.NewInputField().
				SetLabel("Username: ").
				SetFieldWidth(25).
				SetPlaceholder("e.g., ubuntu, admin, root").
				SetFieldTextColor(tcell.ColorWhite).
				SetFieldBackgroundColor(tcell.ColorBlack).
				SetLabelColor(tcell.ColorWhite),
			validator: ValidateUsername,
			required:  true,
		},
		"auth_type": {
			dropdown: nil, // Will be created after password field is defined for callback
			validator: ValidateAuthType,
			required:  true,
		},
		"key_path": {
			inputField: tview.NewInputField().
				SetLabel("Key Path (optional): ").
				SetFieldWidth(50).
				SetPlaceholder("e.g., ~/.ssh/id_rsa").
				SetFieldTextColor(tcell.ColorWhite).
				SetFieldBackgroundColor(tcell.ColorBlack).
				SetLabelColor(tcell.ColorWhite),
			validator: ValidateKeyPath,
			required:  false,
		},
		"passphrase_protected": {
			inputField: tview.NewInputField().
				SetLabel("Passphrase Protected: ").
				SetText("false").
				SetFieldWidth(10).
				SetPlaceholder("true or false").
				SetFieldTextColor(tcell.ColorWhite).
				SetFieldBackgroundColor(tcell.ColorBlack).
				SetLabelColor(tcell.ColorWhite),
			validator: ValidatePassphraseProtected,
			required:  false,
		},
		"password": {
			passwordField: NewPasswordField().
				SetLabel("Password: ").
				SetFieldWidth(30).
				SetPlaceholder("Enter password for authentication").
				SetMaxLength(128).
				SetColors(tcell.ColorWhite, tcell.ColorBlack, tcell.ColorWhite),
			validator: ValidatePasswordField,
			required:  false, // Dynamically required when auth_type is "password"
		},
	}
	
	// Create authentication selector with callback to show/hide password field
	fields["auth_type"].dropdown = NewAuthenticationSelector(func(authType string) {
		// Show/hide password field based on authentication type
		if passwordField := fields["password"]; passwordField != nil {
			if authType == "password" {
				// Show password field by making it visible (in real implementation)
				// For now, we just ensure it's properly configured
				passwordField.required = true
			} else {
				// Hide password field when key authentication is selected
				passwordField.required = false
				// Clear password field when switching to key auth for security
				if passwordField.passwordField != nil {
					passwordField.passwordField.Clear()
				}
			}
		}
	})
	
	return fields
}

// CreateEnhancedServerFormFields creates server form fields with authentication dropdown
func CreateEnhancedServerFormFields() map[string]*EnhancedFormField {
	fields := make(map[string]*EnhancedFormField)
	
	// Standard input fields
	standardFields := []struct {
		name        string
		label       string
		width       int
		placeholder string
		defaultText string
		validator   func(string) error
		required    bool
	}{
		{"name", "Server Name: ", 30, "e.g., production-api", "", ValidateServerName, true},
		{"hostname", "Hostname: ", 40, "e.g., example.com or 192.168.1.100", "", ValidateHostname, true},
		{"port", "Port: ", 10, "1-65535", "22", ValidatePort, true},
		{"username", "Username: ", 25, "e.g., ubuntu, admin, root", "", ValidateUsername, true},
		{"key_path", "Key Path (optional): ", 50, "e.g., ~/.ssh/id_rsa", "", ValidateKeyPath, false},
		{"passphrase_protected", "Passphrase Protected: ", 10, "true or false", "false", ValidatePassphraseProtected, false},
	}
	
	for _, field := range standardFields {
		inputField := tview.NewInputField().
			SetLabel(field.label).
			SetFieldWidth(field.width).
			SetPlaceholder(field.placeholder).
			SetText(field.defaultText).
			SetFieldTextColor(tcell.ColorWhite).
			SetFieldBackgroundColor(tcell.ColorBlack).
			SetLabelColor(tcell.ColorWhite)
			
		fields[field.name] = &EnhancedFormField{
			formItem: inputField,
			getValue: func() string { return inputField.GetText() },
			setValue: func(value string) error { inputField.SetText(value); return nil },
			validator: field.validator,
			required:  field.required,
		}
	}
	
	// Special authentication type dropdown field
	authSelector := NewAuthenticationSelector(func(authType string) {
		// This callback can be used for future functionality like showing/hiding password fields
	})
	
	fields["auth_type"] = &EnhancedFormField{
		formItem:  authSelector.GetFormItem(),
		getValue:  func() string { return authSelector.GetValue() },
		setValue:  func(value string) error { return authSelector.SetValue(value) },
		validator: ValidateAuthType,
		required:  true,
	}
	
	return fields
}

// EnhancedTUIForm represents a modal form with validation and keyboard navigation using EnhancedFormField
type EnhancedTUIForm struct {
	form             *tview.Form
	fields           map[string]*EnhancedFormField
	fieldOrder       []string
	onSubmit         func(map[string]interface{}) error
	onCancel         func()
	realTimeValidate bool
	errorDisplay     *tview.TextView
	validationErrors map[string]string
	focusIndex       int
}

// NewEnhancedTUIForm creates a new enhanced TUI form
func NewEnhancedTUIForm(fields map[string]*EnhancedFormField, onSubmit func(map[string]interface{}) error, onCancel func()) *EnhancedTUIForm {
	return NewEnhancedTUIFormWithOptions(fields, onSubmit, onCancel, false)
}

// NewEnhancedTUIFormWithOptions creates a new enhanced TUI form with options
func NewEnhancedTUIFormWithOptions(fields map[string]*EnhancedFormField, onSubmit func(map[string]interface{}) error, onCancel func(), realTimeValidate bool) *EnhancedTUIForm {
	form := tview.NewForm()
	
	// Create error display area
	errorDisplay := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetMaxLines(3)
	
	tuiForm := &EnhancedTUIForm{
		form:             form,
		fields:           fields,
		fieldOrder:       make([]string, 0, len(fields)),
		onSubmit:         onSubmit,
		onCancel:         onCancel,
		realTimeValidate: realTimeValidate,
		errorDisplay:     errorDisplay,
		validationErrors: make(map[string]string),
		focusIndex:       0,
	}
	
	// Add fields to the form
	tuiForm.setupFormFields()
	
	// Setup keyboard navigation
	tuiForm.setupKeyboardNavigation()
	
	// Setup real-time validation if enabled
	if realTimeValidate {
		tuiForm.setupRealTimeValidation()
	}
	
	// Apply initial focus styling
	tuiForm.applyFocusStyling()
	
	return tuiForm
}

// setupFormFields adds all fields to the tview.Form
func (etf *EnhancedTUIForm) setupFormFields() {
	// Define a preferred field order for server forms
	preferredOrder := []string{"name", "hostname", "port", "username", "auth_type", "password", "key_path", "passphrase_protected"}
	
	// Build field order: preferred fields first, then remaining fields
	usedFields := make(map[string]bool)
	
	// Add preferred fields in order if they exist
	for _, fieldName := range preferredOrder {
		if field, exists := etf.fields[fieldName]; exists {
			etf.fieldOrder = append(etf.fieldOrder, fieldName)
			etf.form.AddFormItem(field.formItem)
			usedFields[fieldName] = true
		}
	}
	
	// Add any remaining fields
	for fieldName, field := range etf.fields {
		if !usedFields[fieldName] {
			etf.fieldOrder = append(etf.fieldOrder, fieldName)
			etf.form.AddFormItem(field.formItem)
		}
	}
	
	// Add error display if real-time validation is enabled
	if etf.realTimeValidate {
		etf.form.AddTextView("Errors:", "", 0, 3, true, false)
	}
	
	// Add Submit and Cancel buttons with prominent styling
	etf.form.AddButton("Submit", func() {
		etf.handleSubmit()
	})
	
	etf.form.AddButton("Cancel", func() {
		etf.handleCancel()
	})
	
	// Apply button styling after buttons are added
	etf.setupButtonStyling()
}

// setupKeyboardNavigation configures keyboard navigation for the form
func (etf *EnhancedTUIForm) setupKeyboardNavigation() {
	etf.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// Enter key submits the form
			etf.handleSubmit()
			return nil
		case tcell.KeyEscape:
			// Escape key cancels the form
			etf.handleCancel()
			return nil
		case tcell.KeyTab:
			// Tab moves to next field with visual focus update
			etf.moveFocusNext()
			return event // Let tview handle Tab navigation
		case tcell.KeyBacktab:
			// Shift+Tab moves to previous field with visual focus update
			etf.moveFocusPrevious()
			return event // Let tview handle Shift+Tab navigation
		}
		return event
	})
}

// handleSubmit processes form submission
func (etf *EnhancedTUIForm) handleSubmit() {
	// Collect and validate form data
	data, err := etf.CollectFormData()
	if err != nil {
		// Handle validation error
		return
	}
	
	// Call the submission callback
	if etf.onSubmit != nil {
		if err := etf.onSubmit(data); err != nil {
			// Handle submission error
			return
		}
	}
}

// handleCancel processes form cancellation
func (etf *EnhancedTUIForm) handleCancel() {
	if etf.onCancel != nil {
		etf.onCancel()
	}
}

// ValidateField validates a single field
func (etf *EnhancedTUIForm) ValidateField(fieldName, value string) error {
	field, exists := etf.fields[fieldName]
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
func (etf *EnhancedTUIForm) CollectFormData() (map[string]interface{}, error) {
	data := make(map[string]interface{})
	
	for fieldName, field := range etf.fields {
		value := field.getValue()
		
		// Validate field
		if err := etf.ValidateField(fieldName, value); err != nil {
			return nil, err
		}
		
		data[fieldName] = value
	}
	
	return data, nil
}

// GetForm returns the underlying tview.Form for display
func (etf *EnhancedTUIForm) GetForm() *tview.Form {
	return etf.form
}

// SetFieldValue sets the value of a specific field
func (etf *EnhancedTUIForm) SetFieldValue(fieldName, value string) error {
	field, exists := etf.fields[fieldName]
	if !exists {
		return fmt.Errorf("field %s does not exist", fieldName)
	}
	
	return field.setValue(value)
}

// GetFieldValue gets the current value of a specific field
func (etf *EnhancedTUIForm) GetFieldValue(fieldName string) (string, error) {
	field, exists := etf.fields[fieldName]
	if !exists {
		return "", fmt.Errorf("field %s does not exist", fieldName)
	}
	
	return field.getValue(), nil
}

// Additional helper methods for EnhancedTUIForm that mirror TUIForm functionality
func (etf *EnhancedTUIForm) setupRealTimeValidation() {
	// Real-time validation for enhanced form fields would need custom implementation
	// For now, this is a placeholder
}

func (etf *EnhancedTUIForm) applyFocusStyling() {
	// Apply styling to form fields - this would need custom implementation
	// for different field types
}

func (etf *EnhancedTUIForm) moveFocusNext() {
	etf.focusIndex = (etf.focusIndex + 1) % len(etf.fieldOrder)
	etf.applyFocusStyling()
}

func (etf *EnhancedTUIForm) moveFocusPrevious() {
	etf.focusIndex = (etf.focusIndex - 1 + len(etf.fieldOrder)) % len(etf.fieldOrder)
	etf.applyFocusStyling()
}

func (etf *EnhancedTUIForm) setupButtonStyling() {
	// Apply form-level styling that affects buttons
	etf.form.SetButtonBackgroundColor(tcell.ColorDarkBlue).
		SetButtonTextColor(tcell.ColorWhite).
		SetLabelColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite)
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

// ValidatePasswordField validates password field with conditional requirements
func ValidatePasswordField(value string) error {
	// Note: This is a basic validation - in real implementation, this would
	// check the current auth_type value from the form to conditionally require password
	if len(value) > 128 {
		return &ValidationError{Field: "password", Message: "Password is too long (max 128 characters)"}
	}
	
	// Additional password strength validation could be added here
	if value != "" && len(value) < 3 {
		return &ValidationError{Field: "password", Message: "Password must be at least 3 characters"}
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

// ShowInfoModal displays an informational modal with title and message
func (mm *ModalManager) ShowInfoModal(title, message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			mm.HideModal()
		}).
		SetBackgroundColor(tcell.ColorDarkGreen)
	
	// Add consistent keyboard handling
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			mm.HideModal()
			return nil
		case tcell.KeyEscape:
			mm.HideModal()
			return nil
		}
		return event
	})
	
	if title != "" {
		modal.SetTitle(" " + title + " ")
	}
	
	mm.ShowModal(modal)
}

// applyFocusStyling applies visual styling to indicate the currently focused field
func (tf *TUIForm) applyFocusStyling() {
	// Apply styling to all fields based on their focus state
	for i, fieldName := range tf.fieldOrder {
		field, exists := tf.fields[fieldName]
		if !exists {
			continue
		}
		
		if i == tf.focusIndex {
			// Apply focused field styling
			if field.inputField != nil {
				field.inputField.SetFieldTextColor(tcell.ColorBlack).
					SetFieldBackgroundColor(tcell.ColorWhite).
					SetLabelColor(tcell.ColorYellow)
			} else if field.dropdown != nil {
				field.dropdown.ApplyFocusStyling()
			}
		} else {
			// Apply unfocused field styling
			if field.inputField != nil {
				field.inputField.SetFieldTextColor(tcell.ColorWhite).
					SetFieldBackgroundColor(tcell.ColorBlack).
					SetLabelColor(tcell.ColorWhite)
			} else if field.dropdown != nil {
				field.dropdown.ApplyUnfocusStyling()
			}
		}
	}
	
	// Ensure buttons maintain prominent highlighting
	tf.updateButtonHighlighting()
}

// moveFocusNext moves focus to the next field and updates styling
func (tf *TUIForm) moveFocusNext() {
	tf.focusIndex = (tf.focusIndex + 1) % len(tf.fieldOrder)
	tf.applyFocusStyling()
}

// moveFocusPrevious moves focus to the previous field and updates styling
func (tf *TUIForm) moveFocusPrevious() {
	tf.focusIndex = (tf.focusIndex - 1 + len(tf.fieldOrder)) % len(tf.fieldOrder)
	tf.applyFocusStyling()
}

// getCurrentFocusedField returns the currently focused field name and field
func (tf *TUIForm) getCurrentFocusedField() (string, *FormField) {
	if tf.focusIndex >= 0 && tf.focusIndex < len(tf.fieldOrder) {
		fieldName := tf.fieldOrder[tf.focusIndex]
		if field, exists := tf.fields[fieldName]; exists {
			return fieldName, field
		}
	}
	return "", nil
}

// setFocusIndex sets the focus to a specific field index and updates styling
func (tf *TUIForm) setFocusIndex(index int) {
	if index >= 0 && index < len(tf.fieldOrder) {
		tf.focusIndex = index
		tf.applyFocusStyling()
	}
}

// setupButtonStyling applies prominent styling to Submit and Cancel buttons
func (tf *TUIForm) setupButtonStyling() {
	// Apply form-level styling that affects buttons
	tf.form.SetButtonBackgroundColor(tcell.ColorDarkBlue).
		SetButtonTextColor(tcell.ColorWhite).
		SetLabelColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite)
}

// updateButtonHighlighting updates button styling based on current form state
func (tf *TUIForm) updateButtonHighlighting() {
	// This ensures buttons maintain their prominent styling
	// even when field focus changes
	tf.form.SetButtonBackgroundColor(tcell.ColorDarkBlue).
		SetButtonTextColor(tcell.ColorWhite)
}

// HideField hides a form field by removing it from the display
func (tf *TUIForm) HideField(fieldName string) {
	if field, exists := tf.fields[fieldName]; exists {
		// Mark field as hidden (for future tview implementations that support dynamic field removal)
		// For now, we'll handle visibility through field order management
		for i, name := range tf.fieldOrder {
			if name == fieldName {
				// Move hidden field to end of order (effectively hiding it)
				tf.fieldOrder = append(tf.fieldOrder[:i], tf.fieldOrder[i+1:]...)
				tf.fieldOrder = append(tf.fieldOrder, fieldName)
				break
			}
		}
		
		// Clear field value when hiding for security (especially for password fields)
		if field.inputField != nil {
			field.inputField.SetText("")
		}
		
		// Refresh form display
		tf.refreshFormDisplay()
	}
}

// ShowField shows a form field by ensuring it's in the proper position
func (tf *TUIForm) ShowField(fieldName string) {
	if _, exists := tf.fields[fieldName]; exists {
		// Restore proper field order
		tf.rebuildFieldOrder()
		
		// Refresh form display
		tf.refreshFormDisplay()
	}
}

// refreshFormDisplay rebuilds the form display with current field visibility
func (tf *TUIForm) refreshFormDisplay() {
	// This is a placeholder for form refresh logic
	// In a full implementation, this would rebuild the tview.Form
	// with only visible fields
}

// rebuildFieldOrder rebuilds the field order based on preferred order
func (tf *TUIForm) rebuildFieldOrder() {
	preferredOrder := []string{"name", "hostname", "port", "username", "auth_type", "password", "key_path", "passphrase_protected"}
	newOrder := []string{}
	usedFields := make(map[string]bool)
	
	// Add preferred fields in order if they exist
	for _, fieldName := range preferredOrder {
		if _, exists := tf.fields[fieldName]; exists {
			newOrder = append(newOrder, fieldName)
			usedFields[fieldName] = true
		}
	}
	
	// Add any remaining fields
	for fieldName := range tf.fields {
		if !usedFields[fieldName] {
			newOrder = append(newOrder, fieldName)
		}
	}
	
	tf.fieldOrder = newOrder
}

// SetConditionalFieldLogic sets up conditional field display logic
func (tf *TUIForm) SetConditionalFieldLogic() {
	// Set up auth type change callback to show/hide password field
	if authField, exists := tf.fields["auth_type"]; exists && authField.dropdown != nil {
		// Update the authentication selector's callback
		authField.dropdown = NewAuthenticationSelector(func(authType string) {
			tf.handleAuthTypeChange(authType)
		})
	}
}

// handleAuthTypeChange handles authentication type changes and updates field visibility
func (tf *TUIForm) handleAuthTypeChange(authType string) {
	if authType == "password" {
		tf.ShowField("password")
		// Make password required when password auth is selected
		if passwordField := tf.fields["password"]; passwordField != nil {
			passwordField.required = true
		}
	} else {
		tf.HideField("password") 
		// Make password optional when key auth is selected
		if passwordField := tf.fields["password"]; passwordField != nil {
			passwordField.required = false
		}
	}
}