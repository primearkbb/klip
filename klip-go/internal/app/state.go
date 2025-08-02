package app

import (
	"strings"
	"time"

	"github.com/john/klip/internal/api"
	"github.com/john/klip/internal/storage"
)

// AppState represents the different states the application can be in
type AppState int

const (
	StateInitializing AppState = iota
	StateOnboarding
	StateChat
	StateModels
	StateSettings
	StateHistory
	StateHelp
	StateError
	StateShutdown
)

// String returns the string representation of AppState
func (s AppState) String() string {
	switch s {
	case StateInitializing:
		return "initializing"
	case StateOnboarding:
		return "onboarding"
	case StateChat:
		return "chat"
	case StateModels:
		return "models"
	case StateSettings:
		return "settings"
	case StateHistory:
		return "history"
	case StateHelp:
		return "help"
	case StateError:
		return "error"
	case StateShutdown:
		return "shutdown"
	default:
		return "unknown"
	}
}

// InitializationStep represents the different initialization steps
type InitializationStep int

const (
	StepStarting InitializationStep = iota
	StepStorage
	StepKeystore
	StepConfig
	StepAnalytics
	StepAPIClient
	StepComplete
)

// String returns the string representation of InitializationStep
func (s InitializationStep) String() string {
	switch s {
	case StepStarting:
		return "Starting..."
	case StepStorage:
		return "Initializing storage..."
	case StepKeystore:
		return "Loading keystore..."
	case StepConfig:
		return "Loading configuration..."
	case StepAnalytics:
		return "Setting up analytics..."
	case StepAPIClient:
		return "Initializing API client..."
	case StepComplete:
		return "Ready!"
	default:
		return "Unknown step"
	}
}

// LoadingState tracks the progress of operations
type LoadingState struct {
	IsLoading   bool
	Message     string
	Progress    float64 // 0.0 to 1.0
	StartTime   time.Time
	CurrentStep InitializationStep
	Error       error
}

// NewLoadingState creates a new loading state
func NewLoadingState(message string) *LoadingState {
	return &LoadingState{
		IsLoading: true,
		Message:   message,
		Progress:  0.0,
		StartTime: time.Now(),
	}
}

// SetProgress updates the loading progress
func (ls *LoadingState) SetProgress(progress float64, message string) {
	ls.Progress = progress
	ls.Message = message
}

// SetError sets an error state
func (ls *LoadingState) SetError(err error) {
	ls.Error = err
	ls.IsLoading = false
}

// Complete marks the loading as complete
func (ls *LoadingState) Complete() {
	ls.IsLoading = false
	ls.Progress = 1.0
}

// ChatState manages chat-specific state
type ChatState struct {
	Messages         []api.Message
	CurrentInput     string
	IsStreaming      bool
	StreamBuffer     string
	LastResponse     *api.ChatResponse
	InputMode        InputMode
	ShowingHistory   bool
	HistoryIndex     int
	WaitingForAPI    bool
	InterruptChannel chan struct{}
}

// InputMode represents different input modes
type InputMode int

const (
	InputModeNormal InputMode = iota
	InputModeCommand
	InputModeSearch
	InputModeMultiline
)

// NewChatState creates a new chat state
func NewChatState() *ChatState {
	return &ChatState{
		Messages:         make([]api.Message, 0),
		InputMode:        InputModeNormal,
		InterruptChannel: make(chan struct{}, 1),
	}
}

// AddMessage adds a message to the chat history
func (cs *ChatState) AddMessage(message api.Message) {
	cs.Messages = append(cs.Messages, message)
}

// ClearMessages clears all messages
func (cs *ChatState) ClearMessages() {
	cs.Messages = make([]api.Message, 0)
}

// GetLastUserMessage returns the last user message
func (cs *ChatState) GetLastUserMessage() *api.Message {
	for i := len(cs.Messages) - 1; i >= 0; i-- {
		if cs.Messages[i].Role == "user" {
			return &cs.Messages[i]
		}
	}
	return nil
}

// ModelsState manages model selection state
type ModelsState struct {
	AvailableModels []api.Model
	CurrentModel    api.Model
	SelectedIndex   int
	Loading         bool
	Error           error
	SearchQuery     string
	FilteredModels  []api.Model
}

// NewModelsState creates a new models state
func NewModelsState() *ModelsState {
	return &ModelsState{
		AvailableModels: make([]api.Model, 0),
		FilteredModels:  make([]api.Model, 0),
	}
}

// FilterModels filters models based on search query
func (ms *ModelsState) FilterModels(query string) {
	oldQuery := ms.SearchQuery
	ms.SearchQuery = query
	if query == "" {
		ms.FilteredModels = ms.AvailableModels
		// Don't reset index when clearing filter
		if ms.SelectedIndex >= len(ms.FilteredModels) {
			ms.SelectedIndex = 0
		}
		return
	}

	filtered := make([]api.Model, 0)
	query = strings.ToLower(query)

	for _, model := range ms.AvailableModels {
		if strings.Contains(strings.ToLower(model.Name), query) ||
			strings.Contains(strings.ToLower(model.ID), query) ||
			strings.Contains(strings.ToLower(model.Provider.String()), query) {
			filtered = append(filtered, model)
		}
	}

	ms.FilteredModels = filtered
	// Reset index when filter query changes
	if query != oldQuery {
		ms.SelectedIndex = 0
	} else if ms.SelectedIndex >= len(filtered) {
		ms.SelectedIndex = 0
	}
}

