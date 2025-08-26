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
	var title, instruction, actionIcon string
	if ie.isImport {
		title = "Import Configuration"
		instruction = "Select a configuration file to import server settings"
		actionIcon = "📥"
	} else {
		title = "Export Configuration"
		instruction = "Choose location and format to export your server configurations"
		actionIcon = "📤"
	}

	// Create professional modal layout
	ie.createProfessionalModal(title, instruction, actionIcon)
}

// createProfessionalModal creates a compact, professional-looking modal
func (ie *ImportExportModal) createProfessionalModal(title, instruction, actionIcon string) {
	// Create form fields
	ie.createFormFields()

	// Create compact header with icon and title
	headerText := tview.NewTextView()
	headerText.SetText(fmt.Sprintf("[aqua::b]%s %s[lightgray] - %s[white]", actionIcon, title, instruction)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Create compact form fields layout
	fieldsLayout := ie.createCompactFieldsLayout()
	
	// Create compact buttons layout
	buttonsLayout := ie.createCompactButtonsLayout()
	
	// Create progress text (initially hidden)
	ie.progressText = tview.NewTextView()
	ie.progressText.SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("")
	
	// Create main content layout - much more compact
	contentLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(headerText, 1, 0, false).
		AddItem(fieldsLayout, 0, 1, true).
		AddItem(buttonsLayout, 1, 0, false).
		AddItem(ie.progressText, 1, 0, false)
	
	// Create border with professional styling - smaller size
	border := tview.NewFlex()
	border.SetBorder(true).
		SetBorderColor(tcell.ColorAqua).
		SetTitle(fmt.Sprintf(" %s ", title)).
		SetTitleColor(tcell.ColorAqua)
	border.AddItem(contentLayout, 0, 1, true)
	
	// Set up key bindings
	ie.setupKeyBindings(border)
	
	// Show modal with smaller size
	if ie.app.modalManager != nil {
		ie.app.modalManager.ShowModal(border)
	} else {
		ie.app.app.SetRoot(border, true)
		ie.app.app.SetFocus(fieldsLayout)
	}
}

// createCompactFieldsLayout creates a more compact form fields layout
func (ie *ImportExportModal) createCompactFieldsLayout() *tview.Flex {
	// Create file path section - single line
	filePathRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewTextView().SetText("[yellow]📁 File:[white]").SetDynamicColors(true), 10, 0, false).
		AddItem(ie.filePathField, 0, 1, true).
		AddItem(tview.NewButton("📂").SetSelectedFunc(ie.showFileSystemBrowser), 4, 0, false)
	
	// Create format section - single line
	formatRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewTextView().SetText("[yellow]📋 Format:[white]").SetDynamicColors(true), 10, 0, false).
		AddItem(ie.formatField, 0, 1, false).
		AddItem(tview.NewBox(), 4, 0, false) // Spacer to align with browse button above
	
	// Create main fields layout - very compact
	fieldsLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(filePathRow, 1, 0, true).
		AddItem(formatRow, 1, 0, false)
	
	// Add profile section for export (if needed)
	if !ie.isImport {
		profileRow := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewTextView().SetText("[yellow]🏷️  Profile:[white]").SetDynamicColors(true), 10, 0, false).
			AddItem(ie.profileField, 0, 1, false).
			AddItem(tview.NewBox(), 4, 0, false) // Spacer
		
		fieldsLayout.AddItem(profileRow, 1, 0, false)
	}
	
	return fieldsLayout
}

// createFieldsLayout creates the form fields with professional layout
func (ie *ImportExportModal) createFieldsLayout() *tview.Flex {
	// Create file path section
	filePathLabel := tview.NewTextView()
	filePathLabel.SetText("[yellow::b]📁 File Path:[white::-]").
		SetDynamicColors(true)
	
	filePathLayout := ie.createFilePathInputWithBrowser()
	
	// Create format section
	formatLabel := tview.NewTextView()
	formatLabel.SetText("[yellow::b]📋 Format:[white::-]").
		SetDynamicColors(true)
	
	// Create profile section (export only)
	var profileSection *tview.Flex
	if !ie.isImport {
		profileLabel := tview.NewTextView()
		profileLabel.SetText("[yellow::b]🏷️  Profile Filter:[white::-]").
			SetDynamicColors(true)
		
		profileSection = tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(profileLabel, 1, 0, false).
			AddItem(ie.profileField, 1, 0, false)
	}
	
	// Create main fields layout
	fieldsLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(filePathLabel, 1, 0, false).
		AddItem(filePathLayout, 2, 0, false).
		AddItem(tview.NewTextView(), 1, 0, false). // Spacer
		AddItem(formatLabel, 1, 0, false).
		AddItem(ie.formatField, 1, 0, false)
	
	if profileSection != nil {
		fieldsLayout.
			AddItem(tview.NewTextView(), 1, 0, false). // Spacer
			AddItem(profileSection, 2, 0, false)
	}
	
	return fieldsLayout
}

