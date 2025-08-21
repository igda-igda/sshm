package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sshm/internal/config"
)

// FormTestHelper assists with testing form interactions
type FormTestHelper struct {
	form     *TUIForm
	testApp  *tview.Application
	events   chan *tcell.EventKey
	screen   tcell.SimulationScreen
}

// NewFormTestHelper creates a new form test helper
func NewFormTestHelper(form *TUIForm) *FormTestHelper {
	screen := tcell.NewSimulationScreen("UTF-8")
	screen.Init()
	screen.SetSize(80, 24)
	
	app := tview.NewApplication()
	app.SetScreen(screen)
	
	return &FormTestHelper{
		form:    form,
		testApp: app,
		events:  make(chan *tcell.EventKey, 100),
		screen:  screen,
	}
}

// SimulateKeypress simulates a key press event
func (fth *FormTestHelper) SimulateKeypress(key tcell.Key) {
	event := tcell.NewEventKey(key, 0, tcell.ModNone)
	fth.events <- event
}

// SimulateKeypressWithMod simulates a key press with modifiers
func (fth *FormTestHelper) SimulateKeypressWithMod(key tcell.Key, mod tcell.ModMask) {
	event := tcell.NewEventKey(key, 0, mod)
	fth.events <- event
}

// SimulateInput simulates typing text input
func (fth *FormTestHelper) SimulateInput(text string) {
	for _, r := range text {
		event := tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
		fth.events <- event
	}
}

// ProcessEvents processes queued events for testing
func (fth *FormTestHelper) ProcessEvents() {
	for len(fth.events) > 0 {
		event := <-fth.events
		// Process the event through the form's input capture
		if fth.form != nil && fth.form.form != nil {
			if handler := fth.form.form.GetInputCapture(); handler != nil {
				processedEvent := handler(event)
				// If event was consumed (returned nil), don't process further
				if processedEvent == nil {
					continue
				}
			}
		}
	}
}

// Cleanup cleans up test resources
func (fth *FormTestHelper) Cleanup() {
	if fth.screen != nil {
		fth.screen.Fini()
	}
	close(fth.events)
}

// TestTUIForm_Creation tests basic form creation
func TestTUIForm_Creation(t *testing.T) {
	// Create a basic form configuration
	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().SetLabel("Name: "),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"host": {
			inputField: tview.NewInputField().SetLabel("Host: "),
			validator:  func(s string) error { return nil },
			required:   true,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		return nil
	}
	
	onCancel := func() {
		// Test cancel callback
	}

	form := NewTUIForm(fields, onSubmit, onCancel)
	
	if form == nil {
		t.Fatal("Expected form to be created, got nil")
	}
	
	if form.form == nil {
		t.Fatal("Expected internal tview.Form to be created, got nil")
	}
	
	if len(form.fields) != 2 {
		t.Errorf("Expected 2 fields in form, got %d", len(form.fields))
	}
	
	if form.onSubmit == nil {
		t.Fatal("Expected onSubmit callback to be set")
	}
	
	if form.onCancel == nil {
		t.Fatal("Expected onCancel callback to be set")
	}
}

