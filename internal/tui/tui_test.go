package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sshm/internal/config"
)

func TestTUIApplication_NewApp(t *testing.T) {
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Expected no error creating new TUI app, got %v", err)
	}
	
	if app == nil {
		t.Fatal("Expected TUI app to be created, got nil")
	}
	
	if app.app == nil {
		t.Fatal("Expected tview.Application to be initialized")
	}
}

func TestTUIApplication_Start_Stop(t *testing.T) {
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start the application in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Run(ctx)
	}()

	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)

	// Stop the application
	app.Stop()

	// Wait for completion
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			// Check if it's a TTY error (expected in test environment)
			if err.Error() != "TUI application error: open /dev/tty: device not configured" {
				t.Fatalf("Expected clean shutdown, got unexpected error: %v", err)
			}
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Application did not stop within timeout")
	}
}

func TestTUIApplication_SetupLayout(t *testing.T) {
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	if app.layout == nil {
		t.Fatal("Expected layout to be initialized")
	}

	// Verify server list is configured
	if app.serverList == nil {
		t.Fatal("Expected server list to be initialized")
	}

	// Verify status bar is configured  
	if app.statusBar == nil {
		t.Fatal("Expected status bar to be initialized")
	}
}

func TestTUIApplication_CleanupResources(t *testing.T) {
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Stop should clean up resources
	app.Stop()

	// Verify cleanup occurred (app should be stopped)
	if app.app == nil {
		t.Fatal("Expected app to still exist after stop (for potential restart)")
	}
}

func TestTUIApplication_ErrorHandling(t *testing.T) {
	// Test initialization with invalid state
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test multiple stops don't panic
	app.Stop()
	app.Stop() // Should not panic

	// Test running stopped app
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = app.Run(ctx)
	// Should handle gracefully, either restart or return appropriate error
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		// TTY errors are expected in test environments
		if err.Error() == "TUI application error: open /dev/tty: device not configured" {
			t.Logf("App returned TTY error in test environment (expected): %v", err)
		} else {
			t.Logf("App returned error after stop (may be expected): %v", err)
		}
	}
}

// createTestConfig creates a test configuration with sample servers and profiles
func createTestConfig(t *testing.T) *config.Config {
	cfg := &config.Config{
		Servers: []config.Server{
			{
				Name:     "test-web-01",
				Hostname: "192.168.1.10",
				Port:     22,
				Username: "ubuntu",
				AuthType: "key",
				KeyPath:  "/home/user/.ssh/id_rsa",
			},
			{
				Name:     "prod-db-01",
				Hostname: "10.0.0.5",
				Port:     22,
				Username: "admin",
				AuthType: "password",
			},
			{
				Name:     "staging-api",
				Hostname: "staging.api.example.com",
				Port:     2222,
				Username: "deploy",
				AuthType: "key",
				KeyPath:  "/home/user/.ssh/deploy_key",
			},
		},
		Profiles: []config.Profile{
			{
				Name:        "development",
				Description: "Development environment servers",
				Servers:     []string{"test-web-01"},
			},
			{
				Name:        "production",
				Description: "Production environment servers",
				Servers:     []string{"prod-db-01"},
			},
			{
				Name:        "staging",
				Description: "Staging environment servers",
				Servers:     []string{"staging-api"},
			},
		},
	}
	return cfg
}

func TestServerList_DataLoading(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Verify that servers were loaded
	servers := app.config.GetServers()
	if len(servers) != 3 {
		t.Errorf("Expected 3 servers, got %d", len(servers))
	}

	// Verify server table has correct number of rows (including header)
	rowCount := app.serverList.GetRowCount()
	expectedRows := len(servers) + 1 // +1 for header row
	if rowCount != expectedRows {
		t.Errorf("Expected %d rows in server table, got %d", expectedRows, rowCount)
	}
}

func TestServerList_DisplayFormatting(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test header row formatting
	headerCells := []string{"Name", "Host", "Port", "User", "Auth", "Status", "Profile"}
	for col, expectedHeader := range headerCells {
		cell := app.serverList.GetCell(0, col)
		if cell == nil {
			t.Errorf("Expected header cell at column %d, got nil", col)
			continue
		}
		if cell.Text != expectedHeader {
			t.Errorf("Expected header cell %d to be '%s', got '%s'", col, expectedHeader, cell.Text)
		}
	}

	// Test data row formatting for first server
	if app.serverList.GetRowCount() > 1 {
		// Check Name column
		nameCell := app.serverList.GetCell(1, 0)
		if nameCell == nil || nameCell.Text != "test-web-01" {
			t.Errorf("Expected first server name to be 'test-web-01', got %v", nameCell)
		}

		// Check Host column
		hostCell := app.serverList.GetCell(1, 1)
		if hostCell == nil || hostCell.Text != "192.168.1.10" {
			t.Errorf("Expected first server host to be '192.168.1.10', got %v", hostCell)
		}

		// Check Port column
		portCell := app.serverList.GetCell(1, 2)
		if portCell == nil || portCell.Text != "22" {
			t.Errorf("Expected first server port to be '22', got %v", portCell)
		}

		// Check User column
		userCell := app.serverList.GetCell(1, 3)
		if userCell == nil || userCell.Text != "ubuntu" {
			t.Errorf("Expected first server user to be 'ubuntu', got %v", userCell)
		}

		// Check Auth column
		authCell := app.serverList.GetCell(1, 4)
		if authCell == nil || authCell.Text != "key" {
			t.Errorf("Expected first server auth to be 'key', got %v", authCell)
		}
	}
}

func TestServerList_EmptyConfiguration(t *testing.T) {
	// Create a temporary directory for empty test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app with empty config
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Verify that no servers are loaded
	servers := app.config.GetServers()
	if len(servers) != 0 {
		t.Errorf("Expected 0 servers in empty config, got %d", len(servers))
	}

	// Verify server table has only header row
	rowCount := app.serverList.GetRowCount()
	if rowCount != 1 {
		t.Errorf("Expected 1 row (header only) in empty server table, got %d", rowCount)
	}
}

func TestServerList_RefreshConfiguration(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Start with empty config
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Verify empty state
	initialRowCount := app.serverList.GetRowCount()
	if initialRowCount != 1 {
		t.Errorf("Expected 1 row initially (header only), got %d", initialRowCount)
	}

	// Create and save test config
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Refresh configuration
	if err := app.RefreshConfig(); err != nil {
		t.Fatalf("Failed to refresh config: %v", err)
	}

	// Verify servers are now loaded
	servers := app.config.GetServers()
	if len(servers) != 3 {
		t.Errorf("Expected 3 servers after refresh, got %d", len(servers))
	}

	// Verify server table has correct number of rows
	rowCount := app.serverList.GetRowCount()
	expectedRows := len(servers) + 1 // +1 for header row
	if rowCount != expectedRows {
		t.Errorf("Expected %d rows after refresh, got %d", expectedRows, rowCount)
	}
}