// createFilePathInputWithBrowser creates file path input with browse functionality
func (ie *ImportExportModal) createFilePathInputWithBrowser() *tview.Flex {
	// Create browse button
	browseButton := tview.NewButton("📂 Browse")
	browseButton.SetSelectedFunc(func() {
		ie.showFileSystemBrowser()
	})
	
	// Style the button
	browseButton.SetBackgroundColor(tcell.ColorDarkBlue)
	
	// Create layout with input field and browse button
	pathLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ie.filePathField, 0, 4, true).
		AddItem(browseButton, 12, 0, false)
	
	return pathLayout
}

// createCompactButtonsLayout creates compact action buttons
func (ie *ImportExportModal) createCompactButtonsLayout() *tview.Flex {
	// Create action button
	var actionButton *tview.Button
	if ie.isImport {
		actionButton = tview.NewButton("📥 Import")
		actionButton.SetSelectedFunc(ie.handleImport)
	} else {
		actionButton = tview.NewButton("📤 Export")
		actionButton.SetSelectedFunc(ie.handleExport)
	}
	actionButton.SetBackgroundColor(tcell.ColorDarkGreen)
	
	// Create cancel button
	cancelButton := tview.NewButton("❌ Cancel")
	cancelButton.SetSelectedFunc(ie.handleCancel)
	cancelButton.SetBackgroundColor(tcell.ColorDarkRed)
	
	// Create compact button layout
	buttonLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false). // Spacer
		AddItem(actionButton, 12, 0, false).
		AddItem(tview.NewBox(), 2, 0, false). // Small spacer
		AddItem(cancelButton, 12, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)  // Spacer
	
	return buttonLayout
}

// createButtonsLayout creates the action buttons with professional styling
func (ie *ImportExportModal) createButtonsLayout() *tview.Flex {
	// Create action button
	var actionButton *tview.Button
	if ie.isImport {
		actionButton = tview.NewButton("📥 Import Configuration")
		actionButton.SetSelectedFunc(ie.handleImport)
		actionButton.SetBackgroundColor(tcell.ColorDarkGreen)
	} else {
		actionButton = tview.NewButton("📤 Export Configuration")
		actionButton.SetSelectedFunc(ie.handleExport)
		actionButton.SetBackgroundColor(tcell.ColorDarkGreen)
	}
	
	// Create cancel button
	cancelButton := tview.NewButton("❌ Cancel")
	cancelButton.SetSelectedFunc(ie.handleCancel)
	cancelButton.SetBackgroundColor(tcell.ColorDarkRed)
	
	// Create button layout
	buttonLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false). // Spacer
		AddItem(actionButton, 24, 0, false).
		AddItem(tview.NewBox(), 2, 0, false). // Spacer
		AddItem(cancelButton, 14, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)  // Spacer
	
	return buttonLayout
}

// showFileSystemBrowser shows a file system browser modal
func (ie *ImportExportModal) showFileSystemBrowser() {
	browser := NewFileSystemBrowser(ie.isImport, func(selectedPath string) {
		if selectedPath != "" {
			ie.filePathField.SetText(selectedPath)
			// Auto-detect format based on file extension
			if ie.isImport {
				format := ie.detectFileFormat(selectedPath)
				ie.setFormatSelection(format)
			}
		}
	})
	
	browser.Show(ie.app)
}

