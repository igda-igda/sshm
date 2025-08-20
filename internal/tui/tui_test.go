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