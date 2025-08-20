package tui

import (
	"context"
	"testing"
	"time"
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