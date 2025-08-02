package storage

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
)

// Storage is the main storage manager that coordinates all storage components
type Storage struct {
	KeyStore        *KeyStore
	ConfigManager   *ConfigManager
	ChatLogger      *ChatLogger
	AnalyticsLogger *AnalyticsLogger
	logger          *log.Logger
}

// New creates a new Storage instance with all components initialized
func New() (*Storage, error) {
	logger := log.New(os.Stderr)

	// Initialize KeyStore
	keyStore, err := NewKeyStore()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keystore: %w", err)
	}

	// Initialize ConfigManager
	configManager, err := NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config manager: %w", err)
	}

	// Load config to get analytics settings
	config, err := configManager.LoadConfig()
	if err != nil {
		logger.Warn("Failed to load config, using defaults", "error", err)
		config = nil
	}

	// Initialize ChatLogger
	chatLogger, err := NewChatLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize chat logger: %w", err)
	}

	// Initialize AnalyticsLogger
	var analyticsConfig *AnalyticsConfig
	if config != nil && config.Analytics != nil {
		analyticsConfig = config.Analytics
	}

	analyticsLogger, err := NewAnalyticsLogger(analyticsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize analytics logger: %w", err)
	}

	// Attempt to migrate from Deno if needed
	if err := configManager.MigrateFromDeno(); err != nil {
		logger.Warn("Failed to migrate from Deno config", "error", err)
	}

	return &Storage{
		KeyStore:        keyStore,
		ConfigManager:   configManager,
		ChatLogger:      chatLogger,
		AnalyticsLogger: analyticsLogger,
		logger:          logger,
	}, nil
}

// Shutdown performs cleanup operations
func (s *Storage) Shutdown() error {
	if s.AnalyticsLogger != nil {
		if err := s.AnalyticsLogger.LogSessionEnd(); err != nil {
			s.logger.Warn("Failed to log session end", "error", err)
		}
	}

	if s.ChatLogger != nil {
		if err := s.ChatLogger.EndSession(); err != nil {
			s.logger.Warn("Failed to end chat session", "error", err)
		}
	}

	return nil
}

// Initialize performs initial setup and validation
func (s *Storage) Initialize() error {
	// Validate configuration
	config, err := s.ConfigManager.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := s.ConfigManager.Validate(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Start chat session
	if err := s.ChatLogger.StartSession(); err != nil {
		return fmt.Errorf("failed to start chat session: %w", err)
	}

	s.logger.Info("Storage system initialized successfully")
	return nil
}
