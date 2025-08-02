package components

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/john/klip/internal/app"
)

// InputType represents different input modes
type InputType int

const (
	InputTypeText InputType = iota
	InputTypeMultiline
	InputTypeCommand
	InputTypeSearch
)

// InputMsg represents input component messages
type InputMsg struct {
	Type string
	Data interface{}
}

// CommandSuggestion represents a command suggestion
type CommandSuggestion struct {
	Command     string
	Description string
	Usage       string
}

// EnhancedInput provides advanced text input capabilities
type EnhancedInput struct {
	textInput         textinput.Model
	textArea          textarea.Model
	inputType         InputType
	history           []string
	historyIndex      int
	suggestions       []CommandSuggestion
	showSuggestions   bool
	selectedSuggestion int
	commandPrefix     string
	width             int
	height            int
	placeholder       string
	maxLength         int
	showCharCount     bool
	showTokenCount    bool
	tokenEstimate     int
	validator         func(string) error
	errorMessage      string
	focused           bool
}

// NewEnhancedInput creates a new enhanced input component
func NewEnhancedInput(inputType InputType, width, height int) *EnhancedInput {
	ei := &EnhancedInput{
		inputType:       inputType,
		width:           width,
		height:          height,
		history:         make([]string, 0),
		historyIndex:    -1,
		suggestions:     getDefaultCommands(),
		commandPrefix:   "/",
		showCharCount:   true,
		showTokenCount:  true,
		focused:         true,
	}

	switch inputType {
	case InputTypeMultiline:
		ta := textarea.New()
		ta.SetWidth(width - 4) // Account for padding and borders
		ta.SetHeight(height - 2)
		ta.Focus()
		ta.Placeholder = "Type your message (Ctrl+Enter to send, Esc to switch modes)..."
		ta.ShowLineNumbers = false
		ta.KeyMap.InsertNewline.SetKeys("enter")
		ei.textArea = ta
	default:
		ti := textinput.New()
		ti.Width = width - 4
		ti.Focus()
		ti.Placeholder = "Type your message or /command..."
		ti.CharLimit = 4000 // Default character limit
		ei.textInput = ti
	}

	ei.updateTokenEstimate()
	return ei
}

// Init initializes the input component
func (ei *EnhancedInput) Init() tea.Cmd {
	// Bubble tea textarea and textinput models don't have Init() methods
	// They are initialized when created with New()
	return nil
}