// setFormatSelection sets the format dropdown based on detected format
func (ie *ImportExportModal) setFormatSelection(format string) {
	if ie.formatField == nil {
		return
	}
	
	// Map internal format to dropdown options
	var targetOption string
	switch format {
	case "yaml":
		targetOption = "YAML"
	case "json":
		targetOption = "JSON"
	case "ssh":
		targetOption = "SSH Config"
	default:
		return // Don't change selection
	}
	
	// Find and set the option - tview DropDown doesn't have GetOption, so we'll set based on known options
	switch targetOption {
	case "YAML":
		if ie.isImport {
			ie.formatField.SetCurrentOption(1) // Auto-detect(0), YAML(1), JSON(2), SSH Config(3)
		} else {
			ie.formatField.SetCurrentOption(0) // YAML(0), JSON(1)
		}
	case "JSON":
		if ie.isImport {
			ie.formatField.SetCurrentOption(2) // Auto-detect(0), YAML(1), JSON(2), SSH Config(3)
		} else {
			ie.formatField.SetCurrentOption(1) // YAML(0), JSON(1)
		}
	case "SSH Config":
		if ie.isImport {
			ie.formatField.SetCurrentOption(3) // Auto-detect(0), YAML(1), JSON(2), SSH Config(3)
		}
	}
}

// setupFuzzyFinder sets up visual fuzzy finder functionality like fzf
func (ie *ImportExportModal) setupFuzzyFinder() {
	var fuzzyModal *tview.Modal
	var fuzzyList *tview.List
	var suggestions []string
	var originalText string
	
	// Create the visual fuzzy finder interface
	ie.filePathField.SetChangedFunc(func(text string) {
		// Store original text for potential restoration
		if originalText == "" {
			originalText = text
		}
		
		// Clear suggestions if text is too short
		if len(strings.TrimSpace(text)) < 2 {
			if fuzzyModal != nil {
				ie.hideFuzzyFinder(fuzzyModal)
				fuzzyModal = nil
			}
			return
		}
		
		// Get directory suggestions
		suggestions = ie.getFuzzyFinderSuggestions(text)
		if len(suggestions) > 0 {
			ie.showVisualFuzzyFinder(text, suggestions, &fuzzyModal, &fuzzyList)
		} else if fuzzyModal != nil {
			ie.hideFuzzyFinder(fuzzyModal)
			fuzzyModal = nil
		}
	})
	
	// Set up key handling for fuzzy finder
	ie.filePathField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle keys when fuzzy finder is visible
		if fuzzyModal != nil && fuzzyList != nil {
			switch event.Key() {
			case tcell.KeyDown, tcell.KeyCtrlN:
				// Navigate down in suggestions
				currentItem := fuzzyList.GetCurrentItem()
				if currentItem < len(suggestions)-1 {
					fuzzyList.SetCurrentItem(currentItem + 1)
				}
				return nil
			case tcell.KeyUp, tcell.KeyCtrlP:
				// Navigate up in suggestions
				currentItem := fuzzyList.GetCurrentItem()
				if currentItem > 0 {
					fuzzyList.SetCurrentItem(currentItem - 1)
				}
				return nil
			case tcell.KeyTab, tcell.KeyEnter:
				// Select current suggestion
				if len(suggestions) > 0 {
					currentItem := fuzzyList.GetCurrentItem()
					if currentItem >= 0 && currentItem < len(suggestions) {
						ie.filePathField.SetText(suggestions[currentItem])
						ie.hideFuzzyFinder(fuzzyModal)
						fuzzyModal = nil
					}
				}
				return nil
			case tcell.KeyEscape:
				// Cancel fuzzy finder
				ie.hideFuzzyFinder(fuzzyModal)
				fuzzyModal = nil
				originalText = ""
				return nil
			}
		}
		
		return event
	})
}

