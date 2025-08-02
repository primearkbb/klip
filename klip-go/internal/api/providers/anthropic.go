package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/john/klip/internal/api"
)

// AnthropicProvider implements the ProviderInterface for Anthropic's Claude models
type AnthropicProvider struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
}

// NewAnthropicProvider creates a new Anthropic provider instance
func NewAnthropicProvider(apiKey string, httpClient *http.Client) (api.ProviderInterface, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	return &AnthropicProvider{
		apiKey:     apiKey,
		httpClient: httpClient,
		baseURL:    "https://api.anthropic.com/v1",
		headers: map[string]string{
			"Content-Type":      "application/json",
			"x-api-key":         apiKey,
			"anthropic-version": "2023-06-01",
		},
	}, nil
}

// AnthropicRequest represents the request format for Anthropic API
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []AnthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
	Tools     []AnthropicTool    `json:"tools,omitempty"`
	Metadata  *AnthropicMetadata `json:"metadata,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicTool represents a tool definition for Anthropic
type AnthropicTool struct {
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	MaxUses int    `json:"max_uses,omitempty"`
}

// AnthropicMetadata contains request metadata
type AnthropicMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// AnthropicResponse represents the response format from Anthropic API
type AnthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []AnthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason,omitempty"`
	StopSequence string             `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage     `json:"usage"`
}

// AnthropicContent represents content in Anthropic response
type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicUsage represents usage information from Anthropic
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicStreamEvent represents a streaming event from Anthropic
type AnthropicStreamEvent struct {
	Type    string                `json:"type"`
	Message *AnthropicResponse    `json:"message,omitempty"`
	Index   int                   `json:"index,omitempty"`
	Delta   *AnthropicStreamDelta `json:"delta,omitempty"`
	Usage   *AnthropicUsage       `json:"usage,omitempty"`
}

// AnthropicStreamDelta represents delta content in streaming
type AnthropicStreamDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Chat sends a non-streaming chat request to Anthropic
func (p *AnthropicProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	anthropicReq := p.buildAnthropicRequest(req, false)

	requestBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := api.MakeHTTPRequest(
		ctx,
		p.httpClient,
		"POST",
		p.baseURL+"/messages",
		p.headers,
		requestBody,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, api.ParseErrorResponse(resp, "anthropic")
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.parseAnthropicResponse(&anthropicResp), nil
}

// ChatStream sends a streaming chat request to Anthropic
func (p *AnthropicProvider) ChatStream(ctx context.Context, req *api.ChatRequest) (<-chan api.StreamChunk, <-chan error) {
	chunkChan := make(chan api.StreamChunk, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errorChan)

		anthropicReq := p.buildAnthropicRequest(req, true)

		requestBody, err := json.Marshal(anthropicReq)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		resp, err := api.MakeHTTPRequest(
			ctx,
			p.httpClient,
			"POST",
			p.baseURL+"/messages",
			p.headers,
			requestBody,
		)
		if err != nil {
			errorChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errorChan <- api.ParseErrorResponse(resp, "anthropic")
			return
		}

		// Parse the streaming response
		parseFunc := func(data []byte) (string, bool, error) {
			var event AnthropicStreamEvent
			if err := json.Unmarshal(data, &event); err != nil {
				// Skip invalid JSON
				return "", false, nil
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta != nil && event.Delta.Type == "text_delta" {
					return event.Delta.Text, false, nil
				}
			case "message_stop":
				return "", true, nil
			}

			return "", false, nil
		}

		streamChunkChan, streamErrorChan := api.ParseSSEStream(ctx, resp.Body, parseFunc)

		for {
			select {
			case chunk, ok := <-streamChunkChan:
				if !ok {
					return
				}
				chunkChan <- chunk
			case err := <-streamErrorChan:
				if err != nil {
					errorChan <- err
				}
				return
			case <-ctx.Done():
				errorChan <- ctx.Err()
				return
			}
		}
	}()

	return chunkChan, errorChan
}

// GetModels returns available Anthropic models
func (p *AnthropicProvider) GetModels(ctx context.Context) ([]api.Model, error) {
	return []api.Model{
		{
			ID:            "claude-3-5-sonnet-20241022",
			Name:          "Claude 3.5 Sonnet (v2)",
			Provider:      api.ProviderAnthropic,
			MaxTokens:     8192,
			ContextWindow: 200000,
		},
		{
			ID:            "claude-3-5-sonnet-20240620",
			Name:          "Claude 3.5 Sonnet (v1)",
			Provider:      api.ProviderAnthropic,
			MaxTokens:     8192,
			ContextWindow: 200000,
		},
		{
			ID:            "claude-3-5-haiku-20241022",
			Name:          "Claude 3.5 Haiku",
			Provider:      api.ProviderAnthropic,
			MaxTokens:     8192,
			ContextWindow: 200000,
		},
		{
			ID:            "claude-3-opus-20240229",
			Name:          "Claude 3 Opus",
			Provider:      api.ProviderAnthropic,
			MaxTokens:     4096,
			ContextWindow: 200000,
		},
		{
			ID:            "claude-3-haiku-20240307",
			Name:          "Claude 3 Haiku",
			Provider:      api.ProviderAnthropic,
			MaxTokens:     4096,
			ContextWindow: 200000,
		},
	}, nil
}

// ValidateCredentials checks if the Anthropic API key is valid
func (p *AnthropicProvider) ValidateCredentials(ctx context.Context) error {
	// Test with a minimal request
	testReq := &api.ChatRequest{
		Model: api.Model{
			ID:       "claude-3-5-haiku-20241022",
			Provider: api.ProviderAnthropic,
		},
		Messages: []api.Message{
			{
				Role:      "user",
				Content:   "Hello",
				Timestamp: time.Now(),
			},
		},
		MaxTokens: 10,
	}

	_, err := p.Chat(ctx, testReq)
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok {
			if apiErr.StatusCode == 401 || apiErr.StatusCode == 403 {
				return fmt.Errorf("invalid API key")
			}
		}
		return fmt.Errorf("credential validation failed: %w", err)
	}

	return nil
}

// buildAnthropicRequest converts a ChatRequest to Anthropic format
func (p *AnthropicProvider) buildAnthropicRequest(req *api.ChatRequest, stream bool) *AnthropicRequest {
	anthropicReq := &AnthropicRequest{
		Model:     req.Model.ID,
		MaxTokens: req.MaxTokens,
		Stream:    stream,
		Messages:  make([]AnthropicMessage, 0),
	}

	// Set default max tokens if not specified
	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = req.Model.MaxTokens
		if anthropicReq.MaxTokens == 0 {
			anthropicReq.MaxTokens = 4096
		}
	}

	// Separate system message from other messages
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			anthropicReq.System = msg.Content
		} else {
			anthropicReq.Messages = append(anthropicReq.Messages, AnthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// Add web search tool if enabled
	if req.EnableWebSearch {
		anthropicReq.Tools = []AnthropicTool{
			{
				Type:    "web_search_20250305",
				Name:    "web_search",
				MaxUses: 5,
			},
		}
	}

	return anthropicReq
}

// parseAnthropicResponse converts Anthropic response to ChatResponse
func (p *AnthropicProvider) parseAnthropicResponse(resp *AnthropicResponse) *api.ChatResponse {
	content := ""
	if len(resp.Content) > 0 && resp.Content[0].Type == "text" {
		content = resp.Content[0].Text
	}

	usage := &api.Usage{
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}

	return &api.ChatResponse{
		Content: content,
		Usage:   usage,
	}
}

// Helper function to extract error message from Anthropic error response
func extractAnthropicError(data map[string]interface{}) string {
	if errData, ok := data["error"].(map[string]interface{}); ok {
		if msg, ok := errData["message"].(string); ok {
			return msg
		}
	}
	return "Unknown Anthropic API error"
}