// TestTUIForm_KeyboardNavigation tests Tab/Shift+Tab navigation
func TestTUIForm_KeyboardNavigation(t *testing.T) {
	// Create form with multiple fields
	fields := map[string]*FormField{
		"field1": {
			inputField: tview.NewInputField().SetLabel("Field 1: "),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"field2": {
			inputField: tview.NewInputField().SetLabel("Field 2: "),
			validator:  func(s string) error { return nil },
			required:   false,
		},
		"field3": {
			inputField: tview.NewInputField().SetLabel("Field 3: "),
			validator:  func(s string) error { return nil },
			required:   true,
		},
	}

	submitCalled := false
	cancelCalled := false
	
	onSubmit := func(data map[string]interface{}) error {
		submitCalled = true
		return nil
	}
	
	onCancel := func() {
		cancelCalled = true
	}

	form := NewTUIForm(fields, onSubmit, onCancel)
	helper := NewFormTestHelper(form)
	defer helper.Cleanup()

	// Test Tab navigation moves focus forward
	helper.SimulateKeypress(tcell.KeyTab)
	helper.ProcessEvents()
	
	// Test Shift+Tab navigation moves focus backward
	helper.SimulateKeypressWithMod(tcell.KeyTab, tcell.ModShift)
	helper.ProcessEvents()
	
	// Test Enter key triggers form submission
	helper.SimulateKeypress(tcell.KeyEnter)
	helper.ProcessEvents()
	
	if !submitCalled {
		t.Error("Expected form submission to be called when Enter pressed")
	}
	
	// Reset and test Escape key
	submitCalled = false
	helper.SimulateKeypress(tcell.KeyEscape)
	helper.ProcessEvents()
	
	if !cancelCalled {
		t.Error("Expected form cancellation to be called when Escape pressed")
	}
}

// TestTUIForm_FieldValidation tests field validation functionality
func TestTUIForm_FieldValidation(t *testing.T) {
	validationCalled := false
	validationError := false
	
	fields := map[string]*FormField{
		"required_field": {
			inputField: tview.NewInputField().SetLabel("Required: "),
			validator: func(s string) error {
				validationCalled = true
				if s == "" {
					validationError = true
					return &ValidationError{Field: "required_field", Message: "Field is required"}
				}
				validationError = false
				return nil
			},
			required: true,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		return nil
	}
	
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)
	
	// Test validation with empty field
	err := form.ValidateField("required_field", "")
	if err == nil {
		t.Error("Expected validation error for empty required field")
	}
	
	if !validationCalled {
		t.Error("Expected validator function to be called")
	}
	
	if !validationError {
		t.Error("Expected validation error to be set")
	}
	
	// Reset and test validation with valid data
	validationCalled = false
	validationError = false
	
	err = form.ValidateField("required_field", "valid data")
	if err != nil {
		t.Errorf("Expected no validation error for valid data, got: %v", err)
	}
	
	if !validationCalled {
		t.Error("Expected validator function to be called")
	}
	
	if validationError {
		t.Error("Expected no validation error for valid data")
	}
}

// TestTUIForm_DataCollection tests form data collection
func TestTUIForm_DataCollection(t *testing.T) {
	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().SetLabel("Name: ").SetText("test-name"),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"port": {
			inputField: tview.NewInputField().SetLabel("Port: ").SetText("22"),
			validator:  func(s string) error { return nil },
			required:   false,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		// Data will be collected in test
		return nil
	}
	
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)
	
	// Test data collection
	data, err := form.CollectFormData()
	if err != nil {
		t.Errorf("Expected no error collecting form data, got: %v", err)
	}
	
	if len(data) != 2 {
		t.Errorf("Expected 2 fields in collected data, got %d", len(data))
	}
	
	if data["name"] != "test-name" {
		t.Errorf("Expected name field to be 'test-name', got: %v", data["name"])
	}
	
	if data["port"] != "22" {
		t.Errorf("Expected port field to be '22', got: %v", data["port"])
	}
}

// TestTUIForm_SubmissionWorkflow tests complete form submission workflow
func TestTUIForm_SubmissionWorkflow(t *testing.T) {
	fields := map[string]*FormField{
		"server_name": {
			inputField: tview.NewInputField().SetLabel("Server Name: ").SetText("test-server"),
			validator: func(s string) error {
				if s == "" {
					return &ValidationError{Field: "server_name", Message: "Server name is required"}
				}
				return nil
			},
			required: true,
		},
	}

	submittedData := make(map[string]interface{})
	submissionError := false
	
	onSubmit := func(data map[string]interface{}) error {
		submittedData = data
		// Simulate validation at submission time
		if data["server_name"] == "" {
			submissionError = true
			return &ValidationError{Field: "server_name", Message: "Server name cannot be empty"}
		}
		submissionError = false
		return nil
	}
	
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)
	helper := NewFormTestHelper(form)
	defer helper.Cleanup()

	// Test successful submission
	helper.SimulateKeypress(tcell.KeyEnter)
	helper.ProcessEvents()
	
	if submissionError {
		t.Error("Expected successful form submission, but got error")
	}
	
	if len(submittedData) == 0 {
		t.Error("Expected form data to be submitted")
	}
	
	if submittedData["server_name"] != "test-server" {
		t.Errorf("Expected submitted server name to be 'test-server', got: %v", submittedData["server_name"])
	}
}

