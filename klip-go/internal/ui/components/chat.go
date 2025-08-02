package components

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/john/klip/internal/api"
	"github.com/john/klip/internal/app"
)

// ContextMenu represents a context menu for messages
type ContextMenu struct {
	visible    bool
	x, y       int
	items      []ContextMenuItem
	selected   int
	messageIdx int
}

// ContextMenuItem represents an item in the context menu
type ContextMenuItem struct {
	Label   string
	Action  string
	Hotkey  string
	Enabled bool
}

// ChatViewMsg represents messages for the chat view
type ChatViewMsg struct {
	Type string
	Data interface{}
}

// ChatView component for displaying conversation history
type ChatView struct {
	viewport      viewport.Model
	messages      []api.Message
	streamBuffer  string
	isStreaming   bool
	width         int
	height        int
	showTimestamp bool
	autoScroll    bool
	
	// Enhanced features
	selectedMessage   int
	showLineNumbers   bool
	wordWrap         bool
	maxLineLength    int
	theme            string
	searchHighlight  string
	messageReactions map[int][]string
	contextMenu      *ContextMenu
	exportFormats    []string
}

// NewChatView creates a new chat view component
func NewChatView(width, height int) *ChatView {
	vp := viewport.New(width, height-2) // Leave space for borders
	vp.Style = ChatViewportStyle
	
	return &ChatView{
		viewport:      vp,
		messages:      make([]api.Message, 0),
		width:         width,
		height:        height,
		showTimestamp: false,
		autoScroll:    true,
		
		// Enhanced features
		selectedMessage:  -1,
		showLineNumbers:  false,
		wordWrap:        true,
		maxLineLength:   80,
		theme:           "charm",
		messageReactions: make(map[int][]string),
		exportFormats:   []string{"markdown", "text", "json", "html"},
		contextMenu:     &ContextMenu{},
	}
}

// Init initializes the chat view
func (cv *ChatView) Init() tea.Cmd {
	return nil
}

// Update handles chat view updates
func (cv *ChatView) Update(msg tea.Msg) (*ChatView, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cv.width = msg.Width
		cv.height = msg.Height
		cv.viewport.Width = msg.Width
		cv.viewport.Height = msg.Height - 2
		cv.updateContent()
		
	case ChatViewMsg:
		switch msg.Type {
		case "add_message":
			if apiMsg, ok := msg.Data.(api.Message); ok {
				cv.AddMessage(apiMsg)
			}
		case "stream_chunk":
			if chunk, ok := msg.Data.(string); ok {
				cv.AddStreamChunk(chunk)
			}
		case "stream_start":
			cv.StartStreaming()
		case "stream_end":
			cv.EndStreaming()
		case "clear":
			cv.Clear()
		case "toggle_timestamp":
			cv.ToggleTimestamp()
		case "toggle_line_numbers":
			cv.ToggleLineNumbers()
		case "toggle_word_wrap":
			cv.ToggleWordWrap()
		case "set_theme":
			if theme, ok := msg.Data.(string); ok {
				cv.SetTheme(theme)
			}
		case "search_highlight":
			if query, ok := msg.Data.(string); ok {
				cv.SetSearchHighlight(query)
			}
		case "add_reaction":
			if data, ok := msg.Data.(map[string]interface{}); ok {
				messageIdx := data["message"].(int)
				reaction := data["reaction"].(string)
				cv.AddReaction(messageIdx, reaction)
			}
		case "export_message":
			if data, ok := msg.Data.(map[string]interface{}); ok {
				messageIdx := data["message"].(int)
				format := data["format"].(string)
				return cv, cv.exportMessage(messageIdx, format)
			}
		}
		
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			cv.viewport.LineDown(1)
		case "k", "up":
			cv.viewport.LineUp(1)
		case "d", "pgdown":
			cv.viewport.HalfViewDown()
		case "u", "pgup":
			cv.viewport.HalfViewUp()
		case "g":
			cv.viewport.GotoTop()
		case "G":
			cv.viewport.GotoBottom()
		case "t":
			cv.ToggleTimestamp()
		case "l":
			cv.ToggleLineNumbers()
		case "w":
			cv.ToggleWordWrap()
		case "y":
			if cv.selectedMessage >= 0 && cv.selectedMessage < len(cv.messages) {
				return cv, cv.copyMessage(cv.selectedMessage)
			}
		case "Y":
			return cv, cv.copyAllMessages()
		case "r":
			if cv.selectedMessage >= 0 && cv.selectedMessage < len(cv.messages) {
				cv.showContextMenu(cv.selectedMessage)
			}
		case "enter":
			if cv.contextMenu.visible {
				return cv, cv.executeContextAction()
			} else if cv.selectedMessage >= 0 {
				cv.showContextMenu(cv.selectedMessage)
			}
		case "esc":
			if cv.contextMenu.visible {
				cv.contextMenu.visible = false
			} else {
				cv.selectedMessage = -1
			}
		case "space":
			cv.toggleMessageSelection()
		case "/":
			return cv, cv.startSearch()
		case "n":
			cv.nextSearchResult()
		case "N":
			cv.prevSearchResult()
		default:
			// Handle context menu navigation
			if cv.contextMenu.visible {
				switch msg.String() {
				case "up", "k":
					cv.navigateContextMenu(-1)
				case "down", "j":
					cv.navigateContextMenu(1)
				default:
					cv.viewport, cmd = cv.viewport.Update(msg)
				}
			} else {
				cv.viewport, cmd = cv.viewport.Update(msg)
			}
		}
	default:
		cv.viewport, cmd = cv.viewport.Update(msg)
	}
	
	return cv, cmd
}