// Update handles input updates
func (ei *EnhancedInput) Update(msg tea.Msg) (*EnhancedInput, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ei.width = msg.Width
		ei.height = msg.Height
		if ei.inputType == InputTypeMultiline {
			ei.textArea.SetWidth(msg.Width - 4)
			ei.textArea.SetHeight(ei.height - 2)
		} else {
			ei.textInput.Width = msg.Width - 4
		}

	case InputMsg:
		switch msg.Type {
		case "set_placeholder":
			if placeholder, ok := msg.Data.(string); ok {
				ei.SetPlaceholder(placeholder)
			}
		case "set_value":
			if value, ok := msg.Data.(string); ok {
				ei.SetValue(value)
			}
		case "clear":
			ei.Clear()
		case "focus":
			ei.Focus()
		case "blur":
			ei.Blur()
		case "toggle_mode":
			ei.ToggleMode()
		case "add_to_history":
			if value, ok := msg.Data.(string); ok {
				ei.AddToHistory(value)
			}
		}

	case tea.KeyMsg:
		// Handle global shortcuts first
		switch msg.String() {
		case "ctrl+c":
			if ei.inputType == InputTypeMultiline {
				return ei, tea.Quit
			}
		case "ctrl+v":
			cmd = ei.pasteFromClipboard()
			cmds = append(cmds, cmd)
		case "ctrl+x":
			cmd = ei.cutToClipboard()
			cmds = append(cmds, cmd)
		case "ctrl+z":
			cmd = ei.undo()
			cmds = append(cmds, cmd)
		case "tab":
			if ei.showSuggestions && len(ei.suggestions) > 0 {
				ei.acceptSuggestion()
				ei.updateSuggestions()
				return ei, tea.Batch(cmds...)
			}
		case "esc":
			if ei.showSuggestions {
				ei.showSuggestions = false
				return ei, tea.Batch(cmds...)
			}
		case "up":
			if ei.showSuggestions {
				ei.navigateSuggestions(-1)
				return ei, tea.Batch(cmds...)
			} else {
				ei.navigateHistory(-1)
				return ei, tea.Batch(cmds...)
			}
		case "down":
			if ei.showSuggestions {
				ei.navigateSuggestions(1)
				return ei, tea.Batch(cmds...)
			} else {
				ei.navigateHistory(1)
				return ei, tea.Batch(cmds...)
			}
		case "enter":
			if ei.inputType != InputTypeMultiline {
				value := ei.Value()
				if value != "" {
					ei.AddToHistory(value)
					return ei, ei.submitValue(value)
				}
				return ei, tea.Batch(cmds...)
			}
		case "ctrl+enter":
			if ei.inputType == InputTypeMultiline {
				value := ei.Value()
				if value != "" {
					ei.AddToHistory(value)
					return ei, ei.submitValue(value)
				}
				return ei, tea.Batch(cmds...)
			}
		}

		// Update the underlying input component
		if ei.inputType == InputTypeMultiline {
			ei.textArea, cmd = ei.textArea.Update(msg)
		} else {
			ei.textInput, cmd = ei.textInput.Update(msg)
		}
		cmds = append(cmds, cmd)

		// Update suggestions and token estimate after text changes
		ei.updateSuggestions()
		ei.updateTokenEstimate()
		ei.validateInput()

	default:
		if ei.inputType == InputTypeMultiline {
			ei.textArea, cmd = ei.textArea.Update(msg)
		} else {
			ei.textInput, cmd = ei.textInput.Update(msg)
		}
		cmds = append(cmds, cmd)
	}

	return ei, tea.Batch(cmds...)
}

// View renders the input component
func (ei *EnhancedInput) View() string {
	var content strings.Builder

	// Render main input area
	if ei.inputType == InputTypeMultiline {
		content.WriteString(MultilineInputStyle.Render(ei.textArea.View()))
	} else {
		inputView := ei.textInput.View()
		if ei.errorMessage != "" {
			inputView = ErrorInputStyle.Render(inputView)
		} else if ei.focused {
			inputView = FocusedInputStyle.Render(inputView)
		} else {
			inputView = InputStyle.Render(inputView)
		}
		content.WriteString(inputView)
	}

	// Render suggestions if shown
	if ei.showSuggestions && len(ei.suggestions) > 0 {
		content.WriteString("\n")
		content.WriteString(ei.renderSuggestions())
	}

	// Render footer with stats and error
	content.WriteString("\n")
	content.WriteString(ei.renderFooter())

	return content.String()
}

// Value returns the current input value
func (ei *EnhancedInput) Value() string {
	if ei.inputType == InputTypeMultiline {
		return ei.textArea.Value()
	}
	return ei.textInput.Value()
}

// SetValue sets the input value
func (ei *EnhancedInput) SetValue(value string) {
	if ei.inputType == InputTypeMultiline {
		ei.textArea.SetValue(value)
	} else {
		ei.textInput.SetValue(value)
	}
	ei.updateTokenEstimate()
	ei.validateInput()
}

// Clear clears the input
func (ei *EnhancedInput) Clear() {
	ei.SetValue("")
	ei.errorMessage = ""
	ei.showSuggestions = false
}

// Focus focuses the input
func (ei *EnhancedInput) Focus() {
	ei.focused = true
	if ei.inputType == InputTypeMultiline {
		ei.textArea.Focus()
	} else {
		ei.textInput.Focus()
	}
}

// Blur blurs the input
func (ei *EnhancedInput) Blur() {
	ei.focused = false
	if ei.inputType == InputTypeMultiline {
		ei.textArea.Blur()
	} else {
		ei.textInput.Blur()
	}
	ei.showSuggestions = false
}

// SetPlaceholder sets the placeholder text
func (ei *EnhancedInput) SetPlaceholder(placeholder string) {
	ei.placeholder = placeholder
	if ei.inputType == InputTypeMultiline {
		ei.textArea.Placeholder = placeholder
	} else {
		ei.textInput.Placeholder = placeholder
	}
}

