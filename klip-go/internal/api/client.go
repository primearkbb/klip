package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/john/klip/internal/storage"
)

// Provider defines the different AI providers supported
type Provider string

const (
	ProviderAnthropic  Provider = "anthropic"
	ProviderOpenAI     Provider = "openai"
	ProviderOpenRouter Provider = "openrouter"
)

// String returns the string representation of Provider
func (p Provider) String() string {
	return string(p)
}

// Model represents an AI model
type Model struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Provider      Provider `json:"provider"`
	MaxTokens     int      `json:"max_tokens"`
	ContextWindow int      `json:"context_window"`
}

// Message represents a chat message
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Model           Model     `json:"model"`
	Messages        []Message `json:"messages"`
	MaxTokens       int       `json:"max_tokens,omitempty"`
	Temperature     float64   `json:"temperature,omitempty"`
	Stream          bool      `json:"stream,omitempty"`
	EnableWebSearch bool      `json:"enable_web_search,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Content string           `json:"content"`
	Usage   *Usage           `json:"usage,omitempty"`
	Metrics *ResponseMetrics `json:"metrics,omitempty"`
}

// ResponseMetrics contains response timing and metadata
type ResponseMetrics struct {
	LatencyMs      int64 `json:"latency_ms"`
	TokensInput    int   `json:"tokens_input"`
	TokensOutput   int   `json:"tokens_output"`
	ResponseLength int   `json:"response_length"`
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

// ProviderInterface defines the interface that all providers must implement
type ProviderInterface interface {
	// Chat sends a non-streaming chat request
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	// ChatStream sends a streaming chat request
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, <-chan error)
	// GetModels returns available models for this provider
	GetModels(ctx context.Context) ([]Model, error)
	// ValidateCredentials checks if the API key is valid
	ValidateCredentials(ctx context.Context) error
}

// APIError represents an API error
type APIError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Provider   string `json:"provider"`
	Retryable  bool   `json:"retryable"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s API Error (%d): %s", e.Provider, e.StatusCode, e.Message)
}

// Client represents an API client for AI providers
type Client struct {
	httpClient  *http.Client
	logger      *log.Logger
	analytics   *storage.AnalyticsLogger
	provider    ProviderInterface
	retryConfig *RetryConfig
}

// RetryConfig contains retry configuration
type RetryConfig struct {
	MaxRetries      int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	ExponentBase    float64
	JitterMax       time.Duration
	RetryableErrors []int // HTTP status codes that should trigger retries
}

// DefaultRetryConfig returns the default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:      3,
		BaseDelay:       1 * time.Second,
		MaxDelay:        30 * time.Second,
		ExponentBase:    2.0,
		JitterMax:       1 * time.Second,
		RetryableErrors: []int{429, 500, 502, 503, 504},
	}
}

// NewClient creates a new API client with a provider implementation
func NewClient(provider ProviderInterface, analytics *storage.AnalyticsLogger) (*Client, error) {
	if provider == nil {
		return nil, errors.New("provider is required")
	}

	httpClient := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	client := &Client{
		httpClient:  httpClient,
		logger:      log.New(os.Stderr),
		analytics:   analytics,
		provider:    provider,
		retryConfig: DefaultRetryConfig(),
	}

	return client, nil
}

// Chat sends a chat request to the AI provider
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()
	requestMetrics := c.buildRequestMetrics(req, startTime)

	if c.analytics != nil {
		if err := c.analytics.LogRequest(requestMetrics); err != nil {
			c.logger.Warn("Failed to log request metrics", "error", err)
		}
	}

	var response *ChatResponse
	var err error
	retryCount := 0

	// Execute with retry logic
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		response, err = c.provider.Chat(ctx, req)
		if err == nil {
			break
		}

		retryCount = attempt
		if !c.shouldRetry(err, attempt) {
			break
		}

		// Calculate backoff delay
		delay := c.calculateBackoff(attempt)
		c.logger.Debug("Retrying request", "attempt", attempt+1, "delay", delay, "error", err)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			continue
		}
	}

	endTime := time.Now()
	latency := endTime.Sub(startTime).Milliseconds()

	if response != nil {
		// Add metrics to response
		tokensInput := 0
		tokensOutput := 0
		if response.Usage != nil {
			tokensInput = response.Usage.InputTokens
			tokensOutput = response.Usage.OutputTokens
		}
		response.Metrics = &ResponseMetrics{
			LatencyMs:      latency,
			TokensInput:    tokensInput,
			TokensOutput:   tokensOutput,
			ResponseLength: len(response.Content),
		}
	}

	// Log response metrics
	if c.analytics != nil {
		responseMetrics := storage.ResponseMetrics{
			EndTime:        endTime,
			ResponseLength: 0,
			Interrupted:    false,
			Success:        err == nil,
			RetryCount:     retryCount,
		}

		if response != nil {
			responseMetrics.ResponseLength = len(response.Content)
			if response.Usage != nil {
				responseMetrics.TokensInput = response.Usage.InputTokens
				responseMetrics.TokensOutput = response.Usage.OutputTokens
			}
		}

		if err != nil {
			responseMetrics.ErrorType = fmt.Sprintf("%T", err)
			responseMetrics.ErrorMessage = err.Error()
			if apiErr, ok := err.(*APIError); ok {
				responseMetrics.StatusCode = apiErr.StatusCode
			}
		}

		if logErr := c.analytics.LogResponse(requestMetrics, responseMetrics); logErr != nil {
			c.logger.Warn("Failed to log response metrics", "error", logErr)
		}
	}

	return response, err
}

