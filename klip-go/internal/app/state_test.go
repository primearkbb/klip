package app

import (
	"fmt"
	"testing"

	"github.com/john/klip/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestAppStateString(t *testing.T) {
	tests := []struct {
		state    AppState
		expected string
	}{
		{StateInitializing, "initializing"},
		{StateOnboarding, "onboarding"},
		{StateChat, "chat"},
		{StateModels, "models"},
		{StateSettings, "settings"},
		{StateHistory, "history"},
		{StateHelp, "help"},
		{StateError, "error"},
		{StateShutdown, "shutdown"},
		{AppState(999), "unknown"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.state.String())
	}
}

func TestInitializationStepString(t *testing.T) {
	tests := []struct {
		step     InitializationStep
		expected string
	}{
		{StepStarting, "Starting..."},
		{StepStorage, "Initializing storage..."},
		{StepKeystore, "Loading keystore..."},
		{StepConfig, "Loading configuration..."},
		{StepAnalytics, "Setting up analytics..."},
		{StepAPIClient, "Initializing API client..."},
		{StepComplete, "Ready!"},
		{InitializationStep(999), "Unknown step"},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.step.String())
	}
}

func TestStateManager(t *testing.T) {
	sm := NewStateManager()

	// Test initial state
	assert.Equal(t, StateInitializing, sm.Current())
	assert.Equal(t, StateInitializing, sm.Previous())
	assert.Equal(t, 0, len(sm.history))

	// Test valid transition
	assert.True(t, sm.CanTransition(StateOnboarding))
	assert.True(t, sm.Transition(StateOnboarding))
	assert.Equal(t, StateOnboarding, sm.Current())
	assert.Equal(t, StateInitializing, sm.Previous())
	assert.Equal(t, 1, len(sm.history))

	// Test another valid transition
	assert.True(t, sm.Transition(StateChat))
	assert.Equal(t, StateChat, sm.Current())
	assert.Equal(t, StateOnboarding, sm.Previous())

	// Test invalid transition
	assert.True(t, sm.Transition(StateShutdown))
	assert.Equal(t, StateShutdown, sm.Current())
	assert.False(t, sm.CanTransition(StateChat))
	assert.False(t, sm.Transition(StateChat))
	assert.Equal(t, StateShutdown, sm.Current())

	// Test back functionality
	sm = NewStateManager()
	sm.Transition(StateChat)
	sm.Transition(StateModels)
	sm.Transition(StateSettings)

	assert.Equal(t, StateSettings, sm.Current())
	assert.True(t, sm.Back())
	assert.Equal(t, StateModels, sm.Current())
	assert.True(t, sm.Back())
	assert.Equal(t, StateChat, sm.Current())

	// Test reset
	sm.Reset()
	assert.Equal(t, StateInitializing, sm.Current())
	assert.Equal(t, StateInitializing, sm.Previous())
	assert.Equal(t, 0, len(sm.history))
}

func TestStateTransitionValidation(t *testing.T) {
	sm := NewStateManager()

	// From Initializing
	sm.current = StateInitializing
	assert.True(t, sm.CanTransition(StateOnboarding))
	assert.True(t, sm.CanTransition(StateChat))
	assert.True(t, sm.CanTransition(StateError))
	assert.False(t, sm.CanTransition(StateModels))
	assert.False(t, sm.CanTransition(StateSettings))

	// From Onboarding
	sm.current = StateOnboarding
	assert.True(t, sm.CanTransition(StateChat))
	assert.True(t, sm.CanTransition(StateError))
	assert.False(t, sm.CanTransition(StateInitializing))
	assert.False(t, sm.CanTransition(StateModels))

	// From Chat
	sm.current = StateChat
	assert.True(t, sm.CanTransition(StateModels))
	assert.True(t, sm.CanTransition(StateSettings))
	assert.True(t, sm.CanTransition(StateHistory))
	assert.True(t, sm.CanTransition(StateHelp))
	assert.True(t, sm.CanTransition(StateError))
	assert.True(t, sm.CanTransition(StateShutdown))
	assert.False(t, sm.CanTransition(StateInitializing))
	assert.False(t, sm.CanTransition(StateOnboarding))

	// From Error (can transition to any state)
	sm.current = StateError
	assert.True(t, sm.CanTransition(StateInitializing))
	assert.True(t, sm.CanTransition(StateOnboarding))
	assert.True(t, sm.CanTransition(StateChat))
	assert.True(t, sm.CanTransition(StateModels))
	assert.True(t, sm.CanTransition(StateSettings))
	assert.True(t, sm.CanTransition(StateHistory))
	assert.True(t, sm.CanTransition(StateHelp))
	assert.True(t, sm.CanTransition(StateShutdown))

	// From Shutdown (cannot transition to any state)
	sm.current = StateShutdown
	assert.False(t, sm.CanTransition(StateInitializing))
	assert.False(t, sm.CanTransition(StateOnboarding))
	assert.False(t, sm.CanTransition(StateChat))
	assert.False(t, sm.CanTransition(StateModels))
	assert.False(t, sm.CanTransition(StateSettings))
	assert.False(t, sm.CanTransition(StateHistory))
	assert.False(t, sm.CanTransition(StateHelp))
	assert.False(t, sm.CanTransition(StateError))
}