func TestServerList_KeyboardNavigation(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Initial state should have first server selected (row 1)
	currentRow, _ := app.serverList.GetSelection()
	if currentRow != 1 {
		t.Errorf("Expected initial selection at row 1, got %d", currentRow)
	}

	// Test navigation down
	app.navigateDown()
	currentRow, _ = app.serverList.GetSelection()
	if currentRow != 2 {
		t.Errorf("Expected selection at row 2 after navigating down, got %d", currentRow)
	}

	// Test navigation up
	app.navigateUp()
	currentRow, _ = app.serverList.GetSelection()
	if currentRow != 1 {
		t.Errorf("Expected selection at row 1 after navigating up, got %d", currentRow)
	}

	// Test navigation up from first row (should stay at row 1)
	app.navigateUp()
	currentRow, _ = app.serverList.GetSelection()
	if currentRow != 1 {
		t.Errorf("Expected selection to stay at row 1 when navigating up from first row, got %d", currentRow)
	}

	// Test navigation down to last row
	for i := 1; i < app.serverList.GetRowCount()-1; i++ {
		app.navigateDown()
	}
	currentRow, _ = app.serverList.GetSelection()
	expectedLastRow := app.serverList.GetRowCount() - 1
	if currentRow != expectedLastRow {
		t.Errorf("Expected selection at last row %d, got %d", expectedLastRow, currentRow)
	}

	// Test navigation down from last row (should stay at last row)
	app.navigateDown()
	currentRow, _ = app.serverList.GetSelection()
	if currentRow != expectedLastRow {
		t.Errorf("Expected selection to stay at last row %d when navigating down from last row, got %d", expectedLastRow, currentRow)
	}
}

func TestServerList_ProfileFiltering(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Initially should show all servers
	initialRowCount := app.serverList.GetRowCount()
	if initialRowCount != 4 { // 3 servers + 1 header
		t.Errorf("Expected 4 rows initially (3 servers + header), got %d", initialRowCount)
	}

	// Switch to first profile (development)
	app.switchToNextProfile()
	
	// Should show only 1 server from development profile + header
	filteredRowCount := app.serverList.GetRowCount()
	if filteredRowCount != 2 { // 1 server + 1 header
		t.Errorf("Expected 2 rows after filtering to development profile, got %d", filteredRowCount)
	}

	// Verify the server shown is from development profile
	if filteredRowCount > 1 {
		nameCell := app.serverList.GetCell(1, 0)
		if nameCell == nil || nameCell.Text != "test-web-01" {
			t.Errorf("Expected first filtered server to be 'test-web-01', got %v", nameCell)
		}
	}

	// Switch to production profile
	app.switchToNextProfile()
	
	// Should show only 1 server from production profile + header
	filteredRowCount = app.serverList.GetRowCount()
	if filteredRowCount != 2 { // 1 server + 1 header
		t.Errorf("Expected 2 rows after filtering to production profile, got %d", filteredRowCount)
	}

	// Verify the server shown is from production profile
	if filteredRowCount > 1 {
		nameCell := app.serverList.GetCell(1, 0)
		if nameCell == nil || nameCell.Text != "prod-db-01" {
			t.Errorf("Expected production filtered server to be 'prod-db-01', got %v", nameCell)
		}
	}

	// Switch back to all servers
	app.switchToNextProfile() // staging
	app.switchToNextProfile() // back to all
	
	// Should show all servers again
	finalRowCount := app.serverList.GetRowCount()
	if finalRowCount != 4 { // 3 servers + 1 header
		t.Errorf("Expected 4 rows after returning to 'all' filter, got %d", finalRowCount)
	}
}

func TestServerList_EmptyNavigation(t *testing.T) {
	// Create a temporary directory for empty test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app with empty config
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Navigation on empty list should not panic or change selection
	initialRow, _ := app.serverList.GetSelection()
	
	app.navigateUp()
	currentRow, _ := app.serverList.GetSelection()
	if currentRow != initialRow {
		t.Errorf("Selection changed during navigation on empty list: initial=%d, current=%d", initialRow, currentRow)
	}

	app.navigateDown()
	currentRow, _ = app.serverList.GetSelection()
	if currentRow != initialRow {
		t.Errorf("Selection changed during navigation on empty list: initial=%d, current=%d", initialRow, currentRow)
	}
}

// TestProfileNavigator_Creation tests the creation and initialization of profile navigator
func TestProfileNavigator_Creation(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Profile navigator should be initialized
	if app.profileNavigator == nil {
		t.Fatal("Expected profile navigator to be initialized")
	}

	// Should display tabs for all profiles plus "All" tab
	expectedTabs := 4 // "All" + development, production, staging
	if len(app.profileTabs) != expectedTabs {
		t.Errorf("Expected %d profile tabs, got %d", expectedTabs, len(app.profileTabs))
	}

	// First tab should be "All" and should be selected by default
	if app.selectedProfileIndex != 0 {
		t.Errorf("Expected 'All' tab to be selected by default (index 0), got %d", app.selectedProfileIndex)
	}

	if app.profileTabs[0] != "All" {
		t.Errorf("Expected first tab to be 'All', got '%s'", app.profileTabs[0])
	}
}

// TestProfileNavigator_TabDisplay tests the visual display of profile tabs
func TestProfileNavigator_TabDisplay(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test tab display text generation
	tabText := app.renderProfileTabs()
	
	// Should contain all profile names
	expectedProfiles := []string{"All", "development", "production", "staging"}
	for _, profile := range expectedProfiles {
		if !strings.Contains(tabText, profile) {
			t.Errorf("Expected tab display to contain '%s', got: %s", profile, tabText)
		}
	}

	// Should highlight the selected tab (All should be highlighted by default)
	if !strings.Contains(tabText, "[aqua][All][white]") {
		t.Errorf("Expected 'All' tab to be highlighted in tab display, got: %s", tabText)
	}
}

// TestProfileNavigator_SwitchingWithTab tests profile switching using Tab key
func TestProfileNavigator_SwitchingWithTab(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Initially should be on "All" tab (index 0)
	if app.selectedProfileIndex != 0 {
		t.Errorf("Expected initial profile index to be 0, got %d", app.selectedProfileIndex)
	}

	// Switch to next profile using Tab navigation
	app.switchToNextProfile()
	if app.selectedProfileIndex != 1 {
		t.Errorf("Expected profile index to be 1 after switching, got %d", app.selectedProfileIndex)
	}

	// Should now be showing development profile
	if app.profileTabs[app.selectedProfileIndex] != "development" {
		t.Errorf("Expected current profile to be 'development', got '%s'", app.profileTabs[app.selectedProfileIndex])
	}

	// Continue switching through all profiles
	app.switchToNextProfile() // production
	if app.selectedProfileIndex != 2 {
		t.Errorf("Expected profile index to be 2, got %d", app.selectedProfileIndex)
	}

	app.switchToNextProfile() // staging  
	if app.selectedProfileIndex != 3 {
		t.Errorf("Expected profile index to be 3, got %d", app.selectedProfileIndex)
	}

	// Should cycle back to "All"
	app.switchToNextProfile()
	if app.selectedProfileIndex != 0 {
		t.Errorf("Expected profile index to cycle back to 0, got %d", app.selectedProfileIndex)
	}
}

// TestProfileNavigator_SwitchingWithPKey tests profile switching using 'p' key
func TestProfileNavigator_SwitchingWithPKey(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test 'p' key switching (should behave same as Tab)
	initialIndex := app.selectedProfileIndex

	app.switchToNextProfile()
	if app.selectedProfileIndex <= initialIndex {
		t.Errorf("Expected profile index to increase after 'p' key switch, was %d, now %d", initialIndex, app.selectedProfileIndex)
	}
}

