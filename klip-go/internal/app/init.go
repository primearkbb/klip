package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/john/klip/internal/api"
	"github.com/john/klip/internal/api/providers"
	"github.com/john/klip/internal/storage"
)

// initializeApp initializes all application components
func (m *Model) initializeApp() tea.Cmd {
	return func() tea.Msg {
		return initStartMsg{}
	}
}

// initializeStorage initializes the storage system
func (m *Model) initializeStorage() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// Initialize logger first
		m.logger = log.New(os.Stderr)
		m.logger.SetLevel(log.InfoLevel)

		// Initialize storage
		storage, err := storage.New()
		if err != nil {
			m.logger.Error("Failed to initialize storage", "error", err)
			return initErrorMsg{fmt.Errorf("storage initialization failed: %w", err)}
		}

		m.storage = storage

		// Initialize storage components
		if err := m.storage.Initialize(); err != nil {
			m.logger.Error("Failed to initialize storage components", "error", err)
			return initErrorMsg{fmt.Errorf("storage component initialization failed: %w", err)}
		}

		m.logger.Info("Storage system initialized successfully")
		return initKeystoreMsg{}
	})
}

// initializeKeystore initializes the keystore and checks for API keys
func (m *Model) initializeKeystore() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if m.storage == nil || m.storage.KeyStore == nil {
			return initErrorMsg{fmt.Errorf("keystore not available")}
		}

		// Check for existing API keys
		providers := []api.Provider{api.ProviderAnthropic, api.ProviderOpenAI, api.ProviderOpenRouter}
		hasAnyKey := false

		for _, provider := range providers {
			hasKey, err := m.storage.KeyStore.HasKey(string(provider))
			if err != nil {
				m.logger.Warn("Failed to check API key", "provider", provider, "error", err)
				continue
			}

			if hasKey {
				hasAnyKey = true
				m.logger.Debug("Found API key", "provider", provider)
			}
		}

		if !hasAnyKey {
			m.logger.Warn("No API keys found - user will need to set them up")
			// Don't fail initialization, just continue to onboarding
		}

		m.logger.Info("Keystore initialized successfully")
		return initConfigMsg{}
	})
}

// initializeConfig loads configuration
func (m *Model) initializeConfig() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if m.storage == nil || m.storage.ConfigManager == nil {
			return initErrorMsg{fmt.Errorf("config manager not available")}
		}

		// Load configuration
		config, err := m.storage.ConfigManager.LoadConfig()
		if err != nil {
			m.logger.Warn("Failed to load config, using defaults", "error", err)
			// Create default config
			config = &storage.Config{
				DefaultModel: "claude-3-5-sonnet-20241022",
				UIPreferences: &storage.UIPreferences{
					Theme:          "dark",
					ShowTimestamps: true,
					CompactMode:    false,
				},
				Analytics: &storage.AnalyticsConfig{
					Enabled:    true,
					RetainDays: 30,
				},
			}

			// Save default config
			if saveErr := m.storage.ConfigManager.SaveConfig(config); saveErr != nil {
				m.logger.Warn("Failed to save default config", "error", saveErr)
			}
		}

		m.config = config

		// Apply configuration
		m.applyConfiguration(config)

		m.logger.Info("Configuration loaded successfully")
		return initAnalyticsMsg{}
	})
}

// initializeAnalytics initializes analytics logging
func (m *Model) initializeAnalytics() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if m.storage == nil || m.storage.AnalyticsLogger == nil {
			return initErrorMsg{fmt.Errorf("analytics logger not available")}
		}

		// Start analytics session
		if m.analyticsEnabled {
			// TODO: Implement session start logging when method is available
			m.logger.Debug("Analytics enabled")
		}

		m.logger.Info("Analytics initialized successfully")
		return initAPIClientMsg{}
	})
}

// initializeAPIClient initializes the API client with the default model
func (m *Model) initializeAPIClient() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// Set default model
		m.currentModel = m.getDefaultModel()

		// Initialize API client for the default model
		if err := m.initAPIClientForModel(m.currentModel); err != nil {
			m.logger.Warn("Failed to initialize API client", "error", err, "model", m.currentModel.Name)
			// Don't fail initialization - user can set up API keys later
		} else {
			m.logger.Info("API client initialized", "model", m.currentModel.Name, "provider", m.currentModel.Provider)
		}

		return initCompleteMsg{}
	})
}