// ToggleMode toggles between single-line and multi-line input
func (ei *EnhancedInput) ToggleMode() {
	currentValue := ei.Value()
	
	if ei.inputType == InputTypeMultiline {
		ei.inputType = InputTypeText
		ei.textInput = textinput.New()
		ei.textInput.Width = ei.width - 4
		ei.textInput.SetValue(currentValue)
		ei.textInput.Focus()
		ei.textInput.Placeholder = "Type your message or /command..."
	} else {
		ei.inputType = InputTypeMultiline
		ei.textArea = textarea.New()
		ei.textArea.SetWidth(ei.width - 4)
		ei.textArea.SetHeight(ei.height - 2)
		ei.textArea.SetValue(currentValue)
		ei.textArea.Focus()
		ei.textArea.Placeholder = "Type your message (Ctrl+Enter to send)..."
	}
	
	ei.updateTokenEstimate()
}

// AddToHistory adds a value to input history
func (ei *EnhancedInput) AddToHistory(value string) {
	if value == "" {
		return
	}
	
	// Remove duplicate if exists
	for i, item := range ei.history {
		if item == value {
			ei.history = append(ei.history[:i], ei.history[i+1:]...)
			break
		}
	}
	
	// Add to beginning
	ei.history = append([]string{value}, ei.history...)
	
	// Limit history size
	if len(ei.history) > 100 {
		ei.history = ei.history[:100]
	}
	
	ei.historyIndex = -1
}

// navigateHistory navigates through input history
func (ei *EnhancedInput) navigateHistory(direction int) {
	if len(ei.history) == 0 {
		return
	}
	
	newIndex := ei.historyIndex + direction
	if newIndex < -1 {
		newIndex = -1
	} else if newIndex >= len(ei.history) {
		newIndex = len(ei.history) - 1
	}
	
	ei.historyIndex = newIndex
	
	if ei.historyIndex == -1 {
		ei.SetValue("")
	} else {
		ei.SetValue(ei.history[ei.historyIndex])
	}
}

// updateSuggestions updates command suggestions based on current input
func (ei *EnhancedInput) updateSuggestions() {
	value := ei.Value()
	
	if !strings.HasPrefix(value, ei.commandPrefix) {
		ei.showSuggestions = false
		return
	}
	
	command := strings.TrimPrefix(value, ei.commandPrefix)
	if command == "" {
		ei.showSuggestions = true
		ei.selectedSuggestion = 0
		return
	}
	
	// Filter suggestions based on input
	filtered := make([]CommandSuggestion, 0)
	for _, suggestion := range ei.suggestions {
		if strings.HasPrefix(strings.ToLower(suggestion.Command), strings.ToLower(command)) {
			filtered = append(filtered, suggestion)
		}
	}
	
	if len(filtered) > 0 {
		ei.suggestions = filtered
		ei.showSuggestions = true
		ei.selectedSuggestion = 0
	} else {
		ei.showSuggestions = false
	}
}

// navigateSuggestions navigates through command suggestions
func (ei *EnhancedInput) navigateSuggestions(direction int) {
	if len(ei.suggestions) == 0 {
		return
	}
	
	newIndex := ei.selectedSuggestion + direction
	if newIndex < 0 {
		newIndex = len(ei.suggestions) - 1
	} else if newIndex >= len(ei.suggestions) {
		newIndex = 0
	}
	
	ei.selectedSuggestion = newIndex
}

// acceptSuggestion accepts the currently selected suggestion
func (ei *EnhancedInput) acceptSuggestion() {
	if !ei.showSuggestions || len(ei.suggestions) == 0 {
		return
	}
	
	suggestion := ei.suggestions[ei.selectedSuggestion]
	ei.SetValue(ei.commandPrefix + suggestion.Command + " ")
	ei.showSuggestions = false
}

// updateTokenEstimate estimates token count for the current input
func (ei *EnhancedInput) updateTokenEstimate() {
	value := ei.Value()
	ei.tokenEstimate = estimateTokens(value)
}