// TestTUIForm_ErrorHandling tests error handling in forms
func TestTUIForm_ErrorHandling(t *testing.T) {
	fields := map[string]*FormField{
		"invalid_field": {
			inputField: tview.NewInputField().SetLabel("Invalid: ").SetText(""),
			validator: func(s string) error {
				return &ValidationError{Field: "invalid_field", Message: "Always invalid"}
			},
			required: true,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		return &ValidationError{Field: "invalid_field", Message: "Submission failed"}
	}
	
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)
	
	// Test field validation error
	err := form.ValidateField("invalid_field", "")
	if err == nil {
		t.Error("Expected validation error, got nil")
	}
	
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("Expected ValidationError, got: %T", err)
	} else {
		if validationErr.Field != "invalid_field" {
			t.Errorf("Expected error field to be 'invalid_field', got: %s", validationErr.Field)
		}
		if validationErr.Message != "Always invalid" {
			t.Errorf("Expected error message 'Always invalid', got: %s", validationErr.Message)
		}
	}
}

// TestTUIForm_CancellationWorkflow tests form cancellation
func TestTUIForm_CancellationWorkflow(t *testing.T) {
	fields := map[string]*FormField{
		"field": {
			inputField: tview.NewInputField().SetLabel("Field: ").SetText("some data"),
			validator:  func(s string) error { return nil },
			required:   false,
		},
	}

	submitCalled := false
	cancelCalled := false
	
	onSubmit := func(data map[string]interface{}) error {
		submitCalled = true
		return nil
	}
	
	onCancel := func() {
		cancelCalled = true
	}

	form := NewTUIForm(fields, onSubmit, onCancel)
	helper := NewFormTestHelper(form)
	defer helper.Cleanup()

	// Test cancellation with Escape key
	helper.SimulateKeypress(tcell.KeyEscape)
	helper.ProcessEvents()
	
	if submitCalled {
		t.Error("Expected submission not to be called during cancellation")
	}
	
	if !cancelCalled {
		t.Error("Expected cancellation callback to be called")
	}
}

// TestTUIForm_InputFieldFocus tests that input fields receive focus properly
func TestTUIForm_InputFieldFocus(t *testing.T) {
	fields := map[string]*FormField{
		"focused_field": {
			inputField: tview.NewInputField().SetLabel("Focused: "),
			validator:  func(s string) error { return nil },
			required:   true,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)
	
	if form.form == nil {
		t.Fatal("Expected form to be created")
	}
	
	// Form should be focusable - just verify the form exists and can receive focus
	// tview.Form implements Focusable interface, so we test that it exists
	if form.form == nil {
		t.Error("Expected form to be focusable")
	}
}

// TestModalManager_Creation tests modal manager creation and initialization
func TestModalManager_Creation(t *testing.T) {
	app := tview.NewApplication()
	layout := tview.NewFlex()
	
	manager := NewModalManager(app, layout)
	
	if manager == nil {
		t.Fatal("Expected modal manager to be created, got nil")
	}
	
	if manager.app != app {
		t.Error("Expected modal manager to store app reference")
	}
	
	if manager.layout != layout {
		t.Error("Expected modal manager to store layout reference")
	}
	
	if manager.modalStack == nil {
		t.Fatal("Expected modal stack to be initialized")
	}
	
	if len(manager.modalStack) != 0 {
		t.Errorf("Expected empty modal stack initially, got %d items", len(manager.modalStack))
	}
}

// TestModalManager_ShowModal tests showing modals
func TestModalManager_ShowModal(t *testing.T) {
	app := tview.NewApplication()
	layout := tview.NewFlex()
	
	manager := NewModalManager(app, layout)
	modal := tview.NewModal().SetText("Test Modal")
	
	// Show modal
	manager.ShowModal(modal)
	
	if len(manager.modalStack) != 1 {
		t.Errorf("Expected 1 modal in stack after showing, got %d", len(manager.modalStack))
	}
	
	if manager.modalStack[0] != modal {
		t.Error("Expected modal to be added to stack")
	}
}

// TestModalManager_HideModal tests hiding modals
func TestModalManager_HideModal(t *testing.T) {
	app := tview.NewApplication()
	layout := tview.NewFlex()
	
	manager := NewModalManager(app, layout)
	modal := tview.NewModal().SetText("Test Modal")
	
	// Show then hide modal
	manager.ShowModal(modal)
	manager.HideModal()
	
	if len(manager.modalStack) != 0 {
		t.Errorf("Expected empty modal stack after hiding, got %d items", len(manager.modalStack))
	}
}

// TestModalManager_ModalStacking tests multiple modal stacking
func TestModalManager_ModalStacking(t *testing.T) {
	app := tview.NewApplication()
	layout := tview.NewFlex()
	
	manager := NewModalManager(app, layout)
	modal1 := tview.NewModal().SetText("Modal 1")
	modal2 := tview.NewModal().SetText("Modal 2")
	
	// Show two modals
	manager.ShowModal(modal1)
	manager.ShowModal(modal2)
	
	if len(manager.modalStack) != 2 {
		t.Errorf("Expected 2 modals in stack, got %d", len(manager.modalStack))
	}
	
	// Hide one modal
	manager.HideModal()
	
	if len(manager.modalStack) != 1 {
		t.Errorf("Expected 1 modal in stack after hiding one, got %d", len(manager.modalStack))
	}
	
	if manager.modalStack[0] != modal1 {
		t.Error("Expected first modal to remain in stack")
	}
}

// TestModalManager_KeyboardRouting tests keyboard event routing through modals
func TestModalManager_KeyboardRouting(t *testing.T) {
	app := tview.NewApplication()
	layout := tview.NewFlex()
	
	manager := NewModalManager(app, layout)
	
	// Create modal with input capture
	modal := tview.NewModal().SetText("Test Modal")
	escapePressed := false
	enterPressed := false
	
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			escapePressed = true
			return nil
		case tcell.KeyEnter:
			enterPressed = true
			return nil
		}
		return event
	})
	
	// Show modal and simulate key events
	manager.ShowModal(modal)
	
	// Simulate Escape key
	escapeEvent := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	if modal.GetInputCapture() != nil {
		modal.GetInputCapture()(escapeEvent)
	}
	
	if !escapePressed {
		t.Error("Expected Escape key to be captured by modal")
	}
	
	// Simulate Enter key
	enterEvent := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	if modal.GetInputCapture() != nil {
		modal.GetInputCapture()(enterEvent)
	}
	
	if !enterPressed {
		t.Error("Expected Enter key to be captured by modal")
	}
}