// View renders the chat view
func (cv *ChatView) View() string {
	return ChatContainerStyle.Render(cv.viewport.View())
}

// AddMessage adds a new message to the chat
func (cv *ChatView) AddMessage(msg api.Message) {
	cv.messages = append(cv.messages, msg)
	cv.updateContent()
	if cv.autoScroll {
		cv.viewport.GotoBottom()
	}
}

// AddStreamChunk adds a chunk to the streaming buffer
func (cv *ChatView) AddStreamChunk(chunk string) {
	cv.streamBuffer += chunk
	cv.updateContent()
	if cv.autoScroll {
		cv.viewport.GotoBottom()
	}
}

// StartStreaming begins streaming mode
func (cv *ChatView) StartStreaming() {
	cv.isStreaming = true
	cv.streamBuffer = ""
	cv.updateContent()
}

// EndStreaming ends streaming mode and finalizes the message
func (cv *ChatView) EndStreaming() {
	if cv.isStreaming && cv.streamBuffer != "" {
		msg := api.Message{
			Role:      "assistant",
			Content:   cv.streamBuffer,
			Timestamp: time.Now(),
		}
		cv.messages = append(cv.messages, msg)
	}
	cv.isStreaming = false
	cv.streamBuffer = ""
	cv.updateContent()
}

// Clear clears all messages
func (cv *ChatView) Clear() {
	cv.messages = make([]api.Message, 0)
	cv.streamBuffer = ""
	cv.isStreaming = false
	cv.updateContent()
}

// ToggleTimestamp toggles timestamp display
func (cv *ChatView) ToggleTimestamp() {
	cv.showTimestamp = !cv.showTimestamp
	cv.updateContent()
}

// SetMessages sets the messages directly
func (cv *ChatView) SetMessages(messages []api.Message) {
	cv.messages = messages
	cv.updateContent()
	if cv.autoScroll {
		cv.viewport.GotoBottom()
	}
}

// GetMessages returns current messages
func (cv *ChatView) GetMessages() []api.Message {
	return cv.messages
}

// updateContent updates the viewport content
func (cv *ChatView) updateContent() {
	var content strings.Builder
	
	for i, msg := range cv.messages {
		rendered := cv.renderMessage(msg, i == len(cv.messages)-1)
		content.WriteString(rendered)
		if i < len(cv.messages)-1 {
			content.WriteString("\n")
		}
	}
	
	// Add streaming content if active
	if cv.isStreaming && cv.streamBuffer != "" {
		if len(cv.messages) > 0 {
			content.WriteString("\n")
		}
		streamMsg := api.Message{
			Role:      "assistant",
			Content:   cv.streamBuffer,
			Timestamp: time.Now(),
		}
		content.WriteString(cv.renderMessage(streamMsg, true))
		content.WriteString(StreamingIndicatorStyle.Render(" ▋"))
	}
	
	cv.viewport.SetContent(content.String())
}

// renderMessage renders a single message with appropriate styling
func (cv *ChatView) renderMessage(msg api.Message, isLast bool) string {
	var content strings.Builder
	
	// Message header
	header := cv.renderMessageHeader(msg)
	content.WriteString(header)
	content.WriteString("\n")
	
	// Message content with syntax highlighting
	renderedContent := cv.renderMessageContent(msg.Content, msg.Role)
	content.WriteString(renderedContent)
	
	if !isLast {
		content.WriteString("\n")
	}
	
	return content.String()
}

