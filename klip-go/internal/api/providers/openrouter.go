package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/john/klip/internal/api"
)

// OpenRouterProvider implements the ProviderInterface for OpenRouter's models
type OpenRouterProvider struct {
	apiKey       string
	httpClient   *http.Client
	baseURL      string
	headers      map[string]string
	cachedModels []api.Model
	cacheExpiry  time.Time
	cacheMutex   sync.RWMutex
}

// NewOpenRouterProvider creates a new OpenRouter provider instance
func NewOpenRouterProvider(apiKey string, httpClient *http.Client) (api.ProviderInterface, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is required")
	}

	return &OpenRouterProvider{
		apiKey:     apiKey,
		httpClient: httpClient,
		baseURL:    "https://openrouter.ai/api/v1",
		headers: map[string]string{
			"Content-Type":   "application/json",
			"Authorization":  "Bearer " + apiKey,
			"HTTP-Referer":   "https://github.com/your-username/klip",
			"X-Title":        "Klip Chat",
		},
	}, nil
}

// OpenRouterRequest represents the request format for OpenRouter API (OpenAI-compatible)
type OpenRouterRequest struct {
	Model       string                `json:"model"`
	Messages    []OpenRouterMessage   `json:"messages"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	Temperature float64               `json:"temperature,omitempty"`
	Stream      bool                  `json:"stream,omitempty"`
	Tools       []OpenRouterTool      `json:"tools,omitempty"`
	ToolChoice  interface{}           `json:"tool_choice,omitempty"`
	Transforms  []string              `json:"transforms,omitempty"`
}

// OpenRouterMessage represents a message in OpenRouter format (OpenAI-compatible)
type OpenRouterMessage struct {
	Role      string                     `json:"role"`
	Content   string                     `json:"content,omitempty"`
	Name      string                     `json:"name,omitempty"`
	ToolCalls []OpenRouterToolCall       `json:"tool_calls,omitempty"`
}

// OpenRouterTool represents a tool definition for OpenRouter
type OpenRouterTool struct {
	Type     string                 `json:"type"`
	Function OpenRouterFunction     `json:"function"`
}

// OpenRouterFunction represents a function definition
type OpenRouterFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// OpenRouterToolCall represents a tool call in a message
type OpenRouterToolCall struct {
	ID       string                   `json:"id"`
	Type     string                   `json:"type"`
	Function OpenRouterFunctionCall   `json:"function"`
}

// OpenRouterFunctionCall represents a function call
type OpenRouterFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenRouterResponse represents the response format from OpenRouter API
type OpenRouterResponse struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []OpenRouterChoice    `json:"choices"`
	Usage   OpenRouterUsage       `json:"usage"`
}

// OpenRouterChoice represents a choice in OpenRouter response
type OpenRouterChoice struct {
	Index        int                  `json:"index"`
	Message      OpenRouterMessage    `json:"message"`
	Delta        *OpenRouterMessage   `json:"delta,omitempty"`
	FinishReason string               `json:"finish_reason"`
}

// OpenRouterUsage represents usage information from OpenRouter
type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenRouterStreamEvent represents a streaming event from OpenRouter
type OpenRouterStreamEvent struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"`
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []OpenRouterChoice    `json:"choices"`
	Usage   *OpenRouterUsage      `json:"usage,omitempty"`
}

// OpenRouterModel represents a model from OpenRouter's model list
type OpenRouterModel struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description,omitempty"`
	Context     int                       `json:"context_length"`
	Architecture OpenRouterArchitecture   `json:"architecture"`
	Pricing     OpenRouterPricing         `json:"pricing"`
	TopProvider OpenRouterTopProvider     `json:"top_provider"`
}

// OpenRouterArchitecture represents model architecture info
type OpenRouterArchitecture struct {
	Modality     string `json:"modality"`
	Tokenizer    string `json:"tokenizer"`
	InstructType string `json:"instruct_type,omitempty"`
}

// OpenRouterPricing represents model pricing info
type OpenRouterPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

// OpenRouterTopProvider represents top provider info
type OpenRouterTopProvider struct {
	MaxCompletionTokens int `json:"max_completion_tokens"`
}

// OpenRouterModelsResponse represents the response from /models endpoint
type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// Chat sends a non-streaming chat request to OpenRouter
func (p *OpenRouterProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	openrouterReq := p.buildOpenRouterRequest(req, false)
	
	requestBody, err := json.Marshal(openrouterReq)
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
		return nil, api.ParseErrorResponse(resp, "openrouter")
	}

	var openrouterResp OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&openrouterResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.parseOpenRouterResponse(&openrouterResp), nil
}

