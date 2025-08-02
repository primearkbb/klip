package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/john/klip/internal/api"
)

// Color palette for consistent theming
var (
	// Primary colors
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#10B981") // Green
	accentColor    = lipgloss.Color("#F59E0B") // Amber
	errorColor     = lipgloss.Color("#EF4444") // Red
	warningColor   = lipgloss.Color("#F97316") // Orange

	// Text colors
	textPrimary   = lipgloss.Color("#F9FAFB") // Light gray
	textSecondary = lipgloss.Color("#9CA3AF") // Medium gray
	textMuted     = lipgloss.Color("#6B7280") // Dark gray

	// Background colors
	bgPrimary   = lipgloss.Color("#111827") // Dark blue-gray
	bgSecondary = lipgloss.Color("#1F2937") // Medium blue-gray
	bgAccent    = lipgloss.Color("#374151") // Light blue-gray

	// Border colors
	borderPrimary   = lipgloss.Color("#4B5563") // Medium gray
	borderSecondary = lipgloss.Color("#6B7280") // Light gray
)

// Base styles
var (
	baseStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(bgPrimary)

	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(textSecondary).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	successStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	mutedStyle = lipgloss.NewStyle().
			Foreground(textMuted)

	inputStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(bgSecondary).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderPrimary)

	focusedInputStyle = inputStyle.Copy().
				BorderForeground(primaryColor)

	buttonStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(bgAccent).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderPrimary)

	selectedButtonStyle = buttonStyle.Copy().
				Foreground(bgPrimary).
				Background(primaryColor).
				BorderForeground(primaryColor)
)

// View renders the current view based on the application state
func (m *Model) View() string {
	if !m.ready {
		return m.renderLoading("Initializing...")
	}

	// Get the main content
	content := m.renderStateView()

	// Create status bar
	statusBar := m.renderStatusBar()

	// Ensure content takes up most of the space, leaving room for status bar
	contentHeight := m.height - 1
	contentContainer := lipgloss.NewStyle().
		Width(m.width).
		Height(contentHeight).
		Render(content)

	// Combine content and status bar
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		contentContainer,
		statusBar,
	)

	// Ensure the final view fills the entire terminal
	finalContainer := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(view)

	return finalContainer
}

// renderStateView renders the view for the current state
func (m *Model) renderStateView() string {
	switch m.GetCurrentState() {
	case StateInitializing:
		return m.renderInitializingView()
	case StateOnboarding:
		return m.renderOnboardingView()
	case StateChat:
		return m.renderChatView()
	case StateModels:
		return m.renderModelsView()
	case StateSettings:
		return m.renderSettingsView()
	case StateHistory:
		return m.renderHistoryView()
	case StateHelp:
		return m.renderHelpView()
	case StateError:
		return m.renderErrorView()
	default:
		return m.renderLoading("Unknown state...")
	}
}

// renderInitializingView renders the initialization view
func (m *Model) renderInitializingView() string {
	if m.loadingState == nil {
		return m.renderLoading("Starting...")
	}

	progress := m.loadingState.Progress
	step := m.loadingState.CurrentStep.String()

	// ASCII art header
	header := titleStyle.Render("🚀 Klip") + "\n" +
		subtitleStyle.Render("Terminal AI Chat") + "\n\n"

	// Progress bar
	progressBar := m.renderProgressBar(progress, 40)

	// Current step
	stepText := fmt.Sprintf("Step: %s", step)

	// Elapsed time
	elapsedText := ""
	if !m.loadingState.StartTime.IsZero() {
		elapsed := m.formatElapsedTime(m.loadingState.StartTime)
		elapsedText = fmt.Sprintf("Elapsed: %s", elapsed)
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		progressBar,
		"",
		stepText,
		elapsedText,
	)

	return m.centerContent(content)
}

// renderOnboardingView renders the onboarding view
func (m *Model) renderOnboardingView() string {
	header := titleStyle.Render("Welcome to Klip!") + "\n\n"

	content := header +
		"Let's get you set up:\n\n" +
		"1. Set up your API keys\n" +
		"2. Choose your preferred model\n" +
		"3. Configure settings\n\n" +
		mutedStyle.Render("Press Enter to continue...")

	return m.centerContent(content)
}

