package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// HistoryManager handles connection history tracking and session monitoring
type HistoryManager struct {
	db *sql.DB
}

// ConnectionHistoryEntry represents a single connection attempt
type ConnectionHistoryEntry struct {
	ID              int       `json:"id"`
	ServerName      string    `json:"server_name"`
	ProfileName     string    `json:"profile_name,omitempty"`
	Host            string    `json:"host"`
	User            string    `json:"user"`
	Port            int       `json:"port"`
	ConnectionType  string    `json:"connection_type"` // 'single' or 'group'
	Status          string    `json:"status"`          // 'success', 'failed', 'timeout', 'cancelled'
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time,omitempty"`
	DurationSeconds int       `json:"duration_seconds,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	JumpHosts       []string  `json:"jump_hosts,omitempty"`
	SessionID       string    `json:"session_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// SessionHealthEntry represents session health monitoring data
type SessionHealthEntry struct {
	ID             int       `json:"id"`
	SessionID      string    `json:"session_id"`
	ServerName     string    `json:"server_name"`
	CheckTime      time.Time `json:"check_time"`
	Status         string    `json:"status"` // 'healthy', 'degraded', 'failed'
	ResponseTimeMs int       `json:"response_time_ms,omitempty"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ConnectionStats represents aggregated connection statistics
type ConnectionStats struct {
	ServerName            string        `json:"server_name"`
	ProfileName           string        `json:"profile_name"`
	TotalConnections      int           `json:"total_connections"`
	SuccessfulConnections int           `json:"successful_connections"`
	SuccessRate           float64       `json:"success_rate"`
	AverageDuration       float64       `json:"average_duration_seconds"`
	LastConnection        time.Time     `json:"last_connection"`
	FirstConnection       time.Time     `json:"first_connection"`
}

// HistoryFilter represents filtering options for connection history queries
type HistoryFilter struct {
	ServerName    string    `json:"server_name,omitempty"`
	ProfileName   string    `json:"profile_name,omitempty"`
	Status        string    `json:"status,omitempty"`
	ConnectionType string   `json:"connection_type,omitempty"`
	StartTime     time.Time `json:"start_time,omitempty"`
	EndTime       time.Time `json:"end_time,omitempty"`
	Limit         int       `json:"limit"`
	Offset        int       `json:"offset"`
}

// NewHistoryManager creates a new history manager with SQLite database
func NewHistoryManager(dbPath string) (*HistoryManager, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	manager := &HistoryManager{db: db}

	// Run database migrations to ensure schema is up to date
	if err := manager.runMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	return manager, nil
}

// Close closes the database connection
func (h *HistoryManager) Close() error {
	return h.db.Close()
}


// RecordConnection records a new connection history entry
func (h *HistoryManager) RecordConnection(entry ConnectionHistoryEntry) (int64, error) {
	jumpHostsJSON := ""
	if len(entry.JumpHosts) > 0 {
		jumpHostsBytes, err := json.Marshal(entry.JumpHosts)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal jump hosts: %w", err)
		}
		jumpHostsJSON = string(jumpHostsBytes)
	}

	query := `
		INSERT INTO connection_history (
			server_name, profile_name, host, user, port, connection_type, status,
			start_time, end_time, duration_seconds, error_message, jump_hosts, session_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var endTime interface{}
	if !entry.EndTime.IsZero() {
		endTime = entry.EndTime
	}

	var durationSeconds interface{}
	if entry.DurationSeconds > 0 {
		durationSeconds = entry.DurationSeconds
	}

	result, err := h.db.Exec(query,
		entry.ServerName, entry.ProfileName, entry.Host, entry.User, entry.Port,
		entry.ConnectionType, entry.Status, entry.StartTime, endTime,
		durationSeconds, entry.ErrorMessage, jumpHostsJSON, entry.SessionID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert connection history: %w", err)
	}

	return result.LastInsertId()
}

// UpdateConnectionEnd updates the end time and duration for a connection
func (h *HistoryManager) UpdateConnectionEnd(id int64, endTime time.Time, status string, errorMessage string) error {
	// Get the start time to calculate duration
	var startTime time.Time
	err := h.db.QueryRow("SELECT start_time FROM connection_history WHERE id = ?", id).Scan(&startTime)
	if err != nil {
		return fmt.Errorf("failed to get connection start time: %w", err)
	}

	duration := int(endTime.Sub(startTime).Seconds())

	query := `
		UPDATE connection_history 
		SET end_time = ?, duration_seconds = ?, status = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = h.db.Exec(query, endTime, duration, status, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to update connection end: %w", err)
	}

	return nil
}

// GetConnectionHistory retrieves connection history based on filter criteria
func (h *HistoryManager) GetConnectionHistory(filter HistoryFilter) ([]ConnectionHistoryEntry, error) {
	query := "SELECT id, server_name, profile_name, host, user, port, connection_type, status, start_time, end_time, duration_seconds, error_message, jump_hosts, session_id, created_at, updated_at FROM connection_history WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.ServerName != "" {
		query += fmt.Sprintf(" AND server_name = $%d", argIndex)
		args = append(args, filter.ServerName)
		argIndex++
	}

	if filter.ProfileName != "" {
		query += fmt.Sprintf(" AND profile_name = $%d", argIndex)
		args = append(args, filter.ProfileName)
		argIndex++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.ConnectionType != "" {
		query += fmt.Sprintf(" AND connection_type = $%d", argIndex)
		args = append(args, filter.ConnectionType)
		argIndex++
	}

	if !filter.StartTime.IsZero() {
		query += fmt.Sprintf(" AND start_time >= $%d", argIndex)
		args = append(args, filter.StartTime)
		argIndex++
	}

	if !filter.EndTime.IsZero() {
		query += fmt.Sprintf(" AND start_time <= $%d", argIndex)
		args = append(args, filter.EndTime)
		argIndex++
	}

	query += " ORDER BY start_time DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, filter.Offset)
		}
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query connection history: %w", err)
	}
	defer rows.Close()

	var history []ConnectionHistoryEntry
	for rows.Next() {
		var entry ConnectionHistoryEntry
		var endTime sql.NullTime
		var durationSeconds sql.NullInt64
		var errorMessage sql.NullString
		var jumpHostsJSON sql.NullString
		var sessionID sql.NullString
		var profileName sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.ServerName, &profileName, &entry.Host, &entry.User, &entry.Port,
			&entry.ConnectionType, &entry.Status, &entry.StartTime, &endTime,
			&durationSeconds, &errorMessage, &jumpHostsJSON, &sessionID,
			&entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection history row: %w", err)
		}

		if profileName.Valid {
			entry.ProfileName = profileName.String
		}

		if endTime.Valid {
			entry.EndTime = endTime.Time
		}

		if durationSeconds.Valid {
			entry.DurationSeconds = int(durationSeconds.Int64)
		}

		if errorMessage.Valid {
			entry.ErrorMessage = errorMessage.String
		}

		if sessionID.Valid {
			entry.SessionID = sessionID.String
		}

		if jumpHostsJSON.Valid && jumpHostsJSON.String != "" {
			if err := json.Unmarshal([]byte(jumpHostsJSON.String), &entry.JumpHosts); err != nil {
				// Log error but don't fail the query
				entry.JumpHosts = []string{}
			}
		}

		history = append(history, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating connection history rows: %w", err)
	}

	return history, nil
}

// GetConnectionStats retrieves connection statistics for a server/profile combination
func (h *HistoryManager) GetConnectionStats(serverName, profileName string) (*ConnectionStats, error) {
	// Query directly from the table instead of using the view to handle datetime parsing
	query := `
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
		WHERE server_name = ? AND (profile_name = ? OR (profile_name IS NULL AND ? = ''))
		GROUP BY server_name, profile_name
	`

	var stats ConnectionStats
	var avgDuration sql.NullFloat64
	var profile sql.NullString
	var lastConnectionStr, firstConnectionStr string

	err := h.db.QueryRow(query, serverName, profileName, profileName).Scan(
		&stats.ServerName, &profile, &stats.TotalConnections,
		&stats.SuccessfulConnections, &stats.SuccessRate, &avgDuration,
		&lastConnectionStr, &firstConnectionStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty stats if no history found
			return &ConnectionStats{
				ServerName:  serverName,
				ProfileName: profileName,
			}, nil
		}
		return nil, fmt.Errorf("failed to get connection stats: %w", err)
	}

	if profile.Valid {
		stats.ProfileName = profile.String
	}

	if avgDuration.Valid {
		stats.AverageDuration = avgDuration.Float64
	}

	// Parse datetime strings to time.Time
	if lastConnectionStr != "" {
		if parsedTime, err := time.Parse("2006-01-02 15:04:05", lastConnectionStr); err == nil {
			stats.LastConnection = parsedTime
		}
	}

	if firstConnectionStr != "" {
		if parsedTime, err := time.Parse("2006-01-02 15:04:05", firstConnectionStr); err == nil {
			stats.FirstConnection = parsedTime
		}
	}

	return &stats, nil
}

// RecordSessionHealth records a session health check
func (h *HistoryManager) RecordSessionHealth(entry SessionHealthEntry) error {
	query := `
		INSERT INTO session_health (session_id, server_name, check_time, status, response_time_ms, error_message)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	var responseTime interface{}
	if entry.ResponseTimeMs > 0 {
		responseTime = entry.ResponseTimeMs
	}

	_, err := h.db.Exec(query, entry.SessionID, entry.ServerName, entry.CheckTime,
		entry.Status, responseTime, entry.ErrorMessage)
	if err != nil {
		return fmt.Errorf("failed to record session health: %w", err)
	}

	return nil
}

// GetSessionHealth retrieves recent session health data
func (h *HistoryManager) GetSessionHealth(sessionID string, limit int) ([]SessionHealthEntry, error) {
	query := `
		SELECT id, session_id, server_name, check_time, status, response_time_ms, error_message, created_at
		FROM session_health 
		WHERE session_id = ?
		ORDER BY check_time DESC
		LIMIT ?
	`

	rows, err := h.db.Query(query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query session health: %w", err)
	}
	defer rows.Close()

	var healthEntries []SessionHealthEntry
	for rows.Next() {
		var entry SessionHealthEntry
		var responseTime sql.NullInt64
		var errorMessage sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.SessionID, &entry.ServerName, &entry.CheckTime,
			&entry.Status, &responseTime, &errorMessage, &entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session health row: %w", err)
		}

		if responseTime.Valid {
			entry.ResponseTimeMs = int(responseTime.Int64)
		}

		if errorMessage.Valid {
			entry.ErrorMessage = errorMessage.String
		}

		healthEntries = append(healthEntries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session health rows: %w", err)
	}

	return healthEntries, nil
}

// GetActiveSessionHealth gets the latest health status for active sessions
func (h *HistoryManager) GetActiveSessionHealth() (map[string]SessionHealthEntry, error) {
	query := `
		WITH latest_health AS (
			SELECT session_id, MAX(check_time) as latest_check
			FROM session_health 
			WHERE check_time > datetime('now', '-5 minutes')
			GROUP BY session_id
		)
		SELECT sh.id, sh.session_id, sh.server_name, sh.check_time, sh.status, 
			   sh.response_time_ms, sh.error_message, sh.created_at
		FROM session_health sh
		INNER JOIN latest_health lh ON sh.session_id = lh.session_id AND sh.check_time = lh.latest_check
		ORDER BY sh.check_time DESC
	`

	rows, err := h.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active session health: %w", err)
	}
	defer rows.Close()

	activeHealth := make(map[string]SessionHealthEntry)
	for rows.Next() {
		var entry SessionHealthEntry
		var responseTime sql.NullInt64
		var errorMessage sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.SessionID, &entry.ServerName, &entry.CheckTime,
			&entry.Status, &responseTime, &errorMessage, &entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan active session health row: %w", err)
		}

		if responseTime.Valid {
			entry.ResponseTimeMs = int(responseTime.Int64)
		}

		if errorMessage.Valid {
			entry.ErrorMessage = errorMessage.String
		}

		activeHealth[entry.SessionID] = entry
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating active session health rows: %w", err)
	}

	return activeHealth, nil
}

// CleanupOldHistory removes connection history older than the specified duration
func (h *HistoryManager) CleanupOldHistory(retentionPeriod time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-retentionPeriod)

	// Clean up connection history
	result, err := h.db.Exec("DELETE FROM connection_history WHERE start_time < ?", cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old connection history: %w", err)
	}

	historyDeleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted connection history count: %w", err)
	}

	// Clean up session health data
	_, err = h.db.Exec("DELETE FROM session_health WHERE check_time < ?", cutoffTime)
	if err != nil {
		return historyDeleted, fmt.Errorf("failed to cleanup old session health: %w", err)
	}

	return historyDeleted, nil
}

// GetRecentActivity gets recent connection activity summary
func (h *HistoryManager) GetRecentActivity(hours int) (map[string]int, error) {
	cutoffTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	query := `
		SELECT status, COUNT(*) as count
		FROM connection_history 
		WHERE start_time >= ?
		GROUP BY status
	`

	rows, err := h.db.Query(query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent activity: %w", err)
	}
	defer rows.Close()

	activity := make(map[string]int)
	for rows.Next() {
		var status string
		var count int

		err := rows.Scan(&status, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity row: %w", err)
		}

		activity[status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating activity rows: %w", err)
	}

	return activity, nil
}