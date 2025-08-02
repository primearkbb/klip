package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/john/klip/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestCommandRegistry(t *testing.T) {
	registry := NewCommandRegistry()
	
	// Test that built-in commands are registered
	assert.NotNil(t, registry.Get("help"))
	assert.NotNil(t, registry.Get("h")) // alias
	assert.NotNil(t, registry.Get("?")) // alias
	
	assert.NotNil(t, registry.Get("model"))
	assert.NotNil(t, registry.Get("m")) // alias
	
	assert.NotNil(t, registry.Get("models"))
	assert.NotNil(t, registry.Get("list")) // alias
	
	assert.NotNil(t, registry.Get("clear"))
	assert.NotNil(t, registry.Get("cls")) // alias
	assert.NotNil(t, registry.Get("c"))   // alias
	
	assert.NotNil(t, registry.Get("quit"))
	assert.NotNil(t, registry.Get("exit")) // alias
	assert.NotNil(t, registry.Get("q"))    // alias
	
	// Test non-existent command
	assert.Nil(t, registry.Get("nonexistent"))
}

func TestCommandRegistration(t *testing.T) {
	registry := NewCommandRegistry()
	
	// Register a custom command
	customCmd := &Command{
		Name:        "test",
		Aliases:     []string{"t", "testing"},
		Description: "Test command",
		Usage:       "/test",
		Handler:     nil,
	}
	
	registry.Register(customCmd)
	
	// Test retrieval by name
	cmd := registry.Get("test")
	assert.NotNil(t, cmd)
	assert.Equal(t, "test", cmd.Name)
	assert.Equal(t, "Test command", cmd.Description)
	
	// Test retrieval by alias
	cmd = registry.Get("t")
	assert.NotNil(t, cmd)
	assert.Equal(t, "test", cmd.Name)
	
	cmd = registry.Get("testing")
	assert.NotNil(t, cmd)
	assert.Equal(t, "test", cmd.Name)
}

func TestCommandSuggestions(t *testing.T) {
	registry := NewCommandRegistry()
	
	// Test suggestions for "h"
	suggestions := registry.GetSuggestions("h")
	assert.Contains(t, suggestions, "/help")
	assert.Contains(t, suggestions, "/history")
	
	// Test suggestions for "mod"
	suggestions = registry.GetSuggestions("mod")
	assert.Contains(t, suggestions, "/model")
	assert.Contains(t, suggestions, "/models")
	
	// Test suggestions for full command
	suggestions = registry.GetSuggestions("help")
	assert.Contains(t, suggestions, "/help")
	
	// Test suggestions with leading slash
	suggestions = registry.GetSuggestions("/he")
	assert.Contains(t, suggestions, "/help")
	
	// Test no suggestions
	suggestions = registry.GetSuggestions("xyz")
	assert.Empty(t, suggestions)
}

func TestCommandList(t *testing.T) {
	registry := NewCommandRegistry()
	
	commands := registry.List()
	assert.Greater(t, len(commands), 10) // Should have many built-in commands
	
	// Check that some expected commands are present
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name] = true
	}
	
	expectedCommands := []string{"help", "model", "models", "clear", "quit", "settings"}
	for _, expectedCmd := range expectedCommands {
		assert.True(t, commandNames[expectedCmd], "Expected command not found: %s", expectedCmd)
	}
}

func TestCommandParsing(t *testing.T) {
	model := New()
	
	// Test valid command
	assert.True(t, model.IsValidCommand("/help"))
	assert.True(t, model.IsValidCommand("/model gpt-4"))
	assert.True(t, model.IsValidCommand("/clear"))
	
	// Test invalid commands
	assert.False(t, model.IsValidCommand("help")) // no leading slash
	assert.False(t, model.IsValidCommand("/nonexistent"))
	assert.False(t, model.IsValidCommand(""))
	assert.False(t, model.IsValidCommand("/"))
}

func TestCommandSuggestionsFromModel(t *testing.T) {
	model := New()
	
	suggestions := model.GetCommandSuggestions("he")
	assert.Contains(t, suggestions, "/help")
	
	suggestions = model.GetCommandSuggestions("/mo")
	assert.Contains(t, suggestions, "/model")
	assert.Contains(t, suggestions, "/models")
	
	suggestions = model.GetCommandSuggestions("q")
	assert.Contains(t, suggestions, "/quit")
}

