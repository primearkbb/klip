package storage

import (
	"os"
	"testing"
)

func setupTestStorage(t *testing.T) (*Storage, string) {
	tempDir := t.TempDir()

	// Mock home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		os.Setenv("HOME", oldHome)
	})

	storage, err := New()
	if err != nil {
		t.Fatalf("Failed to create Storage: %v", err)
	}

	return storage, tempDir
}

func TestStorage_New(t *testing.T) {
	storage, _ := setupTestStorage(t)

	// Verify all components are initialized
	if storage.KeyStore == nil {
		t.Error("Expected KeyStore to be initialized")
	}

	if storage.ConfigManager == nil {
		t.Error("Expected ConfigManager to be initialized")
	}

	if storage.ChatLogger == nil {
		t.Error("Expected ChatLogger to be initialized")
	}

	if storage.AnalyticsLogger == nil {
		t.Error("Expected AnalyticsLogger to be initialized")
	}

	if storage.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}

func TestStorage_Initialize(t *testing.T) {
	storage, _ := setupTestStorage(t)

	err := storage.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Verify that a chat session was started
	currentSession := storage.ChatLogger.GetCurrentSession()
	if currentSession == nil {
		t.Error("Expected current chat session to be set after initialization")
	}

	if currentSession.SessionID == "" {
		t.Error("Expected session ID to be set")
	}
}

func TestStorage_KeyStoreIntegration(t *testing.T) {
	storage, _ := setupTestStorage(t)

	// Test saving and retrieving API keys
	testKey := "test-api-key-12345"

	err := storage.KeyStore.SaveKey("anthropic", testKey)
	if err != nil {
		t.Fatalf("Failed to save API key: %v", err)
	}

	retrievedKey, err := storage.KeyStore.GetKey("anthropic")
	if err != nil {
		t.Fatalf("Failed to get API key: %v", err)
	}

	if retrievedKey != testKey {
		t.Errorf("Expected key '%s', got '%s'", testKey, retrievedKey)
	}

	// Test listing keys
	providers, err := storage.KeyStore.ListKeys()
	if err != nil {
		t.Fatalf("Failed to list keys: %v", err)
	}

	if len(providers) != 1 || providers[0] != "anthropic" {
		t.Errorf("Expected providers ['anthropic'], got %v", providers)
	}
}

func TestStorage_ConfigManagerIntegration(t *testing.T) {
	storage, _ := setupTestStorage(t)

	// Test loading default config
	config, err := storage.ConfigManager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.DefaultProvider != "anthropic" {
		t.Errorf("Expected default provider 'anthropic', got '%s'", config.DefaultProvider)
	}

	// Test updating config
	err = storage.ConfigManager.UpdateProvider("openai")
	if err != nil {
		t.Fatalf("Failed to update provider: %v", err)
	}

	err = storage.ConfigManager.UpdateModel("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to update model: %v", err)
	}

	// Verify changes
	updatedConfig, err := storage.ConfigManager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load updated config: %v", err)
	}

	if updatedConfig.DefaultProvider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", updatedConfig.DefaultProvider)
	}

	if updatedConfig.DefaultModel != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got '%s'", updatedConfig.DefaultModel)
	}
}

func TestStorage_ChatLoggerIntegration(t *testing.T) {
	storage, _ := setupTestStorage(t)

	err := storage.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Test logging messages
	userMessage := Message{
		Role:     "user",
		Content:  "Hello, world!",
		Model:    "claude-3-5-sonnet-20241022",
		Provider: "anthropic",
		Tokens: &Tokens{
			Input: 3,
			Total: 3,
		},
	}

	err = storage.ChatLogger.LogMessage(userMessage)
	if err != nil {
		t.Fatalf("Failed to log user message: %v", err)
	}

	assistantMessage := Message{
		Role:     "assistant",
		Content:  "Hello! How can I help you today?",
		Model:    "claude-3-5-sonnet-20241022",
		Provider: "anthropic",
		Tokens: &Tokens{
			Output: 8,
			Total:  8,
		},
	}

	err = storage.ChatLogger.LogMessage(assistantMessage)
	if err != nil {
		t.Fatalf("Failed to log assistant message: %v", err)
	}

	// Verify messages were logged
	currentSession := storage.ChatLogger.GetCurrentSession()
	if len(currentSession.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(currentSession.Messages))
	}

	if currentSession.TotalTokens != 11 {
		t.Errorf("Expected 11 total tokens, got %d", currentSession.TotalTokens)
	}

	if currentSession.ModelUsed != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got '%s'", currentSession.ModelUsed)
	}
}

