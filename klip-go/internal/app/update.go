package app

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/john/klip/internal/api"
	"github.com/john/klip/internal/storage"
)

// Custom message types for the application
type (
	// Initialization messages
	initStartMsg     struct{}
	initStorageMsg   struct{}
	initKeystoreMsg  struct{}
	initConfigMsg    struct{}
	initAnalyticsMsg struct{}
	initAPIClientMsg struct{}
	initCompleteMsg  struct{}
	initErrorMsg     struct{ error }

	// API messages
	apiRequestMsg     struct{ request *api.ChatRequest }
	apiResponseMsg    struct{ response *api.ChatResponse }
	apiStreamChunkMsg struct{ chunk string }
	apiStreamDoneMsg  struct{}
	apiErrorMsg       struct{ error }

	// Model management messages
	modelsLoadStartMsg   struct{}
	modelsLoadSuccessMsg struct{ models []api.Model }
	modelsLoadErrorMsg   struct{ error }
	modelSwitchMsg       struct{ model api.Model }

	// Settings messages
	settingsLoadMsg   struct{}
	settingsUpdateMsg struct{ config *storage.Config }
	settingsSaveMsg   struct{}

	// History messages
	historyLoadStartMsg   struct{}
	historyLoadSuccessMsg struct{ sessions []storage.ChatSession }
	historyLoadErrorMsg   struct{ error }

	// Status and animation messages
	statusMsg struct {
		message  string
		duration time.Duration
	}
	clearStatusMsg struct{}

	// Input mode messages
	inputModeChangeMsg struct{ mode InputMode }

	// Command messages
	commandExecuteMsg struct {
		command string
		args    []string
	}
)