// ChatStream sends a streaming chat request to the AI provider
func (c *Client) ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, <-chan error) {
	startTime := time.Now()
	requestMetrics := c.buildRequestMetrics(req, startTime)
	requestMetrics.IsStream = true

	if c.analytics != nil {
		if err := c.analytics.LogRequest(requestMetrics); err != nil {
			c.logger.Warn("Failed to log request metrics", "error", err)
		}
	}

	// Create channels for results and errors
	chunkChan := make(chan StreamChunk, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errorChan)

		var totalContent strings.Builder
		var interrupted bool
		retryCount := 0

		// Execute with retry logic
		for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
			retryCount = attempt
			providerChunkChan, providerErrorChan := c.provider.ChatStream(ctx, req)

			// Process the stream
			var streamErr error

			for {
				select {
				case chunk, ok := <-providerChunkChan:
					if !ok {
						// Stream finished successfully
						goto streamComplete
					}
					totalContent.WriteString(chunk.Content)
					chunkChan <- chunk

				case err := <-providerErrorChan:
					streamErr = err
					goto streamError

				case <-ctx.Done():
					interrupted = true
					streamErr = ctx.Err()
					goto streamError
				}
			}

		streamError:
			if streamErr != nil && c.shouldRetry(streamErr, attempt) {
				// Calculate backoff delay
				delay := c.calculateBackoff(attempt)
				c.logger.Debug("Retrying stream request", "attempt", attempt+1, "delay", delay, "error", streamErr)

				select {
				case <-ctx.Done():
					errorChan <- ctx.Err()
					return
				case <-time.After(delay):
					continue
				}
			} else {
				errorChan <- streamErr
				goto logMetrics
			}
		}

	streamComplete:
		// Send final chunk to indicate completion
		chunkChan <- StreamChunk{Content: "", Done: true}

	logMetrics:
		// Log response metrics
		if c.analytics != nil {
			endTime := time.Now()
			responseMetrics := storage.ResponseMetrics{
				EndTime:        endTime,
				ResponseLength: totalContent.Len(),
				Interrupted:    interrupted,
				Success:        !interrupted && len(errorChan) == 0,
				RetryCount:     retryCount,
			}

			if len(errorChan) > 0 {
				// Peek at error without consuming it
				select {
				case err := <-errorChan:
					responseMetrics.ErrorType = fmt.Sprintf("%T", err)
					responseMetrics.ErrorMessage = err.Error()
					if apiErr, ok := err.(*APIError); ok {
						responseMetrics.StatusCode = apiErr.StatusCode
					}
					errorChan <- err // Put it back
				default:
				}
			}

			if logErr := c.analytics.LogResponse(requestMetrics, responseMetrics); logErr != nil {
				c.logger.Warn("Failed to log response metrics", "error", logErr)
			}
		}
	}()

	return chunkChan, errorChan
}

// GetModels returns available models for the current provider
func (c *Client) GetModels(ctx context.Context) ([]Model, error) {
	return c.provider.GetModels(ctx)
}

// ValidateCredentials checks if the API key is valid
func (c *Client) ValidateCredentials(ctx context.Context) error {
	return c.provider.ValidateCredentials(ctx)
}

// shouldRetry determines if an error should trigger a retry
func (c *Client) shouldRetry(err error, attempt int) bool {
	if attempt >= c.retryConfig.MaxRetries {
		return false
	}

	// Check for context cancellation
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for retryable API errors
	if apiErr, ok := err.(*APIError); ok {
		for _, retryableCode := range c.retryConfig.RetryableErrors {
			if apiErr.StatusCode == retryableCode {
				return true
			}
		}
		return apiErr.Retryable
	}

	// Retry on network errors
	return strings.Contains(err.Error(), "connection") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "network")
}

