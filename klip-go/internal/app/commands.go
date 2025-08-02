package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/john/klip/internal/api"
)

// Command represents a slash command
type Command struct {
	Name        string
	Aliases     []string
	Description string
	Usage       string
	Handler     func(m *Model, args []string) tea.Cmd
}

// CommandRegistry holds all available commands
type CommandRegistry struct {
	commands map[string]*Command
	aliases  map[string]string
}

// NewCommandRegistry creates a new command registry with built-in commands
func NewCommandRegistry() *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[string]*Command),
		aliases:  make(map[string]string),
	}

	// Register built-in commands
	registry.registerBuiltinCommands()

	return registry
}

// registerBuiltinCommands registers all built-in commands
func (cr *CommandRegistry) registerBuiltinCommands() {
	commands := []*Command{
		{
			Name:        "help",
			Aliases:     []string{"h", "?"},
			Description: "Show help information",
			Usage:       "/help [command]",
			Handler:     (*Model).handleHelpCommand,
		},
		{
			Name:        "model",
			Aliases:     []string{"m"},
			Description: "Switch to a different AI model",
			Usage:       "/model [model-id]",
			Handler:     (*Model).handleModelCommand,
		},
		{
			Name:        "models",
			Aliases:     []string{"list"},
			Description: "List all available models",
			Usage:       "/models",
			Handler:     (*Model).handleModelsCommand,
		},
		{
			Name:        "clear",
			Aliases:     []string{"cls", "c"},
			Description: "Clear chat history",
			Usage:       "/clear",
			Handler:     (*Model).handleClearCommand,
		},
		{
			Name:        "history",
			Aliases:     []string{"hist"},
			Description: "View chat history",
			Usage:       "/history [session-id]",
			Handler:     (*Model).handleHistoryCommand,
		},
		{
			Name:        "export",
			Aliases:     []string{"save", "download"},
			Description: "Export chat session",
			Usage:       "/export [format] [filename]",
			Handler:     (*Model).handleExportCommand,
		},
		{
			Name:        "settings",
			Aliases:     []string{"config", "cfg"},
			Description: "Open settings",
			Usage:       "/settings [option] [value]",
			Handler:     (*Model).handleSettingsCommand,
		},
		{
			Name:        "keys",
			Aliases:     []string{"apikey", "key"},
			Description: "Manage API keys",
			Usage:       "/keys [provider] [key]",
			Handler:     (*Model).handleKeysCommand,
		},
		{
			Name:        "stats",
			Aliases:     []string{"statistics", "analytics"},
			Description: "Show usage statistics",
			Usage:       "/stats",
			Handler:     (*Model).handleStatsCommand,
		},
		{
			Name:        "edit",
			Aliases:     []string{"e"},
			Description: "Edit the last message",
			Usage:       "/edit",
			Handler:     (*Model).handleEditCommand,
		},
		{
			Name:        "retry",
			Aliases:     []string{"r"},
			Description: "Retry the last request",
			Usage:       "/retry",
			Handler:     (*Model).handleRetryCommand,
		},
		{
			Name:        "search",
			Aliases:     []string{"find", "grep"},
			Description: "Search chat history",
			Usage:       "/search <query>",
			Handler:     (*Model).handleSearchCommand,
		},
		{
			Name:        "quit",
			Aliases:     []string{"exit", "q"},
			Description: "Exit the application",
			Usage:       "/quit",
			Handler:     (*Model).handleQuitCommand,
		},
		{
			Name:        "debug",
			Aliases:     []string{"dbg"},
			Description: "Toggle debug information",
			Usage:       "/debug",
			Handler:     (*Model).handleDebugCommand,
		},
		{
			Name:        "websearch",
			Aliases:     []string{"web", "search-web"},
			Description: "Toggle web search functionality",
			Usage:       "/websearch [on|off]",
			Handler:     (*Model).handleWebSearchCommand,
		},
	}

	for _, cmd := range commands {
		cr.Register(cmd)
	}
}