// Update handles all messages and updates the model state
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.lastUpdate = time.Now()
	m.updateAnimationFrame()

	var cmds []tea.Cmd

	// Handle global messages first
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tickMsg:
		if m.shouldAnimate() {
			cmds = append(cmds, tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
				return tickMsg{t}
			}))
		}

		// Clear expired status messages
		if m.hasActiveStatusMessage() && time.Now().After(m.statusTimeout) {
			cmds = append(cmds, func() tea.Msg { return clearStatusMsg{} })
		}

	case clearStatusMsg:
		m.clearStatusMessage()

	case statusMsg:
		m.setStatusMessage(msg.message, msg.duration)

	case tea.KeyMsg:
		// Handle global key bindings
		cmd := m.handleGlobalKeys(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Handle state-specific messages
	cmd := m.handleStateSpecificUpdate(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Handle initialization messages
	cmd = m.handleInitializationMessages(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Handle API messages
	cmd = m.handleAPIMessages(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleGlobalKeys handles global key bindings that work in any state
func (m *Model) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		if m.chatState.IsStreaming {
			// Interrupt streaming
			select {
			case m.chatState.InterruptChannel <- struct{}{}:
			default:
			}
			return func() tea.Msg {
				return statusMsg{"Interrupted", 2 * time.Second}
			}
		}
		return tea.Quit

	case "ctrl+d":
		if m.GetCurrentState() == StateChat && len(m.inputBuffer) == 0 {
			return tea.Quit
		}

	case "f1":
		if m.GetCurrentState() != StateHelp {
			m.TransitionTo(StateHelp)
		} else {
			m.stateManager.Back()
		}

	case "f2":
		if m.GetCurrentState() != StateModels {
			m.TransitionTo(StateModels)
		} else {
			m.stateManager.Back()
		}

	case "f3":
		if m.GetCurrentState() != StateSettings {
			m.TransitionTo(StateSettings)
		} else {
			m.stateManager.Back()
		}

	case "f4":
		if m.GetCurrentState() != StateHistory {
			m.TransitionTo(StateHistory)
		} else {
			m.stateManager.Back()
		}

	case "f12":
		m.showDebugInfo = !m.showDebugInfo

	case "esc":
		// Return to chat state from other states
		if m.GetCurrentState() != StateChat {
			m.TransitionTo(StateChat)
		}
	}

	return nil
}

// handleStateSpecificUpdate handles state-specific updates
func (m *Model) handleStateSpecificUpdate(msg tea.Msg) tea.Cmd {
	switch m.GetCurrentState() {
	case StateInitializing:
		return m.handleInitializingState(msg)
	case StateOnboarding:
		return m.handleOnboardingState(msg)
	case StateChat:
		return m.handleChatState(msg)
	case StateModels:
		return m.handleModelsState(msg)
	case StateSettings:
		return m.handleSettingsState(msg)
	case StateHistory:
		return m.handleHistoryState(msg)
	case StateHelp:
		return m.handleHelpState(msg)
	case StateError:
		return m.handleErrorState(msg)
	}

	return nil
}

// handleInitializingState handles the initialization state
func (m *Model) handleInitializingState(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case initCompleteMsg:
		// Transition to chat state after successful initialization
		m.TransitionTo(StateChat)
		return func() tea.Msg {
			return statusMsg{"Klip is ready!", 3 * time.Second}
		}
	case initErrorMsg:
		// On initialization error, still transition to chat but with a warning
		m.logger.Warn("Initialization completed with errors, but continuing")
		m.TransitionTo(StateChat)
		return func() tea.Msg {
			return statusMsg{"Klip started with limited functionality. Use /settings to configure.", 5 * time.Second}
		}
	}
	return nil
}

// handleOnboardingState handles the onboarding state
func (m *Model) handleOnboardingState(msg tea.Msg) tea.Cmd {
	// TODO: Implement onboarding logic
	return nil
}

// handleChatState handles the main chat state
func (m *Model) handleChatState(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleChatKeys(msg)
	case apiStreamChunkMsg:
		m.chatState.StreamBuffer += msg.chunk
		m.chatState.IsStreaming = true
	case apiStreamDoneMsg:
		// Finalize the streaming response
		if m.chatState.StreamBuffer != "" {
			assistantMsg := api.Message{
				Role:      "assistant",
				Content:   m.chatState.StreamBuffer,
				Timestamp: time.Now(),
			}
			m.chatState.AddMessage(assistantMsg)

			// Log the message (convert to storage format)
			if m.storage != nil && m.storage.ChatLogger != nil {
				go func() {
					storageMsg := storage.Message{
						Role:      assistantMsg.Role,
						Content:   assistantMsg.Content,
						Timestamp: assistantMsg.Timestamp,
					}
					if err := m.storage.ChatLogger.LogMessage(storageMsg); err != nil {
						m.logger.Error("Failed to log assistant message", "error", err)
					}
				}()
			}
		}
		m.chatState.IsStreaming = false
		m.chatState.StreamBuffer = ""
		m.chatState.WaitingForAPI = false
	case apiErrorMsg:
		m.chatState.IsStreaming = false
		m.chatState.WaitingForAPI = false
		m.setError(msg.error, "API request failed", true)
	}
	return nil
}

// handleChatKeys handles key input in chat state
func (m *Model) handleChatKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		if m.chatState.WaitingForAPI {
			return nil
		}

		input := strings.TrimSpace(m.inputBuffer)
		if input == "" {
			return nil
		}

		// Add to input history
		m.addToInputHistory(input)

		// Check if it's a command
		if m.isCommand() {
			return m.executeCommand(input)
		}

		// Send chat message
		return m.sendChatMessage(input)

	case "up":
		if !m.chatState.WaitingForAPI {
			m.navigateInputHistory(1)
		}

	case "down":
		if !m.chatState.WaitingForAPI {
			m.navigateInputHistory(-1)
		}

	case "left":
		m.moveCursor(-1)

	case "right":
		m.moveCursor(1)

	case "home", "ctrl+a":
		m.cursorPos = 0

	case "end", "ctrl+e":
		m.cursorPos = len(m.inputBuffer)

	case "backspace":
		m.deleteAtCursor(-1)

	case "delete":
		m.deleteAtCursor(1)

	case "ctrl+w":
		// Delete word backwards
		m.deleteWordBackward()

	case "ctrl+u":
		// Clear line
		m.inputBuffer = ""
		m.cursorPos = 0

	case "ctrl+k":
		// Clear from cursor to end
		m.inputBuffer = m.inputBuffer[:m.cursorPos]

	case "ctrl+l":
		// Clear screen (clear chat)
		return m.executeCommand("/clear")

	default:
		// Insert character
		if len(msg.Runes) > 0 && unicode.IsPrint(msg.Runes[0]) {
			m.insertAtCursor(string(msg.Runes))
		}
	}

	return nil
}