// renderMessageHeader renders the message header with role and timestamp
func (cv *ChatView) renderMessageHeader(msg api.Message) string {
	var header strings.Builder
	
	// Role indicator
	switch msg.Role {
	case "user":
		header.WriteString(UserMessageHeaderStyle.Render("You"))
	case "assistant":
		header.WriteString(AssistantMessageHeaderStyle.Render("Assistant"))
	case "system":
		header.WriteString(SystemMessageHeaderStyle.Render("System"))
	default:
		header.WriteString(DefaultMessageHeaderStyle.Render(strings.Title(msg.Role)))
	}
	
	// Timestamp if enabled
	if cv.showTimestamp {
		timestamp := msg.Timestamp.Format("15:04:05")
		header.WriteString(" ")
		header.WriteString(TimestampStyle.Render(timestamp))
	}
	
	return header.String()
}

// renderMessageContent renders message content with syntax highlighting
func (cv *ChatView) renderMessageContent(content, role string) string {
	var result strings.Builder
	
	// Apply role-specific styling
	var baseStyle lipgloss.Style
	switch role {
	case "user":
		baseStyle = UserMessageStyle
	case "assistant":
		baseStyle = AssistantMessageStyle
	case "system":
		baseStyle = SystemMessageStyle
	default:
		baseStyle = DefaultMessageStyle
	}
	
	// Split content into lines for processing
	lines := strings.Split(content, "\n")
	inCodeBlock := false
	codeBlockLang := ""
	
	for i, line := range lines {
		if cv.isCodeBlockDelimiter(line) {
			if !inCodeBlock {
				// Starting code block
				inCodeBlock = true
				codeBlockLang = cv.extractCodeLanguage(line)
				result.WriteString(CodeBlockDelimiterStyle.Render(line))
			} else {
				// Ending code block
				inCodeBlock = false
				codeBlockLang = ""
				result.WriteString(CodeBlockDelimiterStyle.Render(line))
			}
		} else if inCodeBlock {
			// Code content
			highlighted := cv.highlightCode(line, codeBlockLang)
			result.WriteString(CodeBlockStyle.Render(highlighted))
		} else if cv.isInlineCode(line) {
			// Inline code
			highlighted := cv.highlightInlineCode(line)
			result.WriteString(baseStyle.Render(highlighted))
		} else {
			// Regular text
			result.WriteString(baseStyle.Render(line))
		}
		
		// Add newline if not last line
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// isCodeBlockDelimiter checks if a line is a code block delimiter
func (cv *ChatView) isCodeBlockDelimiter(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "```")
}

// extractCodeLanguage extracts the language from a code block delimiter
func (cv *ChatView) extractCodeLanguage(line string) string {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) > 3 {
		return strings.TrimSpace(trimmed[3:])
	}
	return ""
}

// isInlineCode checks if a line contains inline code
func (cv *ChatView) isInlineCode(line string) bool {
	return strings.Contains(line, "`")
}

// highlightCode applies basic syntax highlighting to code
func (cv *ChatView) highlightCode(code, lang string) string {
	// Basic syntax highlighting based on language
	switch strings.ToLower(lang) {
	case "go", "golang":
		return cv.highlightGo(code)
	case "python", "py":
		return cv.highlightPython(code)
	case "javascript", "js", "typescript", "ts":
		return cv.highlightJavaScript(code)
	case "json":
		return cv.highlightJSON(code)
	case "bash", "sh", "shell":
		return cv.highlightBash(code)
	default:
		return code
	}
}

// highlightInlineCode highlights inline code snippets
func (cv *ChatView) highlightInlineCode(line string) string {
	// Simple inline code highlighting with backticks
	re := regexp.MustCompile("`([^`]+)`")
	return re.ReplaceAllStringFunc(line, func(match string) string {
		code := strings.Trim(match, "`")
		return InlineCodeStyle.Render("`" + code + "`")
	})
}