func TestStateHistoryLimit(t *testing.T) {
	sm := NewStateManager()
	sm.maxHistory = 3 // Set small limit for testing

	// Add more states than the limit
	sm.Transition(StateChat)
	sm.Transition(StateModels)
	sm.Transition(StateChat)
	sm.Transition(StateSettings)
	sm.Transition(StateChat)

	// History should be limited
	assert.Equal(t, 3, len(sm.history))
	
	// Should contain the most recent states
	expectedHistory := []AppState{StateModels, StateChat, StateSettings}
	assert.Equal(t, expectedHistory, sm.history)
}

func TestInputMode(t *testing.T) {
	modes := []InputMode{InputModeNormal, InputModeCommand, InputModeSearch, InputModeMultiline}
	
	// Test that all modes are distinct
	seen := make(map[InputMode]bool)
	for _, mode := range modes {
		assert.False(t, seen[mode], "Duplicate input mode found")
		seen[mode] = true
	}
}

func TestErrorState(t *testing.T) {
	err := assert.AnError
	errorState := NewErrorState(err, "test context", true, StateChat)

	assert.Equal(t, err, errorState.Error)
	assert.Equal(t, "test context", errorState.Context)
	assert.True(t, errorState.Recoverable)
	assert.Equal(t, StateChat, errorState.PreviousState)
	assert.Nil(t, errorState.RetryAction)
}

func TestChatStateOperations(t *testing.T) {
	chatState := NewChatState()

	// Test initial state
	assert.Equal(t, 0, len(chatState.Messages))
	assert.Equal(t, InputModeNormal, chatState.InputMode)
	assert.False(t, chatState.IsStreaming)
	assert.False(t, chatState.WaitingForAPI)
	assert.Equal(t, "", chatState.CurrentInput)
	assert.Equal(t, "", chatState.StreamBuffer)

	// Test message operations
	message1 := api.Message{Role: "user", Content: "Hello"}
	message2 := api.Message{Role: "assistant", Content: "Hi there"}
	message3 := api.Message{Role: "user", Content: "How are you?"}

	chatState.AddMessage(message1)
	chatState.AddMessage(message2)
	chatState.AddMessage(message3)

	assert.Equal(t, 3, len(chatState.Messages))

	// Test GetLastUserMessage
	lastUser := chatState.GetLastUserMessage()
	assert.NotNil(t, lastUser)
	assert.Equal(t, "user", lastUser.Role)
	assert.Equal(t, "How are you?", lastUser.Content)

	// Test ClearMessages
	chatState.ClearMessages()
	assert.Equal(t, 0, len(chatState.Messages))
	
	lastUser = chatState.GetLastUserMessage()
	assert.Nil(t, lastUser)
}