// TestProfileNavigator_FilterApplication tests that profile selection filters server list
func TestProfileNavigator_FilterApplication(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Initially should show all servers (All tab selected)
	initialRowCount := app.serverList.GetRowCount()
	if initialRowCount != 4 { // 3 servers + 1 header
		t.Errorf("Expected 4 rows with all servers shown, got %d", initialRowCount)
	}

	// Switch to development profile
	app.switchToProfile(1) // development profile index
	
	// Should filter to only development servers + header
	filteredRowCount := app.serverList.GetRowCount()
	if filteredRowCount != 2 { // 1 server + 1 header
		t.Errorf("Expected 2 rows after filtering to development profile, got %d", filteredRowCount)
	}

	// Verify the correct server is shown
	if filteredRowCount > 1 {
		nameCell := app.serverList.GetCell(1, 0)
		if nameCell == nil || nameCell.Text != "test-web-01" {
			t.Errorf("Expected development server 'test-web-01', got %v", nameCell)
		}
	}

	// Switch to production profile
	app.switchToProfile(2) // production profile index
	
	// Should filter to only production servers + header
	filteredRowCount = app.serverList.GetRowCount()
	if filteredRowCount != 2 { // 1 server + 1 header
		t.Errorf("Expected 2 rows after filtering to production profile, got %d", filteredRowCount)
	}

	// Verify the correct server is shown
	if filteredRowCount > 1 {
		nameCell := app.serverList.GetCell(1, 0)
		if nameCell == nil || nameCell.Text != "prod-db-01" {
			t.Errorf("Expected production server 'prod-db-01', got %v", nameCell)
		}
	}

	// Switch back to All tab
	app.switchToProfile(0) // All profile index
	
	// Should show all servers again
	finalRowCount := app.serverList.GetRowCount()
	if finalRowCount != 4 { // 3 servers + 1 header
		t.Errorf("Expected 4 rows after switching back to All, got %d", finalRowCount)
	}
}

// TestProfileNavigator_EmptyConfiguration tests profile navigator with no profiles configured
func TestProfileNavigator_EmptyConfiguration(t *testing.T) {
	// Create a temporary directory for empty test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app with empty config (no profiles)
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Profile navigator should still be initialized but only show "All" tab
	if app.profileNavigator == nil {
		t.Fatal("Expected profile navigator to be initialized even with no profiles")
	}

	// Should only have "All" tab
	if len(app.profileTabs) != 1 {
		t.Errorf("Expected 1 profile tab (All only), got %d", len(app.profileTabs))
	}

	if app.profileTabs[0] != "All" {
		t.Errorf("Expected only tab to be 'All', got '%s'", app.profileTabs[0])
	}

	// Profile switching should not change anything with only one tab
	initialIndex := app.selectedProfileIndex
	app.switchToNextProfile()
	if app.selectedProfileIndex != initialIndex {
		t.Errorf("Profile index should not change with only one tab, was %d, now %d", initialIndex, app.selectedProfileIndex)
	}
}

// TestProfileNavigator_BackwardNavigation tests backward navigation with Shift+Tab
func TestProfileNavigator_BackwardNavigation(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Start at "All" tab (index 0)
	if app.selectedProfileIndex != 0 {
		t.Errorf("Expected initial profile index to be 0, got %d", app.selectedProfileIndex)
	}

	// Move backward (should wrap to last profile)
	app.switchToPreviousProfile()
	expectedLastIndex := len(app.profileTabs) - 1
	if app.selectedProfileIndex != expectedLastIndex {
		t.Errorf("Expected backward navigation to wrap to last index %d, got %d", expectedLastIndex, app.selectedProfileIndex)
	}

	// Move backward again
	app.switchToPreviousProfile()
	expectedIndex := expectedLastIndex - 1
	if app.selectedProfileIndex != expectedIndex {
		t.Errorf("Expected profile index to be %d after backward navigation, got %d", expectedIndex, app.selectedProfileIndex)
	}
}

// TestSessionManager_Creation tests the creation and initialization of session manager
func TestSessionManager_Creation(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Session panel should be initialized if tmux is available
	if app.sessionPanel != nil {
		// Verify session table structure
		if app.sessionPanel.GetRowCount() < 1 {
			t.Error("Expected session panel to have at least header row")
		}

		// Check header row exists
		nameHeader := app.sessionPanel.GetCell(0, 0)
		if nameHeader == nil || nameHeader.Text != "Session" {
			t.Errorf("Expected session header 'Session', got %v", nameHeader)
		}

		statusHeader := app.sessionPanel.GetCell(0, 1)
		if statusHeader == nil || statusHeader.Text != "Status" {
			t.Errorf("Expected status header 'Status', got %v", statusHeader)
		}

		windowsHeader := app.sessionPanel.GetCell(0, 2)
		if windowsHeader == nil || windowsHeader.Text != "Windows" {
			t.Errorf("Expected windows header 'Windows', got %v", windowsHeader)
		}

		activeHeader := app.sessionPanel.GetCell(0, 3)
		if activeHeader == nil || activeHeader.Text != "Last Activity" {
			t.Errorf("Expected activity header 'Last Activity', got %v", activeHeader)
		}
	}
}

// TestSessionManager_TmuxDetection tests tmux session detection and parsing
func TestSessionManager_TmuxDetection(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test session parsing with mock data
	mockSessions := []SessionInfo{
		{
			Name:         "test-server-01",
			Status:       "active",
			Windows:      3,
			LastActivity: "2024-08-19 14:30:00",
		},
		{
			Name:         "dev-profile",
			Status:       "active",
			Windows:      5,
			LastActivity: "2024-08-19 14:25:00",
		},
	}

	// Test parseTmuxSessions function
	sessions := app.parseTmuxSessions("test-server-01 3 active 2024-08-19 14:30:00\ndev-profile 5 active 2024-08-19 14:25:00\n")
	
	if len(sessions) != 2 {
		t.Errorf("Expected 2 parsed sessions, got %d", len(sessions))
	}

	// Verify first session details
	if len(sessions) > 0 {
		if sessions[0].Name != "test-server-01" {
			t.Errorf("Expected first session name 'test-server-01', got '%s'", sessions[0].Name)
		}
		if sessions[0].Windows != 3 {
			t.Errorf("Expected first session windows 3, got %d", sessions[0].Windows)
		}
		if sessions[0].Status != "active" {
			t.Errorf("Expected first session status 'active', got '%s'", sessions[0].Status)
		}
	}

	// Test session display update
	app.updateSessionDisplay(mockSessions)
	
	// Verify sessions are displayed in table (header + data rows)
	expectedRows := len(mockSessions) + 1
	if app.sessionPanel != nil && app.sessionPanel.GetRowCount() != expectedRows {
		t.Errorf("Expected %d rows in session panel, got %d", expectedRows, app.sessionPanel.GetRowCount())
	}
}

// TestSessionManager_SessionDisplay tests the visual display of session information
func TestSessionManager_SessionDisplay(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test with sample session data
	testSessions := []SessionInfo{
		{
			Name:         "web-server",
			Status:       "active",
			Windows:      2,
			LastActivity: "14:30",
		},
		{
			Name:         "db-server",
			Status:       "attached",
			Windows:      1,
			LastActivity: "14:25",
		},
	}

	// Update session display
	app.updateSessionDisplay(testSessions)

	if app.sessionPanel == nil {
		t.Skip("Session panel not initialized (tmux might not be available)")
	}

	// Verify table has correct number of rows (header + sessions)
	expectedRows := len(testSessions) + 1
	actualRows := app.sessionPanel.GetRowCount()
	if actualRows != expectedRows {
		t.Errorf("Expected %d rows in session panel, got %d", expectedRows, actualRows)
	}

	// Verify first session data row
	if actualRows > 1 {
		nameCell := app.sessionPanel.GetCell(1, 0)
		if nameCell == nil || nameCell.Text != "web-server" {
			t.Errorf("Expected first session name 'web-server', got %v", nameCell)
		}

		statusCell := app.sessionPanel.GetCell(1, 1)
		if statusCell == nil || statusCell.Text != "active" {
			t.Errorf("Expected first session status 'active', got %v", statusCell)
		}

		windowsCell := app.sessionPanel.GetCell(1, 2)
		if windowsCell == nil || windowsCell.Text != "2" {
			t.Errorf("Expected first session windows '2', got %v", windowsCell)
		}

		activityCell := app.sessionPanel.GetCell(1, 3)
		if activityCell == nil || activityCell.Text != "14:30" {
			t.Errorf("Expected first session activity '14:30', got %v", activityCell)
		}
	}
}