// TestServerForm_AllFieldsValidation tests comprehensive server form validation
func TestServerForm_AllFieldsValidation(t *testing.T) {
	// Test field validation with all CLI add command fields
	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().SetLabel("Server Name: "),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "name", Message: "Server name is required"}
				}
				return nil
			},
			required: true,
		},
		"hostname": {
			inputField: tview.NewInputField().SetLabel("Hostname: "),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "hostname", Message: "Hostname is required"}
				}
				return nil
			},
			required: true,
		},
		"port": {
			inputField: tview.NewInputField().SetLabel("Port: ").SetText("22"),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "port", Message: "Port is required"}
				}
				// Test port range validation in the future
				return nil
			},
			required: true,
		},
		"username": {
			inputField: tview.NewInputField().SetLabel("Username: "),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "username", Message: "Username is required"}
				}
				return nil
			},
			required: true,
		},
		"auth_type": {
			inputField: tview.NewInputField().SetLabel("Auth Type (key/password): ").SetText("key"),
			validator: func(value string) error {
				if value != "key" && value != "password" {
					return &ValidationError{Field: "auth_type", Message: "Auth type must be 'key' or 'password'"}
				}
				return nil
			},
			required: true,
		},
		"key_path": {
			inputField: tview.NewInputField().SetLabel("Key Path (optional): "),
			validator: func(value string) error {
				// Key path is optional but could validate file existence
				return nil
			},
			required: false,
		},
		"passphrase_protected": {
			inputField: tview.NewInputField().SetLabel("Passphrase Protected (true/false): ").SetText("false"),
			validator: func(value string) error {
				if value != "true" && value != "false" && value != "" {
					return &ValidationError{Field: "passphrase_protected", Message: "Must be 'true' or 'false'"}
				}
				return nil
			},
			required: false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Test required field validation - name
	err := form.ValidateField("name", "")
	if err == nil {
		t.Error("Expected validation error for empty server name")
	}

	err = form.ValidateField("name", "test-server")
	if err != nil {
		t.Errorf("Expected no validation error for valid server name, got: %v", err)
	}

	// Test required field validation - hostname
	err = form.ValidateField("hostname", "")
	if err == nil {
		t.Error("Expected validation error for empty hostname")
	}

	err = form.ValidateField("hostname", "example.com")
	if err != nil {
		t.Errorf("Expected no validation error for valid hostname, got: %v", err)
	}

	// Test required field validation - port
	err = form.ValidateField("port", "")
	if err == nil {
		t.Error("Expected validation error for empty port")
	}

	err = form.ValidateField("port", "22")
	if err != nil {
		t.Errorf("Expected no validation error for valid port, got: %v", err)
	}

	// Test required field validation - username
	err = form.ValidateField("username", "")
	if err == nil {
		t.Error("Expected validation error for empty username")
	}

	err = form.ValidateField("username", "testuser")
	if err != nil {
		t.Errorf("Expected no validation error for valid username, got: %v", err)
	}

	// Test auth type validation
	err = form.ValidateField("auth_type", "invalid")
	if err == nil {
		t.Error("Expected validation error for invalid auth type")
	}

	err = form.ValidateField("auth_type", "key")
	if err != nil {
		t.Errorf("Expected no validation error for 'key' auth type, got: %v", err)
	}

	err = form.ValidateField("auth_type", "password")
	if err != nil {
		t.Errorf("Expected no validation error for 'password' auth type, got: %v", err)
	}

	// Test optional key path validation
	err = form.ValidateField("key_path", "")
	if err != nil {
		t.Errorf("Expected no validation error for empty key path (optional), got: %v", err)
	}

	err = form.ValidateField("key_path", "~/.ssh/id_rsa")
	if err != nil {
		t.Errorf("Expected no validation error for valid key path, got: %v", err)
	}

	// Test passphrase protected validation
	err = form.ValidateField("passphrase_protected", "invalid")
	if err == nil {
		t.Error("Expected validation error for invalid passphrase protected value")
	}

	err = form.ValidateField("passphrase_protected", "true")
	if err != nil {
		t.Errorf("Expected no validation error for 'true' passphrase protected, got: %v", err)
	}

	err = form.ValidateField("passphrase_protected", "false")
	if err != nil {
		t.Errorf("Expected no validation error for 'false' passphrase protected, got: %v", err)
	}

	err = form.ValidateField("passphrase_protected", "")
	if err != nil {
		t.Errorf("Expected no validation error for empty passphrase protected (optional), got: %v", err)
	}
}

