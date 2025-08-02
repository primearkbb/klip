package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/john/klip/internal/api"
	"github.com/john/klip/internal/storage"
)

// Model represents the main application state
type Model struct {
	// Core application state
	stateManager *StateManager

	// Window dimensions and layout
	width  int
	height int
	ready  bool

	// Dependencies
	storage    *storage.Storage
	apiClient  api.ProviderInterface
	logger     *log.Logger
	ctx        context.Context
	cancelFunc context.CancelFunc

	// Current model and configuration
	currentModel api.Model
	config       *storage.Config

	// State-specific data
	loadingState  *LoadingState
	chatState     *ChatState
	modelsState   *ModelsState
	settingsState *SettingsState
	historyState  *HistoryState
	helpState     *HelpState
	errorState    *ErrorState

	// Input handling
	inputBuffer  string
	cursorPos    int
	inputHistory []string
	historyIndex int

	// UI state
	statusMessage  string
	statusTimeout  time.Time
	showDebugInfo  bool
	animationFrame int
	lastUpdate     time.Time

	// Feature flags
	webSearchEnabled bool
	analyticsEnabled bool
}

// New creates a new application model
func New() *Model {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize logger
	logger := log.New(os.Stderr)
	logger.SetLevel(log.InfoLevel)

	m := &Model{
		stateManager:     NewStateManager(),
		ctx:              ctx,
		cancelFunc:       cancel,
		logger:           logger,
		ready:            false,
		loadingState:     NewLoadingState("Initializing Klip..."),
		chatState:        NewChatState(),
		modelsState:      NewModelsState(),
		settingsState:    NewSettingsState(),
		historyState:     NewHistoryState(),
		helpState:        NewHelpState(),
		inputHistory:     make([]string, 0),
		historyIndex:     -1,
		lastUpdate:       time.Now(),
		webSearchEnabled: true,
		analyticsEnabled: true,
		currentModel: api.Model{
			ID:            "claude-3-5-sonnet-20241022",
			Name:          "Claude 3.5 Sonnet",
			Provider:      api.ProviderAnthropic,
			MaxTokens:     4096,
			ContextWindow: 200000,
		},
	}

	// Start proper initialization sequence
	m.stateManager.current = StateInitializing
	m.ready = false

	return m
}

// GetCurrentState returns the current application state
func (m *Model) GetCurrentState() AppState {
	return m.stateManager.Current()
}

// GetPreviousState returns the previous application state
func (m *Model) GetPreviousState() AppState {
	return m.stateManager.Previous()
}

// TransitionTo transitions to a new state
func (m *Model) TransitionTo(newState AppState) bool {
	if m.stateManager.Transition(newState) {
		m.onStateTransition(m.stateManager.Previous(), newState)
		return true
	}
	return false
}

// onStateTransition handles state transition logic
func (m *Model) onStateTransition(from, to AppState) {
	m.logger.Debug("State transition", "from", from.String(), "to", to.String())

	// Clear status message on state change
	m.clearStatusMessage()

	// State-specific transition logic
	switch to {
	case StateChat:
		m.chatState.InputMode = InputModeNormal
		m.inputBuffer = ""
		m.cursorPos = 0
	case StateModels:
		if len(m.modelsState.AvailableModels) == 0 {
			m.modelsState.Loading = true
		}
	case StateSettings:
		if m.settingsState.Config == nil {
			m.settingsState.Config = m.config
		}
	case StateHistory:
		if len(m.historyState.Sessions) == 0 {
			m.historyState.Loading = true
		}
	case StateError:
		// Error state is handled separately
	case StateShutdown:
		m.cleanup()
	}
}

// cleanup performs cleanup operations
func (m *Model) cleanup() {
	if m.storage != nil {
		if err := m.storage.Shutdown(); err != nil {
			m.logger.Error("Error during storage shutdown", "error", err)
		}
	}

	if m.cancelFunc != nil {
		m.cancelFunc()
	}
}

// setStatusMessage sets a temporary status message
func (m *Model) setStatusMessage(message string, duration time.Duration) {
	m.statusMessage = message
	m.statusTimeout = time.Now().Add(duration)
}

// clearStatusMessage clears the status message
func (m *Model) clearStatusMessage() {
	m.statusMessage = ""
	m.statusTimeout = time.Time{}
}

// hasActiveStatusMessage checks if there's an active status message
func (m *Model) hasActiveStatusMessage() bool {
	return m.statusMessage != "" && time.Now().Before(m.statusTimeout)
}

// addToInputHistory adds an input to the history
func (m *Model) addToInputHistory(input string) {
	if input == "" {
		return
	}

	// Avoid duplicates
	if len(m.inputHistory) > 0 && m.inputHistory[len(m.inputHistory)-1] == input {
		return
	}

	m.inputHistory = append(m.inputHistory, input)

	// Limit history size
	const maxHistory = 100
	if len(m.inputHistory) > maxHistory {
		m.inputHistory = m.inputHistory[len(m.inputHistory)-maxHistory:]
	}

	m.historyIndex = -1
}