func TestModelsStateFiltering(t *testing.T) {
	modelsState := NewModelsState()
	
	// Add test models
	models := []api.Model{
		{ID: "gpt-4", Name: "GPT-4", Provider: api.ProviderOpenAI},
		{ID: "claude-3-sonnet", Name: "Claude 3 Sonnet", Provider: api.ProviderAnthropic},
		{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: api.ProviderOpenAI},
		{ID: "claude-3-haiku", Name: "Claude 3 Haiku", Provider: api.ProviderAnthropic},
	}
	
	modelsState.AvailableModels = models

	// Test no filter (should show all)
	modelsState.FilterModels("")
	assert.Equal(t, 4, len(modelsState.FilteredModels))

	// Test filter by name
	modelsState.FilterModels("GPT")
	assert.Equal(t, 2, len(modelsState.FilteredModels))
	
	// Test filter by ID
	modelsState.FilterModels("claude")
	assert.Equal(t, 2, len(modelsState.FilteredModels))
	
	// Test filter by provider
	modelsState.FilterModels("openai")
	assert.Equal(t, 2, len(modelsState.FilteredModels))
	
	// Test filter with no matches
	modelsState.FilterModels("nonexistent")
	assert.Equal(t, 0, len(modelsState.FilteredModels))

	// Test that selected index is reset when filtered models change
	modelsState.FilterModels("gpt")
	modelsState.SelectedIndex = 1
	modelsState.FilterModels("claude")
	assert.Equal(t, 0, modelsState.SelectedIndex)
}

func TestModelsStateSelection(t *testing.T) {
	modelsState := NewModelsState()
	
	models := []api.Model{
		{ID: "model1", Name: "Model 1"},
		{ID: "model2", Name: "Model 2"},
		{ID: "model3", Name: "Model 3"},
	}
	
	modelsState.AvailableModels = models
	modelsState.FilterModels("")

	// Test valid selection
	modelsState.SelectedIndex = 1
	selected := modelsState.GetSelectedModel()
	assert.NotNil(t, selected)
	assert.Equal(t, "model2", selected.ID)

	// Test invalid selection (negative index)
	modelsState.SelectedIndex = -1
	selected = modelsState.GetSelectedModel()
	assert.Nil(t, selected)

	// Test invalid selection (index too large)
	modelsState.SelectedIndex = 10
	selected = modelsState.GetSelectedModel()
	assert.Nil(t, selected)

	// Test empty filtered models
	modelsState.FilteredModels = []api.Model{}
	modelsState.SelectedIndex = 0
	selected = modelsState.GetSelectedModel()
	assert.Nil(t, selected)
}

func TestLoadingStateProgression(t *testing.T) {
	loadingState := NewLoadingState("Test operation")
	
	// Test initial state
	assert.True(t, loadingState.IsLoading)
	assert.Equal(t, 0.0, loadingState.Progress)
	assert.Equal(t, "Test operation", loadingState.Message)
	assert.Nil(t, loadingState.Error)

	// Test progress updates
	loadingState.SetProgress(0.25, "Quarter done")
	assert.Equal(t, 0.25, loadingState.Progress)
	assert.Equal(t, "Quarter done", loadingState.Message)
	assert.True(t, loadingState.IsLoading)

	loadingState.SetProgress(0.75, "Almost there")
	assert.Equal(t, 0.75, loadingState.Progress)
	assert.Equal(t, "Almost there", loadingState.Message)

	// Test completion
	loadingState.Complete()
	assert.False(t, loadingState.IsLoading)
	assert.Equal(t, 1.0, loadingState.Progress)
}

func TestLoadingStateError(t *testing.T) {
	loadingState := NewLoadingState("Test operation")
	
	err := assert.AnError
	loadingState.SetError(err)
	
	assert.False(t, loadingState.IsLoading)
	assert.Equal(t, err, loadingState.Error)
}

// Benchmark tests for state operations

func BenchmarkStateManagerTransition(b *testing.B) {
	sm := NewStateManager()
	states := []AppState{StateChat, StateModels, StateSettings, StateHistory, StateHelp}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := states[i%len(states)]
		sm.Transition(state)
	}
}

func BenchmarkStateManagerBack(b *testing.B) {
	sm := NewStateManager()
	
	// Build up some history
	for i := 0; i < 10; i++ {
		sm.Transition(StateChat)
		sm.Transition(StateModels)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !sm.Back() {
			// Reset when we can't go back further
			sm.Transition(StateChat)
			sm.Transition(StateModels)
		}
	}
}

func BenchmarkModelsFiltering(b *testing.B) {
	modelsState := NewModelsState()
	
	// Create many test models
	models := make([]api.Model, 1000)
	for i := 0; i < 1000; i++ {
		models[i] = api.Model{
			ID:       fmt.Sprintf("model-%d", i),
			Name:     fmt.Sprintf("Test Model %d", i),
			Provider: api.ProviderOpenAI,
		}
	}
	modelsState.AvailableModels = models
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := fmt.Sprintf("model-%d", i%100)
		modelsState.FilterModels(query)
	}
}