// validateInput validates the current input
func (ei *EnhancedInput) validateInput() {
	if ei.validator == nil {
		ei.errorMessage = ""
		return
	}
	
	if err := ei.validator(ei.Value()); err != nil {
		ei.errorMessage = err.Error()
	} else {
		ei.errorMessage = ""
	}
}

// Clipboard operations
func (ei *EnhancedInput) pasteFromClipboard() tea.Cmd {
	return func() tea.Msg {
		text, err := clipboard.ReadAll()
		if err != nil {
			return InputMsg{Type: "error", Data: err}
		}
		
		currentValue := ei.Value()
		// Insert at cursor position (simplified for now)
		newValue := currentValue + text
		
		return InputMsg{Type: "set_value", Data: newValue}
	}
}

func (ei *EnhancedInput) cutToClipboard() tea.Cmd {
	return func() tea.Msg {
		value := ei.Value()
		if value == "" {
			return nil
		}
		
		err := clipboard.WriteAll(value)
		if err != nil {
			return InputMsg{Type: "error", Data: err}
		}
		
		return InputMsg{Type: "clear"}
	}
}

func (ei *EnhancedInput) undo() tea.Cmd {
	// Simplified undo - just clear for now
	return func() tea.Msg {
		return InputMsg{Type: "clear"}
	}
}

// submitValue submits the current input value
func (ei *EnhancedInput) submitValue(value string) tea.Cmd {
	return func() tea.Msg {
		return InputMsg{Type: "submit", Data: value}
	}
}

// renderSuggestions renders the command suggestions list
func (ei *EnhancedInput) renderSuggestions() string {
	if len(ei.suggestions) == 0 {
		return ""
	}
	
	var content strings.Builder
	content.WriteString(SuggestionsHeaderStyle.Render("Commands:"))
	content.WriteString("\n")
	
	maxVisible := 5
	start := 0
	end := len(ei.suggestions)
	
	if end > maxVisible {
		// Center the selection
		start = ei.selectedSuggestion - maxVisible/2
		if start < 0 {
			start = 0
		}
		end = start + maxVisible
		if end > len(ei.suggestions) {
			end = len(ei.suggestions)
			start = end - maxVisible
		}
	}
	
	for i := start; i < end; i++ {
		suggestion := ei.suggestions[i]
		prefix := "  "
		style := SuggestionStyle
		
		if i == ei.selectedSuggestion {
			prefix = "→ "
			style = SelectedSuggestionStyle
		}
		
		line := fmt.Sprintf("%s%s%s - %s",
			prefix,
			ei.commandPrefix,
			suggestion.Command,
			suggestion.Description)
		
		content.WriteString(style.Render(line))
		content.WriteString("\n")
	}
	
	return SuggestionsContainerStyle.Render(content.String())
}

// renderFooter renders the input footer with stats and errors
func (ei *EnhancedInput) renderFooter() string {
	var parts []string
	
	// Error message
	if ei.errorMessage != "" {
		parts = append(parts, ErrorMessageStyle.Render("✗ "+ei.errorMessage))
	}
	
	// Character count
	if ei.showCharCount {
		charCount := len(ei.Value())
		charText := fmt.Sprintf("%d chars", charCount)
		if ei.maxLength > 0 {
			charStyle := CharCountStyle
			if charCount > ei.maxLength {
				charStyle = ErrorCharCountStyle
			}
			charText = fmt.Sprintf("%d/%d chars", charCount, ei.maxLength)
			parts = append(parts, charStyle.Render(charText))
		} else {
			parts = append(parts, CharCountStyle.Render(charText))
		}
	}
	
	// Token estimate
	if ei.showTokenCount {
		tokenText := fmt.Sprintf("~%d tokens", ei.tokenEstimate)
		parts = append(parts, TokenCountStyle.Render(tokenText))
	}
	
	// Input mode indicator
	var modeText string
	switch ei.inputType {
	case InputTypeMultiline:
		modeText = "MULTI"
	case InputTypeCommand:
		modeText = "CMD"
	case InputTypeSearch:
		modeText = "SEARCH"
	default:
		modeText = "TEXT"
	}
	parts = append(parts, ModeIndicatorStyle.Render(modeText))
	
	if len(parts) == 0 {
		return ""
	}
	
	return FooterStyle.Render(strings.Join(parts, " │ "))
}