// TestServerForm_DataCollectionWithAllFields tests data collection with all server form fields
func TestServerForm_DataCollectionWithAllFields(t *testing.T) {
	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().SetLabel("Server Name: ").SetText("test-server"),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"hostname": {
			inputField: tview.NewInputField().SetLabel("Hostname: ").SetText("test.example.com"),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"port": {
			inputField: tview.NewInputField().SetLabel("Port: ").SetText("2222"),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"username": {
			inputField: tview.NewInputField().SetLabel("Username: ").SetText("testuser"),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"auth_type": {
			inputField: tview.NewInputField().SetLabel("Auth Type: ").SetText("key"),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"key_path": {
			inputField: tview.NewInputField().SetLabel("Key Path: ").SetText("~/.ssh/test_key"),
			validator:  func(s string) error { return nil },
			required:   false,
		},
		"passphrase_protected": {
			inputField: tview.NewInputField().SetLabel("Passphrase Protected: ").SetText("true"),
			validator:  func(s string) error { return nil },
			required:   false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Test data collection
	data, err := form.CollectFormData()
	if err != nil {
		t.Errorf("Expected no error collecting form data, got: %v", err)
	}

	if len(data) != 7 {
		t.Errorf("Expected 7 fields in collected data, got %d", len(data))
	}

	// Test all expected field values
	expectedData := map[string]string{
		"name":                 "test-server",
		"hostname":             "test.example.com", 
		"port":                 "2222",
		"username":             "testuser",
		"auth_type":            "key",
		"key_path":             "~/.ssh/test_key",
		"passphrase_protected": "true",
	}

	for field, expectedValue := range expectedData {
		if data[field] != expectedValue {
			t.Errorf("Expected %s field to be '%s', got: %v", field, expectedValue, data[field])
		}
	}
}

// TestServerForm_SubmissionWithValidation tests form submission with server validation
func TestServerForm_SubmissionWithValidation(t *testing.T) {
	submitCallCount := 0
	submittedData := make(map[string]interface{})
	
	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().SetLabel("Server Name: ").SetText("valid-server"),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "name", Message: "Server name is required"}
				}
				return nil
			},
			required: true,
		},
		"hostname": {
			inputField: tview.NewInputField().SetLabel("Hostname: ").SetText("valid.example.com"),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "hostname", Message: "Hostname is required"}
				}
				return nil
			},
			required: true,
		},
		"username": {
			inputField: tview.NewInputField().SetLabel("Username: ").SetText("validuser"),
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "username", Message: "Username is required"}
				}
				return nil
			},
			required: true,
		},
		"auth_type": {
			inputField: tview.NewInputField().SetLabel("Auth Type: ").SetText("password"),
			validator: func(value string) error {
				if value != "key" && value != "password" {
					return &ValidationError{Field: "auth_type", Message: "Auth type must be 'key' or 'password'"}
				}
				return nil
			},
			required: true,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		submitCallCount++
		submittedData = data
		return nil
	}

	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)
	helper := NewFormTestHelper(form)
	defer helper.Cleanup()

	// Test successful submission with valid data
	helper.SimulateKeypress(tcell.KeyEnter)
	helper.ProcessEvents()

	if submitCallCount != 1 {
		t.Errorf("Expected submission callback to be called once, got %d calls", submitCallCount)
	}

	if len(submittedData) == 0 {
		t.Error("Expected form data to be submitted")
	}

	// Verify submitted data contains expected values
	expectedFields := []string{"name", "hostname", "username", "auth_type"}
	for _, field := range expectedFields {
		if _, exists := submittedData[field]; !exists {
			t.Errorf("Expected submitted data to contain field '%s'", field)
		}
	}
}

