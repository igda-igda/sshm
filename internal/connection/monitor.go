package connection

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"sshm/internal/history"
	"sshm/internal/tmux"
)

// HealthMonitor manages session health monitoring
type HealthMonitor struct {
	historyManager *history.HistoryManager
	tmuxManager    *tmux.Manager
	sessions       map[string]*SessionInfo
	mutex          sync.RWMutex
	stopChan       chan struct{}
	interval       time.Duration
}

// SessionInfo tracks information about an active session
type SessionInfo struct {
	SessionID    string
	ServerName   string
	StartTime    time.Time
	LastCheck    time.Time
	LastStatus   string
	FailureCount int
}

// TmuxSessionInfo represents tmux session information
type TmuxSessionInfo struct {
	Name    string
	Created time.Time
	Windows int
}

// NewHealthMonitor creates a new session health monitor
func NewHealthMonitor(historyManager *history.HistoryManager, tmuxManager *tmux.Manager) *HealthMonitor {
	return &HealthMonitor{
		historyManager: historyManager,
		tmuxManager:    tmuxManager,
		sessions:       make(map[string]*SessionInfo),
		stopChan:       make(chan struct{}),
		interval:       30 * time.Second, // Default check every 30 seconds
	}
}

// StartMonitoring begins monitoring active sessions
func (hm *HealthMonitor) StartMonitoring(ctx context.Context) error {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	// Initial discovery of existing sessions
	if err := hm.discoverActiveSessions(); err != nil {
		return fmt.Errorf("failed to discover active sessions: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-hm.stopChan:
			return nil
		case <-ticker.C:
			if err := hm.performHealthChecks(); err != nil {
				// Log error but continue monitoring
				fmt.Printf("Health check error: %v\n", err)
			}
		}
	}
}

// Stop stops the health monitoring
func (hm *HealthMonitor) Stop() {
	close(hm.stopChan)
}

// AddSession adds a session to be monitored
func (hm *HealthMonitor) AddSession(sessionID, serverName string) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.sessions[sessionID] = &SessionInfo{
		SessionID:    sessionID,
		ServerName:   serverName,
		StartTime:    time.Now(),
		LastCheck:    time.Now(),
		LastStatus:   "healthy",
		FailureCount: 0,
	}
}

// RemoveSession removes a session from monitoring
func (hm *HealthMonitor) RemoveSession(sessionID string) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	delete(hm.sessions, sessionID)
}

// GetSessionInfo gets information about a monitored session
func (hm *HealthMonitor) GetSessionInfo(sessionID string) (*SessionInfo, bool) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	info, exists := hm.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	infoCopy := *info
	return &infoCopy, true
}

// GetActiveSessions returns a copy of all active session information
func (hm *HealthMonitor) GetActiveSessions() map[string]SessionInfo {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	result := make(map[string]SessionInfo)
	for id, info := range hm.sessions {
		result[id] = *info
	}
	return result
}

// SetCheckInterval sets the health check interval
func (hm *HealthMonitor) SetCheckInterval(interval time.Duration) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	hm.interval = interval
}

// discoverActiveSessions discovers currently active tmux sessions
func (hm *HealthMonitor) discoverActiveSessions() error {
	sessions, err := hm.getDetailedSessionInfo()
	if err != nil {
		return fmt.Errorf("failed to get detailed session info: %w", err)
	}

	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	// Clear existing sessions first
	hm.sessions = make(map[string]*SessionInfo)

	// Add discovered sessions
	for _, session := range sessions {
		// Extract server name from session name if possible
		// Session names are typically the server name or profile name
		serverName := session.Name
		
		hm.sessions[session.Name] = &SessionInfo{
			SessionID:    session.Name,
			ServerName:   serverName,
			StartTime:    session.Created,
			LastCheck:    time.Now(),
			LastStatus:   "unknown", // Will be determined on first health check
			FailureCount: 0,
		}
	}

	return nil
}

