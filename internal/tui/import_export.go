package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
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

	// Create centered professional modal layout
	ie.createCenteredModal(title, instruction, actionIcon)
}

// createCenteredModal creates a compact, professional-looking modal centered in screen
func (ie *ImportExportModal) createCenteredModal(title, instruction, actionIcon string) {
	// Create form fields
	ie.createFormFields()

	// Create professional header with icon and title
	headerText := tview.NewTextView()
	headerText.SetText(fmt.Sprintf("[aqua::b]%s %s[white::-]\n[lightgray]%s[white]", actionIcon, title, instruction)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Create compact form fields layout
	fieldsLayout := ie.createCenteredFieldsLayout()
	
	// Create compact buttons layout
	buttonsLayout := ie.createCompactButtonsLayout()
	
	// Create progress/suggestions text area (for fzf dropdown and progress)
	ie.progressText = tview.NewTextView()
	ie.progressText.SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).  // Left align for fzf suggestions
		SetText("").
		SetBorder(false)
	
	// Create main content layout - professional with optimal spacing
	contentLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(headerText, 2, 0, false).              // More space for 2-line header
		AddItem(tview.NewBox(), 1, 0, false).          // Spacing after header
		AddItem(fieldsLayout, 0, 1, true).             // Main content area
		AddItem(tview.NewBox(), 1, 0, false).          // Spacing before buttons
		AddItem(buttonsLayout, 1, 0, false).           // Button row
		AddItem(tview.NewBox(), 1, 0, false).          // Spacing after buttons
		AddItem(ie.progressText, 10, 0, false)         // Generous space for fzf suggestions
	
	// Create border with professional styling
	border := tview.NewFlex()
	border.SetBorder(true).
		SetBorderColor(tcell.ColorAqua).
		SetTitle(fmt.Sprintf(" %s ", title)).
		SetTitleColor(tcell.ColorAqua)
	border.AddItem(contentLayout, 0, 1, true)
	
	// Center the modal in screen - create centering flex containers with bigger size for fzf
	centeredModal := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).  // Left padding
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewBox(), 0, 1, false). // Top padding
			AddItem(border, 35, 0, true).         // Even taller modal for better spacing
			AddItem(tview.NewBox(), 0, 1, false), // Bottom padding
			80, 0, true).                         // Wider modal for better content space
		AddItem(tview.NewBox(), 0, 1, false)   // Right padding
	
	// Set up key bindings
	ie.setupKeyBindings(centeredModal)
	
	// Show centered modal
	if ie.app.modalManager != nil {
		ie.app.modalManager.ShowModal(centeredModal)
	} else {
		ie.app.app.SetRoot(centeredModal, true)
		ie.app.app.SetFocus(fieldsLayout)
	}
}