// renderChatView renders the main chat interface
func (m *Model) renderChatView() string {
	// Calculate available space
	messageHeight := m.height - 3 // Reserve space for input and status bar

	// Chat messages area
	messagesView := m.renderMessages(messageHeight)

	// Input area
	inputView := m.renderInputArea()

	// Create full-height container
	chatContainer := lipgloss.NewStyle().
		Width(m.width).
		Height(messageHeight).
		Render(messagesView)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		chatContainer,
		inputView,
	)
}

// renderMessages renders the chat messages
func (m *Model) renderMessages(height int) string {
	if len(m.chatState.Messages) == 0 {
		// Create a properly formatted welcome screen
		header := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Align(lipgloss.Center).
			Width(m.width).
			Render("Klip Chat")

		welcomeMsg := lipgloss.NewStyle().
			Align(lipgloss.Center).
			Width(m.width).
			Render("Welcome! Start chatting with AI or type /help for commands.")

		currentModel := lipgloss.NewStyle().
			Align(lipgloss.Center).
			Width(m.width).
			Render(fmt.Sprintf("Current model: %s", successStyle.Render(m.currentModel.Name)))

		commands := lipgloss.NewStyle().
			Foreground(textMuted).
			Align(lipgloss.Center).
			Width(m.width).
			Render("Commands: /help, /models, /settings, /clear, /history")

		// Join with proper spacing
		welcome := lipgloss.JoinVertical(
			lipgloss.Left,
			"", // Empty line at top
			header,
			"", // Empty line
			welcomeMsg,
			currentModel,
			"", // Empty line
			commands,
		)

		// Create container that fills available space but centers content better
		topPadding := height / 4 // Add some top padding but not too much
		container := lipgloss.NewStyle().
			Width(m.width).
			Height(height).
			PaddingTop(topPadding).
			Render(welcome)

		return container
	}

	var messageViews []string

	for _, msg := range m.chatState.Messages {
		messageView := m.renderSingleMessage(msg)
		messageViews = append(messageViews, messageView)
	}

	// Add streaming buffer if active
	if m.chatState.IsStreaming && m.chatState.StreamBuffer != "" {
		streamingMsg := api.Message{
			Role:      "assistant",
			Content:   m.chatState.StreamBuffer,
			Timestamp: time.Now(),
		}
		messageView := m.renderSingleMessage(streamingMsg) + m.renderTypingIndicator()
		messageViews = append(messageViews, messageView)
	}

	content := strings.Join(messageViews, "\n\n")

	// Scroll to bottom if content is too long
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[len(lines)-height:]
		content = strings.Join(lines, "\n")
	}

	return content
}

// renderSingleMessage renders a single chat message
func (m *Model) renderSingleMessage(msg api.Message) string {
	var roleStyle lipgloss.Style
	var rolePrefix string

	switch msg.Role {
	case "user":
		roleStyle = successStyle.Bold(true)
		rolePrefix = "You"
	case "assistant":
		roleStyle = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
		rolePrefix = "Klip"
	case "system":
		roleStyle = warningStyle
		rolePrefix = "System"
	default:
		roleStyle = mutedStyle
		rolePrefix = strings.Title(msg.Role)
	}

	// Format timestamp
	timestamp := mutedStyle.Render(msg.Timestamp.Format("15:04"))

	// Header line
	header := fmt.Sprintf("%s %s:", roleStyle.Render(rolePrefix), timestamp)

	// Message content (word wrap)
	content := m.wrapText(msg.Content, m.width-4)

	return header + "\n" + content
}

// renderInputArea renders the input area at the bottom
func (m *Model) renderInputArea() string {
	// Input prompt based on mode
	var prompt string
	var promptStyle lipgloss.Style

	if m.chatState.WaitingForAPI {
		prompt = "Thinking..."
		promptStyle = warningStyle
	} else {
		mode := m.getInputMode()
		switch mode {
		case InputModeCommand:
			prompt = "Command"
			promptStyle = lipgloss.NewStyle().Foreground(accentColor)
		case InputModeSearch:
			prompt = "Search"
			promptStyle = lipgloss.NewStyle().Foreground(primaryColor)
		default:
			prompt = "Message"
			promptStyle = successStyle
		}
	}

	// Input field content
	inputText := m.inputBuffer
	cursor := ""

	if !m.chatState.WaitingForAPI {
		// Show cursor
		if m.cursorPos < len(inputText) {
			before := inputText[:m.cursorPos]
			char := string(inputText[m.cursorPos])
			after := inputText[m.cursorPos+1:]
			cursor = before + lipgloss.NewStyle().Reverse(true).Render(char) + after
		} else {
			cursor = inputText + lipgloss.NewStyle().Reverse(true).Render(" ")
		}
	} else {
		cursor = inputText
	}

	// Calculate input field width
	promptWidth := lipgloss.Width(prompt + ": ")
	inputWidth := m.width - promptWidth - 2
	if inputWidth < 10 {
		inputWidth = 10
	}

	// Create input field with proper styling
	inputField := lipgloss.NewStyle().
		Foreground(textPrimary).
		Background(bgSecondary).
		Padding(0, 1).
		Width(inputWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Render(cursor)

	// Create the full input line
	inputLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		promptStyle.Render(prompt+": "),
		inputField,
	)

	// Ensure the input line fills the width
	container := lipgloss.NewStyle().
		Width(m.width).
		Render(inputLine)

	return container
}

