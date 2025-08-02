package api

import (
	"context"
	"testing"
	"time"
)

func TestNewModelManager(t *testing.T) {
	mm := NewModelManager()

	if mm == nil {
		t.Errorf("Expected model manager but got nil")
	}

	if mm.providers == nil {
		t.Errorf("Expected providers map to be initialized")
	}

	if mm.cachedModels == nil {
		t.Errorf("Expected cachedModels map to be initialized")
	}

	if mm.cacheExpiry == nil {
		t.Errorf("Expected cacheExpiry map to be initialized")
	}

	if mm.cacheDuration != 5*time.Minute {
		t.Errorf("Expected cacheDuration to be 5 minutes, got %v", mm.cacheDuration)
	}
}

func TestModelManagerRegisterProvider(t *testing.T) {
	mm := NewModelManager()
	mockProvider := &MockProvider{}

	mm.RegisterProvider(ProviderAnthropic, mockProvider)

	if len(mm.providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(mm.providers))
	}

	if mm.providers[ProviderAnthropic] != mockProvider {
		t.Errorf("Expected registered provider to match")
	}
}

func TestModelManagerGetModelsByProvider(t *testing.T) {
	mm := NewModelManager()

	// Test with mock provider
	mockModels := []Model{
		{
			ID:       "test-model-1",
			Name:     "Test Model 1",
			Provider: ProviderAnthropic,
		},
		{
			ID:       "test-model-2",
			Name:     "Test Model 2",
			Provider: ProviderAnthropic,
		},
	}

	mockProvider := &MockProvider{models: mockModels}
	mm.RegisterProvider(ProviderAnthropic, mockProvider)

	ctx := context.Background()
	models, err := mm.GetModelsByProvider(ctx, ProviderAnthropic)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	if models[0].ID != "test-model-1" {
		t.Errorf("Expected first model ID to be 'test-model-1', got '%s'", models[0].ID)
	}
}

func TestModelManagerGetModelsByProviderFallback(t *testing.T) {
	mm := NewModelManager()

	// Test fallback to static models when no provider is registered
	ctx := context.Background()
	models, err := mm.GetModelsByProvider(ctx, ProviderAnthropic)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(models) == 0 {
		t.Errorf("Expected static models to be returned as fallback")
	}

	// Verify they are Anthropic models
	for _, model := range models {
		if model.Provider != ProviderAnthropic {
			t.Errorf("Expected provider to be %s, got %s", ProviderAnthropic, model.Provider)
		}
	}
}

func TestModelManagerCaching(t *testing.T) {
	mm := NewModelManager()

	mockModels := []Model{
		{ID: "test-model", Name: "Test Model", Provider: ProviderAnthropic},
	}

	mockProvider := &MockProvider{models: mockModels}
	mm.RegisterProvider(ProviderAnthropic, mockProvider)

	ctx := context.Background()

	// First call should fetch from provider
	models1, err := mm.GetModelsByProvider(ctx, ProviderAnthropic)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Second call should use cache
	models2, err := mm.GetModelsByProvider(ctx, ProviderAnthropic)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(models1) != len(models2) {
		t.Errorf("Expected cached models to match original")
	}

	// Verify cache is working by checking that we get the same results
	if models1[0].ID != models2[0].ID {
		t.Errorf("Expected cached model to match original")
	}
}

func TestModelManagerGetModel(t *testing.T) {
	mm := NewModelManager()

	ctx := context.Background()

	// Test getting a predefined model
	model, err := mm.GetModel(ctx, "claude-3-5-sonnet-20241022")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if model == nil {
		t.Errorf("Expected model but got nil")
	}

	if model.ID != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model ID 'claude-3-5-sonnet-20241022', got '%s'", model.ID)
	}

	if model.Provider != ProviderAnthropic {
		t.Errorf("Expected provider %s, got %s", ProviderAnthropic, model.Provider)
	}

	// Test getting a non-existent model
	_, err = mm.GetModel(ctx, "non-existent-model")
	if err == nil {
		t.Errorf("Expected error for non-existent model")
	}
}

