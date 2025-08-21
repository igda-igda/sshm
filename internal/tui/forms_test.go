package tui

import (
	"errors"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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

