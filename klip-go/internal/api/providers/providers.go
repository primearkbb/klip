package providers

// Provider-specific implementations and configurations

import (
	"context"
	"net/http"
	"time"

	"github.com/john/klip/internal/api"
)

// ProviderConfig holds configuration for each provider
type ProviderConfig struct {
	Name         string
	BaseURL      string
	RequiresAuth bool
	Models       []string
}

// GetProviderConfig returns configuration for a specific provider
func GetProviderConfig(provider api.Provider) *ProviderConfig {
	switch provider {
	case api.ProviderAnthropic:
		return &ProviderConfig{
			Name:         "Anthropic",
			BaseURL:      "https://api.anthropic.com",
			RequiresAuth: true,
			Models: []string{
				"claude-sonnet-4-20250514",
				"claude-3-5-sonnet-20241022",
				"claude-3-5-haiku-20241022",
				"claude-3-opus-20240229",
			},
		}
	case api.ProviderOpenAI:
		return &ProviderConfig{
			Name:         "OpenAI",
			BaseURL:      "https://api.openai.com",
			RequiresAuth: true,
			Models: []string{
				"gpt-4o",
				"gpt-4o-mini",
				"gpt-4-turbo",
				"gpt-3.5-turbo",
				"o1-preview",
				"o1-mini",
			},
		}
	case api.ProviderOpenRouter:
		return &ProviderConfig{
			Name:         "OpenRouter",
			BaseURL:      "https://openrouter.ai/api",
			RequiresAuth: true,
			Models: []string{
				"anthropic/claude-3.5-sonnet",
				"openai/gpt-4o",
				"google/gemini-pro-1.5",
				"meta-llama/llama-3.1-405b-instruct",
			},
		}
	default:
		return nil
	}
}

// GetAllProviders returns all supported providers
func GetAllProviders() []api.Provider {
	return []api.Provider{
		api.ProviderAnthropic,
		api.ProviderOpenAI,
		api.ProviderOpenRouter,
	}
}

// NewProvider creates a new provider instance based on the provider type
func NewProvider(providerType api.Provider, apiKey string, httpClient *http.Client) (api.ProviderInterface, error) {
	switch providerType {
	case api.ProviderAnthropic:
		return NewAnthropicProvider(apiKey, httpClient)
	case api.ProviderOpenAI:
		return NewOpenAIProvider(apiKey, httpClient)
	case api.ProviderOpenRouter:
		return NewOpenRouterProvider(apiKey, httpClient)
	default:
		return nil, &api.APIError{
			StatusCode: 400,
			Message:    "Unsupported provider: " + string(providerType),
			Provider:   "provider",
			Retryable:  false,
		}
	}
}

// ValidateProviderCredentials validates credentials for a specific provider
func ValidateProviderCredentials(providerType api.Provider, apiKey string, httpClient *http.Client) error {
	provider, err := NewProvider(providerType, apiKey, httpClient)
	if err != nil {
		return err
	}
	
	// Use a context with timeout for validation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	return provider.ValidateCredentials(ctx)
}