func TestModelManagerGetDefaultModel(t *testing.T) {
	mm := NewModelManager()

	model := mm.GetDefaultModel()

	if model.ID == "" {
		t.Errorf("Expected default model to have an ID")
	}

	if model.Provider != ProviderAnthropic {
		t.Errorf("Expected default model to be Anthropic")
	}

	// Should be one of the Claude models
	expectedModels := []string{"claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022"}
	found := false
	for _, expected := range expectedModels {
		if model.ID == expected {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected default model to be one of %v, got %s", expectedModels, model.ID)
	}
}

func TestModelManagerGetAllModels(t *testing.T) {
	mm := NewModelManager()

	// Register mock providers
	mockAnthropicModels := []Model{
		{ID: "anthropic-1", Provider: ProviderAnthropic},
	}
	mockOpenAIModels := []Model{
		{ID: "openai-1", Provider: ProviderOpenAI},
	}
	mockOpenRouterModels := []Model{
		{ID: "openrouter-1", Provider: ProviderOpenRouter},
	}

	mm.RegisterProvider(ProviderAnthropic, &MockProvider{models: mockAnthropicModels})
	mm.RegisterProvider(ProviderOpenAI, &MockProvider{models: mockOpenAIModels})
	mm.RegisterProvider(ProviderOpenRouter, &MockProvider{models: mockOpenRouterModels})

	ctx := context.Background()
	allModels, err := mm.GetAllModels(ctx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(allModels) < 3 {
		t.Errorf("Expected at least 3 models (one from each provider), got %d", len(allModels))
	}

	// Check that we have models from each provider
	providerCounts := make(map[Provider]int)
	for _, model := range allModels {
		providerCounts[model.Provider]++
	}

	if providerCounts[ProviderAnthropic] == 0 {
		t.Errorf("Expected Anthropic models in result")
	}
	if providerCounts[ProviderOpenAI] == 0 {
		t.Errorf("Expected OpenAI models in result")
	}
	if providerCounts[ProviderOpenRouter] == 0 {
		t.Errorf("Expected OpenRouter models in result")
	}
}

func TestModelManagerValidateModel(t *testing.T) {
	mm := NewModelManager()

	ctx := context.Background()

	// Test valid model
	err := mm.ValidateModel(ctx, "claude-3-5-sonnet-20241022")
	if err != nil {
		t.Errorf("Expected valid model to pass validation, got error: %v", err)
	}

	// Test invalid model
	err = mm.ValidateModel(ctx, "non-existent-model")
	if err == nil {
		t.Errorf("Expected invalid model to fail validation")
	}
}

func TestModelManagerGetModelProviders(t *testing.T) {
	mm := NewModelManager()

	providers := mm.GetModelProviders()

	expectedProviders := []Provider{ProviderAnthropic, ProviderOpenAI, ProviderOpenRouter}

	if len(providers) != len(expectedProviders) {
		t.Errorf("Expected %d providers, got %d", len(expectedProviders), len(providers))
	}

	for _, expected := range expectedProviders {
		found := false
		for _, actual := range providers {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected provider %s not found in result", expected)
		}
	}
}

func TestModelManagerClearCache(t *testing.T) {
	mm := NewModelManager()

	// Add some cache data
	mm.cachedModels[ProviderAnthropic] = []Model{{ID: "test"}}
	mm.cacheExpiry[ProviderAnthropic] = time.Now().Add(time.Hour)

	if len(mm.cachedModels) == 0 {
		t.Errorf("Expected cache to have data before clearing")
	}

	mm.ClearCache()

	if len(mm.cachedModels) != 0 {
		t.Errorf("Expected cache to be empty after clearing, got %d items", len(mm.cachedModels))
	}

	if len(mm.cacheExpiry) != 0 {
		t.Errorf("Expected expiry cache to be empty after clearing, got %d items", len(mm.cacheExpiry))
	}
}

func TestModelManagerClearProviderCache(t *testing.T) {
	mm := NewModelManager()

	// Add cache data for multiple providers
	mm.cachedModels[ProviderAnthropic] = []Model{{ID: "anthropic"}}
	mm.cachedModels[ProviderOpenAI] = []Model{{ID: "openai"}}
	mm.cacheExpiry[ProviderAnthropic] = time.Now().Add(time.Hour)
	mm.cacheExpiry[ProviderOpenAI] = time.Now().Add(time.Hour)

	mm.ClearProviderCache(ProviderAnthropic)

	// Anthropic cache should be cleared
	if _, exists := mm.cachedModels[ProviderAnthropic]; exists {
		t.Errorf("Expected Anthropic cache to be cleared")
	}
	if _, exists := mm.cacheExpiry[ProviderAnthropic]; exists {
		t.Errorf("Expected Anthropic expiry to be cleared")
	}

	// OpenAI cache should remain
	if _, exists := mm.cachedModels[ProviderOpenAI]; !exists {
		t.Errorf("Expected OpenAI cache to remain")
	}
	if _, exists := mm.cacheExpiry[ProviderOpenAI]; !exists {
		t.Errorf("Expected OpenAI expiry to remain")
	}
}

func TestPredefinedModels(t *testing.T) {
	if len(PredefinedModels) == 0 {
		t.Errorf("Expected predefined models to be non-empty")
	}

	// Test some key models exist
	expectedModels := []string{
		"claude-3-5-sonnet-20241022",
		"gpt-4o",
		"gpt-3.5-turbo",
	}

	for _, expected := range expectedModels {
		if _, exists := PredefinedModels[expected]; !exists {
			t.Errorf("Expected predefined model '%s' not found", expected)
		}
	}

	// Verify model structure
	for id, model := range PredefinedModels {
		if model.ID != id {
			t.Errorf("Model ID mismatch: key='%s', model.ID='%s'", id, model.ID)
		}

		if model.Name == "" {
			t.Errorf("Model '%s' has empty name", id)
		}

		if model.Provider == "" {
			t.Errorf("Model '%s' has empty provider", id)
		}

		if model.MaxTokens <= 0 {
			t.Errorf("Model '%s' has invalid MaxTokens: %d", id, model.MaxTokens)
		}

		if model.ContextWindow <= 0 {
			t.Errorf("Model '%s' has invalid ContextWindow: %d", id, model.ContextWindow)
		}
	}
}

// Benchmark tests
func BenchmarkModelManagerGetModel(b *testing.B) {
	mm := NewModelManager()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mm.GetModel(ctx, "claude-3-5-sonnet-20241022")
	}
}

func BenchmarkModelManagerGetModelsByProvider(b *testing.B) {
	mm := NewModelManager()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mm.GetModelsByProvider(ctx, ProviderAnthropic)
	}
}