// handleModelsState handles the models selection state
func (m *Model) handleModelsState(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleModelsKeys(msg)
	case modelsLoadSuccessMsg:
		m.modelsState.AvailableModels = msg.models
		m.modelsState.FilterModels("") // Initialize filtered models
		m.modelsState.Loading = false
		return func() tea.Msg {
			return statusMsg{fmt.Sprintf("Loaded %d models", len(msg.models)), 2 * time.Second}
		}
	case modelsLoadErrorMsg:
		m.modelsState.Error = msg.error
		m.modelsState.Loading = false
	}
	return nil
}

// handleModelsKeys handles key input in models state
func (m *Model) handleModelsKeys(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.modelsState.SelectedIndex > 0 {
			m.modelsState.SelectedIndex--
		}

	case "down", "j":
		if m.modelsState.SelectedIndex < len(m.modelsState.FilteredModels)-1 {
			m.modelsState.SelectedIndex++
		}

	case "enter":
		selectedModel := m.modelsState.GetSelectedModel()
		if selectedModel != nil {
			return m.switchToModel(*selectedModel)
		}

	case "backspace":
		if len(m.modelsState.SearchQuery) > 0 {
			m.modelsState.SearchQuery = m.modelsState.SearchQuery[:len(m.modelsState.SearchQuery)-1]
			m.modelsState.FilterModels(m.modelsState.SearchQuery)
		}

	case "/":
		// Start search mode
		m.modelsState.SearchQuery = ""
		m.modelsState.FilterModels("")

	default:
		// Add to search query
		if len(msg.Runes) > 0 && unicode.IsPrint(msg.Runes[0]) {
			m.modelsState.SearchQuery += string(msg.Runes)
			m.modelsState.FilterModels(m.modelsState.SearchQuery)
		}
	}

	return nil
}

// handleSettingsState handles the settings state
func (m *Model) handleSettingsState(msg tea.Msg) tea.Cmd {
	// TODO: Implement settings handling
	return nil
}

// handleHistoryState handles the history browsing state
func (m *Model) handleHistoryState(msg tea.Msg) tea.Cmd {
	// TODO: Implement history handling
	return nil
}

// handleHelpState handles the help display state
func (m *Model) handleHelpState(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.helpState.SelectedSection > 0 {
				m.helpState.SelectedSection--
			}
		case "down", "j":
			// TODO: Implement help navigation
		}
	}
	return nil
}

// handleErrorState handles the error state
func (m *Model) handleErrorState(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// Retry
			if m.recoverFromError() {
				return func() tea.Msg {
					return statusMsg{"Retrying...", 2 * time.Second}
				}
			}
		case "enter", "esc":
			// Try to go back to previous state
			if m.errorState != nil {
				m.TransitionTo(m.errorState.PreviousState)
			} else {
				m.TransitionTo(StateChat)
			}
		}
	}
	return nil
}

// handleInitializationMessages handles initialization-related messages
func (m *Model) handleInitializationMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case initStartMsg:
		m.loadingState.CurrentStep = StepStarting
		return func() tea.Msg { return initStorageMsg{} }

	case initStorageMsg:
		m.loadingState.CurrentStep = StepStorage
		return m.initializeStorage()

	case initKeystoreMsg:
		m.loadingState.CurrentStep = StepKeystore
		return m.initializeKeystore()

	case initConfigMsg:
		m.loadingState.CurrentStep = StepConfig
		return m.initializeConfig()

	case initAnalyticsMsg:
		m.loadingState.CurrentStep = StepAnalytics
		return m.initializeAnalytics()

	case initAPIClientMsg:
		m.loadingState.CurrentStep = StepAPIClient
		return m.initializeAPIClient()

	case initCompleteMsg:
		m.loadingState.CurrentStep = StepComplete
		m.loadingState.Complete()
		// Load available models
		return func() tea.Msg { return modelsLoadStartMsg{} }

	case initErrorMsg:
		m.loadingState.SetError(msg.error)
		m.setError(msg.error, "Initialization failed", true)
	}

	return nil
}