// ChatStream sends a streaming chat request to OpenRouter
func (p *OpenRouterProvider) ChatStream(ctx context.Context, req *api.ChatRequest) (<-chan api.StreamChunk, <-chan error) {
	chunkChan := make(chan api.StreamChunk, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errorChan)

		openrouterReq := p.buildOpenRouterRequest(req, true)
		
		requestBody, err := json.Marshal(openrouterReq)
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
			errorChan <- api.ParseErrorResponse(resp, "openrouter")
			return
		}

		// Parse the streaming response
		parseFunc := func(data []byte) (string, bool, error) {
			var event OpenRouterStreamEvent
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

// GetModels returns available OpenRouter models (fetched dynamically)
func (p *OpenRouterProvider) GetModels(ctx context.Context) ([]api.Model, error) {
	// Check cache first
	p.cacheMutex.RLock()
	if time.Now().Before(p.cacheExpiry) && len(p.cachedModels) > 0 {
		models := make([]api.Model, len(p.cachedModels))
		copy(models, p.cachedModels)
		p.cacheMutex.RUnlock()
		return models, nil
	}
	p.cacheMutex.RUnlock()

	// Fetch models from OpenRouter API
	models, err := p.fetchModelsFromAPI(ctx)
	if err != nil {
		// Return fallback models if API fetch fails
		return p.getFallbackModels(), nil
	}

	// Update cache
	p.cacheMutex.Lock()
	p.cachedModels = models
	p.cacheExpiry = time.Now().Add(5 * time.Minute) // Cache for 5 minutes
	p.cacheMutex.Unlock()

	return models, nil
}

// fetchModelsFromAPI retrieves models from OpenRouter's API
func (p *OpenRouterProvider) fetchModelsFromAPI(ctx context.Context) ([]api.Model, error) {
	resp, err := api.MakeHTTPRequest(
		ctx,
		p.httpClient,
		"GET",
		p.baseURL+"/models",
		p.headers,
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, api.ParseErrorResponse(resp, "openrouter")
	}

	var modelsResp OpenRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	// Convert OpenRouter models to our Model format
	models := make([]api.Model, 0, len(modelsResp.Data))
	for _, model := range modelsResp.Data {
		maxTokens := model.TopProvider.MaxCompletionTokens
		if maxTokens == 0 {
			maxTokens = 4096 // Default fallback
		}

		contextWindow := model.Context
		if contextWindow == 0 {
			contextWindow = 4096 // Default fallback
		}

		models = append(models, api.Model{
			ID:            model.ID,
			Name:          model.Name,
			Provider:      api.ProviderOpenRouter,
			MaxTokens:     maxTokens,
			ContextWindow: contextWindow,
		})
	}

	return models, nil
}

// getFallbackModels returns a static list of popular OpenRouter models
func (p *OpenRouterProvider) getFallbackModels() []api.Model {
	return []api.Model{
		{
			ID:            "anthropic/claude-3.5-sonnet",
			Name:          "Claude 3.5 Sonnet (OpenRouter)",
			Provider:      api.ProviderOpenRouter,
			MaxTokens:     8192,
			ContextWindow: 200000,
		},
		{
			ID:            "openai/gpt-4o",
			Name:          "GPT-4o (OpenRouter)",
			Provider:      api.ProviderOpenRouter,
			MaxTokens:     16384,
			ContextWindow: 128000,
		},
		{
			ID:            "openai/gpt-4o-mini",
			Name:          "GPT-4o Mini (OpenRouter)",
			Provider:      api.ProviderOpenRouter,
			MaxTokens:     16384,
			ContextWindow: 128000,
		},
		{
			ID:            "meta-llama/llama-3.1-405b-instruct",
			Name:          "Llama 3.1 405B (OpenRouter)",
			Provider:      api.ProviderOpenRouter,
			MaxTokens:     4096,
			ContextWindow: 131072,
		},
		{
			ID:            "google/gemini-pro-1.5",
			Name:          "Gemini Pro 1.5 (OpenRouter)",
			Provider:      api.ProviderOpenRouter,
			MaxTokens:     8192,
			ContextWindow: 2000000,
		},
		{
			ID:            "mistralai/mixtral-8x7b-instruct",
			Name:          "Mixtral 8x7B (OpenRouter)",
			Provider:      api.ProviderOpenRouter,
			MaxTokens:     4096,
			ContextWindow: 32768,
		},
	}
}

// ValidateCredentials checks if the OpenRouter API key is valid
func (p *OpenRouterProvider) ValidateCredentials(ctx context.Context) error {
	// Test by fetching models (lightweight request)
	_, err := p.fetchModelsFromAPI(ctx)
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

// buildOpenRouterRequest converts a ChatRequest to OpenRouter format
func (p *OpenRouterProvider) buildOpenRouterRequest(req *api.ChatRequest, stream bool) *OpenRouterRequest {
	openrouterReq := &OpenRouterRequest{
		Model:       req.Model.ID,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      stream,
		Messages:    make([]OpenRouterMessage, 0),
	}

	// Set default values
	if openrouterReq.MaxTokens == 0 {
		openrouterReq.MaxTokens = req.Model.MaxTokens
		if openrouterReq.MaxTokens == 0 {
			openrouterReq.MaxTokens = 4096
		}
	}
	if openrouterReq.Temperature == 0 {
		openrouterReq.Temperature = 0.7
	}

	// Convert messages
	for _, msg := range req.Messages {
		openrouterReq.Messages = append(openrouterReq.Messages, OpenRouterMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return openrouterReq
}

// parseOpenRouterResponse converts OpenRouter response to ChatResponse
func (p *OpenRouterProvider) parseOpenRouterResponse(resp *OpenRouterResponse) *api.ChatResponse {
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

// Helper function to extract error message from OpenRouter error response
func extractOpenRouterError(data map[string]interface{}) string {
	if errData, ok := data["error"].(map[string]interface{}); ok {
		if msg, ok := errData["message"].(string); ok {
			return msg
		}
	}
	return "Unknown OpenRouter API error"
}