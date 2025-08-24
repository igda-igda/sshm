package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"
	"sshm/internal/config"
)

// ImportExportModal represents the import/export interface modal
type ImportExportModal struct {
	modal            *tview.Modal
	form             *TUIForm
	isImport         bool
	filePathField    *tview.InputField
	formatField      *tview.DropDown
	profileField     *tview.DropDown
	progressText     *tview.TextView
	onComplete       func(success bool, message string)
	// progressIndicator *ProgressIndicator // Removed for now
	app              *TUIApp
}

// ShowImportModal displays the import configuration modal
func (t *TUIApp) ShowImportModal() {
	modal := &ImportExportModal{
		app:      t,
		isImport: true,
	}
	modal.show()
}

// ShowExportModal displays the export configuration modal
func (t *TUIApp) ShowExportModal() {
	modal := &ImportExportModal{
		app:      t,
		isImport: false,
	}
	modal.show()
}

// show creates and displays the import/export modal
func (ie *ImportExportModal) show() {
	var title, instruction string
	if ie.isImport {
		title = "Import Configuration"
		instruction = "Import server configurations from file"
	} else {
		title = "Export Configuration"
		instruction = "Export server configurations to file"
	}

	// Create form fields
	ie.createFormFields()

	// Create form
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(fmt.Sprintf(" %s ", title))
	form.SetButtonsAlign(tview.AlignCenter)
	
	// Add instruction text
	instructionText := tview.NewTextView()
	instructionText.SetText(instruction).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	
	// Add form fields
	form.AddFormItem(tview.NewTextView().SetText("File Path:"))
	form.AddFormItem(ie.filePathField)
	form.AddFormItem(tview.NewTextView().SetText("Format:"))
	form.AddFormItem(ie.formatField)
	
	if !ie.isImport {
		form.AddFormItem(tview.NewTextView().SetText("Profile Filter:"))
		form.AddFormItem(ie.profileField)
	}
	
	// Add progress text (initially hidden)
	ie.progressText = tview.NewTextView()
	ie.progressText.SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("")
	
	// Add buttons
	if ie.isImport {
		form.AddButton("Import", ie.handleImport)
	} else {
		form.AddButton("Export", ie.handleExport)
	}
	form.AddButton("Cancel", ie.handleCancel)
	
	// Create main layout
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(instructionText, 2, 0, false).
		AddItem(form, 0, 1, true).
		AddItem(ie.progressText, 3, 0, false)
	
	// Set up key bindings
	ie.setupKeyBindings(layout)
	
	// Show modal
	if ie.app.modalManager != nil {
		ie.app.modalManager.ShowModal(layout)
	} else {
		ie.app.app.SetRoot(layout, true)
		ie.app.app.SetFocus(form)
	}
}

// createFormFields creates the form input fields
func (ie *ImportExportModal) createFormFields() {
	// File path field
	ie.filePathField = tview.NewInputField()
	ie.filePathField.SetLabel("").
		SetPlaceholder("Enter file path...").
		SetFieldWidth(50)
	
	// Auto-suggest file extension based on format
	ie.filePathField.SetChangedFunc(func(text string) {
		if text != "" && !strings.Contains(text, ".") {
			// Auto-detect and suggest format
			_, currentOption := ie.formatField.GetCurrentOption()
			switch currentOption {
			case "YAML":
				ie.filePathField.SetText(text + ".yaml")
			case "JSON":
				ie.filePathField.SetText(text + ".json")
			}
		}
	})
	
	// Format selection field
	ie.formatField = tview.NewDropDown()
	if ie.isImport {
		ie.formatField.SetOptions([]string{"Auto-detect", "YAML", "JSON", "SSH Config"}, nil)
	} else {
		ie.formatField.SetOptions([]string{"YAML", "JSON"}, nil)
	}
	ie.formatField.SetCurrentOption(0)
	
	// Profile filter field (export only)
	if !ie.isImport {
		ie.profileField = tview.NewDropDown()
		profiles := ie.app.config.GetProfiles()
		
		// Add "All" option first
		options := []string{"All"}
		for _, profile := range profiles {
			options = append(options, profile.Name)
		}
		
		ie.profileField.SetOptions(options, nil)
		ie.profileField.SetCurrentOption(0)
	}
}