// Register registers a command and its aliases
func (cr *CommandRegistry) Register(cmd *Command) {
	cr.commands[cmd.Name] = cmd

	// Register aliases
	for _, alias := range cmd.Aliases {
		cr.aliases[alias] = cmd.Name
	}
}

// Get retrieves a command by name or alias
func (cr *CommandRegistry) Get(name string) *Command {
	// Remove leading slash if present
	name = strings.TrimPrefix(name, "/")

	// Try direct lookup
	if cmd, exists := cr.commands[name]; exists {
		return cmd
	}

	// Try alias lookup
	if realName, exists := cr.aliases[name]; exists {
		return cr.commands[realName]
	}

	return nil
}

// List returns all registered commands
func (cr *CommandRegistry) List() []*Command {
	commands := make([]*Command, 0, len(cr.commands))
	for _, cmd := range cr.commands {
		commands = append(commands, cmd)
	}
	return commands
}

// GetSuggestions returns command suggestions for autocomplete
func (cr *CommandRegistry) GetSuggestions(prefix string) []string {
	prefix = strings.TrimPrefix(strings.ToLower(prefix), "/")
	var suggestions []string

	// Check command names
	for name := range cr.commands {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			suggestions = append(suggestions, "/"+name)
		}
	}

	// Check aliases
	for alias := range cr.aliases {
		if strings.HasPrefix(strings.ToLower(alias), prefix) {
			suggestions = append(suggestions, "/"+alias)
		}
	}

	return suggestions
}

// ExecuteCommand executes a command with the given arguments
func (m *Model) ExecuteCommand(input string) tea.Cmd {
	if !strings.HasPrefix(input, "/") {
		return func() tea.Msg {
			return statusMsg{"Not a command", 2 * time.Second}
		}
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return func() tea.Msg {
			return statusMsg{"Empty command", 2 * time.Second}
		}
	}

	commandName := parts[0]
	args := parts[1:]

	// Get command from registry
	registry := NewCommandRegistry()
	cmd := registry.Get(commandName)
	if cmd == nil {
		return func() tea.Msg {
			return statusMsg{fmt.Sprintf("Unknown command: %s", commandName), 3 * time.Second}
		}
	}

	// Log command execution
	// TODO: Implement command logging when method is available

	// Execute command
	return cmd.Handler(m, args)
}

// Command handlers

// handleHelpCommand shows help information
func (m *Model) handleHelpCommand(args []string) tea.Cmd {
	if len(args) > 0 {
		// Show help for specific command
		cmdName := args[0]
		registry := NewCommandRegistry()
		cmd := registry.Get(cmdName)
		if cmd == nil {
			return func() tea.Msg {
				return statusMsg{fmt.Sprintf("Unknown command: %s", cmdName), 3 * time.Second}
			}
		}

		// Show detailed help for the command
		helpText := fmt.Sprintf("Command: %s\nUsage: %s\nDescription: %s",
			cmd.Name, cmd.Usage, cmd.Description)

		if len(cmd.Aliases) > 0 {
			helpText += fmt.Sprintf("\nAliases: %s", strings.Join(cmd.Aliases, ", "))
		}

		// TODO: Display help text in a modal or dedicated view
		_ = helpText // Avoid unused variable warning
		return func() tea.Msg {
			return statusMsg{"Help displayed", 2 * time.Second}
		}
	}

	// Show general help
	m.TransitionTo(StateHelp)
	return nil
}

// handleModelCommand switches models
func (m *Model) handleModelCommand(args []string) tea.Cmd {
	if len(args) == 0 {
		// No model specified, show model selection
		m.TransitionTo(StateModels)
		return nil
	}

	modelID := args[0]

	// TODO: Find model and switch to it
	// For now, just show a status message
	return func() tea.Msg {
		return statusMsg{fmt.Sprintf("Switching to model: %s", modelID), 2 * time.Second}
	}
}