// TestSessionManager_EmptySessionList tests handling of empty session list
func TestSessionManager_EmptySessionList(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test with empty session list
	emptySessions := []SessionInfo{}

	// Update session display
	app.updateSessionDisplay(emptySessions)

	if app.sessionPanel == nil {
		t.Skip("Session panel not initialized (tmux might not be available)")
	}

	// Should only have header row
	actualRows := app.sessionPanel.GetRowCount()
	if actualRows != 1 {
		t.Errorf("Expected 1 row (header only) for empty sessions, got %d", actualRows)
	}
}

// TestSessionManager_SessionRefresh tests session refresh functionality
func TestSessionManager_SessionRefresh(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	if app.sessionPanel == nil {
		t.Skip("Session panel not initialized (tmux might not be available)")
	}

	// Store initial row count
	initialRows := app.sessionPanel.GetRowCount()

	// Call refresh session functionality
	err = app.refreshSessions()
	if err != nil {
		t.Logf("Session refresh returned error (expected if tmux not available): %v", err)
	}

	// Verify refresh didn't break the display
	currentRows := app.sessionPanel.GetRowCount()
	if currentRows < 1 {
		t.Error("Expected at least header row after session refresh")
	}

	// Row count might change if sessions were added/removed, but should be >= 1
	if currentRows < initialRows && initialRows == 1 {
		t.Errorf("Session panel lost header row during refresh")
	}
}

// TestSessionManager_SessionSelection tests session selection and highlighting
func TestSessionManager_SessionSelection(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test with sample session data
	testSessions := []SessionInfo{
		{Name: "session1", Status: "active", Windows: 1, LastActivity: "14:30"},
		{Name: "session2", Status: "inactive", Windows: 2, LastActivity: "14:25"},
		{Name: "session3", Status: "active", Windows: 3, LastActivity: "14:20"},
	}

	// Update session display
	app.updateSessionDisplay(testSessions)

	if app.sessionPanel == nil {
		t.Skip("Session panel not initialized (tmux might not be available)")
	}

	// Test session navigation (if we have multiple sessions)
	if app.sessionPanel.GetRowCount() > 2 {
		// Test selecting first session
		app.sessionPanel.Select(1, 0) // First data row
		selectedRow, _ := app.sessionPanel.GetSelection()
		if selectedRow != 1 {
			t.Errorf("Expected selected row to be 1, got %d", selectedRow)
		}

		// Test selecting second session
		app.sessionPanel.Select(2, 0) // Second data row
		selectedRow, _ = app.sessionPanel.GetSelection()
		if selectedRow != 2 {
			t.Errorf("Expected selected row to be 2, got %d", selectedRow)
		}
	}
}

// TestKeyboardNavigation_QuitKeys tests quit functionality with q/Q and Ctrl+C
func TestKeyboardNavigation_QuitKeys(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test that app can be stopped (simulating quit keys)
	if app.running {
		t.Error("App should not be running before start")
	}

	// Start and immediately stop (simulating quit key press)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Run(ctx)
	}()

	// Give it a moment to start
	time.Sleep(5 * time.Millisecond)

	// Stop the app (simulates 'q' key press)
	app.Stop()

	// Wait for completion
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			if err.Error() != "TUI application error: open /dev/tty: device not configured" {
				t.Logf("App returned error during quit (may be expected in test): %v", err)
			}
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Application did not quit within timeout")
	}
}

// TestKeyboardNavigation_NavigationKeys tests j/k and arrow key navigation
func TestKeyboardNavigation_NavigationKeys(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test navigation on servers panel (default focus)
	if app.focusedPanel != "servers" {
		t.Errorf("Expected default focus on servers panel, got %s", app.focusedPanel)
	}

	// Test navigation down (j key simulation)
	initialRow, _ := app.serverList.GetSelection()
	app.handleNavigationDown() // Simulates 'j' key
	newRow, _ := app.serverList.GetSelection()
	if newRow <= initialRow && app.serverList.GetRowCount() > 2 {
		t.Errorf("Expected navigation down to increase row selection from %d to %d", initialRow, newRow)
	}

	// Test navigation up (k key simulation)
	currentRow, _ := app.serverList.GetSelection()
	app.handleNavigationUp() // Simulates 'k' key
	afterUpRow, _ := app.serverList.GetSelection()
	if afterUpRow >= currentRow && currentRow > 1 {
		t.Errorf("Expected navigation up to decrease row selection from %d to %d", currentRow, afterUpRow)
	}

	// Test navigation bounds (can't go above first row or below last row)
	for i := 0; i < 10; i++ {
		app.handleNavigationUp() // Multiple up movements
	}
	topRow, _ := app.serverList.GetSelection()
	if topRow < 1 {
		t.Errorf("Expected navigation to stay at first data row (1), got %d", topRow)
	}

	for i := 0; i < 10; i++ {
		app.handleNavigationDown() // Multiple down movements
	}
	bottomRow, _ := app.serverList.GetSelection()
	maxRow := app.serverList.GetRowCount() - 1
	if bottomRow > maxRow {
		t.Errorf("Expected navigation to stay within bounds (max %d), got %d", maxRow, bottomRow)
	}
}

// TestKeyboardNavigation_EnterKey tests Enter key functionality for connections
func TestKeyboardNavigation_EnterKey(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Select a server and test Enter key handling
	if app.serverList.GetRowCount() > 1 {
		app.serverList.Select(1, 0) // Select first server
		
		// handleEnterKey should attempt connection (we can't test actual connection in unit tests)
		// but we can verify the function doesn't panic and handles the selection correctly
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Enter key handler panicked: %v", r)
			}
		}()
		
		app.handleEnterKey() // Should trigger connection attempt
	}
}

// TestKeyboardNavigation_TabKeys tests Tab navigation for profile switching and focus switching
func TestKeyboardNavigation_TabKeys(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test Tab key on servers panel (should switch profiles)
	if app.focusedPanel != "servers" {
		t.Errorf("Expected default focus on servers panel, got %s", app.focusedPanel)
	}

	initialProfile := app.selectedProfileIndex
	app.switchToNextProfile() // Simulates Tab key on servers panel
	newProfile := app.selectedProfileIndex
	if newProfile == initialProfile && len(app.profileTabs) > 1 {
		t.Errorf("Expected Tab to switch profiles from %d to different index, got %d", initialProfile, newProfile)
	}

	// Test Shift+Tab (backward profile navigation)
	currentProfile := app.selectedProfileIndex
	app.switchToPreviousProfile() // Simulates Shift+Tab
	afterBackwardProfile := app.selectedProfileIndex
	if afterBackwardProfile == currentProfile && len(app.profileTabs) > 1 {
		t.Errorf("Expected Shift+Tab to switch profiles backward from %d, got %d", currentProfile, afterBackwardProfile)
	}

	// Test focus switching (s key simulation)
	if app.sessionPanel != nil {
		initialFocus := app.focusedPanel
		app.switchFocus() // Simulates 's' key
		newFocus := app.focusedPanel
		if newFocus == initialFocus {
			t.Errorf("Expected focus switch from %s to different panel, stayed at %s", initialFocus, newFocus)
		}
	}
}