// createCenteredFieldsLayout creates a professional form fields layout with proper spacing
func (ie *ImportExportModal) createCenteredFieldsLayout() *tview.Flex {
	// Create file path section with full-width input
	filePathLabel := tview.NewTextView()
	filePathLabel.SetText("[yellow::b]📁 File Path[white::-]").
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	
	// Make input field span full width with padding
	filePathInputRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 1, 0, false).        // Left padding
		AddItem(ie.filePathField, 0, 1, true).       // Full-width input
		AddItem(tview.NewBox(), 1, 0, false)         // Right padding
	
	filePathSection := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(filePathLabel, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).        // Spacing
		AddItem(filePathInputRow, 1, 0, true)
	
	// Create bigger, centered browse button with more spacing
	browseButton := tview.NewButton("📂 Browse Files")
	browseButton.SetSelectedFunc(ie.showBuiltInFileSystemBrowser)
	browseButton.SetBackgroundColor(tcell.ColorDarkBlue)
	
	browseButtonRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).        // Left spacer
		AddItem(browseButton, 24, 0, false).         // Bigger button
		AddItem(tview.NewBox(), 0, 1, false)         // Right spacer
	
	// Create format section with centered dropdown
	formatLabel := tview.NewTextView()
	formatLabel.SetText("[yellow::b]📋 Format[white::-]").
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	
	formatDropdownRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).        // Left spacer
		AddItem(ie.formatField, 20, 0, false).       // Centered dropdown
		AddItem(tview.NewBox(), 0, 1, false)         // Right spacer
	
	formatSection := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(formatLabel, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).        // Spacing
		AddItem(formatDropdownRow, 1, 0, false)
	
	// Create main fields layout with better spacing
	fieldsLayout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(filePathSection, 3, 0, true).       // File path with more space
		AddItem(tview.NewBox(), 1, 0, false).       // Spacer
		AddItem(browseButtonRow, 1, 0, false).      // Browse button
		AddItem(tview.NewBox(), 2, 0, false).       // Larger spacer
		AddItem(formatSection, 3, 0, false)         // Format section
	
	// Add profile section for export (if needed) with consistent styling
	if !ie.isImport {
		profileLabel := tview.NewTextView()
		profileLabel.SetText("[yellow::b]🏷️  Profile Filter[white::-]").
			SetDynamicColors(true).
			SetTextAlign(tview.AlignLeft)
		
		profileDropdownRow := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox(), 0, 1, false).    // Left spacer
			AddItem(ie.profileField, 20, 0, false).  // Centered dropdown
			AddItem(tview.NewBox(), 0, 1, false)     // Right spacer
		
		profileSection := tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(profileLabel, 1, 0, false).
			AddItem(tview.NewBox(), 1, 0, false).    // Spacing
			AddItem(profileDropdownRow, 1, 0, false)
		
		fieldsLayout.AddItem(tview.NewBox(), 2, 0, false) // Larger spacer
		fieldsLayout.AddItem(profileSection, 3, 0, false) // Profile section
	}
	
	return fieldsLayout
}


// createCompactButtonsLayout creates professional action buttons with better spacing
func (ie *ImportExportModal) createCompactButtonsLayout() *tview.Flex {
	// Create action button with improved styling
	var actionButton *tview.Button
	if ie.isImport {
		actionButton = tview.NewButton("📥 Import Configuration")
		actionButton.SetSelectedFunc(ie.handleImport)
	} else {
		actionButton = tview.NewButton("📤 Export Configuration")
		actionButton.SetSelectedFunc(ie.handleExport)
	}
	actionButton.SetBackgroundColor(tcell.ColorDarkGreen)
	
	// Create cancel button with improved styling
	cancelButton := tview.NewButton("❌ Cancel")
	cancelButton.SetSelectedFunc(ie.handleCancel)
	cancelButton.SetBackgroundColor(tcell.ColorDarkRed)
	
	// Create professional button layout with better spacing
	buttonLayout := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).     // Spacer
		AddItem(actionButton, 24, 0, false).      // Wider action button
		AddItem(tview.NewBox(), 4, 0, false).     // Larger spacer between buttons
		AddItem(cancelButton, 14, 0, false).      // Cancel button
		AddItem(tview.NewBox(), 0, 1, false)      // Spacer
	
	return buttonLayout
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

// launchExternalFzf launches the external fzf binary for file selection in fullscreen
func (ie *ImportExportModal) launchExternalFzf() {
	// Check if fzf is available
	if !ie.isFzfAvailable() {
		ie.showFzfInstallationModal()
		return
	}
	
	// Suspend the TUI application
	ie.app.app.Suspend(func() {
		// Set up fzf search directory
		searchDir, err := os.UserHomeDir()
		if err != nil {
			searchDir = "/"
		}
		
		// Build fzf command based on import/export mode
		var fzfCommand string
		if ie.isImport {
			// For import: show files with supported extensions
			fzfCommand = fmt.Sprintf("find %s -type f \\( -name '*.yaml' -o -name '*.yml' -o -name '*.json' -o -name 'config' -o -name '*config*' \\) 2>/dev/null | fzf --height=100%% --border --info=inline --preview 'head -20 {}' --preview-window=right:50%% --prompt='Select config file: '", searchDir)
		} else {
			// For export: show directories and let user type filename
			fzfCommand = fmt.Sprintf("find %s -type d 2>/dev/null | fzf --height=100%% --border --info=inline --preview 'ls -la {}' --preview-window=right:50%% --prompt='Select directory: '", searchDir)
		}
		
		// Execute fzf and get result
		result := ie.withFilter(fzfCommand, func(in io.WriteCloser) {
			in.Close()
		})
		
		// Process the result
		if len(result) > 0 && result[0] != "" {
			selectedPath := strings.TrimSpace(result[0])
			if selectedPath != "" {
				if !ie.isImport {
					// For export, add a default filename
					selectedPath = filepath.Join(selectedPath, "config.yaml")
				}
				
				// Schedule update to the TUI after resume
				ie.app.app.QueueUpdateDraw(func() {
					ie.filePathField.SetText(selectedPath)
					
					// Auto-detect format for import
					if ie.isImport {
						format := ie.detectFileFormat(selectedPath)
						ie.setFormatSelection(format)
					}
				})
			}
		}
	})
}


