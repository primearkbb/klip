package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
)

// Config represents the application configuration
type Config struct {
	DefaultProvider   string            `json:"default_provider"`
	DefaultModel      string            `json:"default_model"`
	Settings          *Settings         `json:"settings"`
	UIPreferences     *UIPreferences    `json:"ui_preferences"`
	Analytics         *AnalyticsConfig  `json:"analytics"`
	Logging           *LoggingConfig    `json:"logging"`
	CustomPreferences map[string]interface{} `json:"custom_preferences,omitempty"`
	
	// Direct access fields for backwards compatibility
	EnableLogging     bool   `json:"enable_logging"`
	EnableAnalytics   bool   `json:"enable_analytics"`
	LogDirectory      string `json:"log_directory"`
	MaxHistory        int    `json:"max_history"`
	RequestTimeout    time.Duration `json:"request_timeout"`
	
	// API Key fields
	AnthropicAPIKey   string `json:"anthropic_api_key"`
	OpenAIAPIKey      string `json:"openai_api_key"`
	OpenRouterAPIKey  string `json:"openrouter_api_key"`
	
	// Feature flags
	EnableWebSearch   bool   `json:"enable_web_search"`
	BaseURL           string `json:"base_url"`
	MaxRetries        int    `json:"max_retries"`
	
	// UI preferences
	Theme             string        `json:"theme"`
	ShowTimestamps    bool          `json:"show_timestamps"`
	SyntaxHighlighting bool         `json:"syntax_highlighting"`
	ShowTokenCount    bool          `json:"show_token_count"`
	AutoScroll        bool          `json:"auto_scroll"`
	MaxLineLength     int           `json:"max_line_length"`
	EnableAnimations  bool          `json:"enable_animations"`
	ShowTypingIndicator bool        `json:"show_typing_indicator"`
	AnimationSpeed    time.Duration `json:"animation_speed"`
	
	// System settings
	DebugMode         bool   `json:"debug_mode"`
	ConfigDir         string `json:"config_dir"`
	LogLevel          string `json:"log_level"`
	UserAgent         string `json:"user_agent"`
	
	// Performance settings
	StreamBufferSize      int           `json:"stream_buffer_size"`
	MaxConcurrentRequests int           `json:"max_concurrent_requests"`
	CacheModels           bool          `json:"cache_models"`
	CacheDuration         time.Duration `json:"cache_duration"`
}

// Settings contains general application settings
type Settings struct {
	Temperature       *float64 `json:"temperature,omitempty"`
	MaxTokens         *int     `json:"max_tokens,omitempty"`
	StreamResponses   bool     `json:"stream_responses"`
	AutoSaveChats     bool     `json:"auto_save_chats"`
	ConfirmBeforeExit bool     `json:"confirm_before_exit"`
	EnableWebSearch   bool     `json:"enable_web_search"`
}

// UIPreferences contains user interface preferences
type UIPreferences struct {
	Theme           string `json:"theme"`
	ShowTimestamps  bool   `json:"show_timestamps"`
	ShowTokenCounts bool   `json:"show_token_counts"`
	ShowCosts       bool   `json:"show_costs"`
	CompactMode     bool   `json:"compact_mode"`
	SyntaxHighlight bool   `json:"syntax_highlight"`
}