// handleAPIMessages handles API-related messages
func (m *Model) handleAPIMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case apiRequestMsg:
		// Handle API request
		return m.performAPIRequest(msg.request)

	case apiResponseMsg:
		// Handle API response
		if msg.response != nil {
			assistantMsg := api.Message{
				Role:      "assistant",
				Content:   msg.response.Content,
				Timestamp: time.Now(),
			}
			m.chatState.AddMessage(assistantMsg)

			// Log the message (convert to storage format)
			if m.storage != nil && m.storage.ChatLogger != nil {
				go func() {
					storageMsg := storage.Message{
						Role:      assistantMsg.Role,
						Content:   assistantMsg.Content,
						Timestamp: assistantMsg.Timestamp,
					}
					if err := m.storage.ChatLogger.LogMessage(storageMsg); err != nil {
						m.logger.Error("Failed to log assistant message", "error", err)
					}
				}()
			}
		}
		m.chatState.WaitingForAPI = false

	case modelsLoadStartMsg:
		m.modelsState.Loading = true
		return m.loadAvailableModels()

	case modelSwitchMsg:
		m.currentModel = msg.model
		m.modelsState.CurrentModel = msg.model
		return func() tea.Msg {
			return statusMsg{fmt.Sprintf("Switched to %s", msg.model.Name), 3 * time.Second}
		}
	}

	return nil
}

// deleteWordBackward deletes a word backwards from the cursor
func (m *Model) deleteWordBackward() {
	if m.cursorPos == 0 {
		return
	}

	// Find the start of the current word
	pos := m.cursorPos - 1
	for pos > 0 && unicode.IsSpace(rune(m.inputBuffer[pos])) {
		pos--
	}
	for pos > 0 && !unicode.IsSpace(rune(m.inputBuffer[pos-1])) {
		pos--
	}

	// Delete from pos to cursor
	before := m.inputBuffer[:pos]
	after := m.inputBuffer[m.cursorPos:]
	m.inputBuffer = before + after
	m.cursorPos = pos
}

// sendChatMessage sends a chat message to the API
func (m *Model) sendChatMessage(content string) tea.Cmd {
	// Check if API client is available
	if m.apiClient == nil {
		return func() tea.Msg {
			return statusMsg{"No API client available. Use /settings to configure API keys.", 5 * time.Second}
		}
	}

	// Create user message
	userMsg := api.Message{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
	}

	// Add to chat history
	m.chatState.AddMessage(userMsg)

	// Log the user message (convert to storage format) - with nil check
	if m.storage != nil && m.storage.ChatLogger != nil {
		go func() {
			storageMsg := storage.Message{
				Role:      userMsg.Role,
				Content:   userMsg.Content,
				Timestamp: userMsg.Timestamp,
			}
			if err := m.storage.ChatLogger.LogMessage(storageMsg); err != nil {
				m.logger.Error("Failed to log user message", "error", err)
			}
		}()
	}

	// Clear input
	m.inputBuffer = ""
	m.cursorPos = 0

	// Create API request
	request := &api.ChatRequest{
		Model:           m.currentModel,
		Messages:        m.chatState.Messages,
		EnableWebSearch: m.webSearchEnabled,
		Stream:          true,
	}

	m.chatState.WaitingForAPI = true

	return func() tea.Msg {
		return apiRequestMsg{request}
	}
}

// executeCommand executes a slash command
func (m *Model) executeCommand(input string) tea.Cmd {
	// Clear input
	m.inputBuffer = ""
	m.cursorPos = 0

	return m.ExecuteCommand(input)
}

// switchToModel switches to a different model
func (m *Model) switchToModel(model api.Model) tea.Cmd {
	// Check if we need API key for this provider
	if m.storage != nil && m.storage.KeyStore != nil {
		hasKey, err := m.storage.KeyStore.HasKey(string(model.Provider))
		if err != nil || !hasKey {
			m.setError(fmt.Errorf("no API key for provider %s", model.Provider), "Model switch failed", true)
			return nil
		}
	}

	return func() tea.Msg {
		return modelSwitchMsg{model}
	}
}

// These functions are implemented in init.go