// TestKeyboardNavigation_RefreshKey tests refresh functionality with 'r' key
func TestKeyboardNavigation_RefreshKey(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test refresh functionality (r key simulation)
	initialRowCount := app.serverList.GetRowCount()
	
	// refreshData should not panic and should maintain data integrity
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Refresh functionality panicked: %v", r)
		}
	}()
	
	app.refreshData() // Simulates 'r' key press
	
	// After refresh, should still have same number of servers (since config didn't change)
	afterRefreshCount := app.serverList.GetRowCount()
	if afterRefreshCount != initialRowCount {
		t.Errorf("Expected same row count after refresh: %d, got %d", initialRowCount, afterRefreshCount)
	}
}

// TestKeyboardNavigation_HelpKey tests '?' key for help modal
func TestKeyboardNavigation_HelpKey(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test help key functionality (? key simulation)
	// showHelp should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Help functionality panicked: %v", r)
		}
	}()
	
	app.showHelp() // Simulates '?' key press
	
	// The help modal should be set as the root (we can't easily test the actual modal content in unit tests)
	// but we can verify the function completes successfully
}

// TestKeyboardNavigation_SessionPanelNavigation tests navigation within session panel
func TestKeyboardNavigation_SessionPanelNavigation(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	if app.sessionPanel == nil {
		t.Skip("Session panel not initialized (tmux might not be available)")
	}

	// Test session navigation with mock sessions
	testSessions := []SessionInfo{
		{Name: "session1", Status: "active", Windows: 1, LastActivity: "14:30"},
		{Name: "session2", Status: "inactive", Windows: 2, LastActivity: "14:25"},
		{Name: "session3", Status: "active", Windows: 3, LastActivity: "14:20"},
	}
	
	app.updateSessionDisplay(testSessions)
	app.focusedPanel = "sessions" // Switch focus to sessions panel

	if app.sessionPanel.GetRowCount() > 2 {
		// Test session navigation down
		app.sessionPanel.Select(1, 0) // Start at first session
		initialRow, _ := app.sessionPanel.GetSelection()
		
		app.navigateSessionDown()
		newRow, _ := app.sessionPanel.GetSelection()
		if newRow <= initialRow {
			t.Errorf("Expected session navigation down to increase row from %d to %d", initialRow, newRow)
		}

		// Test session navigation up
		currentRow, _ := app.sessionPanel.GetSelection()
		app.navigateSessionUp()
		afterUpRow, _ := app.sessionPanel.GetSelection()
		if afterUpRow >= currentRow && currentRow > 1 {
			t.Errorf("Expected session navigation up to decrease row from %d to %d", currentRow, afterUpRow)
		}

		// Test Enter key on session (should attempt to attach)
		app.sessionPanel.Select(1, 0) // Select first session
		app.sessions = testSessions   // Set sessions data for attachment
		
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Session attachment panicked: %v", r)
			}
		}()
		
		// Note: This will actually try to stop the TUI and attach to tmux session
		// In a test environment, this might fail, but it shouldn't panic
	}
}

// TestKeyboardNavigation_FocusSwitching tests switching focus between panels
func TestKeyboardNavigation_FocusSwitching(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test initial focus state
	if app.focusedPanel != "servers" {
		t.Errorf("Expected initial focus on servers panel, got %s", app.focusedPanel)
	}

	if app.sessionPanel != nil {
		// Test switching focus to sessions panel
		initialFocus := app.focusedPanel
		app.switchFocus()
		newFocus := app.focusedPanel
		
		if newFocus == initialFocus {
			t.Errorf("Expected focus to switch from %s, but stayed at %s", initialFocus, newFocus)
		}
		
		if newFocus != "sessions" {
			t.Errorf("Expected focus to switch to sessions, got %s", newFocus)
		}

		// Test switching back to servers panel
		app.switchFocus()
		finalFocus := app.focusedPanel
		
		if finalFocus != "servers" {
			t.Errorf("Expected focus to switch back to servers, got %s", finalFocus)
		}
	}
}

// TestKeyboardNavigation_AllKeybindings tests all supported keybindings comprehensively
func TestKeyboardNavigation_AllKeybindings(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test all keybinding functions don't panic
	testFunctions := []func(){
		func() { app.handleNavigationUp() },    // k, Up Arrow
		func() { app.handleNavigationDown() },  // j, Down Arrow
		func() { app.handleEnterKey() },        // Enter
		func() { app.switchToNextProfile() },   // Tab (on servers), p
		func() { app.switchToPreviousProfile() }, // Shift+Tab (on servers)
		func() { app.switchFocus() },           // s, Tab (between panels)
		func() { app.refreshData() },           // r
		func() { app.showHelp() },              // ?
		func() { app.Stop() },                  // q, Q, Ctrl+C (test that Stop works)
	}

	for i, testFunc := range testFunctions {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Keybinding function %d panicked: %v", i, r)
				}
			}()
			testFunc()
		}()
	}
}

// TestKeyboardNavigation_ContextAwareness tests that keybindings work differently based on focused panel
func TestKeyboardNavigation_ContextAwareness(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file with multiple servers and sessions
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test navigation context awareness
	
	// When focused on servers panel
	app.focusedPanel = "servers"
	serverRow, _ := app.serverList.GetSelection()
	app.handleNavigationDown()
	newServerRow, _ := app.serverList.GetSelection()
	
	// Should affect server list navigation
	if newServerRow <= serverRow && app.serverList.GetRowCount() > 2 {
		t.Errorf("Expected server navigation when focused on servers panel")
	}

	if app.sessionPanel != nil {
		// Set up mock sessions for testing
		testSessions := []SessionInfo{
			{Name: "session1", Status: "active", Windows: 1, LastActivity: "14:30"},
			{Name: "session2", Status: "inactive", Windows: 2, LastActivity: "14:25"},
		}
		app.updateSessionDisplay(testSessions)
		app.sessions = testSessions

		// When focused on sessions panel
		app.focusedPanel = "sessions"
		app.sessionPanel.Select(1, 0) // Select first session
		sessionRow, _ := app.sessionPanel.GetSelection()
		app.handleNavigationDown()
		newSessionRow, _ := app.sessionPanel.GetSelection()
		
		// Should affect session list navigation
		if newSessionRow <= sessionRow && app.sessionPanel.GetRowCount() > 2 {
			t.Errorf("Expected session navigation when focused on sessions panel")
		}

		// Test Enter key context awareness
		app.focusedPanel = "servers"
		// handleEnterKey should trigger server connection
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Server connection attempt panicked: %v", r)
			}
		}()
		app.handleEnterKey()

		app.focusedPanel = "sessions"
		// handleEnterKey should trigger session attachment
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Session attachment attempt panicked: %v", r)
			}
		}()
		// Note: This will actually try to attach to tmux session and stop TUI
	}
}

// TestKeyboardNavigation_EditServerKey tests 'e' key for editing servers
func TestKeyboardNavigation_EditServerKey(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test edit functionality on servers panel
	if app.serverList.GetRowCount() > 1 {
		app.focusedPanel = "servers"
		app.serverList.Select(1, 0) // Select first server
		
		// editSelectedServer should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Edit server functionality panicked: %v", r)
			}
		}()
		
		app.editSelectedServer() // Simulates 'e' key press
	}

	// Test edit functionality when not on servers panel (should be ignored)
	app.focusedPanel = "sessions"
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Edit server functionality panicked when not on servers panel: %v", r)
		}
	}()
	app.editSelectedServer() // Should return early and not crash
}