// navigateInputHistory navigates through input history
func (m *Model) navigateInputHistory(direction int) {
	if len(m.inputHistory) == 0 {
		return
	}

	newIndex := m.historyIndex + direction

	if newIndex < -1 {
		newIndex = -1
	} else if newIndex >= len(m.inputHistory) {
		newIndex = len(m.inputHistory) - 1
	}

	m.historyIndex = newIndex

	if m.historyIndex == -1 {
		m.inputBuffer = ""
	} else {
		m.inputBuffer = m.inputHistory[len(m.inputHistory)-1-m.historyIndex]
	}

	m.cursorPos = len(m.inputBuffer)
}

// getCurrentInput returns the current input string
func (m *Model) getCurrentInput() string {
	return m.inputBuffer
}

// setCurrentInput sets the current input string
func (m *Model) setCurrentInput(input string) {
	m.inputBuffer = input
	m.cursorPos = len(input)
	m.historyIndex = -1
}

// insertAtCursor inserts text at the cursor position
func (m *Model) insertAtCursor(text string) {
	if m.cursorPos > len(m.inputBuffer) {
		m.cursorPos = len(m.inputBuffer)
	}

	before := m.inputBuffer[:m.cursorPos]
	after := m.inputBuffer[m.cursorPos:]
	m.inputBuffer = before + text + after
	m.cursorPos += len(text)
}

// deleteAtCursor deletes character at cursor position
func (m *Model) deleteAtCursor(direction int) {
	if direction < 0 && m.cursorPos > 0 {
		// Backspace
		before := m.inputBuffer[:m.cursorPos-1]
		after := m.inputBuffer[m.cursorPos:]
		m.inputBuffer = before + after
		m.cursorPos--
	} else if direction > 0 && m.cursorPos < len(m.inputBuffer) {
		// Delete
		before := m.inputBuffer[:m.cursorPos]
		after := m.inputBuffer[m.cursorPos+1:]
		m.inputBuffer = before + after
	}
}

// moveCursor moves the cursor position
func (m *Model) moveCursor(direction int) {
	newPos := m.cursorPos + direction
	if newPos < 0 {
		newPos = 0
	} else if newPos > len(m.inputBuffer) {
		newPos = len(m.inputBuffer)
	}
	m.cursorPos = newPos
}

// isCommand checks if the current input is a command
func (m *Model) isCommand() bool {
	return strings.HasPrefix(m.inputBuffer, "/")
}

// getInputMode returns the current input mode based on context
func (m *Model) getInputMode() InputMode {
	if m.GetCurrentState() != StateChat {
		return InputModeNormal
	}

	if m.isCommand() {
		return InputModeCommand
	}

	return m.chatState.InputMode
}

// setError sets an error state
func (m *Model) setError(err error, context string, recoverable bool) {
	m.errorState = NewErrorState(err, context, recoverable, m.GetCurrentState())
	m.TransitionTo(StateError)
}

// recoverFromError attempts to recover from an error state
func (m *Model) recoverFromError() bool {
	if m.errorState == nil || !m.errorState.Recoverable {
		return false
	}

	// Try to recover
	if m.errorState.RetryAction != nil {
		if err := m.errorState.RetryAction(); err != nil {
			m.setError(err, "Recovery failed", false)
			return false
		}
	}

	// Return to previous state
	previousState := m.errorState.PreviousState
	m.errorState = nil
	return m.TransitionTo(previousState)
}

// updateAnimationFrame updates the animation frame counter
func (m *Model) updateAnimationFrame() {
	m.animationFrame++
	if m.animationFrame > 1000 {
		m.animationFrame = 0
	}
}

// shouldAnimate determines if animations should be active
func (m *Model) shouldAnimate() bool {
	return m.loadingState != nil && m.loadingState.IsLoading
}

// formatElapsedTime formats elapsed time for display
func (m *Model) formatElapsedTime(start time.Time) string {
	elapsed := time.Since(start)
	if elapsed < time.Second {
		return fmt.Sprintf("%.0fms", float64(elapsed.Nanoseconds())/1e6)
	} else if elapsed < time.Minute {
		return fmt.Sprintf("%.1fs", elapsed.Seconds())
	} else {
		return fmt.Sprintf("%.1fm", elapsed.Minutes())
	}
}

// Init initializes the model and starts the initialization sequence
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.initializeApp(),
		tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			return tickMsg{t}
		}),
		// Add a fallback timeout in case initialization hangs
		tea.Tick(15*time.Second, func(t time.Time) tea.Msg {
			return statusMsg{"Initialization timeout, continuing with limited functionality", 5 * time.Second}
		}),
	)
}

// tickMsg represents a tick message for animations
type tickMsg struct {
	time time.Time
}
