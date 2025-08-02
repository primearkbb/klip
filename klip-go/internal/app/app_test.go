package app

import (
	"fmt"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/john/klip/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	model := New()

	assert.NotNil(t, model)
	assert.NotNil(t, model.stateManager)
	assert.NotNil(t, model.ctx)
	assert.NotNil(t, model.cancelFunc)
	assert.NotNil(t, model.loadingState)
	assert.NotNil(t, model.chatState)
	assert.NotNil(t, model.modelsState)
	assert.NotNil(t, model.settingsState)
	assert.NotNil(t, model.historyState)
	assert.NotNil(t, model.helpState)
	assert.Equal(t, StateInitializing, model.GetCurrentState())
	assert.False(t, model.ready)
	assert.True(t, model.webSearchEnabled)
	assert.True(t, model.analyticsEnabled)
}

func TestStateTransitions(t *testing.T) {
	model := New()
	// Initialize logger to avoid nil pointer
	model.logger = log.New(os.Stderr)

	// Test valid transitions
	assert.True(t, model.TransitionTo(StateOnboarding))
	assert.Equal(t, StateOnboarding, model.GetCurrentState())
	assert.Equal(t, StateInitializing, model.GetPreviousState())

	assert.True(t, model.TransitionTo(StateChat))
	assert.Equal(t, StateChat, model.GetCurrentState())
	assert.Equal(t, StateOnboarding, model.GetPreviousState())

	assert.True(t, model.TransitionTo(StateModels))
	assert.Equal(t, StateModels, model.GetCurrentState())

	assert.True(t, model.TransitionTo(StateChat))
	assert.Equal(t, StateChat, model.GetCurrentState())

	// Test invalid transitions
	assert.True(t, model.TransitionTo(StateShutdown))
	assert.Equal(t, StateShutdown, model.GetCurrentState())

	// Cannot transition from shutdown
	assert.False(t, model.TransitionTo(StateChat))
	assert.Equal(t, StateShutdown, model.GetCurrentState())
}

func TestInputHandling(t *testing.T) {
	model := New()

	// Test input buffer operations
	model.setCurrentInput("hello world")
	assert.Equal(t, "hello world", model.getCurrentInput())
	assert.Equal(t, 11, model.cursorPos)

	// Test cursor movement
	model.moveCursor(-5)
	assert.Equal(t, 6, model.cursorPos)

	model.moveCursor(20) // Should clamp to end
	assert.Equal(t, 11, model.cursorPos)

	model.moveCursor(-20) // Should clamp to start
	assert.Equal(t, 0, model.cursorPos)

	// Test insertion
	model.insertAtCursor("Hi ")
	assert.Equal(t, "Hi hello world", model.getCurrentInput())
	assert.Equal(t, 3, model.cursorPos)

	// Test deletion
	model.deleteAtCursor(-1) // Backspace
	assert.Equal(t, "Hihello world", model.getCurrentInput())
	assert.Equal(t, 2, model.cursorPos)

	model.deleteAtCursor(1) // Delete
	assert.Equal(t, "Hiello world", model.getCurrentInput())
	assert.Equal(t, 2, model.cursorPos)
}

func TestInputHistory(t *testing.T) {
	model := New()

	// Add items to history
	model.addToInputHistory("first command")
	model.addToInputHistory("second command")
	model.addToInputHistory("third command")

	assert.Equal(t, 3, len(model.inputHistory))
	assert.Equal(t, -1, model.historyIndex)

	// Navigate up in history
	model.navigateInputHistory(1)
	assert.Equal(t, "third command", model.getCurrentInput())
	assert.Equal(t, 0, model.historyIndex)

	model.navigateInputHistory(1)
	assert.Equal(t, "second command", model.getCurrentInput())
	assert.Equal(t, 1, model.historyIndex)

	model.navigateInputHistory(1)
	assert.Equal(t, "first command", model.getCurrentInput())
	assert.Equal(t, 2, model.historyIndex)

	// Can't go further up
	model.navigateInputHistory(1)
	assert.Equal(t, "first command", model.getCurrentInput())
	assert.Equal(t, 2, model.historyIndex)

	// Navigate down
	model.navigateInputHistory(-1)
	assert.Equal(t, "second command", model.getCurrentInput())
	assert.Equal(t, 1, model.historyIndex)

	model.navigateInputHistory(-1)
	assert.Equal(t, "third command", model.getCurrentInput())
	assert.Equal(t, 0, model.historyIndex)

	model.navigateInputHistory(-1)
	assert.Equal(t, "", model.getCurrentInput())
	assert.Equal(t, -1, model.historyIndex)

	// Avoid duplicates
	model.addToInputHistory("third command") // Should not be added again
	assert.Equal(t, 3, len(model.inputHistory))
}

