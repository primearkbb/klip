package api

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PredefinedModels contains static model definitions for all providers
var PredefinedModels = map[string]Model{
	// Anthropic Claude 4 Series
	"claude-opus-4-20250514": {
		ID:            "claude-opus-4-20250514",
		Name:          "Claude Opus 4",
		Provider:      ProviderAnthropic,
		MaxTokens:     32000,
		ContextWindow: 200000,
	},
	"claude-sonnet-4-20250514": {
		ID:            "claude-sonnet-4-20250514",
		Name:          "Claude Sonnet 4",
		Provider:      ProviderAnthropic,
		MaxTokens:     8192,
		ContextWindow: 200000,
	},

	// Anthropic Claude 3.7 Series
	"claude-3-7-sonnet-20250219": {
		ID:            "claude-3-7-sonnet-20250219",
		Name:          "Claude 3.7 Sonnet",
		Provider:      ProviderAnthropic,
		MaxTokens:     8192,
		ContextWindow: 200000,
	},

	// Anthropic Claude 3.5 Series
	"claude-3-5-sonnet-20241022": {
		ID:            "claude-3-5-sonnet-20241022",
		Name:          "Claude 3.5 Sonnet (v2)",
		Provider:      ProviderAnthropic,
		MaxTokens:     8192,
		ContextWindow: 200000,
	},
	"claude-3-5-sonnet-20240620": {
		ID:            "claude-3-5-sonnet-20240620",
		Name:          "Claude 3.5 Sonnet (v1)",
		Provider:      ProviderAnthropic,
		MaxTokens:     8192,
		ContextWindow: 200000,
	},
	"claude-3-5-haiku-20241022": {
		ID:            "claude-3-5-haiku-20241022",
		Name:          "Claude 3.5 Haiku",
		Provider:      ProviderAnthropic,
		MaxTokens:     8192,
		ContextWindow: 200000,
	},

	// Anthropic Claude 3 Series
	"claude-3-opus-20240229": {
		ID:            "claude-3-opus-20240229",
		Name:          "Claude 3 Opus",
		Provider:      ProviderAnthropic,
		MaxTokens:     4096,
		ContextWindow: 200000,
	},
	"claude-3-haiku-20240307": {
		ID:            "claude-3-haiku-20240307",
		Name:          "Claude 3 Haiku",
		Provider:      ProviderAnthropic,
		MaxTokens:     4096,
		ContextWindow: 200000,
	},

	// OpenAI GPT-4.1 Series
	"gpt-4.1": {
		ID:            "gpt-4.1",
		Name:          "GPT-4.1",
		Provider:      ProviderOpenAI,
		MaxTokens:     16384,
		ContextWindow: 1000000,
	},
	"gpt-4.1-mini": {
		ID:            "gpt-4.1-mini",
		Name:          "GPT-4.1 Mini",
		Provider:      ProviderOpenAI,
		MaxTokens:     16384,
		ContextWindow: 1000000,
	},
	"gpt-4.1-nano": {
		ID:            "gpt-4.1-nano",
		Name:          "GPT-4.1 Nano",
		Provider:      ProviderOpenAI,
		MaxTokens:     16384,
		ContextWindow: 1000000,
	},

	// OpenAI o-series models
	"o3": {
		ID:            "o3",
		Name:          "OpenAI o3",
		Provider:      ProviderOpenAI,
		MaxTokens:     16384,
		ContextWindow: 200000,
	},
	"o3-pro": {
		ID:            "o3-pro",
		Name:          "OpenAI o3 Pro",
		Provider:      ProviderOpenAI,
		MaxTokens:     16384,
		ContextWindow: 200000,
	},
	"o4-mini": {
		ID:            "o4-mini",
		Name:          "OpenAI o4 Mini",
		Provider:      ProviderOpenAI,
		MaxTokens:     16384,
		ContextWindow: 200000,
	},
	"o1-preview": {
		ID:            "o1-preview",
		Name:          "o1 Preview",
		Provider:      ProviderOpenAI,
		MaxTokens:     32768,
		ContextWindow: 128000,
	},
	"o1-mini": {
		ID:            "o1-mini",
		Name:          "o1 Mini",
		Provider:      ProviderOpenAI,
		MaxTokens:     65536,
		ContextWindow: 128000,
	},

	// OpenAI GPT-4o Series
	"gpt-4o": {
		ID:            "gpt-4o",
		Name:          "GPT-4o",
		Provider:      ProviderOpenAI,
		MaxTokens:     16384,
		ContextWindow: 128000,
	},
	"gpt-4o-mini": {
		ID:            "gpt-4o-mini",
		Name:          "GPT-4o Mini",
		Provider:      ProviderOpenAI,
		MaxTokens:     16384,
		ContextWindow: 128000,
	},

	// OpenAI GPT-4 Series
	"gpt-4-turbo": {
		ID:            "gpt-4-turbo",
		Name:          "GPT-4 Turbo",
		Provider:      ProviderOpenAI,
		MaxTokens:     4096,
		ContextWindow: 128000,
	},
	"gpt-4": {
		ID:            "gpt-4",
		Name:          "GPT-4",
		Provider:      ProviderOpenAI,
		MaxTokens:     8192,
		ContextWindow: 8192,
	},

	// OpenAI GPT-3.5 Series
	"gpt-3.5-turbo": {
		ID:            "gpt-3.5-turbo",
		Name:          "GPT-3.5 Turbo",
		Provider:      ProviderOpenAI,
		MaxTokens:     4096,
		ContextWindow: 16384,
	},

	// Popular OpenRouter Models (static fallback)
	"anthropic/claude-3.5-sonnet": {
		ID:            "anthropic/claude-3.5-sonnet",
		Name:          "Claude 3.5 Sonnet (OpenRouter)",
		Provider:      ProviderOpenRouter,
		MaxTokens:     8192,
		ContextWindow: 200000,
	},
	"openai/gpt-4o": {
		ID:            "openai/gpt-4o",
		Name:          "GPT-4o (OpenRouter)",
		Provider:      ProviderOpenRouter,
		MaxTokens:     16384,
		ContextWindow: 128000,
	},
	"meta-llama/llama-3.1-405b-instruct": {
		ID:            "meta-llama/llama-3.1-405b-instruct",
		Name:          "Llama 3.1 405B (OpenRouter)",
		Provider:      ProviderOpenRouter,
		MaxTokens:     4096,
		ContextWindow: 131072,
	},
	"google/gemini-pro-1.5": {
		ID:            "google/gemini-pro-1.5",
		Name:          "Gemini Pro 1.5 (OpenRouter)",
		Provider:      ProviderOpenRouter,
		MaxTokens:     8192,
		ContextWindow: 2000000,
	},
}

