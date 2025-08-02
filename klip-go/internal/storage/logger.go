package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// Message represents a chat message
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Model     string    `json:"model,omitempty"`
	Provider  string    `json:"provider,omitempty"`
	Tokens    *Tokens   `json:"tokens,omitempty"`
}

// Tokens represents token usage information
type Tokens struct {
	Input  int `json:"input,omitempty"`
	Output int `json:"output,omitempty"`
	Total  int `json:"total,omitempty"`
}

// ChatSession represents a chat session with metadata
type ChatSession struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`
	Model     string    `json:"model,omitempty"`
	Provider  string    `json:"provider,omitempty"`
	Title     string    `json:"title,omitempty"`
}

// ChatLog represents a complete chat session
type ChatLog struct {
	Timestamp    time.Time `json:"timestamp"`
	SessionID    string    `json:"session_id"`
	Title        string    `json:"title,omitempty"`
	Messages     []Message `json:"messages"`
	LastUpdated  time.Time `json:"last_updated"`
	TotalTokens  int       `json:"total_tokens"`
	TotalCost    float64   `json:"total_cost,omitempty"`
	ModelUsed    string    `json:"model_used,omitempty"`
	ProviderUsed string    `json:"provider_used,omitempty"`
}

// ChatLogger handles logging of chat sessions
type ChatLogger struct {
	logDir     string
	sessionID  string
	currentLog *ChatLog
	logger     *log.Logger
}

// NewChatLogger creates a new ChatLogger instance
func NewChatLogger() (*ChatLogger, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	logDir := filepath.Join(configDir, "logs")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	sessionID := generateSessionID()

	chatLogger := &ChatLogger{
		logDir:    logDir,
		sessionID: sessionID,
		currentLog: &ChatLog{
			Timestamp:   time.Now(),
			SessionID:   sessionID,
			Messages:    make([]Message, 0),
			LastUpdated: time.Now(),
			TotalTokens: 0,
		},
		logger: log.New(os.Stderr),
	}

	return chatLogger, nil
}

// generateSessionID creates a unique session ID
func generateSessionID() string {
	now := time.Now()
	timestamp := fmt.Sprintf("%d", now.Unix())
	// Add some randomness to avoid collisions
	random := fmt.Sprintf("%d", now.UnixNano()%1000000)
	return fmt.Sprintf("%s-%s", timestamp, random)
}

// ToSession converts a ChatLog to a ChatSession
func (cl *ChatLog) ToSession() ChatSession {
	session := ChatSession{
		ID:        cl.SessionID,
		StartTime: cl.Timestamp,
		CreatedAt: cl.Timestamp,
		UpdatedAt: cl.LastUpdated,
		Messages:  cl.Messages,
		Title:     cl.Title,
	}

	// Set EndTime if session has ended
	if !cl.LastUpdated.IsZero() && len(cl.Messages) > 0 {
		session.EndTime = cl.LastUpdated
	}

	// Extract model and provider from messages if not set
	if session.Model == "" && len(cl.Messages) > 0 {
		for _, msg := range cl.Messages {
			if msg.Model != "" {
				session.Model = msg.Model
				session.Provider = msg.Provider
				break
			}
		}
	}

	return session
}

// StartSession starts a new chat session
func (cl *ChatLogger) StartSession() error {
	now := time.Now()
	cl.sessionID = generateSessionID()
	cl.currentLog = &ChatLog{
		Timestamp:   now,
		SessionID:   cl.sessionID,
		Messages:    make([]Message, 0),
		LastUpdated: now,
		TotalTokens: 0,
	}

	// Save initial empty session
	return cl.saveLog()
}

// LogMessage adds a message to the current session
func (cl *ChatLogger) LogMessage(message Message) error {
	message.Timestamp = time.Now()
	cl.currentLog.Messages = append(cl.currentLog.Messages, message)
	cl.currentLog.LastUpdated = time.Now()

	// Update session metadata
	if message.Model != "" {
		cl.currentLog.ModelUsed = message.Model
	}
	if message.Provider != "" {
		cl.currentLog.ProviderUsed = message.Provider
	}
	if message.Tokens != nil {
		cl.currentLog.TotalTokens += message.Tokens.Total
	}

	return cl.saveLog()
}

// EndSession finalizes the current session
func (cl *ChatLogger) EndSession() error {
	if cl.currentLog == nil {
		return nil
	}

	cl.currentLog.LastUpdated = time.Now()
	return cl.saveLog()
}

// saveLog saves the current log to disk
func (cl *ChatLogger) saveLog() error {
	if cl.currentLog == nil {
		return fmt.Errorf("no current log to save")
	}

	// Generate filename based on timestamp and session ID
	timestamp := cl.currentLog.Timestamp.Format("2006-01-02-15-04-05")
	filename := fmt.Sprintf("%s-%s.json", timestamp, cl.sessionID)
	filepath := filepath.Join(cl.logDir, filename)

	// Marshal to JSON
	data, err := json.MarshalIndent(cl.currentLog, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}

	return nil
}

// ClearLog clears the current session messages
func (cl *ChatLogger) ClearLog() error {
	if cl.currentLog == nil {
		return fmt.Errorf("no current log to clear")
	}

	cl.currentLog.Messages = make([]Message, 0)
	cl.currentLog.LastUpdated = time.Now()
	cl.currentLog.TotalTokens = 0
	cl.currentLog.TotalCost = 0

	return cl.saveLog()
}

// GetCurrentSession returns the current session
func (cl *ChatLogger) GetCurrentSession() *ChatLog {
	return cl.currentLog
}

// ListSessions returns a list of available chat sessions
func (cl *ChatLogger) ListSessions(limit int) ([]*ChatLog, error) {
	files, err := os.ReadDir(cl.logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	var sessions []*ChatLog

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filepath := filepath.Join(cl.logDir, file.Name())

			data, err := os.ReadFile(filepath)
			if err != nil {
				cl.logger.Warn("Failed to read log file", "file", file.Name(), "error", err)
				continue
			}

			var session ChatLog
			if err := json.Unmarshal(data, &session); err != nil {
				cl.logger.Warn("Failed to parse log file", "file", file.Name(), "error", err)
				continue
			}

			sessions = append(sessions, &session)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp.After(sessions[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}

	return sessions, nil
}

// GetSession retrieves a specific session by ID
func (cl *ChatLogger) GetSession(sessionID string) (*ChatLog, error) {
	files, err := os.ReadDir(cl.logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") && strings.Contains(file.Name(), sessionID) {
			filepath := filepath.Join(cl.logDir, file.Name())

			data, err := os.ReadFile(filepath)
			if err != nil {
				return nil, fmt.Errorf("failed to read log file: %w", err)
			}

			var session ChatLog
			if err := json.Unmarshal(data, &session); err != nil {
				return nil, fmt.Errorf("failed to parse log file: %w", err)
			}

			if session.SessionID == sessionID {
				return &session, nil
			}
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

// ExportSession exports a session to a file in the specified format
func (cl *ChatLogger) ExportSession(sessionID string, format string) (string, error) {
	session, err := cl.GetSession(sessionID)
	if err != nil {
		return "", err
	}

	timestamp := time.Now().Format("2006-01-02-15-04-05")
	var filename, exportPath string

	switch format {
	case "json":
		filename = fmt.Sprintf("klip-export-%s-%s.json", timestamp, sessionID)
		exportPath = filepath.Join(cl.logDir, filename)

		data, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal session: %w", err)
		}

		if err := os.WriteFile(exportPath, data, 0600); err != nil {
			return "", fmt.Errorf("failed to write export file: %w", err)
		}

	case "txt", "text":
		filename = fmt.Sprintf("klip-export-%s-%s.txt", timestamp, sessionID)
		exportPath = filepath.Join(cl.logDir, filename)

		textContent := cl.formatAsText(session)
		if err := os.WriteFile(exportPath, []byte(textContent), 0600); err != nil {
			return "", fmt.Errorf("failed to write export file: %w", err)
		}

	case "md", "markdown":
		filename = fmt.Sprintf("klip-export-%s-%s.md", timestamp, sessionID)
		exportPath = filepath.Join(cl.logDir, filename)

		markdownContent := cl.formatAsMarkdown(session)
		if err := os.WriteFile(exportPath, []byte(markdownContent), 0600); err != nil {
			return "", fmt.Errorf("failed to write export file: %w", err)
		}

	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}

	return exportPath, nil
}

// formatAsText formats a session as plain text
func (cl *ChatLogger) formatAsText(session *ChatLog) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Chat Log - %s\n", session.Timestamp.Format("2006-01-02 15:04:05")))
	builder.WriteString(fmt.Sprintf("Session ID: %s\n", session.SessionID))
	if session.ModelUsed != "" {
		builder.WriteString(fmt.Sprintf("Model: %s", session.ModelUsed))
		if session.ProviderUsed != "" {
			builder.WriteString(fmt.Sprintf(" (%s)", session.ProviderUsed))
		}
		builder.WriteString("\n")
	}
	if session.TotalTokens > 0 {
		builder.WriteString(fmt.Sprintf("Total Tokens: %d\n", session.TotalTokens))
	}
	builder.WriteString(strings.Repeat("=", 50) + "\n\n")

	for _, message := range session.Messages {
		timestamp := message.Timestamp.Format("2006-01-02 15:04:05")
		role := strings.ToUpper(message.Role)

		builder.WriteString(fmt.Sprintf("[%s] %s:\n", timestamp, role))
		builder.WriteString(message.Content + "\n")

		if message.Tokens != nil && message.Tokens.Total > 0 {
			builder.WriteString(fmt.Sprintf("(Tokens: %d)\n", message.Tokens.Total))
		}

		builder.WriteString(strings.Repeat("-", 30) + "\n\n")
	}

	return builder.String()
}

// formatAsMarkdown formats a session as Markdown
func (cl *ChatLogger) formatAsMarkdown(session *ChatLog) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# Chat Log - %s\n\n", session.Timestamp.Format("2006-01-02 15:04:05")))
	builder.WriteString(fmt.Sprintf("**Session ID:** %s\n", session.SessionID))
	if session.ModelUsed != "" {
		builder.WriteString(fmt.Sprintf("**Model:** %s", session.ModelUsed))
		if session.ProviderUsed != "" {
			builder.WriteString(fmt.Sprintf(" (%s)", session.ProviderUsed))
		}
		builder.WriteString("\n")
	}
	if session.TotalTokens > 0 {
		builder.WriteString(fmt.Sprintf("**Total Tokens:** %d\n", session.TotalTokens))
	}
	builder.WriteString("\n---\n\n")

	for _, message := range session.Messages {
		timestamp := message.Timestamp.Format("2006-01-02 15:04:05")
		role := strings.Title(strings.ToLower(message.Role))

		builder.WriteString(fmt.Sprintf("## %s - %s\n\n", role, timestamp))
		builder.WriteString(message.Content + "\n")

		if message.Tokens != nil && message.Tokens.Total > 0 {
			builder.WriteString(fmt.Sprintf("\n*Tokens: %d*\n", message.Tokens.Total))
		}

		builder.WriteString("\n---\n\n")
	}

	return builder.String()
}

// DeleteSession removes a session log file
func (cl *ChatLogger) DeleteSession(sessionID string) error {
	files, err := os.ReadDir(cl.logDir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") && strings.Contains(file.Name(), sessionID) {
			filepath := filepath.Join(cl.logDir, file.Name())

			// Verify this is the correct session
			data, err := os.ReadFile(filepath)
			if err != nil {
				continue
			}

			var session ChatLog
			if err := json.Unmarshal(data, &session); err != nil {
				continue
			}

			if session.SessionID == sessionID {
				if err := os.Remove(filepath); err != nil {
					return fmt.Errorf("failed to delete session file: %w", err)
				}
				return nil
			}
		}
	}

	return fmt.Errorf("session not found: %s", sessionID)
}

// CleanupOldLogs removes log files older than the specified number of days
func (cl *ChatLogger) CleanupOldLogs(retainDays int) error {
	if retainDays <= 0 {
		return fmt.Errorf("retain days must be positive")
	}

	cutoffTime := time.Now().AddDate(0, 0, -retainDays)

	files, err := os.ReadDir(cl.logDir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	deletedCount := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filepath := filepath.Join(cl.logDir, file.Name())

			fileInfo, err := file.Info()
			if err != nil {
				cl.logger.Warn("Failed to get file info", "file", file.Name(), "error", err)
				continue
			}

			if fileInfo.ModTime().Before(cutoffTime) {
				if err := os.Remove(filepath); err != nil {
					cl.logger.Warn("Failed to delete old log file", "file", file.Name(), "error", err)
				} else {
					deletedCount++
				}
			}
		}
	}

	cl.logger.Info("Cleaned up old log files", "deleted", deletedCount, "retain_days", retainDays)
	return nil
}

// GetLogStats returns statistics about stored logs
func (cl *ChatLogger) GetLogStats() (map[string]interface{}, error) {
	files, err := os.ReadDir(cl.logDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	stats := map[string]interface{}{
		"total_sessions":   0,
		"total_messages":   0,
		"total_tokens":     0,
		"total_size_bytes": int64(0),
		"oldest_session":   "",
		"newest_session":   "",
		"models_used":      make(map[string]int),
		"providers_used":   make(map[string]int),
	}

	var oldestTime, newestTime time.Time

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filepath := filepath.Join(cl.logDir, file.Name())

			fileInfo, err := file.Info()
			if err != nil {
				continue
			}

			stats["total_size_bytes"] = stats["total_size_bytes"].(int64) + fileInfo.Size()

			data, err := os.ReadFile(filepath)
			if err != nil {
				continue
			}

			var session ChatLog
			if err := json.Unmarshal(data, &session); err != nil {
				continue
			}

			stats["total_sessions"] = stats["total_sessions"].(int) + 1
			stats["total_messages"] = stats["total_messages"].(int) + len(session.Messages)
			stats["total_tokens"] = stats["total_tokens"].(int) + session.TotalTokens

			if oldestTime.IsZero() || session.Timestamp.Before(oldestTime) {
				oldestTime = session.Timestamp
				stats["oldest_session"] = session.Timestamp.Format("2006-01-02 15:04:05")
			}

			if newestTime.IsZero() || session.Timestamp.After(newestTime) {
				newestTime = session.Timestamp
				stats["newest_session"] = session.Timestamp.Format("2006-01-02 15:04:05")
			}

			if session.ModelUsed != "" {
				models := stats["models_used"].(map[string]int)
				models[session.ModelUsed]++
			}

			if session.ProviderUsed != "" {
				providers := stats["providers_used"].(map[string]int)
				providers[session.ProviderUsed]++
			}
		}
	}

	return stats, nil
}