func TestCommandDetection(t *testing.T) {
	model := New()

	model.setCurrentInput("/help")
	assert.True(t, model.isCommand())

	model.setCurrentInput("hello world")
	assert.False(t, model.isCommand())

	model.setCurrentInput("/")
	assert.True(t, model.isCommand())

	model.setCurrentInput("")
	assert.False(t, model.isCommand())
}

func TestStatusMessages(t *testing.T) {
	model := New()

	// Set status message
	model.setStatusMessage("Test message", 2*time.Second)
	assert.True(t, model.hasActiveStatusMessage())
	assert.Equal(t, "Test message", model.statusMessage)

	// Clear status message
	model.clearStatusMessage()
	assert.False(t, model.hasActiveStatusMessage())
	assert.Equal(t, "", model.statusMessage)

	// Test expiration
	model.setStatusMessage("Expiring message", 1*time.Millisecond)
	assert.True(t, model.hasActiveStatusMessage())

	time.Sleep(2 * time.Millisecond)
	assert.False(t, model.hasActiveStatusMessage())
}

func TestChatState(t *testing.T) {
	chatState := NewChatState()

	assert.NotNil(t, chatState)
	assert.Equal(t, 0, len(chatState.Messages))
	assert.Equal(t, InputModeNormal, chatState.InputMode)
	assert.False(t, chatState.IsStreaming)
	assert.False(t, chatState.WaitingForAPI)

	// Add messages
	userMsg := api.Message{
		Role:      "user",
		Content:   "Hello",
		Timestamp: time.Now(),
	}
	chatState.AddMessage(userMsg)

	assistantMsg := api.Message{
		Role:      "assistant",
		Content:   "Hi there!",
		Timestamp: time.Now(),
	}
	chatState.AddMessage(assistantMsg)

	assert.Equal(t, 2, len(chatState.Messages))

	// Get last user message
	lastUser := chatState.GetLastUserMessage()
	assert.NotNil(t, lastUser)
	assert.Equal(t, "user", lastUser.Role)
	assert.Equal(t, "Hello", lastUser.Content)

	// Clear messages
	chatState.ClearMessages()
	assert.Equal(t, 0, len(chatState.Messages))

	// Should return nil when no user messages
	lastUser = chatState.GetLastUserMessage()
	assert.Nil(t, lastUser)
}

func TestModelsState(t *testing.T) {
	modelsState := NewModelsState()

	assert.NotNil(t, modelsState)
	assert.Equal(t, 0, len(modelsState.AvailableModels))
	assert.Equal(t, 0, len(modelsState.FilteredModels))
	assert.Equal(t, 0, modelsState.SelectedIndex)
	assert.Equal(t, "", modelsState.SearchQuery)

	// Add test models
	models := []api.Model{
		{ID: "gpt-4", Name: "GPT-4", Provider: api.ProviderOpenAI},
		{ID: "claude-3", Name: "Claude 3", Provider: api.ProviderAnthropic},
		{ID: "gpt-3.5", Name: "GPT-3.5", Provider: api.ProviderOpenAI},
	}

	modelsState.AvailableModels = models
	modelsState.FilterModels("")

	assert.Equal(t, 3, len(modelsState.FilteredModels))

	// Test filtering
	modelsState.FilterModels("gpt")
	assert.Equal(t, 2, len(modelsState.FilteredModels))
	assert.Equal(t, "gpt-4", modelsState.FilteredModels[0].ID)
	assert.Equal(t, "gpt-3.5", modelsState.FilteredModels[1].ID)

	modelsState.FilterModels("claude")
	assert.Equal(t, 1, len(modelsState.FilteredModels))
	assert.Equal(t, "claude-3", modelsState.FilteredModels[0].ID)

	modelsState.FilterModels("nonexistent")
	assert.Equal(t, 0, len(modelsState.FilteredModels))

	// Test selection
	modelsState.FilterModels("gpt")
	modelsState.SelectedIndex = 1

	selected := modelsState.GetSelectedModel()
	assert.NotNil(t, selected)
	assert.Equal(t, "gpt-3.5", selected.ID)

	// Test out of bounds
	modelsState.SelectedIndex = 10
	selected = modelsState.GetSelectedModel()
	assert.Nil(t, selected)
}