// getFuzzyFinderSuggestions returns file/directory suggestions based on current input
func (ie *ImportExportModal) getFuzzyFinderSuggestions(input string) []string {
	var suggestions []string
	
	// Determine the directory to search and the search term
	var searchDir, searchTerm string
	if strings.Contains(input, "/") {
		// Input contains path separators
		searchDir = filepath.Dir(input)
		searchTerm = filepath.Base(input)
	} else {
		// Search in current directory or home directory
		searchDir = "."
		if homeDir, err := os.UserHomeDir(); err == nil {
			searchDir = homeDir
		}
		searchTerm = input
	}
	
	// If searchDir is relative and doesn't exist, try to make it absolute
	if !filepath.IsAbs(searchDir) {
		if absDir, err := filepath.Abs(searchDir); err == nil {
			searchDir = absDir
		}
	}
	
	// Read directory
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return suggestions
	}
	
	// Filter entries based on search term and requirements
	searchTermLower := strings.ToLower(searchTerm)
	for _, entry := range entries {
		name := entry.Name()
		nameLower := strings.ToLower(name)
		
		// Skip hidden files unless specifically searching for them
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(searchTerm, ".") {
			continue
		}
		
		// Check if name matches search term (fuzzy matching)
		if ie.fuzzyMatch(nameLower, searchTermLower) {
			var suggestion string
			if strings.Contains(input, "/") {
				// Maintain the directory path
				suggestion = filepath.Join(searchDir, name)
			} else {
				suggestion = name
			}
			
			// For import mode, prefer files with supported extensions
			if ie.isImport && !entry.IsDir() {
				ext := strings.ToLower(filepath.Ext(name))
				supportedExts := []string{".yaml", ".yml", ".json", ".config"}
				supported := false
				for _, supportedExt := range supportedExts {
					if ext == supportedExt || name == "config" {
						supported = true
						break
					}
				}
				if supported {
					// Add supported files first
					suggestions = append([]string{suggestion}, suggestions...)
				}
			} else if entry.IsDir() {
				// Add directories
				suggestions = append(suggestions, suggestion+"/")
			} else if !ie.isImport {
				// For export mode, include all files
				suggestions = append(suggestions, suggestion)
			}
		}
	}
	
	// Limit suggestions to avoid overwhelming the user
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}
	
	return suggestions
}

// showVisualFuzzyFinder displays an fzf-like visual popup with suggestions
func (ie *ImportExportModal) showVisualFuzzyFinder(query string, suggestions []string, fuzzyModal **tview.Modal, fuzzyList **tview.List) {
	// Limit suggestions to show (like fzf)
	maxSuggestions := 10
	displaySuggestions := suggestions
	if len(suggestions) > maxSuggestions {
		displaySuggestions = suggestions[:maxSuggestions]
	}
	
	// Create the text display similar to fzf
	var displayText strings.Builder
	displayText.WriteString(fmt.Sprintf("[white]> %s[::]\n\n", query))
	
	for i, suggestion := range displaySuggestions {
		if i == 0 {
			// Highlight the first (selected) item
			displayText.WriteString(fmt.Sprintf("[black:darkblue]  %s  [::]\n", suggestion))
		} else {
			displayText.WriteString(fmt.Sprintf("  %s\n", suggestion))
		}
	}
	
	// Add footer info like fzf
	total := len(suggestions)
	shown := len(displaySuggestions)
	displayText.WriteString(fmt.Sprintf("\n[gray]  %d/%d[::]\n", shown, total))
	
	// Create or update the modal
	if *fuzzyModal == nil {
		modal := tview.NewModal()
		modal.SetText(displayText.String())
		modal.SetTitle("fzf - File Finder") 
		modal.SetBackgroundColor(tcell.ColorBlack)
		modal.SetBorder(true)
		modal.SetBorderColor(tcell.ColorGray)
		modal.AddButtons([]string{"Select", "Cancel"})
		modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Select" && len(suggestions) > 0 {
				ie.filePathField.SetText(suggestions[0])
			}
			ie.hideFuzzyFinder(modal)
			*fuzzyModal = nil
		})
		
		*fuzzyModal = modal
		
		// Show the modal
		if ie.app.modalManager != nil {
			ie.app.modalManager.ShowModal(*fuzzyModal)
		}
	} else {
		// Update existing modal
		(*fuzzyModal).SetText(displayText.String())
		(*fuzzyModal).SetTitle(fmt.Sprintf("fzf - File Finder (%d/%d)", shown, total))
	}
}

// highlightMatches highlights matching characters in the suggestion similar to fzf
func (ie *ImportExportModal) highlightMatches(text, pattern string) string {
	if len(pattern) == 0 {
		return text
	}
	
	// Simple highlighting - make matched characters yellow like fzf
	patternLower := strings.ToLower(pattern)
	
	result := text
	for _, char := range patternLower {
		charStr := string(char)
		if idx := strings.Index(strings.ToLower(result), charStr); idx != -1 {
			// Replace first occurrence with highlighted version
			before := result[:idx]
			after := result[idx+1:]
			highlighted := fmt.Sprintf("[yellow]%c[white]", result[idx])
			result = before + highlighted + after
		}
	}
	
	return result
}

