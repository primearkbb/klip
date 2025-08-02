package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupTestConfigManager(t *testing.T) (*ConfigManager, string) {
	tempDir := t.TempDir()
	
	// Mock home directory
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	t.Cleanup(func() {
		os.Setenv("HOME", oldHome)
	})

	configManager, err := NewConfigManager()
	if err != nil {
		t.Fatalf("Failed to create ConfigManager: %v", err)
	}

	return configManager, tempDir
}

func TestConfigManager_LoadDefaultConfig(t *testing.T) {
	configManager, _ := setupTestConfigManager(t)

	config, err := configManager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check default values
	if config.DefaultProvider != "anthropic" {
		t.Errorf("Expected default provider 'anthropic', got '%s'", config.DefaultProvider)
	}
	if config.DefaultModel != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected default model 'claude-3-5-sonnet-20241022', got '%s'", config.DefaultModel)
	}
	if config.Settings == nil {
		t.Error("Expected settings to be initialized")
	}
	if config.UIPreferences == nil {
		t.Error("Expected UI preferences to be initialized")
	}
	if config.Analytics == nil {
		t.Error("Expected analytics config to be initialized")
	}
	if config.Logging == nil {
		t.Error("Expected logging config to be initialized")
	}
}

func TestConfigManager_SaveAndLoadConfig(t *testing.T) {
	configManager, _ := setupTestConfigManager(t)

	// Create test config
	temperature := 0.9
	maxTokens := 2048
	testConfig := &Config{
		DefaultProvider: "openai",
		DefaultModel:    "gpt-4o",
		Settings: &Settings{
			Temperature:       &temperature,
			MaxTokens:         &maxTokens,
			StreamResponses:   false,
			AutoSaveChats:     false,
			ConfirmBeforeExit: false,
			EnableWebSearch:   false,
		},
		UIPreferences: &UIPreferences{
			Theme:           "dark",
			ShowTimestamps:  false,
			ShowTokenCounts: false,
			ShowCosts:       false,
			CompactMode:     true,
			SyntaxHighlight: false,
		},
		Analytics: &AnalyticsConfig{
			Enabled:             false,
			RetainDays:          7,
			MaxFileSizeMB:       5,
			EnableCostTracking:  false,
			AnonymizeContent:    true,
		},
		Logging: &LoggingConfig{
			Enabled:       false,
			LogLevel:      "debug",
			IncludeSystem: true,
			RetainDays:    7,
			MaxFileSizeMB: 25,
		},
		CustomPreferences: map[string]interface{}{
			"test_key": "test_value",
		},
	}

	// Save config
	err := configManager.SaveConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := configManager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if loadedConfig.DefaultProvider != testConfig.DefaultProvider {
		t.Errorf("Expected provider %s, got %s", testConfig.DefaultProvider, loadedConfig.DefaultProvider)
	}
	if loadedConfig.DefaultModel != testConfig.DefaultModel {
		t.Errorf("Expected model %s, got %s", testConfig.DefaultModel, loadedConfig.DefaultModel)
	}
	if *loadedConfig.Settings.Temperature != *testConfig.Settings.Temperature {
		t.Errorf("Expected temperature %f, got %f", *testConfig.Settings.Temperature, *loadedConfig.Settings.Temperature)
	}
	if loadedConfig.UIPreferences.Theme != testConfig.UIPreferences.Theme {
		t.Errorf("Expected theme %s, got %s", testConfig.UIPreferences.Theme, loadedConfig.UIPreferences.Theme)
	}
}

func TestConfigManager_UpdateProvider(t *testing.T) {
	configManager, _ := setupTestConfigManager(t)

	// Update provider
	err := configManager.UpdateProvider("openai")
	if err != nil {
		t.Fatalf("Failed to update provider: %v", err)
	}

	// Load and verify
	config, err := configManager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.DefaultProvider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", config.DefaultProvider)
	}
}

func TestConfigManager_UpdateModel(t *testing.T) {
	configManager, _ := setupTestConfigManager(t)

	// Update model
	err := configManager.UpdateModel("gpt-4o")
	if err != nil {
		t.Fatalf("Failed to update model: %v", err)
	}

	// Load and verify
	config, err := configManager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.DefaultModel != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got '%s'", config.DefaultModel)
	}
}

