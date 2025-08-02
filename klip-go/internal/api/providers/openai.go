package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/john/klip/internal/api"
)

// OpenAIProvider implements the ProviderInterface for OpenAI's models
type OpenAIProvider struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
	headers    map[string]string
}

// NewOpenAIProvider creates a new OpenAI provider instance
func NewOpenAIProvider(apiKey string, httpClient *http.Client) (api.ProviderInterface, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	return &OpenAIProvider{
		apiKey:     apiKey,
		httpClient: httpClient,
		baseURL:    "https://api.openai.com/v1",
		headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + apiKey,
		},
	}, nil
}

// OpenAIRequest represents the request format for OpenAI API
type OpenAIRequest struct {
	Model        string           `json:"model"`
	Messages     []OpenAIMessage  `json:"messages"`
	MaxTokens    int              `json:"max_tokens,omitempty"`
	Temperature  float64          `json:"temperature,omitempty"`
	Stream       bool             `json:"stream,omitempty"`
	Functions    []OpenAIFunction `json:"functions,omitempty"`
	FunctionCall interface{}      `json:"function_call,omitempty"`
	Tools        []OpenAITool     `json:"tools,omitempty"`
	ToolChoice   interface{}      `json:"tool_choice,omitempty"`
}

// OpenAIMessage represents a message in OpenAI format
type OpenAIMessage struct {
	Role         string              `json:"role"`
	Content      string              `json:"content,omitempty"`
	Name         string              `json:"name,omitempty"`
	FunctionCall *OpenAIFunctionCall `json:"function_call,omitempty"`
	ToolCalls    []OpenAIToolCall    `json:"tool_calls,omitempty"`
}

// OpenAIFunction represents a function definition for function calling
type OpenAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// OpenAITool represents a tool definition for OpenAI
type OpenAITool struct {
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

// OpenAIFunctionCall represents a function call in a message
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAIToolCall represents a tool call in a message
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIResponse represents the response format from OpenAI API
type OpenAIResponse struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	Choices           []OpenAIChoice `json:"choices"`
	Usage             OpenAIUsage    `json:"usage"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
}

// OpenAIChoice represents a choice in OpenAI response
type OpenAIChoice struct {
	Index        int            `json:"index"`
	Message      OpenAIMessage  `json:"message"`
	Delta        *OpenAIMessage `json:"delta,omitempty"`
	FinishReason string         `json:"finish_reason"`
}

// OpenAIUsage represents usage information from OpenAI
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIStreamEvent represents a streaming event from OpenAI
type OpenAIStreamEvent struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	Choices           []OpenAIChoice `json:"choices"`
	Usage             *OpenAIUsage   `json:"usage,omitempty"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
}

// Chat sends a non-streaming chat request to OpenAI
func (p *OpenAIProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	openaiReq := p.buildOpenAIRequest(req, false)

	requestBody, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := api.MakeHTTPRequest(
		ctx,
		p.httpClient,
		"POST",
		p.baseURL+"/chat/completions",
		p.headers,
		requestBody,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, api.ParseErrorResponse(resp, "openai")
	}

	var openaiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.parseOpenAIResponse(&openaiResp), nil
}