// hideFuzzyFinder hides the visual fuzzy finder modal
func (ie *ImportExportModal) hideFuzzyFinder(fuzzyModal *tview.Modal) {
	if fuzzyModal != nil && ie.app.modalManager != nil {
		ie.app.modalManager.HideModal()
	}
}

// fuzzyMatch performs simple fuzzy matching
func (ie *ImportExportModal) fuzzyMatch(text, pattern string) bool {
	if pattern == "" {
		return true
	}
	if text == "" {
		return false
	}
	
	// Simple fuzzy matching: all characters of pattern must appear in text in order
	textIndex := 0
	for _, patternChar := range pattern {
		found := false
		for textIndex < len(text) {
			if rune(text[textIndex]) == patternChar {
				found = true
				textIndex++
				break
			}
			textIndex++
		}
		if !found {
			return false
		}
	}
	return true
}

// createFormFields creates the form input fields
func (ie *ImportExportModal) createFormFields() {
	// File path field with fuzzy finder functionality
	ie.filePathField = tview.NewInputField()
	ie.filePathField.SetLabel("").
		SetPlaceholder("Enter file path...").
		SetFieldWidth(50)
	
	// Set up fuzzy finder functionality (this will set up the ChangedFunc internally)
	ie.setupFuzzyFinder()
	
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

// FileSystemBrowser represents a file system browser modal
type FileSystemBrowser struct {
	isImport        bool
	onFileSelected  func(string)
	currentPath     string
	fileList        *tview.Table
	pathDisplay     *tview.TextView
	selectedIndex   int
	entries         []FileEntry
}

// FileEntry represents a file or directory entry
type FileEntry struct {
	Name      string
	Path      string
	IsDir     bool
	Size      int64
	Extension string
}

// NewFileSystemBrowser creates a new file system browser
func NewFileSystemBrowser(isImport bool, onFileSelected func(string)) *FileSystemBrowser {
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = "/"
	}
	
	return &FileSystemBrowser{
		isImport:       isImport,
		onFileSelected: onFileSelected,
		currentPath:    homeDir,
		selectedIndex:  0,
	}
}