// isFzfAvailable checks if fzf command is available in the system
func (ie *ImportExportModal) isFzfAvailable() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// showFzfInstallationModal shows installation instructions for fzf
func (ie *ImportExportModal) showFzfInstallationModal() {
	// Create installation instructions
	instructions := `[yellow::b]📦 fzf (Fuzzy Finder) Required[white::-]

The Browse Files feature requires fzf to be installed on your system.

[aqua::b]Installation Instructions:[white::-]

[yellow]macOS (using Homebrew):[white]
  brew install fzf

[yellow]Linux (Ubuntu/Debian):[white]
  sudo apt-get install fzf

[yellow]Linux (CentOS/RHEL/Fedora):[white]
  sudo dnf install fzf
  # or: sudo yum install fzf

[yellow]Using Git (all platforms):[white]
  git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
  ~/.fzf/install

[yellow]Manual Download:[white]
  Visit: https://github.com/junegunn/fzf/releases

[lightgray]After installation, restart your terminal or run:[white]
  source ~/.bashrc   # or ~/.zshrc`

	// Create modal
	modal := tview.NewModal()
	modal.SetText(instructions)
	modal.SetTitle(" 📦 Install fzf ")
	modal.SetBackgroundColor(tcell.ColorBlack)
	modal.SetBorder(true)
	modal.SetBorderColor(tcell.ColorYellow)
	modal.AddButtons([]string{"📂 Use File Browser", "❌ Cancel"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonLabel == "📂 Use File Browser" {
			// Fall back to built-in file system browser
			ie.showBuiltInFileSystemBrowser()
		}
		if ie.app.modalManager != nil {
			ie.app.modalManager.HideModal()
		}
	})

	// Show the modal
	if ie.app.modalManager != nil {
		ie.app.modalManager.ShowModal(modal)
	}
}

// showBuiltInFileSystemBrowser shows the built-in file system browser as fallback
func (ie *ImportExportModal) showBuiltInFileSystemBrowser() {
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

// withFilter executes fzf command and returns selected results
func (ie *ImportExportModal) withFilter(command string, input func(in io.WriteCloser)) []string {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	
	cmd := exec.Command(shell, "-c", command)
	cmd.Stderr = os.Stderr
	
	in, err := cmd.StdinPipe()
	if err != nil {
		return []string{}
	}
	
	go func() {
		input(in)
		in.Close()
	}()
	
	result, err := cmd.Output()
	if err != nil {
		return []string{}
	}
	
	lines := strings.Split(string(result), "\n")
	// Filter out empty lines
	var filteredLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			filteredLines = append(filteredLines, line)
		}
	}
	
	return filteredLines
}