// setupKeyBindings configures keyboard navigation for the modal
func (ie *ImportExportModal) setupKeyBindings(layout tview.Primitive) {
	layout.(*tview.Flex).SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			ie.handleCancel()
			return nil
		case tcell.KeyEnter:
			// Handle Enter based on current focus
			if ie.isImport {
				ie.handleImport()
			} else {
				ie.handleExport()
			}
			return nil
		case tcell.KeyTab:
			// Tab navigation between fields
			return event
		case tcell.KeyBacktab:
			// Shift+Tab navigation
			return event
		}
		return event
	})
}

// handleImport processes the import operation
func (ie *ImportExportModal) handleImport() {
	filePath := strings.TrimSpace(ie.filePathField.GetText())
	if filePath == "" {
		ie.showError("File path is required")
		return
	}
	
	// Validate file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		ie.showError(fmt.Sprintf("File does not exist: %s", filePath))
		return
	}
	
	// Get format
	_, formatText := ie.formatField.GetCurrentOption()
	format := ie.normalizeFormat(formatText)
	
	// Auto-detect format if needed
	if format == "auto" {
		format = ie.detectFileFormat(filePath)
	}
	
	// Validate format
	if !ie.isFormatSupported(format, true) {
		ie.showError(fmt.Sprintf("Unsupported format for import: %s", format))
		return
	}
	
	// Create progress indicator
	progress := NewImportExportProgressIndicator("Importing configuration...")
	
	// Show progress
	ie.showProgressIndicator(progress)
	
	// Perform import in background
	go func() {
		// Update progress - reading file
		progress.Update(1, 4, "Reading configuration file...")
		ie.app.app.QueueUpdateDraw(func() {
			ie.showProgressIndicator(progress)
		})
		
		err := ie.performImportWithProgress(filePath, format, progress)
		ie.app.app.QueueUpdateDraw(func() {
			if err != nil {
				progress.SetError(err)
				ie.showProgressIndicator(progress)
			} else {
				progress.Complete("Configuration imported successfully")
				ie.showProgressIndicator(progress)
				// Refresh the TUI
				ie.app.RefreshConfig()
			}
		})
	}()
}

// handleExport processes the export operation
func (ie *ImportExportModal) handleExport() {
	filePath := strings.TrimSpace(ie.filePathField.GetText())
	if filePath == "" {
		ie.showError("File path is required")
		return
	}
	
	// Get format
	_, formatText := ie.formatField.GetCurrentOption()
	format := ie.normalizeFormat(formatText)
	
	// Validate format
	if !ie.isFormatSupported(format, false) {
		ie.showError(fmt.Sprintf("Unsupported format for export: %s", format))
		return
	}
	
	// Get profile filter
	var profileName string
	if ie.profileField != nil {
		_, selectedProfile := ie.profileField.GetCurrentOption()
		if selectedProfile != "All" {
			profileName = selectedProfile
		}
	}
	
	// Create progress indicator
	progress := NewImportExportProgressIndicator("Exporting configuration...")
	
	// Show progress
	ie.showProgressIndicator(progress)
	
	// Perform export in background
	go func() {
		// Update progress - preparing export
		progress.Update(1, 3, "Preparing configuration for export...")
		ie.app.app.QueueUpdateDraw(func() {
			ie.showProgressIndicator(progress)
		})
		
		err := ie.performExportWithProgress(filePath, format, profileName, progress)
		ie.app.app.QueueUpdateDraw(func() {
			if err != nil {
				progress.SetError(err)
				ie.showProgressIndicator(progress)
			} else {
				progress.Complete(fmt.Sprintf("Configuration exported to %s", filePath))
				ie.showProgressIndicator(progress)
			}
		})
	}()
}