// Show displays the file system browser modal
func (fb *FileSystemBrowser) Show(app *TUIApp) {
	// Create path display
	fb.pathDisplay = tview.NewTextView()
	fb.pathDisplay.SetDynamicColors(true).
		SetText(fmt.Sprintf("[aqua::b]📁 Current: [white::-]%s", fb.currentPath))
	
	// Create file list table
	fb.fileList = tview.NewTable()
	fb.fileList.SetBorder(true).SetTitle(" 📂 File Browser ")
	fb.fileList.SetSelectable(true, false)
	fb.fileList.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite))
	
	// Load initial directory
	fb.loadDirectory()
	
	// Create instruction text
	var instruction string
	if fb.isImport {
		instruction = "[lightgray]Navigate: [yellow]↑/↓[lightgray] • Select File: [yellow]Enter[lightgray] • Up Directory: [yellow]Backspace[lightgray] • Cancel: [yellow]Esc[white]"
	} else {
		instruction = "[lightgray]Navigate: [yellow]↑/↓[lightgray] • Select Directory: [yellow]Enter[lightgray] • Up Directory: [yellow]Backspace[lightgray] • Cancel: [yellow]Esc[white]"
	}
	
	instructionText := tview.NewTextView()
	instructionText.SetText(instruction).
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	
	// Create buttons for export mode
	var buttonsLayout *tview.Flex
	if !fb.isImport {
		// Create filename input for export
		filenameInput := tview.NewInputField()
		filenameInput.SetLabel("💾 Filename: ").
			SetPlaceholder("config").
			SetFieldWidth(30)
		
		saveButton := tview.NewButton("💾 Save Here")
		saveButton.SetBackgroundColor(tcell.ColorDarkGreen)
		saveButton.SetSelectedFunc(func() {
			filename := strings.TrimSpace(filenameInput.GetText())
			if filename == "" {
				filename = "config.yaml"
			}
			if !strings.Contains(filename, ".") {
				filename += ".yaml"
			}
			selectedPath := filepath.Join(fb.currentPath, filename)
			fb.onFileSelected(selectedPath)
			app.modalManager.HideModal()
		})
		
		cancelButton := tview.NewButton("❌ Cancel")
		cancelButton.SetBackgroundColor(tcell.ColorDarkRed)
		cancelButton.SetSelectedFunc(func() {
			app.modalManager.HideModal()
		})
		
		filenameLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(filenameInput, 0, 1, false)
		
		buttonLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(saveButton, 14, 0, false).
			AddItem(tview.NewBox(), 2, 0, false).
			AddItem(cancelButton, 12, 0, false).
			AddItem(tview.NewBox(), 0, 1, false)
		
		buttonsLayout = tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(filenameLayout, 1, 0, false).
			AddItem(tview.NewBox(), 1, 0, false).
			AddItem(buttonLayout, 1, 0, false)
	}
	
	// Create main layout
	var mainLayout *tview.Flex
	if buttonsLayout != nil {
		mainLayout = tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(fb.pathDisplay, 1, 0, false).
			AddItem(fb.fileList, 0, 1, true).
			AddItem(tview.NewBox(), 1, 0, false).
			AddItem(buttonsLayout, 4, 0, false).
			AddItem(tview.NewBox(), 1, 0, false).
			AddItem(instructionText, 2, 0, false)
	} else {
		mainLayout = tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(fb.pathDisplay, 1, 0, false).
			AddItem(fb.fileList, 0, 1, true).
			AddItem(tview.NewBox(), 1, 0, false).
			AddItem(instructionText, 2, 0, false)
	}
	
	// Create border
	border := tview.NewFlex()
	border.SetBorder(true).
		SetBorderColor(tcell.ColorAqua).
		SetTitle(" 📂 File System Browser ").
		SetTitleColor(tcell.ColorAqua)
	border.AddItem(mainLayout, 0, 1, true)
	
	// Set up key bindings
	fb.setupKeyBindings(border, app)
	
	// Show modal
	if app.modalManager != nil {
		app.modalManager.ShowModal(border)
	} else {
		app.app.SetRoot(border, true)
		app.app.SetFocus(fb.fileList)
	}
}

// loadDirectory loads the current directory contents
func (fb *FileSystemBrowser) loadDirectory() {
	// Clear existing entries
	fb.entries = nil
	fb.fileList.Clear()
	
	// Read directory
	entries, err := os.ReadDir(fb.currentPath)
	if err != nil {
		// Show error in table
		fb.fileList.SetCell(0, 0, tview.NewTableCell("❌").SetTextColor(tcell.ColorRed))
		fb.fileList.SetCell(0, 1, tview.NewTableCell("Error reading directory").SetTextColor(tcell.ColorRed))
		fb.fileList.SetCell(0, 2, tview.NewTableCell(err.Error()).SetTextColor(tcell.ColorRed))
		return
	}
	
	// Add parent directory entry (unless at root)
	if fb.currentPath != "/" && fb.currentPath != "" {
		fb.entries = append(fb.entries, FileEntry{
			Name:  "..",
			Path:  filepath.Dir(fb.currentPath),
			IsDir: true,
		})
	}
	
	// Add directory entries
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		// Skip hidden files/directories
		if strings.HasPrefix(entry.Name(), ".") && entry.Name() != ".." {
			continue
		}
		
		fileEntry := FileEntry{
			Name:      entry.Name(),
			Path:      filepath.Join(fb.currentPath, entry.Name()),
			IsDir:     entry.IsDir(),
			Size:      info.Size(),
			Extension: strings.ToLower(filepath.Ext(entry.Name())),
		}
		
		// Filter files for import mode
		if fb.isImport && !fileEntry.IsDir {
			// Only show supported file types
			supportedExts := []string{".yaml", ".yml", ".json", ".config"}
			supported := false
			for _, ext := range supportedExts {
				if fileEntry.Extension == ext || fileEntry.Name == "config" {
					supported = true
					break
				}
			}
			if !supported {
				continue
			}
		}
		
		fb.entries = append(fb.entries, fileEntry)
	}
	
	// Update table display
	fb.updateTableDisplay()
	
	// Update path display
	fb.pathDisplay.SetText(fmt.Sprintf("[aqua::b]📁 Current: [white::-]%s", fb.currentPath))
	
	// Reset selection
	fb.selectedIndex = 0
	if len(fb.entries) > 0 {
		fb.fileList.Select(0, 0)
	}
}