// createFormFields creates the form input fields with professional styling
func (ie *ImportExportModal) createFormFields() {
	// File path field with real-time fzf dropdown - professional styling
	ie.filePathField = tview.NewInputField()
	ie.filePathField.SetLabel("").
		SetPlaceholder("Type file path or start typing to see fzf suggestions...").
		SetFieldWidth(0).  // Use full available width
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite)
	
	// Add real-time fzf dropdown functionality
	ie.setupRealtimeFzfDropdown()
	
	// Format selection field with professional styling
	ie.formatField = tview.NewDropDown()
	if ie.isImport {
		ie.formatField.SetOptions([]string{"Auto-detect", "YAML", "JSON", "SSH Config"}, nil)
	} else {
		ie.formatField.SetOptions([]string{"YAML", "JSON"}, nil)
	}
	ie.formatField.SetCurrentOption(0).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite)
	
	// Profile filter field (export only) with professional styling
	if !ie.isImport {
		ie.profileField = tview.NewDropDown()
		profiles := ie.app.config.GetProfiles()
		
		// Add "All" option first
		options := []string{"All"}
		for _, profile := range profiles {
			options = append(options, profile.Name)
		}
		
		ie.profileField.SetOptions(options, nil).
			SetCurrentOption(0).
			SetFieldBackgroundColor(tcell.ColorBlack).
			SetFieldTextColor(tcell.ColorWhite)
	}
}

// setupRealtimeFzfDropdown adds real-time fzf dropdown functionality to the file path field
func (ie *ImportExportModal) setupRealtimeFzfDropdown() {
	var suggestionsView *tview.TextView
	var currentSuggestions []string
	var selectedIndex int
	var suggestionsVisible bool
	
	// Set up real-time change handler
	ie.filePathField.SetChangedFunc(func(text string) {
		// Only trigger fzf for meaningful input (1+ chars for testing, later can be 3+)
		if len(strings.TrimSpace(text)) < 1 {
			if suggestionsVisible {
				ie.hideFzfDropdown()
				suggestionsView = nil
				currentSuggestions = nil
				suggestionsVisible = false
			}
			return
		}
		
		// Check if fzf is available
		if !ie.isFzfAvailable() {
			// Show message about fzf not being available
			ie.progressText.SetText("[yellow]fzf not available - install fzf for file suggestions[white]")
			return
		}
		
		// Get fzf suggestions asynchronously
		go func() {
			newSuggestions := ie.getFzfSuggestions(text)
			
			// Update UI on main thread
			ie.app.app.QueueUpdateDraw(func() {
				if len(newSuggestions) > 0 {
					currentSuggestions = newSuggestions
					selectedIndex = 0
					ie.showFzfDropdown(text, currentSuggestions, selectedIndex, &suggestionsView)
					suggestionsVisible = true
				} else {
					// Show "no matches" message
					ie.progressText.SetText(fmt.Sprintf("[yellow]fzf: no matches for '%s'[white]", text))
					if suggestionsVisible {
						currentSuggestions = nil
						suggestionsVisible = false
					}
				}
			})
		}()
	})
	
	// Set up key handling for dropdown navigation
	ie.filePathField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If suggestions are visible, handle navigation
		if suggestionsVisible && len(currentSuggestions) > 0 {
			switch event.Key() {
			case tcell.KeyEscape:
				// Hide suggestions
				ie.hideFzfDropdown()
				suggestionsView = nil
				currentSuggestions = nil
				suggestionsVisible = false
				return nil
			case tcell.KeyDown:
				// Navigate down
				if selectedIndex < len(currentSuggestions)-1 {
					selectedIndex++
					ie.showFzfDropdown(ie.filePathField.GetText(), currentSuggestions, selectedIndex, &suggestionsView)
				}
				return nil
			case tcell.KeyUp:
				// Navigate up
				if selectedIndex > 0 {
					selectedIndex--
					ie.showFzfDropdown(ie.filePathField.GetText(), currentSuggestions, selectedIndex, &suggestionsView)
				}
				return nil
			case tcell.KeyEnter, tcell.KeyTab:
				// Select current suggestion
				if len(currentSuggestions) > 0 && selectedIndex < len(currentSuggestions) {
					selectedPath := currentSuggestions[selectedIndex]
					ie.filePathField.SetText(selectedPath)
					ie.hideFzfDropdown()
					suggestionsView = nil
					currentSuggestions = nil
					suggestionsVisible = false
					
					// Auto-detect format for import
					if ie.isImport {
						format := ie.detectFileFormat(selectedPath)
						ie.setFormatSelection(format)
					}
				}
				return nil
			}
		}
		
		return event
	})
}