// getDetailedSessionInfo gets detailed information about all tmux sessions
func (hm *HealthMonitor) getDetailedSessionInfo() ([]TmuxSessionInfo, error) {
	sessionNames, err := hm.tmuxManager.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var sessions []TmuxSessionInfo
	for _, name := range sessionNames {
		// Get session info using the existing method
		info, err := hm.tmuxManager.GetSessionInfo(name)
		if err != nil {
			// Skip sessions we can't get info for, but log the error
			fmt.Printf("Warning: Failed to get info for session %s: %v\n", name, err)
			continue
		}

		session := TmuxSessionInfo{
			Name: name,
		}

		// Parse creation time
		if createdStr, ok := info["created"]; ok {
			// tmux created timestamp is in Unix epoch format
			if createdUnix, err := strconv.ParseInt(createdStr, 10, 64); err == nil {
				session.Created = time.Unix(createdUnix, 0)
			} else {
				// Fallback to current time if parsing fails
				session.Created = time.Now()
			}
		} else {
			session.Created = time.Now()
		}

		// Parse window count
		if windowsStr, ok := info["windows"]; ok {
			if windows, err := strconv.Atoi(windowsStr); err == nil {
				session.Windows = windows
			}
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// performHealthChecks performs health checks on all monitored sessions
func (hm *HealthMonitor) performHealthChecks() error {
	hm.mutex.Lock()
	sessionsCopy := make(map[string]*SessionInfo)
	for id, info := range hm.sessions {
		sessionsCopy[id] = info
	}
	hm.mutex.Unlock()

	// Discover current tmux sessions to check if they still exist
	currentSessions, err := hm.tmuxManager.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list current sessions: %w", err)
	}

	currentSessionIDs := make(map[string]bool)
	for _, sessionName := range currentSessions {
		currentSessionIDs[sessionName] = true
	}

	// Check health for each session
	for sessionID, info := range sessionsCopy {
		status := "healthy"
		responseTime := 0
		errorMessage := ""

		// Check if session still exists in tmux
		if !currentSessionIDs[sessionID] {
			status = "failed"
			errorMessage = "Session no longer exists in tmux"
			
			// Remove from monitoring since session is gone
			hm.RemoveSession(sessionID)
		} else {
			// Session exists, perform health check
			startTime := time.Now()
			healthy, err := hm.checkSessionHealth(sessionID)
			responseTime = int(time.Since(startTime).Milliseconds())

			if err != nil {
				status = "failed"
				errorMessage = err.Error()
				info.FailureCount++
			} else if !healthy {
				status = "degraded"
				info.FailureCount++
			} else {
				status = "healthy"
				info.FailureCount = 0 // Reset failure count on success
			}

			// Update session info
			hm.mutex.Lock()
			if existingInfo, exists := hm.sessions[sessionID]; exists {
				existingInfo.LastCheck = time.Now()
				existingInfo.LastStatus = status
				existingInfo.FailureCount = info.FailureCount
			}
			hm.mutex.Unlock()
		}

		// Record health check in history
		healthEntry := history.SessionHealthEntry{
			SessionID:      sessionID,
			ServerName:     info.ServerName,
			CheckTime:      time.Now(),
			Status:         status,
			ResponseTimeMs: responseTime,
			ErrorMessage:   errorMessage,
		}

		if err := hm.historyManager.RecordSessionHealth(healthEntry); err != nil {
			// Log error but continue with other sessions
			fmt.Printf("Failed to record session health for %s: %v\n", sessionID, err)
		}
	}

	return nil
}

// checkSessionHealth checks if a specific session is healthy
func (hm *HealthMonitor) checkSessionHealth(sessionID string) (bool, error) {
	// Check if the session is responsive by getting window count
	windowCount, err := hm.tmuxManager.GetWindowCount(sessionID)
	if err != nil {
		return false, fmt.Errorf("failed to get window count for session %s: %w", sessionID, err)
	}

	// Session is considered healthy if it has at least one window
	if windowCount == 0 {
		return false, fmt.Errorf("session %s has no windows", sessionID)
	}

	// Additional health checks could be added here:
	// - Check if SSH connections in windows are still active
	// - Verify network connectivity
	// - Check system resource usage
	
	return true, nil
}

// GetHealthStats returns health statistics for monitoring dashboard
func (hm *HealthMonitor) GetHealthStats() HealthStats {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	stats := HealthStats{
		TotalSessions: len(hm.sessions),
	}

	for _, info := range hm.sessions {
		switch info.LastStatus {
		case "healthy":
			stats.HealthySessions++
		case "degraded":
			stats.DegradedSessions++
		case "failed":
			stats.FailedSessions++
		default:
			stats.UnknownSessions++
		}

		if info.FailureCount > 0 {
			stats.SessionsWithFailures++
		}
	}

	return stats
}

// HealthStats represents aggregated health statistics
type HealthStats struct {
	TotalSessions        int `json:"total_sessions"`
	HealthySessions      int `json:"healthy_sessions"`
	DegradedSessions     int `json:"degraded_sessions"`
	FailedSessions       int `json:"failed_sessions"`
	UnknownSessions      int `json:"unknown_sessions"`
	SessionsWithFailures int `json:"sessions_with_failures"`
}