// GetSelectedModel returns the currently selected model
func (ms *ModelsState) GetSelectedModel() *api.Model {
	if len(ms.FilteredModels) == 0 || ms.SelectedIndex < 0 || ms.SelectedIndex >= len(ms.FilteredModels) {
		return nil
	}
	return &ms.FilteredModels[ms.SelectedIndex]
}

// SettingsState manages application settings
type SettingsState struct {
	Config          *storage.Config
	UnsavedChanges  bool
	SelectedSection int
	SelectedItem    int
	EditingValue    bool
	TempValue       string
}

// NewSettingsState creates a new settings state
func NewSettingsState() *SettingsState {
	return &SettingsState{}
}

// HistoryState manages chat history viewing
type HistoryState struct {
	Sessions      []storage.ChatSession
	SelectedIndex int
	Loading       bool
	Error         error
	ViewingDetail bool
	DetailSession *storage.ChatSession
}

// NewHistoryState creates a new history state
func NewHistoryState() *HistoryState {
	return &HistoryState{
		Sessions: make([]storage.ChatSession, 0),
	}
}

// HelpState manages help display
type HelpState struct {
	CurrentSection  string
	SelectedSection int
	SelectedTopic   int
	SearchQuery     string
}

// NewHelpState creates a new help state
func NewHelpState() *HelpState {
	return &HelpState{}
}

// ErrorState manages error display and recovery
type ErrorState struct {
	Error         error
	Context       string
	Recoverable   bool
	RetryAction   func() error
	PreviousState AppState
}

// NewErrorState creates a new error state
func NewErrorState(err error, context string, recoverable bool, previousState AppState) *ErrorState {
	return &ErrorState{
		Error:         err,
		Context:       context,
		Recoverable:   recoverable,
		PreviousState: previousState,
	}
}

// StateTransition represents a state change
type StateTransition struct {
	From AppState
	To   AppState
	Data interface{}
}

// StateManager manages state transitions and validation
type StateManager struct {
	current    AppState
	previous   AppState
	history    []AppState
	maxHistory int
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		current:    StateInitializing,
		previous:   StateInitializing,
		history:    make([]AppState, 0),
		maxHistory: 10,
	}
}

// Current returns the current state
func (sm *StateManager) Current() AppState {
	return sm.current
}

// Previous returns the previous state
func (sm *StateManager) Previous() AppState {
	return sm.previous
}

// CanTransition checks if a transition is valid
func (sm *StateManager) CanTransition(to AppState) bool {
	switch sm.current {
	case StateInitializing:
		return to == StateOnboarding || to == StateChat || to == StateError
	case StateOnboarding:
		return to == StateChat || to == StateError
	case StateChat:
		return to == StateModels || to == StateSettings || to == StateHistory ||
			to == StateHelp || to == StateError || to == StateShutdown
	case StateModels:
		return to == StateChat || to == StateSettings || to == StateHistory ||
			to == StateHelp || to == StateError
	case StateSettings:
		return to == StateChat || to == StateModels || to == StateHistory ||
			to == StateHelp || to == StateError
	case StateHistory:
		return to == StateChat || to == StateModels || to == StateSettings ||
			to == StateHelp || to == StateError
	case StateHelp:
		return to == StateChat || to == StateModels || to == StateSettings ||
			to == StateHistory || to == StateError
	case StateError:
		return true // Can transition to any state from error
	case StateShutdown:
		return false // Cannot transition from shutdown
	default:
		return false
	}
}

// Transition changes the current state
func (sm *StateManager) Transition(to AppState) bool {
	if !sm.CanTransition(to) {
		return false
	}

	sm.previous = sm.current
	sm.current = to

	// Add to history
	sm.history = append(sm.history, sm.previous)
	if len(sm.history) > sm.maxHistory {
		sm.history = sm.history[1:]
	}

	return true
}

// Back transitions to the previous state if possible
func (sm *StateManager) Back() bool {
	if len(sm.history) == 0 {
		return false
	}

	target := sm.history[len(sm.history)-1]
	// For Back(), we allow transitions that might normally be invalid
	// This is intentional to support navigation history
	sm.previous = sm.current
	sm.current = target
	sm.history = sm.history[:len(sm.history)-1]
	return true
}

// Reset resets the state manager to initial state
func (sm *StateManager) Reset() {
	sm.current = StateInitializing
	sm.previous = StateInitializing
	sm.history = make([]AppState, 0)
}

// StatusState manages status bar and system information
type StatusState struct {
	ConnectionState int
	CurrentModel    string
	CurrentProvider string
	TokenCount      int
	EstimatedCost   float64
	RequestCount    int
}

// NewStatusState creates a new status state
func NewStatusState() *StatusState {
	return &StatusState{
		ConnectionState: 0, // Disconnected
	}
}

// State represents the complete application state
type State struct {
	Loading  *LoadingState
	Chat     *ChatState
	Models   *ModelsState
	Settings *SettingsState
	History  *HistoryState
	Help     *HelpState
	Error    *ErrorState
	Status   *StatusState
}

// NewState creates a new complete application state
func NewState() *State {
	return &State{
		Loading:  NewLoadingState("Initializing..."),
		Chat:     NewChatState(),
		Models:   NewModelsState(),
		Settings: NewSettingsState(),
		History:  NewHistoryState(),
		Help:     NewHelpState(),
		Status:   NewStatusState(),
	}
}