// calculateBackoff calculates the backoff delay for a retry attempt
func (c *Client) calculateBackoff(attempt int) time.Duration {
	base := float64(c.retryConfig.BaseDelay)
	delay := base * math.Pow(c.retryConfig.ExponentBase, float64(attempt))

	// Add jitter
	if c.retryConfig.JitterMax > 0 {
		jitter := time.Duration(rand.Int63n(int64(c.retryConfig.JitterMax)))
		delay += float64(jitter)
	}

	// Clamp to max delay
	if time.Duration(delay) > c.retryConfig.MaxDelay {
		return c.retryConfig.MaxDelay
	}

	return time.Duration(delay)
}

// buildRequestMetrics creates request metrics for analytics
func (c *Client) buildRequestMetrics(req *ChatRequest, startTime time.Time) storage.RequestMetrics {
	systemMessage := ""
	userMessages := make([]Message, 0)
	totalLength := 0

	for _, msg := range req.Messages {
		totalLength += len(msg.Content)
		if msg.Role == "system" {
			systemMessage = msg.Content
		} else if msg.Role == "user" {
			userMessages = append(userMessages, msg)
		}
	}

	lastUserMessage := ""
	if len(userMessages) > 0 {
		lastUserMessage = userMessages[len(userMessages)-1].Content
	}

	return storage.RequestMetrics{
		StartTime:               startTime,
		ModelID:                 req.Model.ID,
		ModelName:               req.Model.Name,
		Provider:                req.Model.Provider.String(),
		MessageCount:            len(req.Messages),
		UserMessageLength:       len(lastUserMessage),
		TotalConversationLength: totalLength,
		HasSystemMessage:        systemMessage != "",
		Temperature:             req.Temperature,
		MaxTokens:               req.MaxTokens,
		IsStream:                req.Stream,
	}
}

// Utility functions for HTTP requests

// MakeHTTPRequest creates and executes an HTTP request (exported for provider use)
func MakeHTTPRequest(ctx context.Context, client *http.Client, method, url string, headers map[string]string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// ParseSSEStream parses Server-Sent Events from a response body (exported for provider use)
func ParseSSEStream(ctx context.Context, body io.ReadCloser, parseFunc func([]byte) (string, bool, error)) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errorChan)
		defer body.Close()

		scanner := bufio.NewScanner(body)
		var buffer strings.Builder

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errorChan <- ctx.Err()
				return
			default:
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			if line == "data: [DONE]" {
				chunkChan <- StreamChunk{Content: "", Done: true}
				return
			}

			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if content, done, err := parseFunc([]byte(data)); err != nil {
					errorChan <- err
					return
				} else if content != "" {
					buffer.WriteString(content)
					chunkChan <- StreamChunk{Content: content, Done: done}
					if done {
						return
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errorChan <- fmt.Errorf("stream scanning error: %w", err)
		}
	}()

	return chunkChan, errorChan
}

// ParseErrorResponse extracts error information from an HTTP response (exported for provider use)
func ParseErrorResponse(resp *http.Response, provider string) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Failed to read error response: %v", err),
			Provider:   provider,
			Retryable:  isRetryableStatusCode(resp.StatusCode),
		}
	}

	// Try to parse JSON error
	var errorData map[string]interface{}
	if json.Unmarshal(body, &errorData) == nil {
		message := extractErrorMessage(errorData)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    message,
			Provider:   provider,
			Retryable:  isRetryableStatusCode(resp.StatusCode),
		}
	}

	// Fallback to plain text
	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    string(body),
		Provider:   provider,
		Retryable:  isRetryableStatusCode(resp.StatusCode),
	}
}

// extractErrorMessage extracts error message from various API error formats
func extractErrorMessage(errorData map[string]interface{}) string {
	// Try common error message fields
	if msg, ok := errorData["message"].(string); ok && msg != "" {
		return msg
	}
	if err, ok := errorData["error"].(string); ok && err != "" {
		return err
	}
	if details, ok := errorData["error"].(map[string]interface{}); ok {
		if msg, ok := details["message"].(string); ok && msg != "" {
			return msg
		}
	}
	return "Unknown error"
}

// isRetryableStatusCode checks if an HTTP status code is retryable
func isRetryableStatusCode(statusCode int) bool {
	retryableCodes := []int{429, 500, 502, 503, 504}
	for _, code := range retryableCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}