// getFzfSuggestions gets file suggestions using fzf
func (ie *ImportExportModal) getFzfSuggestions(query string) []string {
	// Determine search directory - start from home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/"
	}
	
	// If query contains a path separator, use its directory
	searchDir := homeDir
	if strings.Contains(query, "/") {
		queryDir := filepath.Dir(query)
		if queryDir != "." && queryDir != query {
			// Expand ~ to home directory
			if strings.HasPrefix(queryDir, "~") {
				queryDir = strings.Replace(queryDir, "~", homeDir, 1)
			}
			searchDir = queryDir
		}
	}
	
	// Build fzf command for suggestions
	var fzfCommand string
	if ie.isImport {
		// For import: find config files and use fzf for fuzzy matching
		// Search more broadly and let fzf do the filtering
		fzfCommand = fmt.Sprintf("find %s -maxdepth 4 -type f \\( -name '*.yaml' -o -name '*.yml' -o -name '*.json' -o -name 'config' -o -name '*config*' -o -name '*.conf' \\) 2>/dev/null | fzf --filter='%s' | head -10", searchDir, filepath.Base(query))
	} else {
		// For export: find directories and files
		fzfCommand = fmt.Sprintf("find %s -maxdepth 3 -type f -o -type d 2>/dev/null | fzf --filter='%s' | head -10", searchDir, filepath.Base(query))
	}
	
	return ie.withFilter(fzfCommand, func(in io.WriteCloser) {
		in.Close()
	})
}

// showFzfDropdown displays fzf-style suggestions in a dropdown within the modal
func (ie *ImportExportModal) showFzfDropdown(query string, suggestions []string, selectedIndex int, suggestionsView **tview.TextView) {
	// This will be implemented by modifying the modal layout to include a suggestions area
	// For now, we'll use a simple approach that shows suggestions in the progress text area
	
	// Limit suggestions displayed
	maxShow := 6
	showSuggestions := suggestions
	if len(suggestions) > maxShow {
		showSuggestions = suggestions[:maxShow]
	}
	
	// Build suggestions text with fzf-like styling - compact format
	var suggestionsText strings.Builder
	suggestionsText.WriteString(fmt.Sprintf("[aqua::b]fzf:[white::-] %s\n", query))
	
	for i, suggestion := range showSuggestions {
		// Truncate long paths for display
		displayPath := suggestion
		if len(displayPath) > 60 {
			displayPath = "..." + displayPath[len(displayPath)-57:]
		}
		
		if i == selectedIndex {
			// Highlight selected item with fzf-style selection
			suggestionsText.WriteString(fmt.Sprintf("[black:aqua]▶ %s[::]\n", displayPath))
		} else {
			suggestionsText.WriteString(fmt.Sprintf("[white]  %s[::]\n", displayPath))
		}
	}
	
	// Add compact footer
	if len(suggestions) > maxShow {
		suggestionsText.WriteString(fmt.Sprintf("[gray]... +%d more[::] ", len(suggestions)-maxShow))
	}
	suggestionsText.WriteString(fmt.Sprintf("[yellow]↑↓:nav Enter:select Esc:close[::] [gray]%d/%d[::]\n", selectedIndex+1, len(suggestions)))
	
	// Update the progress text area to show suggestions
	if ie.progressText != nil {
		ie.progressText.SetText(suggestionsText.String())
	}
}

// hideFzfDropdown hides the fzf suggestions dropdown
func (ie *ImportExportModal) hideFzfDropdown() {
	// Clear the progress text area
	if ie.progressText != nil {
		ie.progressText.SetText("")
	}
}

// setupKeyBindings configures keyboard navigation for the modal
func (ie *ImportExportModal) setupKeyBindings(layout tview.Primitive) {
	if flexLayout, ok := layout.(*tview.Flex); ok {
		flexLayout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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
		
		// Skip hidden files but keep directories (hidden directories can contain config files)
		if strings.HasPrefix(entry.Name(), ".") && entry.Name() != ".." && !entry.IsDir() {
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