// ModelManager handles model discovery and caching
type ModelManager struct {
	providers     map[Provider]ProviderInterface
	cachedModels  map[Provider][]Model
	cacheExpiry   map[Provider]time.Time
	cacheMutex    sync.RWMutex
	cacheDuration time.Duration
}

// NewModelManager creates a new model manager
func NewModelManager() *ModelManager {
	return &ModelManager{
		providers:     make(map[Provider]ProviderInterface),
		cachedModels:  make(map[Provider][]Model),
		cacheExpiry:   make(map[Provider]time.Time),
		cacheDuration: 5 * time.Minute,
	}
}

// RegisterProvider registers a provider with the model manager
func (mm *ModelManager) RegisterProvider(provider Provider, providerImpl ProviderInterface) {
	mm.providers[provider] = providerImpl
}

// GetModelsByProvider returns models for a specific provider
func (mm *ModelManager) GetModelsByProvider(ctx context.Context, provider Provider) ([]Model, error) {
	// Check cache first
	mm.cacheMutex.RLock()
	if expiry, exists := mm.cacheExpiry[provider]; exists && time.Now().Before(expiry) {
		if models, exists := mm.cachedModels[provider]; exists {
			result := make([]Model, len(models))
			copy(result, models)
			mm.cacheMutex.RUnlock()
			return result, nil
		}
	}
	mm.cacheMutex.RUnlock()

	// Fetch from provider
	providerImpl, exists := mm.providers[provider]
	if !exists {
		return mm.getStaticModelsByProvider(provider), nil
	}

	models, err := providerImpl.GetModels(ctx)
	if err != nil {
		// Fallback to static models on error
		return mm.getStaticModelsByProvider(provider), nil
	}

	// Update cache
	mm.cacheMutex.Lock()
	mm.cachedModels[provider] = models
	mm.cacheExpiry[provider] = time.Now().Add(mm.cacheDuration)
	mm.cacheMutex.Unlock()

	return models, nil
}

// GetAllModels returns all available models from all providers
func (mm *ModelManager) GetAllModels(ctx context.Context) ([]Model, error) {
	var allModels []Model

	for _, provider := range []Provider{ProviderAnthropic, ProviderOpenAI, ProviderOpenRouter} {
		models, err := mm.GetModelsByProvider(ctx, provider)
		if err != nil {
			// Continue with other providers if one fails
			continue
		}
		allModels = append(allModels, models...)
	}

	return allModels, nil
}

// GetModel returns a specific model by ID
func (mm *ModelManager) GetModel(ctx context.Context, modelID string) (*Model, error) {
	// Check static models first
	if model, exists := PredefinedModels[modelID]; exists {
		return &model, nil
	}

	// Search in all providers
	allModels, err := mm.GetAllModels(ctx)
	if err != nil {
		return nil, err
	}

	for _, model := range allModels {
		if model.ID == modelID {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", modelID)
}

// GetDefaultModel returns the default model for the application
func (mm *ModelManager) GetDefaultModel() Model {
	if model, exists := PredefinedModels["claude-sonnet-4-20250514"]; exists {
		return model
	}
	// Fallback to Claude 3.5 Sonnet
	return PredefinedModels["claude-3-5-sonnet-20241022"]
}

// getStaticModelsByProvider returns static models for a provider
func (mm *ModelManager) getStaticModelsByProvider(provider Provider) []Model {
	var models []Model
	for _, model := range PredefinedModels {
		if model.Provider == provider {
			models = append(models, model)
		}
	}
	return models
}

// ValidateModel checks if a model is valid and available
func (mm *ModelManager) ValidateModel(ctx context.Context, modelID string) error {
	_, err := mm.GetModel(ctx, modelID)
	return err
}

// GetModelProviders returns all available providers
func (mm *ModelManager) GetModelProviders() []Provider {
	return []Provider{ProviderAnthropic, ProviderOpenAI, ProviderOpenRouter}
}

// ClearCache clears the model cache for all providers
func (mm *ModelManager) ClearCache() {
	mm.cacheMutex.Lock()
	defer mm.cacheMutex.Unlock()

	mm.cachedModels = make(map[Provider][]Model)
	mm.cacheExpiry = make(map[Provider]time.Time)
}

// ClearProviderCache clears the cache for a specific provider
func (mm *ModelManager) ClearProviderCache(provider Provider) {
	mm.cacheMutex.Lock()
	defer mm.cacheMutex.Unlock()

	delete(mm.cachedModels, provider)
	delete(mm.cacheExpiry, provider)
}