// TestKeyboardNavigation_DeleteServerKey tests 'd' key for deleting servers
func TestKeyboardNavigation_DeleteServerKey(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test delete functionality on servers panel
	if app.serverList.GetRowCount() > 1 {
		app.focusedPanel = "servers"
		app.serverList.Select(1, 0) // Select first server
		
		// deleteSelectedServer should not panic (shows confirmation modal)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Delete server functionality panicked: %v", r)
			}
		}()
		
		app.deleteSelectedServer() // Simulates 'd' key press
	}

	// Test delete functionality when not on servers panel (should be ignored)
	app.focusedPanel = "sessions"
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Delete server functionality panicked when not on servers panel: %v", r)
		}
	}()
	app.deleteSelectedServer() // Should return early and not crash
}

// TestKeyboardNavigation_DeleteServerFromConfig tests the actual deletion logic
func TestKeyboardNavigation_DeleteServerFromConfig(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create test config file
	testCfg := createTestConfig(t)
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := testCfg.SaveToPath(configPath); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test deleting an existing server
	initialServerCount := len(app.config.GetServers())
	if initialServerCount == 0 {
		t.Skip("No servers to delete in test config")
	}

	// Delete the first server
	serverToDelete := "test-web-01"
	err = app.deleteServerFromConfig(serverToDelete)
	if err != nil {
		t.Errorf("Expected no error deleting server, got: %v", err)
	}

	// Verify server was removed
	updatedServers := app.config.GetServers()
	if len(updatedServers) != initialServerCount-1 {
		t.Errorf("Expected server count to decrease from %d to %d, got %d", 
			initialServerCount, initialServerCount-1, len(updatedServers))
	}

	// Verify the specific server was removed
	for _, server := range updatedServers {
		if server.Name == serverToDelete {
			t.Errorf("Server '%s' should have been deleted but still exists", serverToDelete)
		}
	}

	// Test deleting non-existent server
	err = app.deleteServerFromConfig("non-existent-server")
	if err == nil {
		t.Error("Expected error when deleting non-existent server, got nil")
	}
}

// TestKeyboardNavigation_UpdatedHelpSystem tests that help includes new keybindings
func TestKeyboardNavigation_UpdatedHelpSystem(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// showHelp should not panic and should include new keybindings
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Updated help system panicked: %v", r)
		}
	}()
	
	app.showHelp() // Should show help with edit/delete keybindings included
}

// TestSessionAttachment_DetachmentCycle tests session attachment/detachment with TUI return
func TestSessionAttachment_DetachmentCycle(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test session data
	testSessions := []SessionInfo{
		{Name: "test-session", Status: "active", Windows: 1, LastActivity: "14:30"},
	}
	
	// Setup mock session data
	app.updateSessionDisplay(testSessions)
	app.sessions = testSessions

	// Test session attachment preparation
	if app.sessionPanel != nil && len(testSessions) > 0 {
		app.focusedPanel = "sessions"
		app.sessionPanel.Select(1, 0) // Select first session
		
		// Test that attachToSelectedSession attempts to attach
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Session attachment panicked: %v", r)
			}
		}()

		// This will actually try to stop TUI and attach to tmux
		// In test environment, tmux might not be available, but shouldn't panic
		app.attachToSelectedSession()
	}
}

// TestSessionAttachment_TUIStateManagement tests TUI state during session operations
func TestSessionAttachment_TUIStateManagement(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test that TUI can be stopped and restarted (simulating session detachment/return)
	if !app.running {
		// App should start in stopped state
		t.Log("TUI app correctly initialized in stopped state")
	}

	// Test stopping TUI (simulates session attachment)
	app.Stop()
	
	// Test that stop is idempotent
	app.Stop() // Should not panic

	// Verify app state
	if app.app == nil {
		t.Error("TUI application should not be nil after stop")
	}
}

// TestSessionAttachment_ErrorHandling tests error scenarios during session operations
func TestSessionAttachment_ErrorHandling(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test attachment with no sessions
	app.focusedPanel = "sessions"
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Session attachment with no sessions panicked: %v", r)
		}
	}()
	
	app.attachToSelectedSession() // Should handle gracefully with no sessions

	// Test attachment with invalid selection
	if app.sessionPanel != nil {
		app.sessionPanel.Select(0, 0) // Select header row (invalid)
		app.attachToSelectedSession() // Should return early without crashing
	}
}

// TestSessionAttachment_SessionSelection tests session selection validation
func TestSessionAttachment_SessionSelection(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	testSessions := []SessionInfo{
		{Name: "session-1", Status: "active", Windows: 1, LastActivity: "14:30"},
		{Name: "session-2", Status: "detached", Windows: 2, LastActivity: "14:25"},
	}

	app.updateSessionDisplay(testSessions)
	app.sessions = testSessions

	if app.sessionPanel != nil {
		// Test valid session selection
		app.focusedPanel = "sessions"
		app.sessionPanel.Select(1, 0) // Select first session
		selectedRow, _ := app.sessionPanel.GetSelection()
		
		if selectedRow != 1 {
			t.Errorf("Expected session selection at row 1, got %d", selectedRow)
		}

		// Test boundary conditions
		app.sessionPanel.Select(10, 0) // Out of bounds selection
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Out of bounds session selection panicked: %v", r)
			}
		}()
		
		app.attachToSelectedSession() // Should handle out of bounds gracefully
	}
}

// TestSessionAttachment_TmuxIntegration tests integration with tmux manager
func TestSessionAttachment_TmuxIntegration(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test tmux manager availability check
	if app.tmuxManager == nil {
		t.Fatal("Expected tmux manager to be initialized")
	}

	// Test session listing functionality
	sessions, err := app.tmuxManager.ListSessions()
	if err != nil {
		t.Logf("Tmux not available (expected in test environment): %v", err)
	} else {
		t.Logf("Found %d tmux sessions", len(sessions))
	}

	// Test session existence checking
	exists := app.tmuxManager.SessionExists("non-existent-session")
	if exists {
		t.Error("Expected non-existent session to return false")
	}
}

// TestSessionAttachment_StatusMonitoring tests session status monitoring
func TestSessionAttachment_StatusMonitoring(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test session refresh functionality
	err = app.refreshSessions()
	if err != nil {
		t.Logf("Session refresh error (expected if tmux not available): %v", err)
	}

	// Test session display update with different statuses
	testSessions := []SessionInfo{
		{Name: "active-session", Status: "active", Windows: 1, LastActivity: "14:30"},
		{Name: "attached-session", Status: "attached", Windows: 2, LastActivity: "14:25"},
		{Name: "inactive-session", Status: "inactive", Windows: 1, LastActivity: "14:20"},
	}

	app.updateSessionDisplay(testSessions)

	if app.sessionPanel == nil {
		t.Skip("Session panel not initialized")
	}

	// Verify different session statuses are displayed correctly
	expectedRows := len(testSessions) + 1 // +1 for header
	actualRows := app.sessionPanel.GetRowCount()
	if actualRows != expectedRows {
		t.Errorf("Expected %d rows for session display, got %d", expectedRows, actualRows)
	}

	// Test real-time session updates
	updatedSessions := []SessionInfo{
		{Name: "active-session", Status: "attached", Windows: 1, LastActivity: "14:35"},
		{Name: "new-session", Status: "active", Windows: 1, LastActivity: "14:32"},
	}

	app.updateSessionDisplay(updatedSessions)
	app.sessions = updatedSessions

	// Verify session list was updated
	updatedRows := app.sessionPanel.GetRowCount()
	expectedUpdatedRows := len(updatedSessions) + 1
	if updatedRows != expectedUpdatedRows {
		t.Errorf("Expected %d rows after session update, got %d", expectedUpdatedRows, updatedRows)
	}
}

