package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockProvider implements ProviderInterface for testing
type MockProvider struct {
	models []Model
	err    error
}

func (m *MockProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ChatResponse{Content: "test response"}, nil
}

func (m *MockProvider) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errorChan)

		if m.err != nil {
			errorChan <- m.err
			return
		}

		chunkChan <- StreamChunk{Content: "test", Done: false}
		chunkChan <- StreamChunk{Content: "", Done: true}
	}()

	return chunkChan, errorChan
}

func (m *MockProvider) GetModels(ctx context.Context) ([]Model, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.models, nil
}

func (m *MockProvider) ValidateCredentials(ctx context.Context) error {
	return m.err
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name         string
		provider     ProviderInterface
		expectError  bool
		errorMessage string
	}{
		{
			name:        "Valid provider",
			provider:    &MockProvider{},
			expectError: false,
		},
		{
			name:         "Nil provider",
			provider:     nil,
			expectError:  true,
			errorMessage: "provider is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.provider, nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMessage != "" && !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("Expected error message to contain '%s', got: %s", tt.errorMessage, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("Expected client but got nil")
				}
			}
		})
	}
}

func TestRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}

	if config.BaseDelay != 1*time.Second {
		t.Errorf("Expected BaseDelay to be 1s, got %v", config.BaseDelay)
	}

	if len(config.RetryableErrors) == 0 {
		t.Errorf("Expected RetryableErrors to be non-empty")
	}

	expectedErrors := []int{429, 500, 502, 503, 504}
	for _, expected := range expectedErrors {
		found := false
		for _, actual := range config.RetryableErrors {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected RetryableErrors to contain %d", expected)
		}
	}
}

func TestCalculateBackoff(t *testing.T) {
	client := &Client{
		retryConfig: DefaultRetryConfig(),
	}

	// Test exponential backoff
	delay1 := client.calculateBackoff(0)
	delay2 := client.calculateBackoff(1)
	delay3 := client.calculateBackoff(2)

	if delay1 >= delay2 {
		t.Errorf("Expected backoff to increase: %v >= %v", delay1, delay2)
	}

	if delay2 >= delay3 {
		t.Errorf("Expected backoff to increase: %v >= %v", delay2, delay3)
	}

	// Test max delay cap
	maxAttempt := 10
	maxDelay := client.calculateBackoff(maxAttempt)
	if maxDelay > client.retryConfig.MaxDelay {
		t.Errorf("Expected delay to be capped at %v, got %v", client.retryConfig.MaxDelay, maxDelay)
	}
}

func TestShouldRetry(t *testing.T) {
	client := &Client{
		retryConfig: DefaultRetryConfig(),
	}

	tests := []struct {
		name        string
		err         error
		attempt     int
		shouldRetry bool
	}{
		{
			name:        "Network error should retry",
			err:         &APIError{StatusCode: 503, Message: "Service unavailable"},
			attempt:     0,
			shouldRetry: true,
		},
		{
			name:        "Rate limit should retry",
			err:         &APIError{StatusCode: 429, Message: "Rate limited"},
			attempt:     0,
			shouldRetry: true,
		},
		{
			name:        "Auth error should not retry",
			err:         &APIError{StatusCode: 401, Message: "Unauthorized"},
			attempt:     0,
			shouldRetry: false,
		},
		{
			name:        "Max attempts reached",
			err:         &APIError{StatusCode: 503, Message: "Service unavailable"},
			attempt:     3,
			shouldRetry: false,
		},
		{
			name:        "Context canceled should not retry",
			err:         context.Canceled,
			attempt:     0,
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.shouldRetry(tt.err, tt.attempt)
			if result != tt.shouldRetry {
				t.Errorf("Expected shouldRetry to be %v, got %v", tt.shouldRetry, result)
			}
		})
	}
}

func TestMakeHTTPRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/success" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		} else if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Internal server error"}`))
		}
	}))
	defer server.Close()

	client := &http.Client{}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Successful request",
			path:           "/success",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Error request",
			path:           "/error",
			expectedStatus: http.StatusInternalServerError,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			headers := map[string]string{
				"Content-Type": "application/json",
			}

			resp, err := MakeHTTPRequest(
				ctx,
				client,
				"GET",
				server.URL+tt.path,
				headers,
				nil,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if resp == nil {
					t.Errorf("Expected response but got nil")
				} else {
					defer resp.Body.Close()
					if resp.StatusCode != tt.expectedStatus {
						t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
					}
				}
			}
		})
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Message:    "Not found",
		Provider:   "test",
		Retryable:  false,
	}

	expected := "test API Error (404): Not found"
	actual := err.Error()

	if actual != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, actual)
	}
}

func TestParseErrorResponse(t *testing.T) {
	// Create a test server that returns various error formats
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json-error":
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": {"message": "Invalid request"}}`))
		case "/simple-error":
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"message": "Simple error"}`))
		case "/plain-text":
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Plain text error"))
		}
	}))
	defer server.Close()

	client := &http.Client{}

	tests := []struct {
		name            string
		path            string
		expectedMessage string
	}{
		{
			name:            "JSON error format",
			path:            "/json-error",
			expectedMessage: "Invalid request",
		},
		{
			name:            "Simple error format",
			path:            "/simple-error",
			expectedMessage: "Simple error",
		},
		{
			name:            "Plain text error",
			path:            "/plain-text",
			expectedMessage: "Plain text error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			apiErr := ParseErrorResponse(resp, "test")
			if apiErr == nil {
				t.Errorf("Expected API error but got nil")
			} else {
				if !strings.Contains(apiErr.Error(), tt.expectedMessage) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedMessage, apiErr.Error())
				}
			}
		})
	}
}

func TestBuildRequestMetrics(t *testing.T) {
	client := &Client{}
	startTime := time.Now()

	req := &ChatRequest{
		Model: Model{
			ID:       "test-model",
			Name:     "Test Model",
			Provider: ProviderAnthropic,
		},
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant", Timestamp: startTime},
			{Role: "user", Content: "Hello", Timestamp: startTime},
			{Role: "assistant", Content: "Hi there!", Timestamp: startTime},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
		Stream:      true,
	}

	metrics := client.buildRequestMetrics(req, startTime)

	if metrics.StartTime != startTime {
		t.Errorf("Expected StartTime to be %v, got %v", startTime, metrics.StartTime)
	}

	if metrics.ModelID != "test-model" {
		t.Errorf("Expected ModelID to be 'test-model', got '%s'", metrics.ModelID)
	}

	if metrics.ModelName != "Test Model" {
		t.Errorf("Expected ModelName to be 'Test Model', got '%s'", metrics.ModelName)
	}

	if metrics.Provider != "anthropic" {
		t.Errorf("Expected Provider to be 'anthropic', got '%s'", metrics.Provider)
	}

	if metrics.MessageCount != 3 {
		t.Errorf("Expected MessageCount to be 3, got %d", metrics.MessageCount)
	}

	if !metrics.HasSystemMessage {
		t.Errorf("Expected HasSystemMessage to be true")
	}

	if metrics.MaxTokens != 1000 {
		t.Errorf("Expected MaxTokens to be 1000, got %d", metrics.MaxTokens)
	}

	if metrics.Temperature != 0.7 {
		t.Errorf("Expected Temperature to be 0.7, got %f", metrics.Temperature)
	}

	if !metrics.IsStream {
		t.Errorf("Expected IsStream to be true")
	}
}

// Benchmark tests
func BenchmarkCalculateBackoff(b *testing.B) {
	client := &Client{
		retryConfig: DefaultRetryConfig(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.calculateBackoff(i % 5)
	}
}

func BenchmarkShouldRetry(b *testing.B) {
	client := &Client{
		retryConfig: DefaultRetryConfig(),
	}

	err := &APIError{StatusCode: 503, Message: "Service unavailable"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.shouldRetry(err, i%3)
	}
}