// updateTableDisplay updates the file list table
func (fb *FileSystemBrowser) updateTableDisplay() {
	fb.fileList.Clear()
	
	for i, entry := range fb.entries {
		var icon, sizeStr string
		var nameColor tcell.Color = tcell.ColorWhite
		
		if entry.IsDir {
			if entry.Name == ".." {
				icon = "⬆️"
				sizeStr = "<UP>"
			} else {
				icon = "📁"
				sizeStr = "<DIR>"
			}
			nameColor = tcell.ColorLightBlue
		} else {
			// File icon based on extension
			switch entry.Extension {
			case ".yaml", ".yml":
				icon = "📄"
				nameColor = tcell.ColorYellow
			case ".json":
				icon = "📋"
				nameColor = tcell.ColorGreen
			case ".config", "":
				icon = "⚙️"
				nameColor = tcell.ColorAqua
			default:
				icon = "📄"
				nameColor = tcell.ColorWhite
			}
			
			// Format file size
			if entry.Size < 1024 {
				sizeStr = fmt.Sprintf("%d B", entry.Size)
			} else if entry.Size < 1024*1024 {
				sizeStr = fmt.Sprintf("%.1f KB", float64(entry.Size)/1024)
			} else {
				sizeStr = fmt.Sprintf("%.1f MB", float64(entry.Size)/(1024*1024))
			}
		}
		
		fb.fileList.SetCell(i, 0, tview.NewTableCell(icon).SetAlign(tview.AlignCenter))
		fb.fileList.SetCell(i, 1, tview.NewTableCell(entry.Name).SetTextColor(nameColor))
		fb.fileList.SetCell(i, 2, tview.NewTableCell(sizeStr).SetTextColor(tcell.ColorGray).SetAlign(tview.AlignRight))
	}
}

// setupKeyBindings configures keyboard navigation for the browser
func (fb *FileSystemBrowser) setupKeyBindings(layout *tview.Flex, app *TUIApp) {
	layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			// Cancel and close browser
			if app.modalManager != nil {
				app.modalManager.HideModal()
			}
			return nil
		case tcell.KeyEnter:
			// Select current item
			fb.selectCurrentItem(app)
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			// Go up one directory
			if fb.currentPath != "/" && fb.currentPath != "" {
				fb.currentPath = filepath.Dir(fb.currentPath)
				fb.loadDirectory()
			}
			return nil
		case tcell.KeyUp:
			// Move selection up
			fb.moveSelection(-1)
			return nil
		case tcell.KeyDown:
			// Move selection down
			fb.moveSelection(1)
			return nil
		}
		
		// Handle character keys
		switch event.Rune() {
		case 'q', 'Q':
			// Quick quit
			if app.modalManager != nil {
				app.modalManager.HideModal()
			}
			return nil
		}
		
		return event
	})
}

// moveSelection moves the selection up or down
func (fb *FileSystemBrowser) moveSelection(direction int) {
	if len(fb.entries) == 0 {
		return
	}
	
	newIndex := fb.selectedIndex + direction
	if newIndex < 0 {
		newIndex = len(fb.entries) - 1
	} else if newIndex >= len(fb.entries) {
		newIndex = 0
	}
	
	fb.selectedIndex = newIndex
	fb.fileList.Select(newIndex, 0)
}

// selectCurrentItem handles selection of the current item
func (fb *FileSystemBrowser) selectCurrentItem(app *TUIApp) {
	if fb.selectedIndex < 0 || fb.selectedIndex >= len(fb.entries) {
		return
	}
	
	entry := fb.entries[fb.selectedIndex]
	
	if entry.IsDir {
		// Navigate into directory
		fb.currentPath = entry.Path
		fb.loadDirectory()
	} else {
		// Select file (import mode only)
		if fb.isImport {
			fb.onFileSelected(entry.Path)
			if app.modalManager != nil {
				app.modalManager.HideModal()
			}
		}
	}
}