// Basic syntax highlighters
func (cv *ChatView) highlightGo(code string) string {
	// Go keywords
	keywords := []string{"package", "import", "func", "var", "const", "type", "struct", "interface",
		"if", "else", "for", "range", "switch", "case", "default", "return", "break", "continue",
		"go", "defer", "select", "chan", "map", "make", "new", "len", "cap", "append", "copy"}
	
	result := code
	for _, keyword := range keywords {
		re := regexp.MustCompile(`\b` + keyword + `\b`)
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			return KeywordStyle.Render(match)
		})
	}
	
	// String literals
	stringRe := regexp.MustCompile(`"[^"]*"`)
	result = stringRe.ReplaceAllStringFunc(result, func(match string) string {
		return StringStyle.Render(match)
	})
	
	// Comments
	commentRe := regexp.MustCompile(`//.*$`)
	result = commentRe.ReplaceAllStringFunc(result, func(match string) string {
		return CommentStyle.Render(match)
	})
	
	return result
}

func (cv *ChatView) highlightPython(code string) string {
	keywords := []string{"def", "class", "import", "from", "if", "elif", "else", "for", "while",
		"try", "except", "finally", "with", "as", "return", "yield", "break", "continue",
		"pass", "lambda", "and", "or", "not", "is", "in"}
	
	result := code
	for _, keyword := range keywords {
		re := regexp.MustCompile(`\b` + keyword + `\b`)
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			return KeywordStyle.Render(match)
		})
	}
	
	// String literals
	stringRe := regexp.MustCompile(`(['"])[^'"]*\1`)
	result = stringRe.ReplaceAllStringFunc(result, func(match string) string {
		return StringStyle.Render(match)
	})
	
	// Comments
	commentRe := regexp.MustCompile(`#.*$`)
	result = commentRe.ReplaceAllStringFunc(result, func(match string) string {
		return CommentStyle.Render(match)
	})
	
	return result
}

func (cv *ChatView) highlightJavaScript(code string) string {
	keywords := []string{"function", "var", "let", "const", "if", "else", "for", "while", "do",
		"switch", "case", "default", "break", "continue", "return", "try", "catch", "finally",
		"throw", "new", "this", "class", "extends", "import", "export", "from"}
	
	result := code
	for _, keyword := range keywords {
		re := regexp.MustCompile(`\b` + keyword + `\b`)
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			return KeywordStyle.Render(match)
		})
	}
	
	// String literals
	stringRe := regexp.MustCompile(`(['"][^'"]*['"])`)
	result = stringRe.ReplaceAllStringFunc(result, func(match string) string {
		return StringStyle.Render(match)
	})
	
	// Comments
	commentRe := regexp.MustCompile(`//.*$`)
	result = commentRe.ReplaceAllStringFunc(result, func(match string) string {
		return CommentStyle.Render(match)
	})
	
	return result
}

func (cv *ChatView) highlightJSON(code string) string {
	// JSON strings
	stringRe := regexp.MustCompile(`"[^"]*"`)
	result := stringRe.ReplaceAllStringFunc(code, func(match string) string {
		return StringStyle.Render(match)
	})
	
	// JSON numbers
	numberRe := regexp.MustCompile(`\b\d+\.?\d*\b`)
	result = numberRe.ReplaceAllStringFunc(result, func(match string) string {
		return NumberStyle.Render(match)
	})
	
	// JSON booleans and null
	boolRe := regexp.MustCompile(`\b(true|false|null)\b`)
	result = boolRe.ReplaceAllStringFunc(result, func(match string) string {
		return BooleanStyle.Render(match)
	})
	
	return result
}

func (cv *ChatView) highlightBash(code string) string {
	// Bash commands and keywords
	keywords := []string{"if", "then", "else", "elif", "fi", "for", "do", "done", "while",
		"case", "esac", "function", "echo", "cd", "ls", "grep", "awk", "sed", "cat"}
	
	result := code
	for _, keyword := range keywords {
		re := regexp.MustCompile(`\b` + keyword + `\b`)
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			return KeywordStyle.Render(match)
		})
	}
	
	// String literals
	stringRe := regexp.MustCompile(`(['"])[^'"]*\1`)
	result = stringRe.ReplaceAllStringFunc(result, func(match string) string {
		return StringStyle.Render(match)
	})
	
	// Comments
	commentRe := regexp.MustCompile(`#.*$`)
	result = commentRe.ReplaceAllStringFunc(result, func(match string) string {
		return CommentStyle.Render(match)
	})
	
	return result
}