func TestWindowSizeUpdate(t *testing.T) {
	model := New()

	// Test window size message
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	_, cmd := model.Update(msg)

	assert.Equal(t, 80, model.width)
	assert.Equal(t, 24, model.height)
	assert.True(t, model.ready)
	assert.Nil(t, cmd)
}

func TestKeyMessageHandling(t *testing.T) {
	model := New()
	model.TransitionTo(StateChat)
	model.ready = true

	// Test regular character input
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	_, cmd := model.Update(keyMsg)

	assert.Equal(t, "h", model.getCurrentInput())
	assert.Nil(t, cmd)

	// Test backspace
	keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
	_, cmd = model.Update(keyMsg)

	assert.Equal(t, "", model.getCurrentInput())
	assert.Nil(t, cmd)

	// Test Ctrl+C
	keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd = model.Update(keyMsg)

	// Check if cmd is the quit command
	assert.NotNil(t, cmd)
	// We can't directly compare function pointers, so let's verify it's not nil
	// In a real scenario, you'd run the command to verify it quits
}

func TestAnimationFrame(t *testing.T) {
	model := New()

	initialFrame := model.animationFrame
	model.updateAnimationFrame()

	assert.Equal(t, initialFrame+1, model.animationFrame)

	// Test overflow
	model.animationFrame = 1000
	model.updateAnimationFrame()
	assert.Equal(t, 0, model.animationFrame)
}

func TestLoadingState(t *testing.T) {
	loadingState := NewLoadingState("Test operation")

	assert.NotNil(t, loadingState)
	assert.True(t, loadingState.IsLoading)
	assert.Equal(t, "Test operation", loadingState.Message)
	assert.Equal(t, 0.0, loadingState.Progress)
	assert.Nil(t, loadingState.Error)

	// Test progress update
	loadingState.SetProgress(0.5, "Half done")
	assert.Equal(t, 0.5, loadingState.Progress)
	assert.Equal(t, "Half done", loadingState.Message)
	assert.True(t, loadingState.IsLoading)

	// Test error
	err := assert.AnError
	loadingState.SetError(err)
	assert.Equal(t, err, loadingState.Error)
	assert.False(t, loadingState.IsLoading)

	// Test completion
	loadingState = NewLoadingState("Another operation")
	loadingState.Complete()
	assert.False(t, loadingState.IsLoading)
	assert.Equal(t, 1.0, loadingState.Progress)
}

func TestErrorHandling(t *testing.T) {
	model := New()

	// Test setting error
	testErr := assert.AnError
	model.setError(testErr, "Test context", true)

	assert.Equal(t, StateError, model.GetCurrentState())
	assert.NotNil(t, model.errorState)
	assert.Equal(t, testErr, model.errorState.Error)
	assert.Equal(t, "Test context", model.errorState.Context)
	assert.True(t, model.errorState.Recoverable)
}

func TestInitialization(t *testing.T) {
	model := New()

	// Test Init command
	cmd := model.Init()
	assert.NotNil(t, cmd)

	// Test initialization steps
	assert.Equal(t, StateInitializing, model.GetCurrentState())
	assert.NotNil(t, model.loadingState)
	assert.True(t, model.loadingState.IsLoading)
}

func TestCleanup(t *testing.T) {
	model := New()

	// Test cleanup doesn't panic
	assert.NotPanics(t, func() {
		model.cleanup()
	})

	// Context should be cancelled
	select {
	case <-model.ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled after cleanup")
	}
}

// Benchmark tests for performance-critical operations

func BenchmarkInputInsertion(b *testing.B) {
	model := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.insertAtCursor("a")
	}
}

func BenchmarkStateTransition(b *testing.B) {
	model := New()
	states := []AppState{StateChat, StateModels, StateSettings, StateHistory, StateHelp}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := states[i%len(states)]
		model.TransitionTo(state)
	}
}

func BenchmarkMessageAddition(b *testing.B) {
	model := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := api.Message{
			Role:      "user",
			Content:   "Test message",
			Timestamp: time.Now(),
		}
		model.chatState.AddMessage(msg)
	}
}

func BenchmarkModelFiltering(b *testing.B) {
	modelsState := NewModelsState()

	// Create test models
	models := make([]api.Model, 100)
	for i := 0; i < 100; i++ {
		models[i] = api.Model{
			ID:       fmt.Sprintf("model-%d", i),
			Name:     fmt.Sprintf("Test Model %d", i),
			Provider: api.ProviderOpenAI,
		}
	}
	modelsState.AvailableModels = models

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		modelsState.FilterModels("model")
	}
}