// ChatStream sends a streaming chat request to OpenAI
func (p *OpenAIProvider) ChatStream(ctx context.Context, req *api.ChatRequest) (<-chan api.StreamChunk, <-chan error) {
	chunkChan := make(chan api.StreamChunk, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errorChan)

		openaiReq := p.buildOpenAIRequest(req, true)

		requestBody, err := json.Marshal(openaiReq)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		resp, err := api.MakeHTTPRequest(
			ctx,
			p.httpClient,
			"POST",
			p.baseURL+"/chat/completions",
			p.headers,
			requestBody,
		)
		if err != nil {
			errorChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errorChan <- api.ParseErrorResponse(resp, "openai")
			return
		}

		// Parse the streaming response
		parseFunc := func(data []byte) (string, bool, error) {
			var event OpenAIStreamEvent
			if err := json.Unmarshal(data, &event); err != nil {
				// Skip invalid JSON
				return "", false, nil
			}

			if len(event.Choices) > 0 {
				choice := event.Choices[0]
				if choice.Delta != nil && choice.Delta.Content != "" {
					return choice.Delta.Content, false, nil
				}
				if choice.FinishReason != "" {
					return "", true, nil
				}
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

// GetModels returns available OpenAI models
func (p *OpenAIProvider) GetModels(ctx context.Context) ([]api.Model, error) {
	return []api.Model{
		{
			ID:            "gpt-4o",
			Name:          "GPT-4o",
			Provider:      api.ProviderOpenAI,
			MaxTokens:     16384,
			ContextWindow: 128000,
		},
		{
			ID:            "gpt-4o-mini",
			Name:          "GPT-4o Mini",
			Provider:      api.ProviderOpenAI,
			MaxTokens:     16384,
			ContextWindow: 128000,
		},
		{
			ID:            "gpt-4-turbo",
			Name:          "GPT-4 Turbo",
			Provider:      api.ProviderOpenAI,
			MaxTokens:     4096,
			ContextWindow: 128000,
		},
		{
			ID:            "gpt-4",
			Name:          "GPT-4",
			Provider:      api.ProviderOpenAI,
			MaxTokens:     8192,
			ContextWindow: 8192,
		},
		{
			ID:            "gpt-3.5-turbo",
			Name:          "GPT-3.5 Turbo",
			Provider:      api.ProviderOpenAI,
			MaxTokens:     4096,
			ContextWindow: 16384,
		},
		{
			ID:            "o1-preview",
			Name:          "o1 Preview",
			Provider:      api.ProviderOpenAI,
			MaxTokens:     32768,
			ContextWindow: 128000,
		},
		{
			ID:            "o1-mini",
			Name:          "o1 Mini",
			Provider:      api.ProviderOpenAI,
			MaxTokens:     65536,
			ContextWindow: 128000,
		},
	}, nil
}

// ValidateCredentials checks if the OpenAI API key is valid
func (p *OpenAIProvider) ValidateCredentials(ctx context.Context) error {
	// Test with a minimal request
	testReq := &api.ChatRequest{
		Model: api.Model{
			ID:       "gpt-3.5-turbo",
			Provider: api.ProviderOpenAI,
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

// buildOpenAIRequest converts a ChatRequest to OpenAI format
func (p *OpenAIProvider) buildOpenAIRequest(req *api.ChatRequest, stream bool) *OpenAIRequest {
	openaiReq := &OpenAIRequest{
		Model:       req.Model.ID,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      stream,
		Messages:    make([]OpenAIMessage, 0),
	}

	// Set default values
	if openaiReq.MaxTokens == 0 {
		openaiReq.MaxTokens = req.Model.MaxTokens
		if openaiReq.MaxTokens == 0 {
			openaiReq.MaxTokens = 4096
		}
	}
	if openaiReq.Temperature == 0 {
		openaiReq.Temperature = 0.7
	}

	// Convert messages
	for _, msg := range req.Messages {
		openaiReq.Messages = append(openaiReq.Messages, OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return openaiReq
}

// parseOpenAIResponse converts OpenAI response to ChatResponse
func (p *OpenAIProvider) parseOpenAIResponse(resp *OpenAIResponse) *api.ChatResponse {
	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	usage := &api.Usage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
	}

	return &api.ChatResponse{
		Content: content,
		Usage:   usage,
	}
}

// Helper function to extract error message from OpenAI error response
func extractOpenAIError(data map[string]interface{}) string {
	if errData, ok := data["error"].(map[string]interface{}); ok {
		if msg, ok := errData["message"].(string); ok {
			return msg
		}
	}
	return "Unknown OpenAI API error"
}
