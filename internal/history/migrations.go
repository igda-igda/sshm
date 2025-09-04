package history

import (
	"fmt"
	"sort"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
}

// getMigrations returns all available database migrations in order
func getMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Description: "Initial schema creation with migration tracking",
			Up: `
				CREATE TABLE IF NOT EXISTS schema_migrations (
					version INTEGER PRIMARY KEY,
					description TEXT NOT NULL,
					applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS connection_history (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					server_name TEXT NOT NULL,
					profile_name TEXT,
					host TEXT NOT NULL,
					user TEXT NOT NULL,
					port INTEGER NOT NULL,
					connection_type TEXT NOT NULL,
					status TEXT NOT NULL,
					start_time DATETIME NOT NULL,
					end_time DATETIME,
					duration_seconds INTEGER,
					error_message TEXT,
					jump_hosts TEXT,
					session_id TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);

				CREATE TABLE IF NOT EXISTS session_health (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					session_id TEXT NOT NULL,
					server_name TEXT NOT NULL,
					check_time DATETIME NOT NULL,
					status TEXT NOT NULL,
					response_time_ms INTEGER,
					error_message TEXT,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP
				);
			`,
			Down: `
				DROP TABLE IF EXISTS session_health;
				DROP TABLE IF EXISTS connection_history;
				DROP TABLE IF EXISTS schema_migrations;
			`,
		},
		{
			Version:     2,
			Description: "Add database indexes",
			Up: `
				CREATE INDEX IF NOT EXISTS idx_connection_history_server ON connection_history(server_name);
				CREATE INDEX IF NOT EXISTS idx_connection_history_profile ON connection_history(profile_name);
				CREATE INDEX IF NOT EXISTS idx_connection_history_start_time ON connection_history(start_time);
				CREATE INDEX IF NOT EXISTS idx_connection_history_status ON connection_history(status);

				CREATE INDEX IF NOT EXISTS idx_session_health_session ON session_health(session_id);
				CREATE INDEX IF NOT EXISTS idx_session_health_check_time ON session_health(check_time);
			`,
			Down: `
				DROP INDEX IF EXISTS idx_session_health_check_time;
				DROP INDEX IF EXISTS idx_session_health_session;
				DROP INDEX IF EXISTS idx_connection_history_status;
				DROP INDEX IF EXISTS idx_connection_history_start_time;
				DROP INDEX IF EXISTS idx_connection_history_profile;
				DROP INDEX IF EXISTS idx_connection_history_server;
			`,
		},
		{
			Version:     3,
			Description: "Add connection statistics view",
			Up: `
				CREATE VIEW IF NOT EXISTS connection_stats AS
				SELECT 
					server_name,
					profile_name,
					COUNT(*) as total_connections,
					SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful_connections,
					CAST(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS REAL) / COUNT(*) as success_rate,
					AVG(CASE WHEN duration_seconds IS NOT NULL THEN duration_seconds ELSE NULL END) as avg_duration_seconds,
					MAX(start_time) as last_connection,
					MIN(start_time) as first_connection
				FROM connection_history 
				GROUP BY server_name, profile_name;
			`,
			Down: `
				DROP VIEW IF EXISTS connection_stats;
			`,
		},
	}
}

// getCurrentVersion gets the current database schema version
func (h *HistoryManager) getCurrentVersion() (int, error) {
	// First, ensure the migrations table exists
	_, err := h.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to create migrations table: %w", err)
	}

	var version int
	err = h.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return version, nil
}

// applyMigration applies a single migration
func (h *HistoryManager) applyMigration(migration Migration) error {
	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start migration transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute the migration
	if _, err := tx.Exec(migration.Up); err != nil {
		return fmt.Errorf("failed to execute migration %d: %w", migration.Version, err)
	}

	// Record the migration
	if _, err := tx.Exec(
		"INSERT OR REPLACE INTO schema_migrations (version, description) VALUES (?, ?)",
		migration.Version, migration.Description,
	); err != nil {
		return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
	}

	return nil
}

// runMigrations runs all pending migrations
func (h *HistoryManager) runMigrations() error {
	currentVersion, err := h.getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	migrations := getMigrations()
	
	// Sort migrations by version to ensure proper order
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Apply migrations that haven't been applied yet
	for _, migration := range migrations {
		if migration.Version > currentVersion {
			fmt.Printf("Applying migration %d: %s\n", migration.Version, migration.Description)
			if err := h.applyMigration(migration); err != nil {
				return fmt.Errorf("migration %d failed: %w", migration.Version, err)
			}
		}
	}

	return nil
}

// rollbackMigration rolls back a specific migration
func (h *HistoryManager) rollbackMigration(version int) error {
	migrations := getMigrations()
	
	var targetMigration *Migration
	for _, migration := range migrations {
		if migration.Version == version {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration version %d not found", version)
	}

	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start rollback transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute the rollback
	if _, err := tx.Exec(targetMigration.Down); err != nil {
		return fmt.Errorf("failed to execute rollback for migration %d: %w", version, err)
	}

	// Remove the migration record
	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = ?", version); err != nil {
		return fmt.Errorf("failed to remove migration record for version %d: %w", version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback for migration %d: %w", version, err)
	}

	return nil
}

// GetAppliedMigrations returns a list of applied migrations
func (h *HistoryManager) GetAppliedMigrations() ([]Migration, error) {
	rows, err := h.db.Query("SELECT version, description FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var appliedMigrations []Migration
	for rows.Next() {
		var migration Migration
		err := rows.Scan(&migration.Version, &migration.Description)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		appliedMigrations = append(appliedMigrations, migration)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}

	return appliedMigrations, nil
}

// ValidateSchema validates that the database schema matches expectations
func (h *HistoryManager) ValidateSchema() error {
	expectedTables := []string{"connection_history", "session_health", "schema_migrations"}
	
	for _, table := range expectedTables {
		var count int
		err := h.db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
		
		if count == 0 {
			return fmt.Errorf("required table %s not found", table)
		}
	}

	// Check for required indexes
	expectedIndexes := []string{
		"idx_connection_history_server",
		"idx_connection_history_profile", 
		"idx_connection_history_start_time",
		"idx_connection_history_status",
		"idx_session_health_session",
		"idx_session_health_check_time",
	}

	for _, index := range expectedIndexes {
		var count int
		err := h.db.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?",
			index,
		).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check index %s: %w", index, err)
		}
		
		if count == 0 {
			return fmt.Errorf("required index %s not found", index)
		}
	}

	return nil
}