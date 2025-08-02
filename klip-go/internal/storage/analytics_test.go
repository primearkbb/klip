package storage

import (
	"os"
	"testing"
	"time"
)

func setupTestAnalyticsLogger(t *testing.T) (*AnalyticsLogger, string) {
	tempDir := t.TempDir()
	
	// Mock home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		os.Setenv("HOME", oldHome)
	})

	config := &AnalyticsConfig{
		Enabled:             true,
		RetainDays:          365, // Use longer retention to avoid cleanup during tests
		MaxFileSizeMB:       10,
		EnableCostTracking:  true,
		AnonymizeContent:    false,
	}

	analyticsLogger, err := NewAnalyticsLogger(config)
	if err != nil {
		t.Fatalf("Failed to create AnalyticsLogger: %v", err)
	}

	return analyticsLogger, tempDir
}

func TestAnalyticsLogger_LogRequest(t *testing.T) {
	analyticsLogger, _ := setupTestAnalyticsLogger(t)

	metrics := RequestMetrics{
		StartTime:                 time.Now(),
		ModelID:                   "claude-3-5-sonnet-20241022",
		ModelName:                 "Claude 3.5 Sonnet",
		Provider:                  "anthropic",
		MessageCount:              2,
		UserMessageLength:         100,
		TotalConversationLength:   200,
		HasSystemMessage:          true,
		Temperature:               0.7,
		MaxTokens:                 4096,
		IsStream:                  true,
	}

	err := analyticsLogger.LogRequest(metrics)
	if err != nil {
		t.Fatalf("Failed to log request: %v", err)
	}

	// Flush events to ensure they're written
	err = analyticsLogger.flushEvents()
	if err != nil {
		t.Fatalf("Failed to flush events: %v", err)
	}

	// Verify event was logged
	events, err := analyticsLogger.GetAnalyticsData("", "", "request")
	if err != nil {
		t.Fatalf("Failed to get analytics data: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("Expected at least one request event")
	}

	event := events[len(events)-1] // Get the last event
	if event.EventType != "request" {
		t.Errorf("Expected event type 'request', got '%s'", event.EventType)
	}
	if event.ModelID != metrics.ModelID {
		t.Errorf("Expected model ID '%s', got '%s'", metrics.ModelID, event.ModelID)
	}
	if event.Provider != metrics.Provider {
		t.Errorf("Expected provider '%s', got '%s'", metrics.Provider, event.Provider)
	}
}

func TestAnalyticsLogger_LogResponse(t *testing.T) {
	analyticsLogger, _ := setupTestAnalyticsLogger(t)

	startTime := time.Now()
	requestMetrics := RequestMetrics{
		StartTime:                 startTime,
		ModelID:                   "claude-3-5-sonnet-20241022",
		ModelName:                 "Claude 3.5 Sonnet",
		Provider:                  "anthropic",
		MessageCount:              1,
		UserMessageLength:         50,
		TotalConversationLength:   50,
		HasSystemMessage:          false,
		Temperature:               0.7,
		MaxTokens:                 4096,
		IsStream:                  true,
	}

	responseMetrics := ResponseMetrics{
		EndTime:        startTime.Add(2 * time.Second),
		ResponseLength: 150,
		TokensInput:    10,
		TokensOutput:   25,
		Interrupted:    false,
		Success:        true,
		RetryCount:     0,
	}

	err := analyticsLogger.LogResponse(requestMetrics, responseMetrics)
	if err != nil {
		t.Fatalf("Failed to log response: %v", err)
	}

	// Flush events
	err = analyticsLogger.flushEvents()
	if err != nil {
		t.Fatalf("Failed to flush events: %v", err)
	}

	// Verify event was logged
	events, err := analyticsLogger.GetAnalyticsData("", "", "response")
	if err != nil {
		t.Fatalf("Failed to get analytics data: %v", err)
	}

	if len(events) == 0 {
		t.Error("Expected at least one response event")
	}

	event := events[len(events)-1]
	if event.EventType != "response" {
		t.Errorf("Expected event type 'response', got '%s'", event.EventType)
	}
	if event.ResponseData == nil {
		t.Fatal("Expected response data to be set")
	}
	if event.ResponseData.TokensInput != responseMetrics.TokensInput {
		t.Errorf("Expected input tokens %d, got %d", responseMetrics.TokensInput, event.ResponseData.TokensInput)
	}
	if event.ResponseData.TokensOutput != responseMetrics.TokensOutput {
		t.Errorf("Expected output tokens %d, got %d", responseMetrics.TokensOutput, event.ResponseData.TokensOutput)
	}
	if event.ResponseData.LatencyMs <= 0 {
		t.Error("Expected latency to be greater than 0")
	}

	// Verify cost data is calculated
	if event.CostData == nil {
		t.Error("Expected cost data to be calculated")
	}
}

func TestAnalyticsLogger_LogCommand(t *testing.T) {
	analyticsLogger, _ := setupTestAnalyticsLogger(t)

	command := "/model claude-3-5-sonnet-20241022"
	success := true
	executionTime := int64(150)

	err := analyticsLogger.LogCommand(command, success, executionTime)
	if err != nil {
		t.Fatalf("Failed to log command: %v", err)
	}

	// Flush events
	err = analyticsLogger.flushEvents()
	if err != nil {
		t.Fatalf("Failed to flush events: %v", err)
	}

	// Verify event was logged
	events, err := analyticsLogger.GetAnalyticsData("", "", "command_usage")
	if err != nil {
		t.Fatalf("Failed to get analytics data: %v", err)
	}

	if len(events) == 0 {
		t.Error("Expected at least one command event")
	}

	event := events[len(events)-1]
	if event.EventType != "command_usage" {
		t.Errorf("Expected event type 'command_usage', got '%s'", event.EventType)
	}
	if event.CommandData == nil {
		t.Fatal("Expected command data to be set")
	}
	if event.CommandData.Command != command {
		t.Errorf("Expected command '%s', got '%s'", command, event.CommandData.Command)
	}
	if event.CommandData.Success != success {
		t.Errorf("Expected success %t, got %t", success, event.CommandData.Success)
	}
	if event.CommandData.ExecutionTimeMs != executionTime {
		t.Errorf("Expected execution time %d, got %d", executionTime, event.CommandData.ExecutionTimeMs)
	}
}

func TestAnalyticsLogger_LogModelSwitch(t *testing.T) {
	analyticsLogger, _ := setupTestAnalyticsLogger(t)

	oldModelID := "claude-3-5-sonnet-20241022"
	oldModelName := "Claude 3.5 Sonnet"
	oldProvider := "anthropic"
	newModelID := "gpt-4o"
	newModelName := "GPT-4o"
	newProvider := "openai"

	err := analyticsLogger.LogModelSwitch(oldModelID, oldModelName, oldProvider, newModelID, newModelName, newProvider)
	if err != nil {
		t.Fatalf("Failed to log model switch: %v", err)
	}

	// Flush events
	err = analyticsLogger.flushEvents()
	if err != nil {
		t.Fatalf("Failed to flush events: %v", err)
	}

	// Verify event was logged
	events, err := analyticsLogger.GetAnalyticsData("", "", "model_switch")
	if err != nil {
		t.Fatalf("Failed to get analytics data: %v", err)
	}

	if len(events) == 0 {
		t.Error("Expected at least one model switch event")
	}

	event := events[len(events)-1]
	if event.EventType != "model_switch" {
		t.Errorf("Expected event type 'model_switch', got '%s'", event.EventType)
	}
	if event.ModelID != newModelID {
		t.Errorf("Expected new model ID '%s', got '%s'", newModelID, event.ModelID)
	}
	if event.Provider != newProvider {
		t.Errorf("Expected new provider '%s', got '%s'", newProvider, event.Provider)
	}

	if event.Metadata == nil {
		t.Fatal("Expected metadata to be set")
	}
	if event.Metadata["previous_model_id"] != oldModelID {
		t.Errorf("Expected previous model ID '%s', got '%v'", oldModelID, event.Metadata["previous_model_id"])
	}
}

func TestAnalyticsLogger_GetUsageStats(t *testing.T) {
	analyticsLogger, _ := setupTestAnalyticsLogger(t)

	// Log some test events
	startTime := time.Now()
	
	// Log request
	requestMetrics := RequestMetrics{
		StartTime:                 startTime,
		ModelID:                   "claude-3-5-sonnet-20241022",
		ModelName:                 "Claude 3.5 Sonnet",
		Provider:                  "anthropic",
		MessageCount:              1,
		UserMessageLength:         50,
		TotalConversationLength:   50,
		HasSystemMessage:          false,
		Temperature:               0.7,
		MaxTokens:                 4096,
		IsStream:                  true,
	}

	err := analyticsLogger.LogRequest(requestMetrics)
	if err != nil {
		t.Fatalf("Failed to log request: %v", err)
	}

	// Log response
	responseMetrics := ResponseMetrics{
		EndTime:        startTime.Add(1 * time.Second),
		ResponseLength: 100,
		TokensInput:    10,
		TokensOutput:   20,
		Interrupted:    false,
		Success:        true,
		RetryCount:     0,
	}

	err = analyticsLogger.LogResponse(requestMetrics, responseMetrics)
	if err != nil {
		t.Fatalf("Failed to log response: %v", err)
	}

	// Log another request with different model
	requestMetrics2 := requestMetrics
	requestMetrics2.ModelID = "gpt-4o"
	requestMetrics2.ModelName = "GPT-4o"
	requestMetrics2.Provider = "openai"

	err = analyticsLogger.LogRequest(requestMetrics2)
	if err != nil {
		t.Fatalf("Failed to log second request: %v", err)
	}

	// Flush events
	err = analyticsLogger.flushEvents()
	if err != nil {
		t.Fatalf("Failed to flush events: %v", err)
	}

	// Get usage stats
	stats, err := analyticsLogger.GetUsageStats(7)
	if err != nil {
		t.Fatalf("Failed to get usage stats: %v", err)
	}

	// Verify stats
	totalRequests, ok := stats["total_requests"].(int)
	if !ok || totalRequests < 2 {
		t.Errorf("Expected at least 2 total requests, got %v", stats["total_requests"])
	}

	totalTokens, ok := stats["total_tokens"].(int)
	if !ok || totalTokens < 30 {
		t.Errorf("Expected at least 30 total tokens, got %v", stats["total_tokens"])
	}

	avgLatency, ok := stats["avg_latency"].(float64)
	if !ok || avgLatency <= 0 {
		t.Errorf("Expected positive average latency, got %v", stats["avg_latency"])
	}

	modelsUsed, ok := stats["models_used"].(map[string]int)
	if !ok {
		t.Error("Expected models_used to be map[string]int")
	} else {
		if modelsUsed["claude-3-5-sonnet-20241022"] == 0 {
			t.Error("Expected claude model to be used")
		}
		if modelsUsed["gpt-4o"] == 0 {
			t.Error("Expected gpt-4o model to be used")
		}
	}

	dailyUsage, ok := stats["daily_usage"].(map[string]int)
	if !ok {
		t.Error("Expected daily_usage to be map[string]int")
	} else {
		today := time.Now().Format("2006-01-02")
		if dailyUsage[today] < 2 {
			t.Errorf("Expected at least 2 requests today, got %d", dailyUsage[today])
		}
	}
}

func TestAnalyticsLogger_DisabledConfig(t *testing.T) {
	tempDir := t.TempDir()
	
	// Mock home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		os.Setenv("HOME", oldHome)
	})

	// Create analytics logger with disabled config
	config := &AnalyticsConfig{
		Enabled:             false,
		RetainDays:          30,
		MaxFileSizeMB:       10,
		EnableCostTracking:  true,
		AnonymizeContent:    false,
	}

	analyticsLogger, err := NewAnalyticsLogger(config)
	if err != nil {
		t.Fatalf("Failed to create AnalyticsLogger: %v", err)
	}

	// Try to log events - should not error but should not create files
	requestMetrics := RequestMetrics{
		StartTime:                 time.Now(),
		ModelID:                   "claude-3-5-sonnet-20241022",
		ModelName:                 "Claude 3.5 Sonnet",
		Provider:                  "anthropic",
		MessageCount:              1,
		UserMessageLength:         50,
		TotalConversationLength:   50,
		HasSystemMessage:          false,
		Temperature:               0.7,
		MaxTokens:                 4096,
		IsStream:                  true,
	}

	err = analyticsLogger.LogRequest(requestMetrics)
	if err != nil {
		t.Fatalf("Failed to log request with disabled analytics: %v", err)
	}

	// Verify no events are returned (should not error even if directory doesn't exist)
	events, err := analyticsLogger.GetAnalyticsData("", "", "")
	if err != nil {
		// It's okay if directory doesn't exist when analytics is disabled
		if len(events) != 0 {
			t.Errorf("Expected no events with disabled analytics, got %d", len(events))
		}
		return
	}

	if len(events) != 0 {
		t.Errorf("Expected no events with disabled analytics, got %d", len(events))
	}
}