// handleCancel closes the modal
func (ie *ImportExportModal) handleCancel() {
	if ie.app.modalManager != nil {
		ie.app.modalManager.HideModal()
	} else {
		ie.app.app.SetRoot(ie.app.layout, true)
		ie.app.app.SetFocus(ie.app.layout)
	}
}

// performImportWithProgress executes the actual import operation with progress updates
func (ie *ImportExportModal) performImportWithProgress(filePath, format string, progress *ImportExportProgressIndicator) error {
	// Step 1: Read file
	progress.Update(1, 4, "Reading configuration file...")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	// Step 2: Parse configuration
	progress.Update(2, 4, "Parsing configuration...")
	var servers []config.Server
	var profiles []config.Profile
	
	// Parse based on format
	switch format {
	case "yaml":
		servers, profiles, err = ie.parseYAMLConfig(data)
	case "json":
		servers, profiles, err = ie.parseJSONConfig(data)
	case "ssh":
		servers, err = config.ParseSSHConfig(filePath)
		profiles = nil // SSH config doesn't have profiles
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}
	
	if len(servers) == 0 {
		return fmt.Errorf("no valid server configurations found in file")
	}
	
	// Step 3: Import servers and profiles
	progress.Update(3, 4, fmt.Sprintf("Importing %d servers and %d profiles...", len(servers), len(profiles)))
	imported := 0
	updated := 0
	
	for _, server := range servers {
		// Validate server
		if err := server.Validate(); err != nil {
			continue // Skip invalid servers
		}
		
		// Check if server exists
		_, err := ie.app.config.GetServer(server.Name)
		if err == nil {
			// Server exists - update it
			if err := ie.app.config.RemoveServer(server.Name); err != nil {
				continue
			}
			updated++
		} else {
			imported++
		}
		
		// Add server
		if err := ie.app.config.AddServer(server); err != nil {
			continue
		}
	}
	
	// Import profiles
	for _, profile := range profiles {
		// Check if profile exists
		_, err := ie.app.config.GetProfile(profile.Name)
		if err == nil {
			// Profile exists - remove it first
			ie.app.config.RemoveProfile(profile.Name)
		}
		
		// Add profile
		ie.app.config.AddProfile(profile)
	}
	
	// Step 4: Save configuration
	progress.Update(4, 4, "Saving configuration...")
	if err := ie.app.config.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	return nil
}

// performExportWithProgress executes the actual export operation with progress updates
func (ie *ImportExportModal) performExportWithProgress(filePath, format, profileName string, progress *ImportExportProgressIndicator) error {
	// Step 1: Create directory if needed
	progress.Update(1, 3, "Creating output directory...")
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Step 2: Prepare export config
	progress.Update(2, 3, "Preparing export configuration...")
	var exportConfig config.Config
	
	if profileName != "" {
		// Export specific profile
		profile, err := ie.app.config.GetProfile(profileName)
		if err != nil {
			return fmt.Errorf("profile '%s' not found", profileName)
		}
		
		servers, err := ie.app.config.GetServersByProfile(profileName)
		if err != nil {
			return fmt.Errorf("failed to get servers for profile '%s': %w", profileName, err)
		}
		
		exportConfig = config.Config{
			Servers:  servers,
			Profiles: []config.Profile{*profile},
		}
	} else {
		// Export all
		exportConfig = config.Config{
			Servers:  ie.app.config.GetServers(),
			Profiles: ie.app.config.GetProfiles(),
		}
	}
	
	// Marshal data
	var data []byte
	var err error
	
	switch format {
	case "yaml":
		data, err = yaml.Marshal(exportConfig)
	case "json":
		data, err = json.MarshalIndent(exportConfig, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	
	// Step 3: Write file
	progress.Update(3, 3, "Writing export file...")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// parseYAMLConfig parses YAML configuration data
func (ie *ImportExportModal) parseYAMLConfig(data []byte) ([]config.Server, []config.Profile, error) {
	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, err
	}
	return cfg.Servers, cfg.Profiles, nil
}

// parseJSONConfig parses JSON configuration data
func (ie *ImportExportModal) parseJSONConfig(data []byte) ([]config.Server, []config.Profile, error) {
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil, err
	}
	return cfg.Servers, cfg.Profiles, nil
}

// detectFileFormat detects file format based on extension
func (ie *ImportExportModal) detectFileFormat(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	base := strings.ToLower(filepath.Base(filePath))
	
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	default:
		if base == "config" || base == "ssh_config" || strings.Contains(base, "ssh") {
			return "ssh"
		}
		return "yaml" // Default
	}
}