func TestStorage_AnalyticsLoggerIntegration(t *testing.T) {
	storage, _ := setupTestStorage(t)

	// Test logging analytics events
	err := storage.AnalyticsLogger.LogCommand("/help", true, 150)
	if err != nil {
		t.Fatalf("Failed to log command: %v", err)
	}

	err = storage.AnalyticsLogger.LogModelSwitch(
		"claude-3-5-sonnet-20241022", "Claude 3.5 Sonnet", "anthropic",
		"gpt-4o", "GPT-4o", "openai",
	)
	if err != nil {
		t.Fatalf("Failed to log model switch: %v", err)
	}

	// Flush events
	err = storage.AnalyticsLogger.flushEvents()
	if err != nil {
		t.Fatalf("Failed to flush analytics events: %v", err)
	}

	// Verify events were logged
	events, err := storage.AnalyticsLogger.GetAnalyticsData("", "", "")
	if err != nil {
		t.Fatalf("Failed to get analytics data: %v", err)
	}

	// Should have at least session_start, command_usage, and model_switch events
	if len(events) < 3 {
		t.Errorf("Expected at least 3 events, got %d", len(events))
	}

	// Check for specific event types
	eventTypes := make(map[string]bool)
	for _, event := range events {
		eventTypes[event.EventType] = true
	}

	if !eventTypes["session_start"] {
		t.Error("Expected session_start event")
	}

	if !eventTypes["command_usage"] {
		t.Error("Expected command_usage event")
	}

	if !eventTypes["model_switch"] {
		t.Error("Expected model_switch event")
	}
}

func TestStorage_Shutdown(t *testing.T) {
	storage, _ := setupTestStorage(t)

	err := storage.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Log some activity
	err = storage.ChatLogger.LogMessage(Message{
		Role:    "user",
		Content: "Test shutdown message",
	})
	if err != nil {
		t.Fatalf("Failed to log message: %v", err)
	}

	// Shutdown should complete without error
	err = storage.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown storage: %v", err)
	}

	// Verify that session was ended and analytics session_end was logged
	// The actual verification of these operations is done in the individual component tests
}

func TestStorage_ConfigValidation(t *testing.T) {
	storage, _ := setupTestStorage(t)

	// Test with invalid configuration
	invalidConfig := &Config{
		DefaultProvider: "invalid_provider",
		DefaultModel:    "",
		Settings: &Settings{
			Temperature: func() *float64 { t := 5.0; return &t }(), // Invalid temperature
		},
	}

	err := storage.ConfigManager.SaveConfig(invalidConfig)
	if err != nil {
		t.Fatalf("Failed to save invalid config: %v", err)
	}

	// Initialize should fail due to validation
	err = storage.Initialize()
	if err == nil {
		t.Error("Expected initialization to fail with invalid config")
	}
}

func TestStorage_CrossComponentIntegration(t *testing.T) {
	storage, _ := setupTestStorage(t)

	// Initialize storage
	err := storage.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Set up API key
	testKey := "test-integration-key"
	err = storage.KeyStore.SaveKey("anthropic", testKey)
	if err != nil {
		t.Fatalf("Failed to save API key: %v", err)
	}

	// Update configuration
	err = storage.ConfigManager.UpdateModel("claude-3-5-sonnet-20241022")
	if err != nil {
		t.Fatalf("Failed to update model: %v", err)
	}

	// Log chat interaction
	err = storage.ChatLogger.LogMessage(Message{
		Role:     "user",
		Content:  "Integration test message",
		Model:    "claude-3-5-sonnet-20241022",
		Provider: "anthropic",
		Tokens:   &Tokens{Input: 5, Total: 5},
	})
	if err != nil {
		t.Fatalf("Failed to log chat message: %v", err)
	}

	// Log analytics for the interaction
	requestMetrics := RequestMetrics{
		StartTime:               storage.ChatLogger.GetCurrentSession().Timestamp,
		ModelID:                 "claude-3-5-sonnet-20241022",
		ModelName:               "Claude 3.5 Sonnet",
		Provider:                "anthropic",
		MessageCount:            1,
		UserMessageLength:       25,
		TotalConversationLength: 25,
		HasSystemMessage:        false,
		Temperature:             0.7,
		MaxTokens:               4096,
		IsStream:                true,
	}

	err = storage.AnalyticsLogger.LogRequest(requestMetrics)
	if err != nil {
		t.Fatalf("Failed to log analytics request: %v", err)
	}

	// Verify all components are working together
	// 1. Check API key is stored
	retrievedKey, err := storage.KeyStore.GetKey("anthropic")
	if err != nil || retrievedKey != testKey {
		t.Errorf("API key integration failed: %v", err)
	}

	// 2. Check config is correct
	config, err := storage.ConfigManager.LoadConfig()
	if err != nil || config.DefaultModel != "claude-3-5-sonnet-20241022" {
		t.Errorf("Config integration failed: %v", err)
	}

	// 3. Check chat log has message
	session := storage.ChatLogger.GetCurrentSession()
	if len(session.Messages) == 0 {
		t.Error("Chat logger integration failed: no messages found")
	}

	// 4. Check analytics has events
	err = storage.AnalyticsLogger.flushEvents()
	if err != nil {
		t.Fatalf("Failed to flush analytics: %v", err)
	}

	events, err := storage.AnalyticsLogger.GetAnalyticsData("", "", "")
	if err != nil || len(events) == 0 {
		t.Errorf("Analytics integration failed: %v", err)
	}

	// Shutdown cleanly
	err = storage.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown: %v", err)
	}
}