func TestAnalyticsLogger_CostCalculation(t *testing.T) {
	analyticsLogger, _ := setupTestAnalyticsLogger(t)

	// Test cost calculation with known model
	costData := analyticsLogger.calculateCost("claude-3-5-sonnet-20241022", 1000, 2000)
	if costData == nil {
		t.Fatal("Expected cost data to be calculated")
	}

	if costData.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", costData.Currency)
	}

	if costData.EstimatedCostInput <= 0 {
		t.Error("Expected positive input cost")
	}

	if costData.EstimatedCostOutput <= 0 {
		t.Error("Expected positive output cost")
	}

	if costData.EstimatedCostTotal <= 0 {
		t.Error("Expected positive total cost")
	}

	expectedTotal := costData.EstimatedCostInput + costData.EstimatedCostOutput
	if costData.EstimatedCostTotal != expectedTotal {
		t.Errorf("Expected total cost %f, got %f", expectedTotal, costData.EstimatedCostTotal)
	}

	// Test with unknown model
	costData = analyticsLogger.calculateCost("unknown-model", 1000, 2000)
	if costData != nil {
		t.Error("Expected no cost data for unknown model")
	}

	// Test with disabled cost tracking
	analyticsLogger.config.EnableCostTracking = false
	costData = analyticsLogger.calculateCost("claude-3-5-sonnet-20241022", 1000, 2000)
	if costData != nil {
		t.Error("Expected no cost data when cost tracking is disabled")
	}
}

