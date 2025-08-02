package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/john/klip/internal/api"
)

func TestNewAnthropicProvider(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		expectError bool
	}{
		{
			name:        "Valid API key",
			apiKey:      "test-key",
			expectError: false,
		},
		{
			name:        "Empty API key",
			apiKey:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpClient := &http.Client{}
			provider, err := NewAnthropicProvider(tt.apiKey, httpClient)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if provider == nil {
					t.Errorf("Expected provider but got nil")
				}
			}
		})
	}
}

func TestAnthropicProviderGetModels(t *testing.T) {
	httpClient := &http.Client{}
	provider, err := NewAnthropicProvider("test-key", httpClient)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	models, err := provider.GetModels(ctx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(models) == 0 {
		t.Errorf("Expected models to be returned")
	}

	// Verify all models are Anthropic models
	for _, model := range models {
		if model.Provider != api.ProviderAnthropic {
			t.Errorf("Expected provider to be %s, got %s", api.ProviderAnthropic, model.Provider)
		}

		if model.ID == "" {
			t.Errorf("Expected model to have an ID")
		}

		if model.Name == "" {
			t.Errorf("Expected model to have a name")
		}

		if model.MaxTokens <= 0 {
			t.Errorf("Expected model to have positive MaxTokens, got %d", model.MaxTokens)
		}

		if model.ContextWindow <= 0 {
			t.Errorf("Expected model to have positive ContextWindow, got %d", model.ContextWindow)
		}
	}

	// Check for expected models
	expectedModels := []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
	}

	for _, expected := range expectedModels {
		found := false
		for _, model := range models {
			if model.ID == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected model '%s' not found in results", expected)
		}
	}
}

func TestAnthropicBuildRequest(t *testing.T) {
	httpClient := &http.Client{}
	provider, err := NewAnthropicProvider("test-key", httpClient)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	anthProvider := provider.(*AnthropicProvider)

	req := &api.ChatRequest{
		Model: api.Model{
			ID:        "claude-3-5-sonnet-20241022",
			MaxTokens: 1000,
		},
		Messages: []api.Message{
			{Role: "system", Content: "You are a helpful assistant", Timestamp: time.Now()},
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
		},
		MaxTokens:       500,
		EnableWebSearch: true,
	}

	// Test non-streaming request
	anthropicReq := anthProvider.buildAnthropicRequest(req, false)

	if anthropicReq.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got '%s'", anthropicReq.Model)
	}

	if anthropicReq.MaxTokens != 500 {
		t.Errorf("Expected MaxTokens 500, got %d", anthropicReq.MaxTokens)
	}

	if anthropicReq.Stream {
		t.Errorf("Expected Stream to be false for non-streaming request")
	}

	if anthropicReq.System != "You are a helpful assistant" {
		t.Errorf("Expected system message to be extracted")
	}

	if len(anthropicReq.Messages) != 1 {
		t.Errorf("Expected 1 non-system message, got %d", len(anthropicReq.Messages))
	}

	if anthropicReq.Messages[0].Role != "user" {
		t.Errorf("Expected user message, got '%s'", anthropicReq.Messages[0].Role)
	}

	if len(anthropicReq.Tools) == 0 {
		t.Errorf("Expected web search tool to be added")
	}

	// Test streaming request
	streamingReq := anthProvider.buildAnthropicRequest(req, true)
	if !streamingReq.Stream {
		t.Errorf("Expected Stream to be true for streaming request")
	}
}

func TestAnthropicParseResponse(t *testing.T) {
	httpClient := &http.Client{}
	provider, err := NewAnthropicProvider("test-key", httpClient)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	anthProvider := provider.(*AnthropicProvider)

	anthropicResp := &AnthropicResponse{
		Content: []AnthropicContent{
			{Type: "text", Text: "Hello there!"},
		},
		Usage: AnthropicUsage{
			InputTokens:  10,
			OutputTokens: 5,
		},
	}

	chatResp := anthProvider.parseAnthropicResponse(anthropicResp)

	if chatResp.Content != "Hello there!" {
		t.Errorf("Expected content 'Hello there!', got '%s'", chatResp.Content)
	}

	if chatResp.Usage == nil {
		t.Errorf("Expected usage to be set")
	} else {
		if chatResp.Usage.InputTokens != 10 {
			t.Errorf("Expected InputTokens 10, got %d", chatResp.Usage.InputTokens)
		}
		if chatResp.Usage.OutputTokens != 5 {
			t.Errorf("Expected OutputTokens 5, got %d", chatResp.Usage.OutputTokens)
		}
	}
}

func TestAnthropicChatIntegration(t *testing.T) {
	// Create a mock server that simulates Anthropic API responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/messages" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [{"type": "text", "text": "Hello! How can I help you?"}],
				"model": "claude-3-5-sonnet-20241022",
				"stop_reason": "end_turn",
				"usage": {
					"input_tokens": 15,
					"output_tokens": 8
				}
			}`))
		}
	}))
	defer server.Close()

	httpClient := &http.Client{}
	provider, err := NewAnthropicProvider("test-key", httpClient)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Override the base URL to use our test server
	anthProvider := provider.(*AnthropicProvider)
	anthProvider.baseURL = server.URL

	req := &api.ChatRequest{
		Model: api.Model{
			ID:        "claude-3-5-sonnet-20241022",
			MaxTokens: 1000,
		},
		Messages: []api.Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
		},
	}

	ctx := context.Background()
	resp, err := provider.Chat(ctx, req)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp == nil {
		t.Errorf("Expected response but got nil")
	} else {
		if resp.Content != "Hello! How can I help you?" {
			t.Errorf("Expected content 'Hello! How can I help you?', got '%s'", resp.Content)
		}

		if resp.Usage == nil {
			t.Errorf("Expected usage to be set")
		} else {
			if resp.Usage.InputTokens != 15 {
				t.Errorf("Expected InputTokens 15, got %d", resp.Usage.InputTokens)
			}
			if resp.Usage.OutputTokens != 8 {
				t.Errorf("Expected OutputTokens 8, got %d", resp.Usage.OutputTokens)
			}
		}
	}
}

func TestAnthropicErrorHandling(t *testing.T) {
	// Create a mock server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{
			"type": "error",
			"error": {
				"type": "invalid_request_error",
				"message": "Invalid request"
			}
		}`))
	}))
	defer server.Close()

	httpClient := &http.Client{}
	provider, err := NewAnthropicProvider("test-key", httpClient)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Override the base URL to use our test server
	anthProvider := provider.(*AnthropicProvider)
	anthProvider.baseURL = server.URL

	req := &api.ChatRequest{
		Model: api.Model{
			ID:        "claude-3-5-sonnet-20241022",
			MaxTokens: 1000,
		},
		Messages: []api.Message{
			{Role: "user", Content: "Hello", Timestamp: time.Now()},
		},
	}

	ctx := context.Background()
	_, err = provider.Chat(ctx, req)

	if err == nil {
		t.Errorf("Expected error but got none")
	}

	// Check if it's an API error
	if apiErr, ok := err.(*api.APIError); ok {
		if apiErr.StatusCode != 400 {
			t.Errorf("Expected status code 400, got %d", apiErr.StatusCode)
		}
		if apiErr.Provider != "anthropic" {
			t.Errorf("Expected provider 'anthropic', got '%s'", apiErr.Provider)
		}
	} else {
		t.Errorf("Expected APIError, got %T", err)
	}
}