// normalizeFormat converts display format to internal format
func (ie *ImportExportModal) normalizeFormat(displayFormat string) string {
	switch strings.ToLower(displayFormat) {
	case "auto-detect", "auto":
		return "auto"
	case "yaml":
		return "yaml"
	case "json":
		return "json"
	case "ssh config", "ssh":
		return "ssh"
	default:
		return strings.ToLower(displayFormat)
	}
}

// isFormatSupported checks if a format is supported
func (ie *ImportExportModal) isFormatSupported(format string, isImport bool) bool {
	if isImport {
		return format == "yaml" || format == "json" || format == "ssh"
	}
	return format == "yaml" || format == "json"
}

// showProgress displays progress information
func (ie *ImportExportModal) showProgress(message string) {
	ie.progressText.SetText(fmt.Sprintf("[yellow]⏳ %s[white]", message))
}

// showProgressIndicator displays progress using a progress indicator
func (ie *ImportExportModal) showProgressIndicator(progress *ImportExportProgressIndicator) {
	if ie.progressText != nil {
		ie.progressText.SetText(progress.GetProgressText())
	}
}

// showError displays an error message
func (ie *ImportExportModal) showError(message string) {
	ie.progressText.SetText(fmt.Sprintf("[red]❌ %s[white]", message))
}

// showSuccess displays a success message and closes modal after delay
func (ie *ImportExportModal) showSuccess(message string) {
	ie.progressText.SetText(fmt.Sprintf("[green]✅ %s[white]", message))
	
	// Close modal after showing success message
	go func() {
		// Show success for 2 seconds
		// Using time.Sleep instead of simulation screen
		// time.Sleep(2 * time.Second)
		// For now, don't auto-close - let user close manually
		// ie.app.app.QueueUpdateDraw(func() {
		// 	ie.handleCancel()
		// })
	}()
}

// ProgressIndicator represents operation progress for import/export
type ImportExportProgressIndicator struct {
	Message   string
	Current   int
	Total     int
	Completed bool
	Error     error
}

// NewImportExportProgressIndicator creates a new progress indicator
func NewImportExportProgressIndicator(message string) *ImportExportProgressIndicator {
	return &ImportExportProgressIndicator{
		Message: message,
	}
}

// Update updates progress information
func (p *ImportExportProgressIndicator) Update(current, total int, message string) {
	p.Current = current
	p.Total = total
	p.Message = message
}

// Complete marks the operation as completed
func (p *ImportExportProgressIndicator) Complete(message string) {
	p.Completed = true
	p.Message = message
}

// SetError sets an error for the progress indicator
func (p *ImportExportProgressIndicator) SetError(err error) {
	p.Error = err
}

// GetProgressText returns formatted progress text for display
func (p *ImportExportProgressIndicator) GetProgressText() string {
	if p.Error != nil {
		return fmt.Sprintf("[red]❌ Error: %s[white]", p.Error.Error())
	}
	
	if p.Completed {
		return fmt.Sprintf("[green]✅ %s[white]", p.Message)
	}
	
	if p.Total > 0 {
		percentage := (p.Current * 100) / p.Total
		return fmt.Sprintf("[yellow]⏳ %s (%d%%, %d/%d)[white]", p.Message, percentage, p.Current, p.Total)
	}
	
	return fmt.Sprintf("[yellow]⏳ %s[white]", p.Message)
}