func TestAnalyticsLogger_ErrorLogging(t *testing.T) {
	analyticsLogger, _ := setupTestAnalyticsLogger(t)

	startTime := time.Now()
	requestMetrics := RequestMetrics{
		StartTime:                 startTime,
		ModelID:                   "claude-3-5-sonnet-20241022",
		ModelName:                 "Claude 3.5 Sonnet",
		Provider:                  "anthropic",
		MessageCount:              1,
		UserMessageLength:         50,
		TotalConversationLength:   50,
		HasSystemMessage:          false,
		Temperature:               0.7,
		MaxTokens:                 4096,
		IsStream:                  true,
	}

	responseMetrics := ResponseMetrics{
		EndTime:        startTime.Add(1 * time.Second),
		ResponseLength: 0,
		TokensInput:    10,
		TokensOutput:   0,
		Interrupted:    false,
		Success:        false,
		ErrorType:      "rate_limit",
		ErrorMessage:   "Rate limit exceeded",
		StatusCode:     429,
		RetryCount:     3,
	}

	err := analyticsLogger.LogResponse(requestMetrics, responseMetrics)
	if err != nil {
		t.Fatalf("Failed to log error response: %v", err)
	}

	// Flush events
	err = analyticsLogger.flushEvents()
	if err != nil {
		t.Fatalf("Failed to flush events: %v", err)
	}

	// Verify error event was logged
	events, err := analyticsLogger.GetAnalyticsData("", "", "error")
	if err != nil {
		t.Fatalf("Failed to get analytics data: %v", err)
	}

	if len(events) == 0 {
		t.Error("Expected at least one error event")
	}

	event := events[len(events)-1]
	if event.EventType != "error" {
		t.Errorf("Expected event type 'error', got '%s'", event.EventType)
	}
	if event.ErrorData == nil {
		t.Fatal("Expected error data to be set")
	}
	if event.ErrorData.ErrorType != responseMetrics.ErrorType {
		t.Errorf("Expected error type '%s', got '%s'", responseMetrics.ErrorType, event.ErrorData.ErrorType)
	}
	if event.ErrorData.StatusCode != responseMetrics.StatusCode {
		t.Errorf("Expected status code %d, got %d", responseMetrics.StatusCode, event.ErrorData.StatusCode)
	}
}