// getDefaultCommands returns the default command suggestions
func getDefaultCommands() []CommandSuggestion {
	return []CommandSuggestion{
		{Command: "help", Description: "Show help information", Usage: "/help [topic]"},
		{Command: "model", Description: "Switch AI model", Usage: "/model [model-name]"},
		{Command: "models", Description: "List available models", Usage: "/models"},
		{Command: "clear", Description: "Clear chat history", Usage: "/clear"},
		{Command: "history", Description: "View chat history", Usage: "/history"},
		{Command: "settings", Description: "Open settings", Usage: "/settings"},
		{Command: "export", Description: "Export chat history", Usage: "/export [format]"},
		{Command: "quit", Description: "Quit application", Usage: "/quit"},
		{Command: "save", Description: "Save current chat", Usage: "/save [name]"},
		{Command: "load", Description: "Load saved chat", Usage: "/load [name]"},
		{Command: "search", Description: "Search chat history", Usage: "/search [query]"},
		{Command: "stats", Description: "Show usage statistics", Usage: "/stats"},
		{Command: "theme", Description: "Change theme", Usage: "/theme [theme-name]"},
		{Command: "debug", Description: "Toggle debug mode", Usage: "/debug"},
		{Command: "version", Description: "Show version info", Usage: "/version"},
	}
}

// estimateTokens provides a rough token count estimate
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	
	// Rough estimation: 1 token ≈ 4 characters for English text
	// This is a simplified estimation
	words := strings.Fields(text)
	wordCount := len(words)
	
	// Count punctuation and special characters
	specialChars := 0
	for _, r := range text {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			specialChars++
		}
	}
	
	// Rough token estimation
	tokens := wordCount + (specialChars / 2)
	if tokens < len(text)/4 {
		tokens = len(text) / 4
	}
	
	return tokens
}

// Input component styles
var (
	// Input field styles
	InputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(0, 1).
		BorderStyle(lipgloss.ThickBorder())

	ErrorInputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#EF4444")).
		Padding(0, 1)

	MultilineInputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(1)

	// Suggestions styles
	SuggestionsContainerStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#D1D5DB")).
		Padding(1).
		MarginTop(1)

	SuggestionsHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Bold(true)

	SuggestionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151")).
		PaddingLeft(1)

	SelectedSuggestionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Background(lipgloss.Color("#F3F4F6")).
		Bold(true).
		PaddingLeft(1)

	// Footer styles
	FooterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		MarginTop(1).
		PaddingLeft(1)

	ErrorMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444")).
		Bold(true)

	CharCountStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	ErrorCharCountStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444")).
		Bold(true)

	TokenCountStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED"))

	ModeIndicatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#059669")).
		Bold(true)
)

// Helper functions for integration with app state
func NewEnhancedInputFromState(state *app.ChatState, width, height int) *EnhancedInput {
	var inputType InputType
	switch state.InputMode {
	case app.InputModeMultiline:
		inputType = InputTypeMultiline
	case app.InputModeCommand:
		inputType = InputTypeCommand
	case app.InputModeSearch:
		inputType = InputTypeSearch
	default:
		inputType = InputTypeText
	}
	
	ei := NewEnhancedInput(inputType, width, height)
	ei.SetValue(state.CurrentInput)
	
	return ei
}

// UpdateFromState updates the input from app state
func (ei *EnhancedInput) UpdateFromState(state *app.ChatState) {
	// Update input mode if changed
	var targetType InputType
	switch state.InputMode {
	case app.InputModeMultiline:
		targetType = InputTypeMultiline
	case app.InputModeCommand:
		targetType = InputTypeCommand
	case app.InputModeSearch:
		targetType = InputTypeSearch
	default:
		targetType = InputTypeText
	}
	
	if ei.inputType != targetType {
		ei.inputType = targetType
		// Re-initialize with new type (simplified)
		currentValue := ei.Value()
		*ei = *NewEnhancedInput(targetType, ei.width, ei.height)
		ei.SetValue(currentValue)
	}
	
	// Update value if different
	if ei.Value() != state.CurrentInput {
		ei.SetValue(state.CurrentInput)
	}
}