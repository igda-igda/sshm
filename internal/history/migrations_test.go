package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrationSystem(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "sshm-migration-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "migration_test.db")

	// Test initial database creation and migrations
	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Verify all migrations were applied
	appliedMigrations, err := manager.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("Failed to get applied migrations: %v", err)
	}

	expectedMigrations := len(getMigrations())
	if len(appliedMigrations) != expectedMigrations {
		t.Errorf("Expected %d applied migrations, got %d", expectedMigrations, len(appliedMigrations))
	}

	// Verify schema is valid
	err = manager.ValidateSchema()
	if err != nil {
		t.Fatalf("Schema validation failed: %v", err)
	}

	// Test that reopening database doesn't apply migrations again
	manager.Close()

	manager2, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen history manager: %v", err)
	}
	defer manager2.Close()

	appliedMigrations2, err := manager2.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("Failed to get applied migrations on reopen: %v", err)
	}

	if len(appliedMigrations2) != expectedMigrations {
		t.Errorf("Expected %d applied migrations on reopen, got %d", expectedMigrations, len(appliedMigrations2))
	}
}

func TestMigrationVersioning(t *testing.T) {
	migrations := getMigrations()

	// Verify migrations are properly versioned
	versionsSeen := make(map[int]bool)
	for _, migration := range migrations {
		if migration.Version <= 0 {
			t.Errorf("Invalid migration version %d, must be positive", migration.Version)
		}

		if versionsSeen[migration.Version] {
			t.Errorf("Duplicate migration version %d", migration.Version)
		}
		versionsSeen[migration.Version] = true

		if migration.Description == "" {
			t.Errorf("Migration %d has empty description", migration.Version)
		}

		if migration.Up == "" {
			t.Errorf("Migration %d has empty Up script", migration.Version)
		}

		if migration.Down == "" {
			t.Errorf("Migration %d has empty Down script", migration.Version)
		}
	}
}

func TestMigrationRollback(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "sshm-rollback-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "rollback_test.db")

	// Create manager and apply all migrations
	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Get initial migration count
	appliedMigrations, err := manager.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("Failed to get applied migrations: %v", err)
	}

	initialCount := len(appliedMigrations)
	if initialCount == 0 {
		t.Fatal("No migrations were applied")
	}

	// Find a non-critical migration to rollback (not version 1 which has schema_migrations)
	rollbackVersion := 0
	for _, migration := range appliedMigrations {
		if migration.Version > 1 && migration.Version > rollbackVersion {
			rollbackVersion = migration.Version
		}
	}

	if rollbackVersion == 0 {
		t.Skip("No suitable migration found for rollback test (need version > 1)")
	}

	// Rollback the selected migration
	err = manager.rollbackMigration(rollbackVersion)
	if err != nil {
		t.Fatalf("Failed to rollback migration %d: %v", rollbackVersion, err)
	}

	// Verify migration was rolled back
	appliedAfterRollback, err := manager.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("Failed to get applied migrations after rollback: %v", err)
	}

	if len(appliedAfterRollback) != initialCount-1 {
		t.Errorf("Expected %d applied migrations after rollback, got %d", 
			initialCount-1, len(appliedAfterRollback))
	}

	// Verify the specific migration was removed
	for _, migration := range appliedAfterRollback {
		if migration.Version == rollbackVersion {
			t.Errorf("Migration %d should have been rolled back but is still applied", rollbackVersion)
		}
	}
}

func TestSchemaValidation(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "sshm-validation-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "validation_test.db")

	// Create manager with proper schema
	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Test schema validation passes
	err = manager.ValidateSchema()
	if err != nil {
		t.Fatalf("Schema validation should pass but failed: %v", err)
	}

	// Test that validation would fail with missing table
	// Drop a table to test validation failure
	_, err = manager.db.Exec("DROP TABLE connection_history")
	if err != nil {
		t.Fatalf("Failed to drop table for validation test: %v", err)
	}

	// Schema validation should now fail
	err = manager.ValidateSchema()
	if err == nil {
		t.Error("Schema validation should fail with missing table but passed")
	}
}

func TestCurrentVersionTracking(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "sshm-version-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "version_test.db")

	manager, err := NewHistoryManager(dbPath)
	if err != nil {
		t.Fatalf("Failed to create history manager: %v", err)
	}
	defer manager.Close()

	// Get current version
	version, err := manager.getCurrentVersion()
	if err != nil {
		t.Fatalf("Failed to get current version: %v", err)
	}

	migrations := getMigrations()
	expectedMaxVersion := 0
	for _, migration := range migrations {
		if migration.Version > expectedMaxVersion {
			expectedMaxVersion = migration.Version
		}
	}

	if version != expectedMaxVersion {
		t.Errorf("Expected current version %d, got %d", expectedMaxVersion, version)
	}
}