// initAPIClientForModel initializes the API client for a specific model
func (m *Model) initAPIClientForModel(model api.Model) error {
	if m.storage == nil || m.storage.KeyStore == nil {
		return fmt.Errorf("keystore not available")
	}

	// Get API key for the model's provider
	apiKey, err := m.storage.KeyStore.GetKey(string(model.Provider))
	if err != nil {
		return fmt.Errorf("failed to get API key for %s: %w", model.Provider, err)
	}

	if apiKey == "" {
		return fmt.Errorf("no API key found for provider %s", model.Provider)
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 120 * time.Second,
	}

	// Create provider-specific client
	var provider api.ProviderInterface

	switch model.Provider {
	case api.ProviderAnthropic:
		provider, err = providers.NewAnthropicProvider(apiKey, httpClient)
	case api.ProviderOpenAI:
		provider, err = providers.NewOpenAIProvider(apiKey, httpClient)
	case api.ProviderOpenRouter:
		provider, err = providers.NewOpenRouterProvider(apiKey, httpClient)
	default:
		return fmt.Errorf("unsupported provider: %s", model.Provider)
	}

	if err != nil {
		return fmt.Errorf("failed to create provider client: %w", err)
	}

	// Validate credentials
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	if err := provider.ValidateCredentials(ctx); err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}

	m.apiClient = provider
	return nil
}

// getDefaultModel returns the default model based on configuration
func (m *Model) getDefaultModel() api.Model {
	// Try to get from config
	if m.config != nil && m.config.DefaultModel != "" {
		// TODO: Look up model by ID from available models
		// For now, return a hardcoded default
	}

	// Return Claude 3.5 Sonnet as default
	return api.Model{
		ID:            "claude-3-5-sonnet-20241022",
		Name:          "Claude 3.5 Sonnet",
		Provider:      api.ProviderAnthropic,
		MaxTokens:     4096,
		ContextWindow: 200000,
	}
}

// applyConfiguration applies loaded configuration to the application
func (m *Model) applyConfiguration(config *storage.Config) {
	if config.Analytics != nil {
		m.analyticsEnabled = config.Analytics.Enabled
	}

	// Apply UI configuration
	if config.UIPreferences != nil {
		// UI config will be used in rendering
	}

	// Apply other configuration options as needed
}

// loadAvailableModels loads available models from all providers
func (m *Model) loadAvailableModels() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		var allModels []api.Model

		// Get static models first
		staticModels := m.getStaticModels()
		allModels = append(allModels, staticModels...)

		// Try to load dynamic models from providers that support it
		if m.storage != nil && m.storage.KeyStore != nil {
			// Check OpenRouter for dynamic models
			if hasKey, err := m.storage.KeyStore.HasKey(string(api.ProviderOpenRouter)); err == nil && hasKey {
				if dynamicModels, err := m.loadOpenRouterModels(); err == nil {
					allModels = append(allModels, dynamicModels...)
				} else {
					m.logger.Warn("Failed to load OpenRouter models", "error", err)
				}
			}
		}

		m.logger.Info("Loaded models", "count", len(allModels))
		return modelsLoadSuccessMsg{allModels}
	})
}

// getStaticModels returns a list of static models
func (m *Model) getStaticModels() []api.Model {
	return []api.Model{
		// Anthropic models
		{
			ID:            "claude-3-5-sonnet-20241022",
			Name:          "Claude 3.5 Sonnet",
			Provider:      api.ProviderAnthropic,
			MaxTokens:     4096,
			ContextWindow: 200000,
		},
		{
			ID:            "claude-3-5-haiku-20241022",
			Name:          "Claude 3.5 Haiku",
			Provider:      api.ProviderAnthropic,
			MaxTokens:     4096,
			ContextWindow: 200000,
		},
		{
			ID:            "claude-3-opus-20240229",
			Name:          "Claude 3 Opus",
			Provider:      api.ProviderAnthropic,
			MaxTokens:     4096,
			ContextWindow: 200000,
		},

		// OpenAI models
		{
			ID:            "gpt-4o",
			Name:          "GPT-4o",
			Provider:      api.ProviderOpenAI,
			MaxTokens:     4096,
			ContextWindow: 128000,
		},
		{
			ID:            "gpt-4o-mini",
			Name:          "GPT-4o mini",
			Provider:      api.ProviderOpenAI,
			MaxTokens:     4096,
			ContextWindow: 128000,
		},
		{
			ID:            "gpt-4-turbo",
			Name:          "GPT-4 Turbo",
			Provider:      api.ProviderOpenAI,
			MaxTokens:     4096,
			ContextWindow: 128000,
		},

		// Popular OpenRouter models
		{
			ID:            "anthropic/claude-3.5-sonnet",
			Name:          "Claude 3.5 Sonnet (OpenRouter)",
			Provider:      api.ProviderOpenRouter,
			MaxTokens:     4096,
			ContextWindow: 200000,
		},
		{
			ID:            "openai/gpt-4o",
			Name:          "GPT-4o (OpenRouter)",
			Provider:      api.ProviderOpenRouter,
			MaxTokens:     4096,
			ContextWindow: 128000,
		},
	}
}