// Styling for chat components
var (
	// Chat container styles
	ChatContainerStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(1)

	ChatViewportStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1)

	// Message header styles
	UserMessageHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Bold(true).
		PaddingLeft(1)

	AssistantMessageHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true).
		PaddingLeft(1)

	SystemMessageHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Bold(true).
		PaddingLeft(1)

	DefaultMessageHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151")).
		Bold(true).
		PaddingLeft(1)

	TimestampStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Faint(true)

	// Message content styles
	UserMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1F2937")).
		PaddingLeft(2).
		PaddingRight(1)

	AssistantMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1F2937")).
		PaddingLeft(2).
		PaddingRight(1)

	SystemMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true).
		PaddingLeft(2).
		PaddingRight(1)

	DefaultMessageStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151")).
		PaddingLeft(2).
		PaddingRight(1)

	// Code highlighting styles
	CodeBlockStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#F3F4F6")).
		Foreground(lipgloss.Color("#1F2937")).
		Padding(0, 1).
		MarginLeft(2)

	CodeBlockDelimiterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		MarginLeft(2)

	InlineCodeStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#F3F4F6")).
		Foreground(lipgloss.Color("#DC2626")).
		Padding(0, 1)

	// Syntax highlighting styles
	KeywordStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	StringStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#059669"))

	CommentStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Italic(true)

	NumberStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DC2626"))

	BooleanStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C2D12")).
		Bold(true)

	// Streaming indicator
	StreamingIndicatorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Blink(true)
)

// Helper functions for integration with app state
func NewChatViewFromState(state *app.ChatState, width, height int) *ChatView {
	cv := NewChatView(width, height)
	
	// Convert app messages to API messages
	apiMessages := make([]api.Message, len(state.Messages))
	for i, msg := range state.Messages {
		apiMessages[i] = msg
	}
	
	cv.SetMessages(apiMessages)
	
	if state.IsStreaming {
		cv.StartStreaming()
		cv.AddStreamChunk(state.StreamBuffer)
	}
	
	return cv
}

// UpdateFromState updates the chat view from app state
func (cv *ChatView) UpdateFromState(state *app.ChatState) {
	// Convert app messages to API messages
	apiMessages := make([]api.Message, len(state.Messages))
	for i, msg := range state.Messages {
		apiMessages[i] = msg
	}
	
	cv.SetMessages(apiMessages)
	
	if state.IsStreaming {
		if !cv.isStreaming {
			cv.StartStreaming()
		}
		cv.streamBuffer = state.StreamBuffer
		cv.updateContent()
	} else if cv.isStreaming {
		cv.EndStreaming()
	}
}

// Enhanced feature methods

// ToggleLineNumbers toggles line number display
func (cv *ChatView) ToggleLineNumbers() {
	cv.showLineNumbers = !cv.showLineNumbers
	cv.updateContent()
}

// ToggleWordWrap toggles word wrapping
func (cv *ChatView) ToggleWordWrap() {
	cv.wordWrap = !cv.wordWrap
	cv.updateContent()
}

// SetTheme sets the chat theme
func (cv *ChatView) SetTheme(theme string) {
	cv.theme = theme
	cv.updateContent()
}

// SetSearchHighlight sets the search highlight query
func (cv *ChatView) SetSearchHighlight(query string) {
	cv.searchHighlight = query
	cv.updateContent()
}

// AddReaction adds a reaction to a message
func (cv *ChatView) AddReaction(messageIdx int, reaction string) {
	if messageIdx >= 0 && messageIdx < len(cv.messages) {
		if cv.messageReactions[messageIdx] == nil {
			cv.messageReactions[messageIdx] = make([]string, 0)
		}
		cv.messageReactions[messageIdx] = append(cv.messageReactions[messageIdx], reaction)
		cv.updateContent()
	}
}

// toggleMessageSelection toggles message selection
func (cv *ChatView) toggleMessageSelection() {
	// Calculate which message is at the current viewport position
	// This is a simplified implementation
	if cv.selectedMessage == -1 {
		cv.selectedMessage = 0
	} else {
		cv.selectedMessage = -1
	}
	cv.updateContent()
}

// showContextMenu shows the context menu for a message
func (cv *ChatView) showContextMenu(messageIdx int) {
	cv.contextMenu.visible = true
	cv.contextMenu.messageIdx = messageIdx
	cv.contextMenu.selected = 0
	cv.contextMenu.items = []ContextMenuItem{
		{Label: "Copy Message", Action: "copy", Hotkey: "y", Enabled: true},
		{Label: "Add Reaction", Action: "react", Hotkey: "r", Enabled: true},
		{Label: "Export Message", Action: "export", Hotkey: "e", Enabled: true},
		{Label: "Reply to Message", Action: "reply", Hotkey: "R", Enabled: true},
		{Label: "Edit Message", Action: "edit", Hotkey: "E", Enabled: messageIdx < len(cv.messages) && cv.messages[messageIdx].Role == "user"},
	}
}