// handleModelsCommand lists available models
func (m *Model) handleModelsCommand(args []string) tea.Cmd {
	m.TransitionTo(StateModels)
	return func() tea.Msg {
		return modelsLoadStartMsg{}
	}
}

// handleClearCommand clears chat history
func (m *Model) handleClearCommand(args []string) tea.Cmd {
	m.chatState.ClearMessages()

	// Clear log if storage is available
	if m.storage != nil && m.storage.ChatLogger != nil {
		go func() {
			if err := m.storage.ChatLogger.ClearLog(); err != nil {
				m.logger.Error("Failed to clear chat log", "error", err)
			}
		}()
	}

	return func() tea.Msg {
		return statusMsg{"Chat history cleared", 2 * time.Second}
	}
}

// handleHistoryCommand shows chat history
func (m *Model) handleHistoryCommand(args []string) tea.Cmd {
	m.TransitionTo(StateHistory)
	return func() tea.Msg {
		return historyLoadStartMsg{}
	}
}

// handleExportCommand exports chat session
func (m *Model) handleExportCommand(args []string) tea.Cmd {
	format := "json"

	if len(args) > 0 {
		format = args[0]
	}

	// TODO: Implement export functionality
	return func() tea.Msg {
		return statusMsg{fmt.Sprintf("Export not implemented (format: %s)", format), 3 * time.Second}
	}
}

// handleSettingsCommand opens settings
func (m *Model) handleSettingsCommand(args []string) tea.Cmd {
	if len(args) >= 2 {
		// Set specific setting
		option := args[0]
		value := args[1]

		// TODO: Handle setting updates
		return func() tea.Msg {
			return statusMsg{fmt.Sprintf("Setting %s = %s", option, value), 2 * time.Second}
		}
	}

	m.TransitionTo(StateSettings)
	return nil
}

// handleKeysCommand manages API keys
func (m *Model) handleKeysCommand(args []string) tea.Cmd {
	if len(args) == 0 {
		// Show current key status
		status := "API Key Status:\n"

		providers := []string{"anthropic", "openai", "openrouter"}
		for _, provider := range providers {
			hasKey := false
			if m.storage != nil && m.storage.KeyStore != nil {
				var err error
				hasKey, err = m.storage.KeyStore.HasKey(provider)
				if err != nil {
					hasKey = false
				}
			}

			statusIcon := "❌"
			if hasKey {
				statusIcon = "✅"
			}
			status += fmt.Sprintf("%s %s\n", statusIcon, provider)
		}

		// TODO: Display in a proper dialog
		return func() tea.Msg {
			return statusMsg{"API key status displayed", 2 * time.Second}
		}
	}

	if len(args) >= 2 {
		provider := args[0]

		// TODO: Set API key
		return func() tea.Msg {
			return statusMsg{fmt.Sprintf("API key set for %s", provider), 2 * time.Second}
		}
	}

	return func() tea.Msg {
		return statusMsg{"Usage: /keys [provider] [key]", 3 * time.Second}
	}
}

// handleStatsCommand shows usage statistics
func (m *Model) handleStatsCommand(args []string) tea.Cmd {
	// TODO: Implement statistics display
	return func() tea.Msg {
		return statusMsg{"Statistics not implemented", 2 * time.Second}
	}
}

// handleEditCommand edits the last message
func (m *Model) handleEditCommand(args []string) tea.Cmd {
	lastUserMsg := m.chatState.GetLastUserMessage()
	if lastUserMsg == nil {
		return func() tea.Msg {
			return statusMsg{"No message to edit", 2 * time.Second}
		}
	}

	// TODO: Implement edit functionality
	// For now, just set the last message content as current input
	m.setCurrentInput(lastUserMsg.Content)

	return func() tea.Msg {
		return statusMsg{"Message loaded for editing", 2 * time.Second}
	}
}