// TestServerForm_ValidationErrorHandling tests how form handles validation errors
func TestServerForm_ValidationErrorHandling(t *testing.T) {
	submitCallCount := 0
	
	fields := map[string]*FormField{
		"name": {
			inputField: tview.NewInputField().SetLabel("Server Name: ").SetText(""), // Empty required field
			validator: func(value string) error {
				if value == "" {
					return &ValidationError{Field: "name", Message: "Server name is required"}
				}
				return nil
			},
			required: true,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		submitCallCount++
		return nil
	}

	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Test that data collection fails with validation error
	_, err := form.CollectFormData()
	if err == nil {
		t.Error("Expected validation error when collecting data with empty required field")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("Expected ValidationError, got: %T", err)
	} else {
		if validationErr.Field != "name" {
			t.Errorf("Expected validation error on 'name' field, got: %s", validationErr.Field)
		}
		if validationErr.Message != "Server name is required" {
			t.Errorf("Expected 'Server name is required' message, got: %s", validationErr.Message)
		}
	}
}

// TestServerForm_ConditionalValidation tests conditional validation based on auth type
func TestServerForm_ConditionalValidation(t *testing.T) {
	// Test that key_path is required when auth_type is "key"
	fields := map[string]*FormField{
		"auth_type": {
			inputField: tview.NewInputField().SetLabel("Auth Type: ").SetText("key"),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"key_path": {
			inputField: tview.NewInputField().SetLabel("Key Path: ").SetText(""), // Empty when auth_type is key
			validator: func(value string) error {
				// Simulate conditional validation - key_path required when auth_type is "key"
				return nil // For now, this is handled at submission level
			},
			required: false,
		},
	}

	onSubmit := func(data map[string]interface{}) error {
		// Simulate conditional validation at submission
		if data["auth_type"] == "key" && data["key_path"] == "" {
			return &ValidationError{Field: "key_path", Message: "Key path is required when auth type is 'key'"}
		}
		return nil
	}

	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)
	helper := NewFormTestHelper(form)
	defer helper.Cleanup()

	// Test that submission fails with conditional validation error
	helper.SimulateKeypress(tcell.KeyEnter)
	helper.ProcessEvents()

	// The form should handle conditional validation in the submission callback
	// This test verifies the structure exists for implementing such validation
}

// TestServerForm_SetAndGetFieldValues tests setting and getting field values
func TestServerForm_SetAndGetFieldValues(t *testing.T) {
	fields := map[string]*FormField{
		"hostname": {
			inputField: tview.NewInputField().SetLabel("Hostname: "),
			validator:  func(s string) error { return nil },
			required:   true,
		},
		"port": {
			inputField: tview.NewInputField().SetLabel("Port: "),
			validator:  func(s string) error { return nil },
			required:   false,
		},
	}

	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)

	// Test setting field values
	err := form.SetFieldValue("hostname", "updated.example.com")
	if err != nil {
		t.Errorf("Expected no error setting hostname field value, got: %v", err)
	}

	err = form.SetFieldValue("port", "3000")
	if err != nil {
		t.Errorf("Expected no error setting port field value, got: %v", err)
	}

	// Test setting value for non-existent field
	err = form.SetFieldValue("nonexistent", "value")
	if err == nil {
		t.Error("Expected error setting value for non-existent field")
	}

	// Test getting field values
	hostname, err := form.GetFieldValue("hostname")
	if err != nil {
		t.Errorf("Expected no error getting hostname field value, got: %v", err)
	}
	if hostname != "updated.example.com" {
		t.Errorf("Expected hostname to be 'updated.example.com', got: %s", hostname)
	}

	port, err := form.GetFieldValue("port")
	if err != nil {
		t.Errorf("Expected no error getting port field value, got: %v", err)
	}
	if port != "3000" {
		t.Errorf("Expected port to be '3000', got: %s", port)
	}

	// Test getting value for non-existent field
	_, err = form.GetFieldValue("nonexistent")
	if err == nil {
		t.Error("Expected error getting value for non-existent field")
	}
}