func TestConfigManager_Validate(t *testing.T) {
	configManager, _ := setupTestConfigManager(t)

	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				DefaultProvider: "anthropic",
				DefaultModel:    "claude-3-5-sonnet-20241022",
				Settings: &Settings{
					Temperature: func() *float64 { t := 0.7; return &t }(),
					MaxTokens:   func() *int { m := 4096; return &m }(),
				},
				Analytics: &AnalyticsConfig{
					RetainDays:    30,
					MaxFileSizeMB: 10,
				},
				Logging: &LoggingConfig{
					LogLevel:      "info",
					RetainDays:    30,
					MaxFileSizeMB: 50,
				},
			},
			expectError: false,
		},
		{
			name: "empty provider",
			config: &Config{
				DefaultProvider: "",
				DefaultModel:    "claude-3-5-sonnet-20241022",
			},
			expectError: true,
		},
		{
			name: "invalid provider",
			config: &Config{
				DefaultProvider: "invalid",
				DefaultModel:    "claude-3-5-sonnet-20241022",
			},
			expectError: true,
		},
		{
			name: "invalid temperature",
			config: &Config{
				DefaultProvider: "anthropic",
				DefaultModel:    "claude-3-5-sonnet-20241022",
				Settings: &Settings{
					Temperature: func() *float64 { t := 3.0; return &t }(),
				},
			},
			expectError: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				DefaultProvider: "anthropic",
				DefaultModel:    "claude-3-5-sonnet-20241022",
				Logging: &LoggingConfig{
					LogLevel:      "invalid",
					RetainDays:    30,
					MaxFileSizeMB: 50,
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := configManager.Validate(tt.config)
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error, got: %v", err)
			}
		})
	}
}

func TestConfigManager_MigrateFromDeno(t *testing.T) {
	configManager, tempDir := setupTestConfigManager(t)

	// Create mock Deno config
	denoConfig := map[string]interface{}{
		"defaultProvider": "openai",
		"defaultModel":    "gpt-4o",
		"settings": map[string]interface{}{
			"temperature":       0.8,
			"maxTokens":        2048,
			"streamResponses":  false,
		},
		"customSetting": "custom_value",
	}

	configDir := filepath.Join(tempDir, ".klip")
	os.MkdirAll(configDir, 0700)
	
	denoConfigFile := filepath.Join(configDir, "config.json")
	data, err := json.Marshal(denoConfig)
	if err != nil {
		t.Fatalf("Failed to marshal Deno config: %v", err)
	}
	
	err = os.WriteFile(denoConfigFile, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write Deno config: %v", err)
	}

	// Migrate config
	err = configManager.MigrateFromDeno()
	if err != nil {
		t.Fatalf("Failed to migrate config: %v", err)
	}

	// Load migrated config
	config, err := configManager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load migrated config: %v", err)
	}

	// Verify migration
	if config.DefaultProvider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", config.DefaultProvider)
	}
	if config.DefaultModel != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got '%s'", config.DefaultModel)
	}
	if *config.Settings.Temperature != 0.8 {
		t.Errorf("Expected temperature 0.8, got %f", *config.Settings.Temperature)
	}
	if config.CustomPreferences["customSetting"] != "custom_value" {
		t.Error("Expected custom setting to be migrated")
	}

	// Verify backup was created
	backupFile := filepath.Join(configDir, "config.deno.backup.json")
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Error("Expected backup file to be created")
	}
}

func TestConfigManager_UpdateSettings(t *testing.T) {
	configManager, _ := setupTestConfigManager(t)

	// Create new settings
	temperature := 0.5
	maxTokens := 1024
	newSettings := &Settings{
		Temperature:       &temperature,
		MaxTokens:         &maxTokens,
		StreamResponses:   false,
		AutoSaveChats:     false,
		ConfirmBeforeExit: false,
		EnableWebSearch:   false,
	}

	// Update settings
	err := configManager.UpdateSettings(newSettings)
	if err != nil {
		t.Fatalf("Failed to update settings: %v", err)
	}

	// Load and verify
	config, err := configManager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if *config.Settings.Temperature != temperature {
		t.Errorf("Expected temperature %f, got %f", temperature, *config.Settings.Temperature)
	}
	if *config.Settings.MaxTokens != maxTokens {
		t.Errorf("Expected max tokens %d, got %d", maxTokens, *config.Settings.MaxTokens)
	}
	if config.Settings.StreamResponses != false {
		t.Error("Expected stream responses to be false")
	}
}

func TestConfigManager_UpdateUIPreferences(t *testing.T) {
	configManager, _ := setupTestConfigManager(t)

	// Create new UI preferences
	newPrefs := &UIPreferences{
		Theme:           "dark",
		ShowTimestamps:  false,
		ShowTokenCounts: false,
		ShowCosts:       false,
		CompactMode:     true,
		SyntaxHighlight: false,
	}

	// Update preferences
	err := configManager.UpdateUIPreferences(newPrefs)
	if err != nil {
		t.Fatalf("Failed to update UI preferences: %v", err)
	}

	// Load and verify
	config, err := configManager.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.UIPreferences.Theme != "dark" {
		t.Errorf("Expected theme 'dark', got '%s'", config.UIPreferences.Theme)
	}
	if config.UIPreferences.CompactMode != true {
		t.Error("Expected compact mode to be true")
	}
}