// handleRetryCommand retries the last request
func (m *Model) handleRetryCommand(args []string) tea.Cmd {
	lastUserMsg := m.chatState.GetLastUserMessage()
	if lastUserMsg == nil {
		return func() tea.Msg {
			return statusMsg{"No message to retry", 2 * time.Second}
		}
	}

	// Remove the last assistant response if it exists
	if len(m.chatState.Messages) > 0 {
		lastMsg := m.chatState.Messages[len(m.chatState.Messages)-1]
		if lastMsg.Role == "assistant" {
			m.chatState.Messages = m.chatState.Messages[:len(m.chatState.Messages)-1]
		}
	}

	// Resend the last user message
	return m.sendChatMessage(lastUserMsg.Content)
}

// handleSearchCommand searches chat history
func (m *Model) handleSearchCommand(args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return statusMsg{"Usage: /search <query>", 2 * time.Second}
		}
	}

	query := strings.Join(args, " ")

	// TODO: Implement search functionality
	return func() tea.Msg {
		return statusMsg{fmt.Sprintf("Search not implemented (query: %s)", query), 3 * time.Second}
	}
}

// handleQuitCommand exits the application
func (m *Model) handleQuitCommand(args []string) tea.Cmd {
	return tea.Quit
}

// handleDebugCommand toggles debug information
func (m *Model) handleDebugCommand(args []string) tea.Cmd {
	m.showDebugInfo = !m.showDebugInfo

	status := "off"
	if m.showDebugInfo {
		status = "on"
	}

	return func() tea.Msg {
		return statusMsg{fmt.Sprintf("Debug info %s", status), 2 * time.Second}
	}
}

// handleWebSearchCommand toggles web search
func (m *Model) handleWebSearchCommand(args []string) tea.Cmd {
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "on", "true", "1", "yes":
			m.webSearchEnabled = true
		case "off", "false", "0", "no":
			m.webSearchEnabled = false
		default:
			return func() tea.Msg {
				return statusMsg{"Usage: /websearch [on|off]", 2 * time.Second}
			}
		}
	} else {
		m.webSearchEnabled = !m.webSearchEnabled
	}

	status := "disabled"
	if m.webSearchEnabled {
		status = "enabled"
	}

	return func() tea.Msg {
		return statusMsg{fmt.Sprintf("Web search %s", status), 2 * time.Second}
	}
}

// Additional utility functions for command handling

// parseModelID parses a model ID and returns the appropriate model
func (m *Model) parseModelID(modelID string) (*api.Model, error) {
	// Check if it's a numeric index
	if index, err := strconv.Atoi(modelID); err == nil {
		if index > 0 && index <= len(m.modelsState.AvailableModels) {
			return &m.modelsState.AvailableModels[index-1], nil
		}
		return nil, fmt.Errorf("model index out of range: %d", index)
	}

	// Check if it's a model ID or name
	for _, model := range m.modelsState.AvailableModels {
		if model.ID == modelID || strings.EqualFold(model.Name, modelID) {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", modelID)
}

// validateProvider validates if a provider is supported
func (m *Model) validateProvider(provider string) bool {
	supportedProviders := []string{"anthropic", "openai", "openrouter"}
	for _, p := range supportedProviders {
		if strings.EqualFold(p, provider) {
			return true
		}
	}
	return false
}

// formatFileSize formats bytes as human-readable size
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "kMGTPE"[exp])
}

// formatDuration formats a duration as human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1e6)
	} else if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

// GetCommandSuggestions returns command suggestions for autocomplete
func (m *Model) GetCommandSuggestions(input string) []string {
	registry := NewCommandRegistry()
	return registry.GetSuggestions(input)
}

// IsValidCommand checks if a string is a valid command
func (m *Model) IsValidCommand(input string) bool {
	if !strings.HasPrefix(input, "/") {
		return false
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	registry := NewCommandRegistry()
	cmd := registry.Get(parts[0])
	return cmd != nil
}