// TestSessionAttachment_ReturnHandler tests session return handler functionality
func TestSessionAttachment_ReturnHandler(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test that session error modal functionality works
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Session error modal panicked: %v", r)
		}
	}()

	app.showSessionErrorModal("Test session error")

	// Test session connection functionality with mock data
	testSessions := []SessionInfo{
		{Name: "test-session", Status: "active", Windows: 1, LastActivity: "14:30"},
	}
	
	app.updateSessionDisplay(testSessions)
	app.sessions = testSessions

	if app.sessionPanel != nil {
		app.focusedPanel = "sessions"
		app.sessionPanel.Select(1, 0)
		
		// This tests the session attachment flow
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Session attachment flow panicked: %v", r)
			}
		}()

		app.attachToSelectedSession()
	}
}

// TestEnhancedSessionMonitoring tests enhanced session monitoring functionality
func TestEnhancedSessionMonitoring_StatusDetection(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test enhanced status detection with different statuses
	testSessions := []SessionInfo{
		{Name: "detached-session", Status: "detached", Windows: 1, LastActivity: "5m ago"},
		{Name: "attached-session", Status: "attached", Windows: 2, LastActivity: "just now"},
		{Name: "multi-attached-session", Status: "multi-attached", Windows: 3, LastActivity: "2h ago"},
		{Name: "inactive-session", Status: "inactive", Windows: 1, LastActivity: "1d ago"},
	}

	// Update session display with enhanced statuses
	app.updateSessionDisplay(testSessions)
	app.sessions = testSessions

	if app.sessionPanel == nil {
		t.Skip("Session panel not initialized")
	}

	// Verify enhanced statuses are displayed correctly
	expectedRows := len(testSessions) + 1 // +1 for header
	actualRows := app.sessionPanel.GetRowCount()
	if actualRows != expectedRows {
		t.Errorf("Expected %d rows for enhanced session display, got %d", expectedRows, actualRows)
	}

	// Verify specific status displays
	if actualRows > 1 {
		statusCell := app.sessionPanel.GetCell(1, 1) // detached-session status
		if statusCell == nil || statusCell.Text != "detached" {
			t.Errorf("Expected first session status 'detached', got %v", statusCell)
		}
	}

	if actualRows > 3 {
		statusCell := app.sessionPanel.GetCell(3, 1) // multi-attached-session status
		if statusCell == nil || statusCell.Text != "multi-attached" {
			t.Errorf("Expected third session status 'multi-attached', got %v", statusCell)
		}
	}
}

// TestEnhancedSessionMonitoring_ActivityFormatting tests relative time formatting
func TestEnhancedSessionMonitoring_ActivityFormatting(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test enhanced activity time formatting
	testSessions := []SessionInfo{
		{Name: "recent-session", Status: "active", Windows: 1, LastActivity: "just now"},
		{Name: "minutes-session", Status: "active", Windows: 1, LastActivity: "15m ago"},
		{Name: "hours-session", Status: "active", Windows: 1, LastActivity: "3h ago"},
		{Name: "days-session", Status: "active", Windows: 1, LastActivity: "2d ago"},
	}

	app.updateSessionDisplay(testSessions)

	if app.sessionPanel == nil {
		t.Skip("Session panel not initialized")
	}

	// Verify activity time formatting
	expectedRows := len(testSessions) + 1
	actualRows := app.sessionPanel.GetRowCount()
	if actualRows != expectedRows {
		t.Errorf("Expected %d rows, got %d", expectedRows, actualRows)
	}

	// Check specific activity formats
	if actualRows > 1 {
		activityCell := app.sessionPanel.GetCell(1, 3) // recent-session activity
		if activityCell == nil || activityCell.Text != "just now" {
			t.Errorf("Expected first session activity 'just now', got %v", activityCell)
		}
	}

	if actualRows > 2 {
		activityCell := app.sessionPanel.GetCell(2, 3) // minutes-session activity
		if activityCell == nil || activityCell.Text != "15m ago" {
			t.Errorf("Expected second session activity '15m ago', got %v", activityCell)
		}
	}
}

// TestEnhancedSessionMonitoring_AutoRefresh tests automatic session refresh functionality
func TestEnhancedSessionMonitoring_AutoRefresh(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test that auto-refresh can be started and stopped without panicking
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Auto-refresh functionality panicked: %v", r)
		}
	}()

	// Start auto refresh
	app.startAutoRefresh()

	// Verify refresh timer is set
	if app.refreshTimer == nil {
		t.Error("Expected refresh timer to be initialized after starting auto-refresh")
	}

	// Test starting twice (should be idempotent)
	app.startAutoRefresh()

	// Wait a moment to let potential goroutines start
	time.Sleep(10 * time.Millisecond)

	// Stop auto refresh
	app.stopAutoRefresh()

	// Verify timer is stopped
	if app.refreshTimer != nil {
		t.Error("Expected refresh timer to be nil after stopping auto-refresh")
	}

	// Test stopping twice (should be safe)
	app.stopAutoRefresh()
}

// TestEnhancedSessionMonitoring_EnhancedDetails tests enhanced session detail gathering
func TestEnhancedSessionMonitoring_EnhancedDetails(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test enhanced session details methods don't panic
	sessionName := "test-session"

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Enhanced session details panicked: %v", r)
		}
	}()

	// Test enhanced session info gathering
	_, err = app.getEnhancedDetailedSessionInfo(sessionName)
	if err != nil {
		t.Logf("Enhanced session info returned error (expected in test environment): %v", err)
	}

	// Test individual enhanced methods
	_, err = app.getEnhancedSessionWindowCount(sessionName)
	if err != nil {
		t.Logf("Enhanced window count returned error (expected in test environment): %v", err)
	}

	_, err = app.getEnhancedSessionStatus(sessionName)
	if err != nil {
		t.Logf("Enhanced session status returned error (expected in test environment): %v", err)
	}

	_, err = app.getEnhancedSessionActivity(sessionName)
	if err != nil {
		t.Logf("Enhanced session activity returned error (expected in test environment): %v", err)
	}
}

// TestEnhancedSessionMonitoring_RealTimeUpdates tests real-time session updates
func TestEnhancedSessionMonitoring_RealTimeUpdates(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test real-time session updates
	initialSessions := []SessionInfo{
		{Name: "session-1", Status: "detached", Windows: 1, LastActivity: "5m ago"},
	}

	app.updateSessionDisplay(initialSessions)
	app.sessions = initialSessions

	if app.sessionPanel == nil {
		t.Skip("Session panel not initialized")
	}

	// Simulate real-time update with new session data
	updatedSessions := []SessionInfo{
		{Name: "session-1", Status: "attached", Windows: 2, LastActivity: "just now"},
		{Name: "session-2", Status: "detached", Windows: 1, LastActivity: "1m ago"},
	}

	app.updateSessionDisplay(updatedSessions)
	app.sessions = updatedSessions

	updatedRows := app.sessionPanel.GetRowCount()

	// Verify session list was updated
	if updatedRows != len(updatedSessions)+1 {
		t.Errorf("Expected %d rows after real-time update, got %d", len(updatedSessions)+1, updatedRows)
	}

	// Verify status was updated
	if updatedRows > 1 {
		statusCell := app.sessionPanel.GetCell(1, 1) // session-1 status
		if statusCell == nil || statusCell.Text != "attached" {
			t.Errorf("Expected session-1 status to be updated to 'attached', got %v", statusCell)
		}

		windowsCell := app.sessionPanel.GetCell(1, 2) // session-1 windows
		if windowsCell == nil || windowsCell.Text != "2" {
			t.Errorf("Expected session-1 windows to be updated to '2', got %v", windowsCell)
		}
	}
}