func TestValidateProvider(t *testing.T) {
	model := New()
	
	// Test valid providers
	assert.True(t, model.validateProvider("anthropic"))
	assert.True(t, model.validateProvider("openai"))
	assert.True(t, model.validateProvider("openrouter"))
	
	// Test case insensitive
	assert.True(t, model.validateProvider("ANTHROPIC"))
	assert.True(t, model.validateProvider("OpenAI"))
	assert.True(t, model.validateProvider("OpenRouter"))
	
	// Test invalid providers
	assert.False(t, model.validateProvider("invalid"))
	assert.False(t, model.validateProvider(""))
	assert.False(t, model.validateProvider("google"))
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 kB"},
		{1536, "1.5 kB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}
	
	for _, test := range tests {
		result := formatFileSize(test.bytes)
		assert.Equal(t, test.expected, result, "Failed for %d bytes", test.bytes)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{100 * time.Millisecond, "100ms"},
		{500 * time.Millisecond, "500ms"},
		{1500 * time.Millisecond, "1.5s"},
		{30 * time.Second, "30.0s"},
		{90 * time.Second, "1.5m"},
		{2 * time.Hour, "2.0h"},
	}
	
	for _, test := range tests {
		result := formatDuration(test.duration)
		assert.Equal(t, test.expected, result, "Failed for %v", test.duration)
	}
}

func TestParseModelID(t *testing.T) {
	model := New()
	
	// Set up test models
	testModels := []api.Model{
		{ID: "gpt-4", Name: "GPT-4", Provider: api.ProviderOpenAI},
		{ID: "claude-3", Name: "Claude 3", Provider: api.ProviderAnthropic},
		{ID: "gpt-3.5", Name: "GPT-3.5", Provider: api.ProviderOpenAI},
	}
	model.modelsState.AvailableModels = testModels
	
	// Test parsing by ID
	parsedModel, err := model.parseModelID("gpt-4")
	assert.NoError(t, err)
	assert.NotNil(t, parsedModel)
	assert.Equal(t, "gpt-4", parsedModel.ID)
	
	// Test parsing by name (case insensitive)
	parsedModel, err = model.parseModelID("claude 3")
	assert.NoError(t, err)
	assert.NotNil(t, parsedModel)
	assert.Equal(t, "claude-3", parsedModel.ID)
	
	// Test parsing by numeric index
	parsedModel, err = model.parseModelID("1")
	assert.NoError(t, err)
	assert.NotNil(t, parsedModel)
	assert.Equal(t, "gpt-4", parsedModel.ID) // First model
	
	parsedModel, err = model.parseModelID("2")
	assert.NoError(t, err)
	assert.NotNil(t, parsedModel)
	assert.Equal(t, "claude-3", parsedModel.ID) // Second model
	
	// Test invalid cases
	_, err = model.parseModelID("nonexistent")
	assert.Error(t, err)
	
	_, err = model.parseModelID("0") // Invalid index
	assert.Error(t, err)
	
	_, err = model.parseModelID("10") // Index out of range
	assert.Error(t, err)
	
	_, err = model.parseModelID("")
	assert.Error(t, err)
}

// Test command handlers (these would normally require more complex setup)

func TestHelpCommandBasic(t *testing.T) {
	model := New()
	
	// Test basic help command execution
	cmd := model.handleHelpCommand([]string{})
	assert.Nil(t, cmd) // Should transition to help state
	assert.Equal(t, StateHelp, model.GetCurrentState())
}

func TestClearCommand(t *testing.T) {
	model := New()
	
	// Add some messages first
	model.chatState.AddMessage(api.Message{Role: "user", Content: "Hello"})
	model.chatState.AddMessage(api.Message{Role: "assistant", Content: "Hi"})
	assert.Equal(t, 2, len(model.chatState.Messages))
	
	// Execute clear command
	cmd := model.handleClearCommand([]string{})
	assert.NotNil(t, cmd) // Should return status message command
	assert.Equal(t, 0, len(model.chatState.Messages))
}

func TestModelsCommand(t *testing.T) {
	model := New()
	
	cmd := model.handleModelsCommand([]string{})
	assert.NotNil(t, cmd) // Should return load models command
	assert.Equal(t, StateModels, model.GetCurrentState())
}

func TestQuitCommand(t *testing.T) {
	model := New()
	
	cmd := model.handleQuitCommand([]string{})
	assert.NotNil(t, cmd)
	// The command should be tea.Quit, but we can't easily test that without
	// more complex setup, so we'll just verify a command is returned
}

func TestDebugCommand(t *testing.T) {
	model := New()
	
	// Initially debug should be off
	assert.False(t, model.showDebugInfo)
	
	// Toggle debug on
	cmd := model.handleDebugCommand([]string{})
	assert.NotNil(t, cmd) // Should return status message
	assert.True(t, model.showDebugInfo)
	
	// Toggle debug off
	cmd = model.handleDebugCommand([]string{})
	assert.NotNil(t, cmd)
	assert.False(t, model.showDebugInfo)
}

func TestWebSearchCommand(t *testing.T) {
	model := New()
	
	// Initially web search should be enabled
	assert.True(t, model.webSearchEnabled)
	
	// Test explicit disable
	cmd := model.handleWebSearchCommand([]string{"off"})
	assert.NotNil(t, cmd)
	assert.False(t, model.webSearchEnabled)
	
	// Test explicit enable
	cmd = model.handleWebSearchCommand([]string{"on"})
	assert.NotNil(t, cmd)
	assert.True(t, model.webSearchEnabled)
	
	// Test toggle (no args)
	cmd = model.handleWebSearchCommand([]string{})
	assert.NotNil(t, cmd)
	assert.False(t, model.webSearchEnabled) // Should toggle to false
	
	// Test invalid argument
	cmd = model.handleWebSearchCommand([]string{"invalid"})
	assert.NotNil(t, cmd)
	assert.False(t, model.webSearchEnabled) // Should remain unchanged
}

// Benchmark tests

func BenchmarkCommandLookup(b *testing.B) {
	registry := NewCommandRegistry()
	commands := []string{"help", "model", "models", "clear", "quit", "settings"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := commands[i%len(commands)]
		registry.Get(cmd)
	}
}

func BenchmarkCommandSuggestions(b *testing.B) {
	registry := NewCommandRegistry()
	prefixes := []string{"h", "m", "c", "s", "q"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prefix := prefixes[i%len(prefixes)]
		registry.GetSuggestions(prefix)
	}
}

func BenchmarkModelParsing(b *testing.B) {
	model := New()
	
	// Set up test models
	testModels := make([]api.Model, 100)
	for i := 0; i < 100; i++ {
		testModels[i] = api.Model{
			ID:       fmt.Sprintf("model-%d", i),
			Name:     fmt.Sprintf("Model %d", i),
			Provider: api.ProviderOpenAI,
		}
	}
	model.modelsState.AvailableModels = testModels
	
	modelIDs := []string{"model-0", "model-50", "model-99", "Model 25", "75"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		modelID := modelIDs[i%len(modelIDs)]
		model.parseModelID(modelID)
	}
}