// navigateContextMenu navigates the context menu
func (cv *ChatView) navigateContextMenu(direction int) {
	if !cv.contextMenu.visible {
		return
	}
	
	newSelected := cv.contextMenu.selected + direction
	if newSelected < 0 {
		newSelected = len(cv.contextMenu.items) - 1
	} else if newSelected >= len(cv.contextMenu.items) {
		newSelected = 0
	}
	cv.contextMenu.selected = newSelected
}

// executeContextAction executes the selected context menu action
func (cv *ChatView) executeContextAction() tea.Cmd {
	if !cv.contextMenu.visible || cv.contextMenu.selected >= len(cv.contextMenu.items) {
		return nil
	}
	
	action := cv.contextMenu.items[cv.contextMenu.selected].Action
	messageIdx := cv.contextMenu.messageIdx
	cv.contextMenu.visible = false
	
	switch action {
	case "copy":
		return cv.copyMessage(messageIdx)
	case "react":
		return cv.addReactionPrompt(messageIdx)
	case "export":
		return cv.exportMessage(messageIdx, "markdown")
	case "reply":
		return cv.replyToMessage(messageIdx)
	case "edit":
		return cv.editMessage(messageIdx)
	}
	
	return nil
}

// Message action commands
func (cv *ChatView) copyMessage(messageIdx int) tea.Cmd {
	return func() tea.Msg {
		if messageIdx >= 0 && messageIdx < len(cv.messages) {
			return ChatViewMsg{Type: "copy_message", Data: cv.messages[messageIdx].Content}
		}
		return nil
	}
}

func (cv *ChatView) copyAllMessages() tea.Cmd {
	return func() tea.Msg {
		var content strings.Builder
		for i, msg := range cv.messages {
			content.WriteString(fmt.Sprintf("[%s] %s\n", strings.Title(msg.Role), msg.Content))
			if i < len(cv.messages)-1 {
				content.WriteString("\n")
			}
		}
		return ChatViewMsg{Type: "copy_all", Data: content.String()}
	}
}

func (cv *ChatView) addReactionPrompt(messageIdx int) tea.Cmd {
	return func() tea.Msg {
		return ChatViewMsg{Type: "reaction_prompt", Data: messageIdx}
	}
}

func (cv *ChatView) exportMessage(messageIdx int, format string) tea.Cmd {
	return func() tea.Msg {
		if messageIdx >= 0 && messageIdx < len(cv.messages) {
			return ChatViewMsg{Type: "export_message", Data: map[string]interface{}{
				"message": cv.messages[messageIdx],
				"format":  format,
			}}
		}
		return nil
	}
}

func (cv *ChatView) replyToMessage(messageIdx int) tea.Cmd {
	return func() tea.Msg {
		if messageIdx >= 0 && messageIdx < len(cv.messages) {
			return ChatViewMsg{Type: "reply_to_message", Data: cv.messages[messageIdx]}
		}
		return nil
	}
}

func (cv *ChatView) editMessage(messageIdx int) tea.Cmd {
	return func() tea.Msg {
		if messageIdx >= 0 && messageIdx < len(cv.messages) {
			return ChatViewMsg{Type: "edit_message", Data: cv.messages[messageIdx]}
		}
		return nil
	}
}

// Search functionality
func (cv *ChatView) startSearch() tea.Cmd {
	return func() tea.Msg {
		return ChatViewMsg{Type: "start_search"}
	}
}

func (cv *ChatView) nextSearchResult() {
	// Implementation for next search result
	// This would highlight the next occurrence of the search term
}

func (cv *ChatView) prevSearchResult() {
	// Implementation for previous search result
	// This would highlight the previous occurrence of the search term
}

// Enhanced styles for new features
var (
	// Message selection styles
	MessageSelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	LineNumberStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Faint(true)

	ReactionsStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		PaddingLeft(2)

	// Context menu styles
	ContextMenuStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Background(lipgloss.Color("#FFFFFF")).
		Padding(1)

	ContextMenuItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#374151")).
		PaddingLeft(1)

	ContextMenuSelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Background(lipgloss.Color("#F3F4F6")).
		Bold(true).
		PaddingLeft(1)

	// Search highlight style
	SearchHighlightStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("#FEF3C7")).
		Foreground(lipgloss.Color("#92400E"))
)