// renderModelsView renders the model selection view
func (m *Model) renderModelsView() string {
	header := titleStyle.Render("Model Selection") + "\n\n"

	if m.modelsState.Loading {
		return header + m.renderLoading("Loading models...")
	}

	if m.modelsState.Error != nil {
		return header + errorStyle.Render(fmt.Sprintf("Error: %v", m.modelsState.Error))
	}

	if len(m.modelsState.FilteredModels) == 0 {
		return header + mutedStyle.Render("No models available")
	}

	// Search bar
	searchBar := ""
	if m.modelsState.SearchQuery != "" {
		searchBar = fmt.Sprintf("Search: %s\n\n", inputStyle.Render(m.modelsState.SearchQuery))
	}

	// Model list
	var modelItems []string
	for i, model := range m.modelsState.FilteredModels {
		prefix := "  "
		style := lipgloss.NewStyle()

		if i == m.modelsState.SelectedIndex {
			prefix = "▶ "
			style = selectedButtonStyle
		} else if model.ID == m.currentModel.ID {
			prefix = "✓ "
			style = successStyle
		}

		name := fmt.Sprintf("%s (%s)", model.Name, model.Provider)
		item := fmt.Sprintf("%s%s", prefix, style.Render(name))
		modelItems = append(modelItems, item)
	}

	modelList := strings.Join(modelItems, "\n")

	footer := "\n\n" + mutedStyle.Render("↑/↓: Navigate | Enter: Select | /: Search | Esc: Back")

	return header + searchBar + modelList + footer
}

// renderSettingsView renders the settings view
func (m *Model) renderSettingsView() string {
	header := titleStyle.Render("Settings") + "\n\n"

	content := "Settings configuration coming soon...\n\n" +
		"Current configuration:\n" +
		fmt.Sprintf("• Model: %s\n", m.currentModel.Name) +
		fmt.Sprintf("• Provider: %s\n", m.currentModel.Provider) +
		fmt.Sprintf("• Web Search: %t\n", m.webSearchEnabled) +
		fmt.Sprintf("• Analytics: %t\n", m.analyticsEnabled)

	footer := "\n\n" + mutedStyle.Render("Esc: Back")

	return header + content + footer
}

// renderHistoryView renders the chat history view
func (m *Model) renderHistoryView() string {
	header := titleStyle.Render("Chat History") + "\n\n"

	if m.historyState.Loading {
		return header + m.renderLoading("Loading history...")
	}

	if m.historyState.Error != nil {
		return header + errorStyle.Render(fmt.Sprintf("Error: %v", m.historyState.Error))
	}

	content := "Chat history coming soon...\n\n" +
		"This will show your previous chat sessions."

	footer := "\n\n" + mutedStyle.Render("Esc: Back")

	return header + content + footer
}

// renderHelpView renders the help view
func (m *Model) renderHelpView() string {
	header := titleStyle.Render("Klip Help") + "\n\n"

	sections := []string{
		"🎯 Commands:",
		"  /help     - Show this help",
		"  /model    - Switch AI model",
		"  /models   - List all models",
		"  /clear    - Clear chat history",
		"  /history  - View chat history",
		"  /settings - Open settings",
		"  /quit     - Exit application",
		"",
		"⚡ Shortcuts:",
		"  F1        - Toggle help",
		"  F2        - Model selection",
		"  F3        - Settings",
		"  F4        - History",
		"  F12       - Debug info",
		"  Ctrl+C    - Interrupt/Quit",
		"  Ctrl+L    - Clear screen",
		"  ↑/↓       - Input history",
		"",
		"💡 Tips:",
		"  • Use / to start commands",
		"  • Ctrl+C during streaming interrupts",
		"  • Web search is enabled by default",
		"  • All chats are logged locally",
	}

	content := strings.Join(sections, "\n")
	footer := "\n\n" + mutedStyle.Render("Esc: Back")

	return header + content + footer
}