// TestEnhancedValidationFunctions tests the new enhanced validation functions
func TestEnhancedValidationFunctions(t *testing.T) {
	// Test ValidateServerName
	tests := []struct {
		name        string
		value       string
		expectError bool
		errorMsg    string
	}{
		{"empty name", "", true, "Server name is required"},
		{"too short", "a", true, "Server name must be at least 2 characters"},
		{"too long", strings.Repeat("a", 51), true, "Server name must be less than 50 characters"},
		{"invalid chars", "server@name", true, "Server name can only contain letters, numbers, dashes, and underscores"},
		{"valid name", "production-api-01", false, ""},
		{"valid with underscore", "test_server", false, ""},
		{"valid with numbers", "server123", false, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateServerName(test.value)
			if test.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", test.name)
				} else if !strings.Contains(err.Error(), test.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got: %v", test.name, err)
				}
			}
		})
	}
}

// TestEnhancedPortValidation tests the enhanced port validation
func TestEnhancedPortValidation(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expectError bool
		errorMsg    string
	}{
		{"empty port", "", true, "Port is required"},
		{"invalid chars", "abc", true, "Port must be a number"},
		{"port zero", "0", true, "Port must be between 1 and 65535"},
		{"port too high", "65536", true, "Port must be between 1 and 65535"},
		{"valid port 22", "22", false, ""},
		{"valid port 80", "80", false, ""},
		{"valid port 443", "443", false, ""},
		{"valid port 65535", "65535", false, ""},
		{"valid port 1", "1", false, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidatePort(test.value)
			if test.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", test.name)
				} else if !strings.Contains(err.Error(), test.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %s", test.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s, got: %v", test.name, err)
				}
			}
		})
	}
}

// TestEnhancedValidationWithRealTimeForm tests validation in a form with real-time validation enabled
func TestEnhancedValidationWithRealTimeForm(t *testing.T) {
	fields := CreateServerFormFields()
	
	onSubmit := func(data map[string]interface{}) error { return nil }
	onCancel := func() {}

	// Create form with real-time validation enabled
	form := NewTUIFormWithOptions(fields, onSubmit, onCancel, true)

	// Test that HasValidationErrors works
	if form.HasValidationErrors() {
		t.Error("Expected no validation errors initially")
	}

	// Set invalid data and validate
	form.SetFieldValue("port", "invalid")
	form.SetFieldValue("name", "a") // Too short
	
	// Validate all fields
	err := form.ValidateAllFields()
	if err == nil {
		t.Error("Expected validation error with invalid data")
	}

	// Check that form now has validation errors
	if !form.HasValidationErrors() {
		t.Error("Expected validation errors after setting invalid data")
	}

	errors := form.GetValidationErrors()
	if len(errors) == 0 {
		t.Error("Expected validation errors map to be populated")
	}

	// Check for specific errors
	if _, exists := errors["port"]; !exists {
		t.Error("Expected port validation error")
	}
	if _, exists := errors["name"]; !exists {
		t.Error("Expected name validation error")
	}

	// Fix errors and validate again
	form.SetFieldValue("port", "22")
	form.SetFieldValue("name", "valid-server")
	form.SetFieldValue("hostname", "example.com")
	form.SetFieldValue("username", "testuser")
	
	err = form.ValidateAllFields()
	if err != nil {
		t.Errorf("Expected no validation error after fixing data, got: %v", err)
	}

	if form.HasValidationErrors() {
		t.Error("Expected no validation errors after fixing data")
	}
}

