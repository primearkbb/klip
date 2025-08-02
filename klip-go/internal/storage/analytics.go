package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// AnalyticsEvent represents a single analytics event
type AnalyticsEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	SessionID   string                 `json:"session_id"`
	ModelID     string                 `json:"model_id,omitempty"`
	ModelName   string                 `json:"model_name,omitempty"`
	Provider    string                 `json:"provider,omitempty"`
	RequestData *RequestData           `json:"request_data,omitempty"`
	ResponseData *ResponseData         `json:"response_data,omitempty"`
	ErrorData   *ErrorData             `json:"error_data,omitempty"`
	CostData    *CostData              `json:"cost_data,omitempty"`
	CommandData *CommandData           `json:"command_data,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RequestData contains request-specific metrics
type RequestData struct {
	MessageCount              int     `json:"message_count"`
	UserMessageLength         int     `json:"user_message_length"`
	TotalConversationLength   int     `json:"total_conversation_length"`
	HasSystemMessage          bool    `json:"has_system_message"`
	Temperature               float64 `json:"temperature,omitempty"`
	MaxTokens                 int     `json:"max_tokens,omitempty"`
}

// ResponseData contains response-specific metrics
type ResponseData struct {
	ResponseLength int  `json:"response_length"`
	TokensInput    int  `json:"tokens_input,omitempty"`
	TokensOutput   int  `json:"tokens_output,omitempty"`
	TotalTokens    int  `json:"total_tokens,omitempty"`
	LatencyMs      int64 `json:"latency_ms"`
	IsStream       bool `json:"is_stream"`
	Interrupted    bool `json:"interrupted"`
}

// ErrorData contains error-specific information
type ErrorData struct {
	ErrorType    string `json:"error_type"`
	ErrorMessage string `json:"error_message"`
	StatusCode   int    `json:"status_code,omitempty"`
	RetryCount   int    `json:"retry_count"`
}

// CostData contains cost estimation information
type CostData struct {
	EstimatedCostInput  float64 `json:"estimated_cost_input,omitempty"`
	EstimatedCostOutput float64 `json:"estimated_cost_output,omitempty"`
	EstimatedCostTotal  float64 `json:"estimated_cost_total,omitempty"`
	Currency            string  `json:"currency"`
}

// CommandData contains command execution information
type CommandData struct {
	Command         string `json:"command"`
	Success         bool   `json:"success"`
	ExecutionTimeMs int64  `json:"execution_time_ms"`
}

// RequestMetrics contains metrics for tracking a request
type RequestMetrics struct {
	StartTime                 time.Time
	ModelID                   string
	ModelName                 string
	Provider                  string
	MessageCount              int
	UserMessageLength         int
	TotalConversationLength   int
	HasSystemMessage          bool
	Temperature               float64
	MaxTokens                 int
	IsStream                  bool
}

// ResponseMetrics contains metrics for tracking a response
type ResponseMetrics struct {
	EndTime        time.Time
	ResponseLength int
	TokensInput    int
	TokensOutput   int
	Interrupted    bool
	Success        bool
	ErrorType      string
	ErrorMessage   string
	StatusCode     int
	RetryCount     int
}

// CostEstimate represents cost per 1M tokens for a model
type CostEstimate struct {
	Input    float64 `json:"input"`
	Output   float64 `json:"output"`
	Currency string  `json:"currency"`
}

// Cost estimates per 1M tokens (approximate values, should be updated regularly)
var costEstimates = map[string]CostEstimate{
	// Anthropic Claude models (USD per 1M tokens)
	"claude-opus-4-20250514":     {Input: 15.0, Output: 75.0, Currency: "USD"},
	"claude-sonnet-4-20250514":   {Input: 3.0, Output: 15.0, Currency: "USD"},
	"claude-3-7-sonnet-20250219": {Input: 3.0, Output: 15.0, Currency: "USD"},
	"claude-3-5-sonnet-20241022": {Input: 3.0, Output: 15.0, Currency: "USD"},
	"claude-3-5-sonnet-20240620": {Input: 3.0, Output: 15.0, Currency: "USD"},
	"claude-3-5-haiku-20241022":  {Input: 1.0, Output: 5.0, Currency: "USD"},
	"claude-3-opus-20240229":     {Input: 15.0, Output: 75.0, Currency: "USD"},
	"claude-3-haiku-20240307":    {Input: 0.25, Output: 1.25, Currency: "USD"},

	// OpenAI models (USD per 1M tokens)
	"gpt-4.1":      {Input: 10.0, Output: 30.0, Currency: "USD"},
	"gpt-4.1-mini": {Input: 0.15, Output: 0.6, Currency: "USD"},
	"gpt-4.1-nano": {Input: 0.075, Output: 0.3, Currency: "USD"},
	"o3":           {Input: 60.0, Output: 240.0, Currency: "USD"},
	"o3-pro":       {Input: 200.0, Output: 800.0, Currency: "USD"},
	"o4-mini":      {Input: 0.15, Output: 0.6, Currency: "USD"},
	"gpt-4o":       {Input: 2.5, Output: 10.0, Currency: "USD"},
	"gpt-4o-mini":  {Input: 0.15, Output: 0.6, Currency: "USD"},

	// OpenRouter models (varies, using approximate values)
	"anthropic/claude-3.5-sonnet":           {Input: 3.0, Output: 15.0, Currency: "USD"},
	"openai/gpt-4o":                         {Input: 2.5, Output: 10.0, Currency: "USD"},
	"meta-llama/llama-3.1-405b-instruct":    {Input: 2.7, Output: 2.7, Currency: "USD"},
}

// AnalyticsLogger handles collection and storage of analytics data
type AnalyticsLogger struct {
	analyticsDir   string
	sessionID      string
	config         *AnalyticsConfig
	currentDate    string
	pendingEvents  []AnalyticsEvent
	logger         *log.Logger
}

// NewAnalyticsLogger creates a new AnalyticsLogger instance
func NewAnalyticsLogger(config *AnalyticsConfig) (*AnalyticsLogger, error) {
	if config == nil {
		config = &AnalyticsConfig{
			Enabled:             true,
			RetainDays:          30,
			MaxFileSizeMB:       10,
			EnableCostTracking:  true,
			AnonymizeContent:    false,
		}
	}

	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	analyticsDir := filepath.Join(configDir, "analytics")
	if config.Enabled {
		if err := os.MkdirAll(analyticsDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create analytics directory: %w", err)
		}
	}

	sessionID := generateSessionID()
	currentDate := time.Now().Format("2006-01-02")

	al := &AnalyticsLogger{
		analyticsDir:  analyticsDir,
		sessionID:     sessionID,
		config:        config,
		currentDate:   currentDate,
		pendingEvents: make([]AnalyticsEvent, 0),
		logger:        log.New(os.Stderr),
	}

	if config.Enabled {
		// Log session start
		if err := al.logEvent(AnalyticsEvent{
			Timestamp: time.Now(),
			EventType: "session_start",
			SessionID: sessionID,
			Metadata: map[string]interface{}{
				"platform":    "go",
				"app_version": "2.0.0", // Should be dynamic
			},
		}); err != nil {
			al.logger.Warn("Failed to log session start", "error", err)
		}

		// Start cleanup routine (skip during tests)
		if os.Getenv("GO_TEST_MODE") == "" {
			go al.startCleanupRoutine()
		}
	}

	return al, nil
}

// LogRequest logs a request event
func (al *AnalyticsLogger) LogRequest(metrics RequestMetrics) error {
	if !al.config.Enabled {
		return nil
	}

	event := AnalyticsEvent{
		Timestamp: metrics.StartTime,
		EventType: "request",
		SessionID: al.sessionID,
		ModelID:   metrics.ModelID,
		ModelName: metrics.ModelName,
		Provider:  metrics.Provider,
		RequestData: &RequestData{
			MessageCount:              metrics.MessageCount,
			UserMessageLength:         metrics.UserMessageLength,
			TotalConversationLength:   metrics.TotalConversationLength,
			HasSystemMessage:          metrics.HasSystemMessage,
			Temperature:               metrics.Temperature,
			MaxTokens:                 metrics.MaxTokens,
		},
		ResponseData: &ResponseData{
			ResponseLength: 0,
			LatencyMs:      0,
			IsStream:       metrics.IsStream,
			Interrupted:    false,
		},
	}

	return al.logEvent(event)
}

// LogResponse logs a response event
func (al *AnalyticsLogger) LogResponse(requestMetrics RequestMetrics, responseMetrics ResponseMetrics) error {
	if !al.config.Enabled {
		return nil
	}

	latency := responseMetrics.EndTime.Sub(requestMetrics.StartTime).Milliseconds()
	costData := al.calculateCost(requestMetrics.ModelID, responseMetrics.TokensInput, responseMetrics.TokensOutput)

	eventType := "response"
	if !responseMetrics.Success {
		eventType = "error"
	}

	event := AnalyticsEvent{
		Timestamp: responseMetrics.EndTime,
		EventType: eventType,
		SessionID: al.sessionID,
		ModelID:   requestMetrics.ModelID,
		ModelName: requestMetrics.ModelName,
		Provider:  requestMetrics.Provider,
		RequestData: &RequestData{
			MessageCount:              requestMetrics.MessageCount,
			UserMessageLength:         requestMetrics.UserMessageLength,
			TotalConversationLength:   requestMetrics.TotalConversationLength,
			HasSystemMessage:          requestMetrics.HasSystemMessage,
			Temperature:               requestMetrics.Temperature,
			MaxTokens:                 requestMetrics.MaxTokens,
		},
		ResponseData: &ResponseData{
			ResponseLength: responseMetrics.ResponseLength,
			TokensInput:    responseMetrics.TokensInput,
			TokensOutput:   responseMetrics.TokensOutput,
			TotalTokens:    responseMetrics.TokensInput + responseMetrics.TokensOutput,
			LatencyMs:      latency,
			IsStream:       requestMetrics.IsStream,
			Interrupted:    responseMetrics.Interrupted,
		},
		CostData: costData,
	}

	if !responseMetrics.Success {
		event.ErrorData = &ErrorData{
			ErrorType:    responseMetrics.ErrorType,
			ErrorMessage: responseMetrics.ErrorMessage,
			StatusCode:   responseMetrics.StatusCode,
			RetryCount:   responseMetrics.RetryCount,
		}
	}

	return al.logEvent(event)
}

// LogCommand logs a command execution event
func (al *AnalyticsLogger) LogCommand(command string, success bool, executionTimeMs int64) error {
	if !al.config.Enabled {
		return nil
	}

	event := AnalyticsEvent{
		Timestamp: time.Now(),
		EventType: "command_usage",
		SessionID: al.sessionID,
		CommandData: &CommandData{
			Command:         command,
			Success:         success,
			ExecutionTimeMs: executionTimeMs,
		},
	}

	return al.logEvent(event)
}

// LogModelSwitch logs a model switch event
func (al *AnalyticsLogger) LogModelSwitch(oldModelID, oldModelName, oldProvider, newModelID, newModelName, newProvider string) error {
	if !al.config.Enabled {
		return nil
	}

	event := AnalyticsEvent{
		Timestamp: time.Now(),
		EventType: "model_switch",
		SessionID: al.sessionID,
		ModelID:   newModelID,
		ModelName: newModelName,
		Provider:  newProvider,
		Metadata: map[string]interface{}{
			"previous_model_id":   oldModelID,
			"previous_model_name": oldModelName,
			"previous_provider":   oldProvider,
		},
	}

	return al.logEvent(event)
}

// LogSessionEnd logs a session end event
func (al *AnalyticsLogger) LogSessionEnd() error {
	if !al.config.Enabled {
		return nil
	}

	event := AnalyticsEvent{
		Timestamp: time.Now(),
		EventType: "session_end",
		SessionID: al.sessionID,
		Metadata: map[string]interface{}{
			"session_duration_ms": time.Now().Sub(time.Unix(0, 0)).Milliseconds(), // Approximate
		},
	}

	if err := al.logEvent(event); err != nil {
		return err
	}

	// Flush any pending events
	return al.flushEvents()
}

// logEvent adds an event to the pending queue
func (al *AnalyticsLogger) logEvent(event AnalyticsEvent) error {
	if !al.config.Enabled {
		return nil
	}

	al.pendingEvents = append(al.pendingEvents, event)

	// Flush events if we have accumulated enough or if it's an important event
	if len(al.pendingEvents) >= 10 || event.EventType == "session_end" || event.EventType == "error" {
		return al.flushEvents()
	}

	return nil
}

// flushEvents writes pending events to disk
func (al *AnalyticsLogger) flushEvents() error {
	if len(al.pendingEvents) == 0 {
		return nil
	}

	currentDate := time.Now().Format("2006-01-02")
	if currentDate != al.currentDate {
		al.currentDate = currentDate
	}

	filename := fmt.Sprintf("analytics-%s.jsonl", al.currentDate)
	filePath := filepath.Join(al.analyticsDir, filename)

	// Check if we need to rotate the file (size limit)
	if fileInfo, err := os.Stat(filePath); err == nil {
		sizeMB := float64(fileInfo.Size()) / (1024 * 1024)
		if sizeMB > float64(al.config.MaxFileSizeMB) {
			rotatedFilename := fmt.Sprintf("analytics-%s-%d.jsonl", al.currentDate, time.Now().Unix())
			rotatedFilepath := filepath.Join(al.analyticsDir, rotatedFilename)
			if err := os.Rename(filePath, rotatedFilepath); err != nil {
				al.logger.Warn("Failed to rotate analytics file", "error", err)
			}
		}
	}

	// Open file for appending
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open analytics file: %w", err)
	}
	defer file.Close()

	// Write events as JSON Lines
	for _, event := range al.pendingEvents {
		jsonData, err := json.Marshal(event)
		if err != nil {
			al.logger.Warn("Failed to marshal analytics event", "error", err)
			continue
		}

		if _, err := file.Write(append(jsonData, '\n')); err != nil {
			al.logger.Warn("Failed to write analytics event", "error", err)
		}
	}

	// Clear pending events
	al.pendingEvents = al.pendingEvents[:0]

	return nil
}

// calculateCost estimates the cost of a request/response
func (al *AnalyticsLogger) calculateCost(modelID string, inputTokens, outputTokens int) *CostData {
	if !al.config.EnableCostTracking || inputTokens == 0 || outputTokens == 0 {
		return nil
	}

	estimate, exists := costEstimates[modelID]
	if !exists {
		return nil
	}

	inputCost := (float64(inputTokens) / 1_000_000) * estimate.Input
	outputCost := (float64(outputTokens) / 1_000_000) * estimate.Output
	totalCost := inputCost + outputCost

	return &CostData{
		EstimatedCostInput:  inputCost,
		EstimatedCostOutput: outputCost,
		EstimatedCostTotal:  totalCost,
		Currency:            estimate.Currency,
	}
}

// startCleanupRoutine starts a background routine to clean up old analytics files
func (al *AnalyticsLogger) startCleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour) // Run cleanup every 24 hours
	defer ticker.Stop()

	// Run initial cleanup
	if err := al.cleanupOldFiles(); err != nil {
		al.logger.Warn("Initial analytics cleanup failed", "error", err)
	}

	for range ticker.C {
		if err := al.cleanupOldFiles(); err != nil {
			al.logger.Warn("Analytics cleanup failed", "error", err)
		}
	}
}

// cleanupOldFiles removes analytics files older than the retention period
func (al *AnalyticsLogger) cleanupOldFiles() error {
	if !al.config.Enabled {
		return nil
	}

	cutoffDate := time.Now().AddDate(0, 0, -al.config.RetainDays)
	cutoffDateStr := cutoffDate.Format("2006-01-02")

	files, err := os.ReadDir(al.analyticsDir)
	if err != nil {
		return fmt.Errorf("failed to read analytics directory: %w", err)
	}

	deletedCount := 0
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "analytics-") || !strings.HasSuffix(file.Name(), ".jsonl") {
			continue
		}

		// Extract date from filename
		parts := strings.Split(file.Name(), "-")
		if len(parts) < 2 {
			continue
		}

		dateStr := parts[1]
		if len(parts) > 2 {
			// Handle rotated files with timestamp
			dateStr = parts[1]
		}

		if dateStr < cutoffDateStr {
			filePath := filepath.Join(al.analyticsDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				al.logger.Warn("Failed to delete old analytics file", "file", file.Name(), "error", err)
			} else {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		al.logger.Info("Cleaned up old analytics files", "deleted", deletedCount, "retain_days", al.config.RetainDays)
	}

	return nil
}

// GetAnalyticsData retrieves analytics events within a date range
func (al *AnalyticsLogger) GetAnalyticsData(startDate, endDate, eventType string) ([]AnalyticsEvent, error) {
	var events []AnalyticsEvent

	files, err := os.ReadDir(al.analyticsDir)
	if err != nil {
		return events, fmt.Errorf("failed to read analytics directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") {
			continue
		}

		filePath := filepath.Join(al.analyticsDir, file.Name())
		fileEvents, err := al.readAnalyticsFile(filePath, startDate, endDate, eventType)
		if err != nil {
			al.logger.Warn("Failed to read analytics file", "file", file.Name(), "error", err)
			continue
		}

		events = append(events, fileEvents...)
	}

	// Sort by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	return events, nil
}

// readAnalyticsFile reads and filters events from a single analytics file
func (al *AnalyticsLogger) readAnalyticsFile(filePath, startDate, endDate, eventType string) ([]AnalyticsEvent, error) {
	var events []AnalyticsEvent

	file, err := os.Open(filePath)
	if err != nil {
		return events, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event AnalyticsEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip malformed lines
		}

		// Apply filters
		eventDateStr := event.Timestamp.Format("2006-01-02")
		if startDate != "" && eventDateStr < startDate {
			continue
		}
		if endDate != "" && eventDateStr > endDate {
			continue
		}
		if eventType != "" && event.EventType != eventType {
			continue
		}

		events = append(events, event)
	}

	return events, scanner.Err()
}

// GetUsageStats returns usage statistics for the specified number of days
func (al *AnalyticsLogger) GetUsageStats(days int) (map[string]interface{}, error) {
	if days <= 0 {
		days = 7
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	events, err := al.GetAnalyticsData(startDate, "", "")
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_requests":   0,
		"total_tokens":     0,
		"total_cost":       0.0,
		"avg_latency":      0.0,
		"error_rate":       0.0,
		"models_used":      make(map[string]int),
		"daily_usage":      make(map[string]int),
		"providers_used":   make(map[string]int),
	}

	var totalLatency int64
	var requestCount, errorCount int

	for _, event := range events {
		dateStr := event.Timestamp.Format("2006-01-02")

		switch event.EventType {
		case "request":
			stats["total_requests"] = stats["total_requests"].(int) + 1
			requestCount++

			if event.ModelID != "" {
				models := stats["models_used"].(map[string]int)
				models[event.ModelID]++
			}

			if event.Provider != "" {
				providers := stats["providers_used"].(map[string]int)
				providers[event.Provider]++
			}

			dailyUsage := stats["daily_usage"].(map[string]int)
			dailyUsage[dateStr]++

		case "response":
			if event.ResponseData != nil {
				totalLatency += event.ResponseData.LatencyMs
				if event.ResponseData.TotalTokens > 0 {
					stats["total_tokens"] = stats["total_tokens"].(int) + event.ResponseData.TotalTokens
				}
			}

			if event.CostData != nil && event.CostData.EstimatedCostTotal > 0 {
				stats["total_cost"] = stats["total_cost"].(float64) + event.CostData.EstimatedCostTotal
			}

		case "error":
			errorCount++
		}
	}

	// Calculate averages
	if requestCount > 0 {
		stats["avg_latency"] = float64(totalLatency) / float64(requestCount)
		stats["error_rate"] = float64(errorCount) / float64(requestCount)
	}

	return stats, nil
}