// renderErrorView renders the error view
func (m *Model) renderErrorView() string {
	if m.errorState == nil {
		return errorStyle.Render("Unknown error occurred")
	}

	header := errorStyle.Render("Error") + "\n\n"

	errorMsg := fmt.Sprintf("Error: %v", m.errorState.Error)
	context := ""
	if m.errorState.Context != "" {
		context = fmt.Sprintf("Context: %s", m.errorState.Context)
	}

	var actions []string
	if m.errorState.Recoverable {
		actions = append(actions, "Press 'r' to retry")
	}
	actions = append(actions, "Press Enter or Esc to continue")

	actionsText := mutedStyle.Render(strings.Join(actions, " | "))

	content := []string{header, errorMsg}
	if context != "" {
		content = append(content, context)
	}
	content = append(content, "", actionsText)

	return strings.Join(content, "\n")
}

// renderStatusBar renders the status bar at the bottom
func (m *Model) renderStatusBar() string {
	leftItems := []string{
		fmt.Sprintf("State: %s", m.GetCurrentState().String()),
		fmt.Sprintf("Model: %s", m.currentModel.Name),
	}

	rightItems := []string{}

	// Add status message if active
	if m.hasActiveStatusMessage() {
		rightItems = append(rightItems, m.statusMessage)
	}

	// Debug info
	if m.showDebugInfo {
		rightItems = append(rightItems, fmt.Sprintf("Debug: %dx%d", m.width, m.height))
	}

	left := strings.Join(leftItems, " | ")
	right := strings.Join(rightItems, " | ")

	// Create status bar
	statusStyle := lipgloss.NewStyle().
		Foreground(textSecondary).
		Background(bgSecondary).
		Padding(0, 1).
		Width(m.width)

	if right == "" {
		return statusStyle.Render(left)
	}

	// Calculate spacing
	spacing := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if spacing < 0 {
		spacing = 0
	}

	content := left + strings.Repeat(" ", spacing) + right
	return statusStyle.Render(content)
}

// Helper rendering functions

// renderLoading renders a loading indicator
func (m *Model) renderLoading(message string) string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinner := spinners[m.animationFrame%len(spinners)]

	spinnerStyle := lipgloss.NewStyle().Foreground(primaryColor)
	content := fmt.Sprintf("%s %s", spinnerStyle.Render(spinner), message)
	return m.centerContent(content)
}

// renderProgressBar renders a progress bar
func (m *Model) renderProgressBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	filled := int(progress * float64(width))
	empty := width - filled

	bar := successStyle.Render(strings.Repeat("█", filled)) +
		mutedStyle.Render(strings.Repeat("░", empty))

	percentage := fmt.Sprintf("%.0f%%", progress*100)

	return fmt.Sprintf("%s %s", bar, mutedStyle.Render(percentage))
}

// renderTypingIndicator renders a typing indicator for streaming
func (m *Model) renderTypingIndicator() string {
	dots := []string{"", ".", "..", "..."}
	dot := dots[m.animationFrame%len(dots)]
	return mutedStyle.Render(" " + dot)
}

// centerContent centers content on the screen
func (m *Model) centerContent(content string) string {
	style := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-1). // Leave space for status bar
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render(content)
}

// wrapText wraps text to fit within the specified width
func (m *Model) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine []string
	currentLength := 0

	for _, word := range words {
		wordLength := len(word)

		// If adding this word would exceed the width, start a new line
		if currentLength > 0 && currentLength+wordLength+1 > width {
			lines = append(lines, strings.Join(currentLine, " "))
			currentLine = []string{word}
			currentLength = wordLength
		} else {
			currentLine = append(currentLine, word)
			if currentLength > 0 {
				currentLength += 1 // Space
			}
			currentLength += wordLength
		}
	}

	// Add the last line
	if len(currentLine) > 0 {
		lines = append(lines, strings.Join(currentLine, " "))
	}

	return strings.Join(lines, "\n")
}