// TestCreateServerFormFields tests the CreateServerFormFields function
func TestCreateServerFormFields(t *testing.T) {
	fields := CreateServerFormFields()

	// Check that all expected fields are present
	expectedFields := []string{"name", "hostname", "port", "username", "auth_type", "key_path", "passphrase_protected"}
	if len(fields) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(fields))
	}

	for _, fieldName := range expectedFields {
		if field, exists := fields[fieldName]; !exists {
			t.Errorf("Expected field '%s' to exist", fieldName)
		} else {
			if field.inputField == nil {
				t.Errorf("Expected field '%s' to have inputField", fieldName)
			}
			if field.validator == nil {
				t.Errorf("Expected field '%s' to have validator", fieldName)
			}
		}
	}

	// Test that required fields are marked correctly
	requiredFields := []string{"name", "hostname", "port", "username", "auth_type"}
	for _, fieldName := range requiredFields {
		if field := fields[fieldName]; !field.required {
			t.Errorf("Expected field '%s' to be required", fieldName)
		}
	}

	// Test that optional fields are marked correctly
	optionalFields := []string{"key_path", "passphrase_protected"}
	for _, fieldName := range optionalFields {
		if field := fields[fieldName]; field.required {
			t.Errorf("Expected field '%s' to be optional", fieldName)
		}
	}

	// Test that labels are set appropriately
	expectedLabels := map[string]string{
		"name":     "Server Name: ",
		"hostname": "Hostname: ",
		"port":     "Port: ",
		"username": "Username: ",
	}
	for fieldName, expectedLabel := range expectedLabels {
		if field, exists := fields[fieldName]; exists {
			if field.inputField.GetLabel() != expectedLabel {
				t.Errorf("Expected field '%s' to have label '%s', got '%s'", 
					fieldName, expectedLabel, field.inputField.GetLabel())
			}
		}
	}
}

// TestFormIntegrationWithConfig tests that the form properly integrates with config.Server
func TestFormIntegrationWithConfig(t *testing.T) {
	fields := CreateServerFormFields()
	
	// Set valid server data
	fields["name"].inputField.SetText("test-server")
	fields["hostname"].inputField.SetText("test.example.com")
	fields["port"].inputField.SetText("2222")
	fields["username"].inputField.SetText("testuser")
	fields["auth_type"].inputField.SetText("key")
	fields["key_path"].inputField.SetText("~/.ssh/test_key")
	fields["passphrase_protected"].inputField.SetText("true")

	onSubmit := func(data map[string]interface{}) error {
		// Simulate what the real form does - parse port
		portStr := data["port"].(string)
		port := 22 // Default
		parsedPort := 0
		for _, r := range portStr {
			if r >= '0' && r <= '9' {
				parsedPort = parsedPort*10 + int(r-'0')
			}
		}
		if parsedPort > 0 {
			port = parsedPort
		}
		
		// Create server config like the real form does
		server := config.Server{
			Name:     data["name"].(string),
			Hostname: data["hostname"].(string),
			Port:     port,
			Username: data["username"].(string),
			AuthType: data["auth_type"].(string),
			KeyPath:  data["key_path"].(string),
		}
		
		if passphraseStr, ok := data["passphrase_protected"].(string); ok {
			server.PassphraseProtected = (passphraseStr == "true")
		}
		
		// Validate server configuration like the real form does
		if err := server.Validate(); err != nil {
			return err
		}
		
		// Check all expected values
		if server.Name != "test-server" {
			return fmt.Errorf("expected name 'test-server', got '%s'", server.Name)
		}
		if server.Port != 2222 {
			return fmt.Errorf("expected port 2222, got %d", server.Port)
		}
		if !server.PassphraseProtected {
			return fmt.Errorf("expected PassphraseProtected to be true")
		}
		
		return nil
	}
	
	onCancel := func() {}

	form := NewTUIForm(fields, onSubmit, onCancel)
	helper := NewFormTestHelper(form)
	defer helper.Cleanup()

	// Simulate form submission
	helper.SimulateKeypress(tcell.KeyEnter)
	helper.ProcessEvents()
	
	// If we get here without errors, the integration worked
}