// loadOpenRouterModels loads dynamic models from OpenRouter
func (m *Model) loadOpenRouterModels() ([]api.Model, error) {
	if m.storage == nil || m.storage.KeyStore == nil {
		return nil, fmt.Errorf("keystore not available")
	}

	apiKey, err := m.storage.KeyStore.GetKey(string(api.ProviderOpenRouter))
	if err != nil || apiKey == "" {
		return nil, fmt.Errorf("no OpenRouter API key found")
	}

	// Create OpenRouter provider
	httpClient := &http.Client{Timeout: 30 * time.Second}
	provider, err := providers.NewOpenRouterProvider(apiKey, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenRouter provider: %w", err)
	}

	// Get models with timeout
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	models, err := provider.GetModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OpenRouter models: %w", err)
	}

	return models, nil
}

// performAPIRequest performs an API request with the current client
func (m *Model) performAPIRequest(request *api.ChatRequest) tea.Cmd {
	if m.apiClient == nil {
		return func() tea.Msg {
			return apiErrorMsg{fmt.Errorf("no API client available")}
		}
	}

	// Handle streaming request
	if request.Stream {
		return m.performStreamingRequest(request)
	}

	// Handle non-streaming request
	return tea.Cmd(func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 60*time.Second)
		defer cancel()

		response, err := m.apiClient.Chat(ctx, request)
		if err != nil {
			return apiErrorMsg{err}
		}

		return apiResponseMsg{response}
	})
}

// performStreamingRequest performs a streaming API request
func (m *Model) performStreamingRequest(request *api.ChatRequest) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx, cancel := context.WithCancel(m.ctx)
		defer cancel()

		// Set up interrupt handling
		go func() {
			select {
			case <-m.chatState.InterruptChannel:
				cancel()
			case <-ctx.Done():
			}
		}()

		chunkChan, errChan := m.apiClient.ChatStream(ctx, request)

		// Start a goroutine to handle the stream
		go func() {
			for {
				select {
				case chunk, ok := <-chunkChan:
					if !ok {
						// Stream finished
						return
					}

					// Send chunk to UI
					tea.Batch(func() tea.Msg {
						return apiStreamChunkMsg{chunk.Content}
					})()

					if chunk.Done {
						tea.Batch(func() tea.Msg {
							return apiStreamDoneMsg{}
						})()
						return
					}

				case err, ok := <-errChan:
					if !ok {
						return
					}

					tea.Batch(func() tea.Msg {
						return apiErrorMsg{err}
					})()
					return

				case <-ctx.Done():
					return
				}
			}
		}()

		// Return immediately to keep UI responsive
		return nil
	})
}

// switchModel switches to a different model
func (m *Model) switchModel(model api.Model) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// Initialize API client for the new model
		if err := m.initAPIClientForModel(model); err != nil {
			return apiErrorMsg{fmt.Errorf("failed to switch to model %s: %w", model.Name, err)}
		}

		// Update current model
		m.currentModel = model

		// Log the model switch
		// TODO: Implement model switch logging when method is available

		return modelSwitchMsg{model}
	})
}

// checkForUpdates checks for application updates (placeholder)
func (m *Model) checkForUpdates() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// TODO: Implement update checking
		return nil
	})
}

// validateSystemRequirements validates system requirements
func (m *Model) validateSystemRequirements() error {
	// Check if we can write to the config directory
	// TODO: Implement config directory validation when GetConfigDir method is available

	return nil
}

// migrateFromDeno migrates configuration from the Deno version if it exists
func (m *Model) migrateFromDeno() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if m.storage == nil || m.storage.ConfigManager == nil {
			return nil
		}

		// Attempt migration
		if err := m.storage.ConfigManager.MigrateFromDeno(); err != nil {
			m.logger.Warn("Failed to migrate from Deno", "error", err)
		} else {
			m.logger.Info("Successfully migrated configuration from Deno")
		}

		return nil
	})
}

// setupSignalHandling sets up signal handling for graceful shutdown
func (m *Model) setupSignalHandling() {
	// This would typically be done at the program level, not in the model
	// But we can prepare for it here by ensuring cleanup is ready

	// The cleanup function is already implemented in the model
}