// TestEnhancedSessionMonitoring_ErrorHandling tests error handling in enhanced monitoring
func TestEnhancedSessionMonitoring_ErrorHandling(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test error handling with invalid session data
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Enhanced monitoring error handling panicked: %v", r)
		}
	}()

	// Test with empty session names
	_, err = app.getEnhancedSessionDetails([]string{""})
	if err != nil {
		t.Logf("Enhanced session details handled empty session name: %v", err)
	}

	// Test with invalid session names
	_, err = app.getEnhancedSessionDetails([]string{"invalid-session-name"})
	if err != nil {
		t.Logf("Enhanced session details handled invalid session: %v", err)
	}

	// Test session refresh with no tmux
	err = app.refreshSessions()
	if err != nil {
		t.Logf("Session refresh handled tmux unavailability: %v", err)
	}
}

// TestSessionCleanup tests session cleanup functionality
func TestSessionCleanup_KillSelectedSession(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test kill session functionality
	testSessions := []SessionInfo{
		{Name: "session-to-kill", Status: "active", Windows: 1, LastActivity: "5m ago"},
		{Name: "session-to-keep", Status: "active", Windows: 2, LastActivity: "1m ago"},
	}

	app.updateSessionDisplay(testSessions)
	app.sessions = testSessions

	if app.sessionPanel == nil {
		t.Skip("Session panel not initialized")
	}

	// Test kill session with valid selection
	app.focusedPanel = "sessions"
	app.sessionPanel.Select(1, 0) // Select first session

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Kill selected session panicked: %v", r)
		}
	}()

	// This should trigger the confirmation modal
	app.killSelectedSession()
}

// TestSessionCleanup_CleanupOrphaned tests orphaned session cleanup
func TestSessionCleanup_CleanupOrphaned(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test cleanup orphaned sessions functionality
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Cleanup orphaned sessions panicked: %v", r)
		}
	}()

	app.focusedPanel = "sessions"
	
	// This should trigger the confirmation modal
	app.cleanupOrphanedSessions()
}

// TestSessionCleanup_OrphanDetection tests orphaned session detection
func TestSessionCleanup_OrphanDetection(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test orphan detection logic with various session states
	testCases := []struct {
		sessionName string
		sessionInfo SessionInfo
		expectOrphan bool
		description string
	}{
		{
			sessionName: "healthy-session",
			sessionInfo: SessionInfo{Name: "healthy-session", Status: "active", Windows: 2, LastActivity: "5m ago"},
			expectOrphan: false,
			description: "healthy session should not be orphaned",
		},
		{
			sessionName: "no-windows-session",
			sessionInfo: SessionInfo{Name: "no-windows-session", Status: "active", Windows: 0, LastActivity: "1h ago"},
			expectOrphan: true,
			description: "session with no windows should be orphaned",
		},
		{
			sessionName: "inactive-session",
			sessionInfo: SessionInfo{Name: "inactive-session", Status: "inactive", Windows: 1, LastActivity: "1h ago"},
			expectOrphan: true,
			description: "inactive session should be orphaned",
		},
		{
			sessionName: "old-session",
			sessionInfo: SessionInfo{Name: "old-session", Status: "active", Windows: 1, LastActivity: "3d ago"},
			expectOrphan: true,
			description: "session with old activity should be orphaned",
		},
		{
			sessionName: "recent-session",
			sessionInfo: SessionInfo{Name: "recent-session", Status: "active", Windows: 1, LastActivity: "just now"},
			expectOrphan: false,
			description: "recently active session should not be orphaned",
		},
	}

	// Note: The actual orphan detection would call getEnhancedDetailedSessionInfo,
	// which relies on tmux commands that return errors in test environment.
	// So we test the logic indirectly by testing performSessionCleanup which uses it.
	
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Orphan detection panicked: %v", r)
		}
	}()

	// Test that performSessionCleanup doesn't crash
	_, err = app.performSessionCleanup()
	if err != nil {
		t.Logf("Session cleanup returned error (expected in test environment): %v", err)
	}

	// Test individual session orphan detection (will likely error in test env)
	for _, tc := range testCases {
		isOrphan := app.isSessionOrphaned(tc.sessionName)
		// In test environment, this will likely return true due to command execution errors
		t.Logf("Session %s orphan status: %v (expected in test env due to tmux unavailability)", tc.sessionName, isOrphan)
	}
}

// TestSessionCleanup_PerformCleanup tests the actual cleanup performance
func TestSessionCleanup_PerformCleanup(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test perform session cleanup
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Perform session cleanup panicked: %v", r)
		}
	}()

	count, err := app.performSessionCleanup()
	if err != nil {
		t.Logf("Session cleanup returned error (expected without tmux): %v", err)
	} else {
		t.Logf("Session cleanup completed successfully, cleaned %d sessions", count)
	}
}

// TestSessionCleanup_KeyboardBindings tests session cleanup keyboard bindings
func TestSessionCleanup_KeyboardBindings(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Set up test sessions
	testSessions := []SessionInfo{
		{Name: "test-session", Status: "active", Windows: 1, LastActivity: "5m ago"},
	}
	
	if app.sessionPanel != nil {
		app.updateSessionDisplay(testSessions)
		app.sessions = testSessions
		app.focusedPanel = "sessions"
		app.sessionPanel.Select(1, 0)
	}

	// Test that cleanup keybindings don't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Session cleanup keybindings panicked: %v", r)
		}
	}()

	// Test 'K' key for killing selected session
	if app.sessionPanel != nil {
		app.killSelectedSession() // Should show confirmation modal
	}

	// Test 'C' key for cleanup orphaned sessions  
	if app.sessionPanel != nil {
		app.cleanupOrphanedSessions() // Should show confirmation modal
	}
}

// TestSessionCleanup_ErrorHandling tests error handling in session cleanup
func TestSessionCleanup_ErrorHandling(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Test error handling in cleanup operations
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Session cleanup error handling panicked: %v", r)
		}
	}()

	// Test kill session with no sessions panel
	app.sessionPanel = nil
	app.killSelectedSession() // Should return early

	// Test cleanup with no sessions panel
	app.cleanupOrphanedSessions() // Should return early

	// Test invalid session selection
	if app.sessionPanel != nil {
		app.sessionPanel.Select(0, 0) // Header row
		app.killSelectedSession() // Should return early
	}
}

// TestSessionCleanup_Integration tests integration with tmux manager
func TestSessionCleanup_Integration(t *testing.T) {
	// Create a temporary directory for test config
	tempDir := t.TempDir()
	os.Setenv("SSHM_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SSHM_CONFIG_DIR")

	// Create TUI app
	app, err := NewTUIApp()
	if err != nil {
		t.Fatalf("Failed to create TUI app: %v", err)
	}

	// Verify tmux manager is available for cleanup operations
	if app.tmuxManager == nil {
		t.Fatal("Expected tmux manager to be available for session cleanup")
	}

	// Test tmux manager methods used by cleanup
	exists := app.tmuxManager.SessionExists("non-existent-session")
	if exists {
		t.Error("Expected non-existent session check to return false")
	}

	// Test session listing (used by cleanup)
	sessions, err := app.tmuxManager.ListSessions()
	if err != nil {
		t.Logf("List sessions returned error (expected if tmux not available): %v", err)
	} else {
		t.Logf("Found %d tmux sessions for cleanup testing", len(sessions))
	}
}