// AnalyticsConfig contains analytics settings
type AnalyticsConfig struct {
	Enabled             bool `json:"enabled"`
	RetainDays          int  `json:"retain_days"`
	MaxFileSizeMB       int  `json:"max_file_size_mb"`
	EnableCostTracking  bool `json:"enable_cost_tracking"`
	AnonymizeContent    bool `json:"anonymize_content"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Enabled       bool   `json:"enabled"`
	LogLevel      string `json:"log_level"`
	IncludeSystem bool   `json:"include_system"`
	RetainDays    int    `json:"retain_days"`
	MaxFileSizeMB int    `json:"max_file_size_mb"`
}

// ConfigManager handles configuration storage and retrieval
type ConfigManager struct {
	configDir  string
	configFile string
	logger     *log.Logger
}

// NewConfigManager creates a new ConfigManager instance
func NewConfigManager() (*ConfigManager, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	return &ConfigManager{
		configDir:  configDir,
		configFile: filepath.Join(configDir, "config.json"),
		logger:     log.New(os.Stderr),
	}, nil
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".klip")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// LoadConfig loads the application configuration
func (cm *ConfigManager) LoadConfig() (*Config, error) {
	// Return default config if file doesn't exist
	if _, err := os.Stat(cm.configFile); os.IsNotExist(err) {
		config := cm.getDefaultConfig()
		// Save default config to file
		if err := cm.SaveConfig(config); err != nil {
			cm.logger.Warn("Failed to save default config", "error", err)
		}
		return config, nil
	}

	data, err := os.ReadFile(cm.configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure config has all required fields with defaults
	cm.applyDefaults(&config)

	return &config, nil
}

// SaveConfig saves the application configuration
func (cm *ConfigManager) SaveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getDefaultConfig returns a configuration with default values
func (cm *ConfigManager) getDefaultConfig() *Config {
	temperature := 0.7
	maxTokens := 4096

	return &Config{
		DefaultProvider: "anthropic",
		DefaultModel:    "claude-3-5-sonnet-20241022",
		Settings: &Settings{
			Temperature:       &temperature,
			MaxTokens:         &maxTokens,
			StreamResponses:   true,
			AutoSaveChats:     true,
			ConfirmBeforeExit: true,
			EnableWebSearch:   true,
		},
		UIPreferences: &UIPreferences{
			Theme:           "auto",
			ShowTimestamps:  true,
			ShowTokenCounts: true,
			ShowCosts:       true,
			CompactMode:     false,
			SyntaxHighlight: true,
		},
		Analytics: &AnalyticsConfig{
			Enabled:             true,
			RetainDays:          30,
			MaxFileSizeMB:       10,
			EnableCostTracking:  true,
			AnonymizeContent:    false,
		},
		Logging: &LoggingConfig{
			Enabled:       true,
			LogLevel:      "info",
			IncludeSystem: false,
			RetainDays:    30,
			MaxFileSizeMB: 50,
		},
		CustomPreferences: make(map[string]interface{}),
	}
}

// applyDefaults ensures all config fields have default values
func (cm *ConfigManager) applyDefaults(config *Config) {
	if config.DefaultProvider == "" {
		config.DefaultProvider = "anthropic"
	}
	if config.DefaultModel == "" {
		config.DefaultModel = "claude-3-5-sonnet-20241022"
	}

	if config.Settings == nil {
		config.Settings = &Settings{}
	}
	if config.Settings.Temperature == nil {
		temp := 0.7
		config.Settings.Temperature = &temp
	}
	if config.Settings.MaxTokens == nil {
		maxTokens := 4096
		config.Settings.MaxTokens = &maxTokens
	}

	if config.UIPreferences == nil {
		config.UIPreferences = &UIPreferences{
			Theme:           "auto",
			ShowTimestamps:  true,
			ShowTokenCounts: true,
			ShowCosts:       true,
			CompactMode:     false,
			SyntaxHighlight: true,
		}
	}

	if config.Analytics == nil {
		config.Analytics = &AnalyticsConfig{
			Enabled:             true,
			RetainDays:          30,
			MaxFileSizeMB:       10,
			EnableCostTracking:  true,
			AnonymizeContent:    false,
		}
	}

	if config.Logging == nil {
		config.Logging = &LoggingConfig{
			Enabled:       true,
			LogLevel:      "info",
			IncludeSystem: false,
			RetainDays:    30,
			MaxFileSizeMB: 50,
		}
	}

	if config.CustomPreferences == nil {
		config.CustomPreferences = make(map[string]interface{})
	}
}

// UpdateProvider updates the default provider
func (cm *ConfigManager) UpdateProvider(provider string) error {
	config, err := cm.LoadConfig()
	if err != nil {
		return err
	}

	config.DefaultProvider = provider
	return cm.SaveConfig(config)
}

// UpdateModel updates the default model
func (cm *ConfigManager) UpdateModel(model string) error {
	config, err := cm.LoadConfig()
	if err != nil {
		return err
	}

	config.DefaultModel = model
	return cm.SaveConfig(config)
}

// UpdateSettings updates application settings
func (cm *ConfigManager) UpdateSettings(settings *Settings) error {
	config, err := cm.LoadConfig()
	if err != nil {
		return err
	}

	config.Settings = settings
	return cm.SaveConfig(config)
}

// UpdateUIPreferences updates UI preferences
func (cm *ConfigManager) UpdateUIPreferences(prefs *UIPreferences) error {
	config, err := cm.LoadConfig()
	if err != nil {
		return err
	}

	config.UIPreferences = prefs
	return cm.SaveConfig(config)
}

// MigrateFromDeno attempts to migrate configuration from the existing Deno version
func (cm *ConfigManager) MigrateFromDeno() error {
	// Check if Deno config exists
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	denoConfigFile := filepath.Join(homeDir, ".klip", "config.json")
	if _, err := os.Stat(denoConfigFile); os.IsNotExist(err) {
		cm.logger.Info("No Deno config found to migrate")
		return nil
	}

	// Read Deno config
	data, err := os.ReadFile(denoConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read Deno config: %w", err)
	}

	// Parse as generic JSON first
	var denoConfig map[string]interface{}
	if err := json.Unmarshal(data, &denoConfig); err != nil {
		return fmt.Errorf("failed to parse Deno config: %w", err)
	}

	// Create new Go config with migrated values
	config := cm.getDefaultConfig()

	// Migrate known fields
	if provider, ok := denoConfig["defaultProvider"].(string); ok {
		config.DefaultProvider = provider
	}
	if model, ok := denoConfig["defaultModel"].(string); ok {
		config.DefaultModel = model
	}

	// Migrate settings if they exist
	if settingsMap, ok := denoConfig["settings"].(map[string]interface{}); ok {
		if temp, ok := settingsMap["temperature"].(float64); ok {
			config.Settings.Temperature = &temp
		}
		if maxTokens, ok := settingsMap["maxTokens"].(float64); ok {
			tokens := int(maxTokens)
			config.Settings.MaxTokens = &tokens
		}
		if stream, ok := settingsMap["streamResponses"].(bool); ok {
			config.Settings.StreamResponses = stream
		}
	}

	// Store any unmigrated fields in custom preferences
	for key, value := range denoConfig {
		if key != "defaultProvider" && key != "defaultModel" && key != "settings" {
			config.CustomPreferences[key] = value
		}
	}

	// Create backup of original Deno config BEFORE saving migrated config
	backupFile := filepath.Join(cm.configDir, "config.deno.backup.json")
	if err := os.Rename(denoConfigFile, backupFile); err != nil {
		return fmt.Errorf("failed to backup original Deno config: %w", err)
	}
	cm.logger.Info("Created backup of Deno config", "backup", backupFile)

	// Save migrated config
	if err := cm.SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save migrated config: %w", err)
	}

	cm.logger.Info("Successfully migrated configuration from Deno version")
	return nil
}

// Validate validates the configuration
func (cm *ConfigManager) Validate(config *Config) error {
	if config.DefaultProvider == "" {
		return fmt.Errorf("default provider cannot be empty")
	}

	if config.DefaultModel == "" {
		return fmt.Errorf("default model cannot be empty")
	}

	// Validate provider is supported
	supportedProviders := []string{"anthropic", "openai", "openrouter"}
	validProvider := false
	for _, provider := range supportedProviders {
		if config.DefaultProvider == provider {
			validProvider = true
			break
		}
	}
	if !validProvider {
		return fmt.Errorf("unsupported provider: %s", config.DefaultProvider)
	}

	// Validate settings
	if config.Settings != nil {
		if config.Settings.Temperature != nil {
			if *config.Settings.Temperature < 0 || *config.Settings.Temperature > 2 {
				return fmt.Errorf("temperature must be between 0 and 2")
			}
		}
		if config.Settings.MaxTokens != nil {
			if *config.Settings.MaxTokens < 1 || *config.Settings.MaxTokens > 200000 {
				return fmt.Errorf("max tokens must be between 1 and 200000")
			}
		}
	}

	// Validate analytics settings
	if config.Analytics != nil {
		if config.Analytics.RetainDays < 1 {
			return fmt.Errorf("analytics retain days must be at least 1")
		}
		if config.Analytics.MaxFileSizeMB < 1 {
			return fmt.Errorf("analytics max file size must be at least 1MB")
		}
	}

	// Validate logging settings
	if config.Logging != nil {
		validLogLevels := []string{"debug", "info", "warn", "error"}
		validLevel := false
		for _, level := range validLogLevels {
			if config.Logging.LogLevel == level {
				validLevel = true
				break
			}
		}
		if !validLevel {
			return fmt.Errorf("invalid log level: %s", config.Logging.LogLevel)
		}

		if config.Logging.RetainDays < 1 {
			return fmt.Errorf("logging retain days must be at least 1")
		}
		if config.Logging.MaxFileSizeMB < 1 {
			return fmt.Errorf("logging max file size must be at least 1MB")
		}
	}

	return nil
}