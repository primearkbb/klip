package styles

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ComponentStyler provides styled components for the UI
type ComponentStyler struct {
	theme  *Theme
	width  int
	height int
}

// NewComponentStyler creates a new component styler
func NewComponentStyler(theme *Theme, width, height int) *ComponentStyler {
	return &ComponentStyler{
		theme:  theme,
		width:  width,
		height: height,
	}
}

// Button Styles

// ButtonStyle represents different button variants
type ButtonStyle int

const (
	ButtonPrimary ButtonStyle = iota
	ButtonSecondary
	ButtonSuccess
	ButtonError
	ButtonWarning
	ButtonInfo
	ButtonGhost
	ButtonOutline
	ButtonLink
)

// ButtonSize represents button size variants
type ButtonSize int

const (
	ButtonSizeSmall ButtonSize = iota
	ButtonSizeMedium
	ButtonSizeLarge
)

// ButtonState represents button interaction states
type ButtonState int

const (
	ButtonStateNormal ButtonState = iota
	ButtonStateHover
	ButtonStateFocus
	ButtonStateActive
	ButtonStateDisabled
	ButtonStateLoading
)

// Button creates a styled button
func (cs *ComponentStyler) Button(text string, variant ButtonStyle, size ButtonSize, state ButtonState) string {
	baseStyle := lipgloss.NewStyle().
		Padding(cs.getButtonPadding(size)...).
		Align(lipgloss.Center).
		Bold(true)
	
	// Apply variant styling
	switch variant {
	case ButtonPrimary:
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.TextInverse).
			Background(cs.theme.Colors.Primary)
		if state == ButtonStateHover {
			baseStyle = baseStyle.Background(cs.theme.Colors.PrimaryDark)
		} else if state == ButtonStateFocus {
			baseStyle = baseStyle.
				Background(cs.theme.Colors.Primary).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(cs.theme.Colors.PrimaryLight)
		}
		
	case ButtonSecondary:
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.Text).
			Background(cs.theme.Colors.Secondary)
		if state == ButtonStateHover {
			baseStyle = baseStyle.Background(cs.theme.Colors.SecondaryDark)
		}
		
	case ButtonSuccess:
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.TextInverse).
			Background(cs.theme.Colors.Success)
		if state == ButtonStateHover {
			baseStyle = baseStyle.Background(cs.theme.Colors.SuccessDark)
		}
		
	case ButtonError:
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.TextInverse).
			Background(cs.theme.Colors.Error)
		if state == ButtonStateHover {
			baseStyle = baseStyle.Background(cs.theme.Colors.ErrorDark)
		}
		
	case ButtonWarning:
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.TextInverse).
			Background(cs.theme.Colors.Warning)
		if state == ButtonStateHover {
			baseStyle = baseStyle.Background(cs.theme.Colors.WarningDark)
		}
		
	case ButtonInfo:
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.TextInverse).
			Background(cs.theme.Colors.Info)
		if state == ButtonStateHover {
			baseStyle = baseStyle.Background(cs.theme.Colors.InfoDark)
		}
		
	case ButtonGhost:
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.Primary).
			Background(lipgloss.Color(""))
		if state == ButtonStateHover {
			baseStyle = baseStyle.Background(cs.theme.Colors.BackgroundSubtle)
		}
		
	case ButtonOutline:
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.Primary).
			Background(lipgloss.Color("")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cs.theme.Colors.Primary)
		if state == ButtonStateHover {
			baseStyle = baseStyle.
				Foreground(cs.theme.Colors.TextInverse).
				Background(cs.theme.Colors.Primary)
		}
		
	case ButtonLink:
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.Primary).
			Background(lipgloss.Color("")).
			Underline(true).
			Padding(0)
		if state == ButtonStateHover {
			baseStyle = baseStyle.Foreground(cs.theme.Colors.PrimaryDark)
		}
	}
	
	// Handle disabled state
	if state == ButtonStateDisabled {
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.TextMuted).
			Background(cs.theme.Colors.BackgroundSubtle)
	}
	
	// Handle loading state
	if state == ButtonStateLoading {
		text = cs.Spinner("small") + " " + text
	}
	
	return baseStyle.Render(text)
}

func (cs *ComponentStyler) getButtonPadding(size ButtonSize) []int {
	switch size {
	case ButtonSizeSmall:
		return []int{0, 1}
	case ButtonSizeMedium:
		return []int{1, 2}
	case ButtonSizeLarge:
		return []int{1, 3}
	default:
		return []int{1, 2}
	}
}

// Input Styles

// InputType represents different input variants
type InputType int

const (
	InputTypeText InputType = iota
	InputTypePassword
	InputTypeEmail
	InputTypeNumber
	InputTypeSearch
	InputTypeTextarea
)

// InputState represents input interaction states
type InputState int

const (
	InputStateNormal InputState = iota
	InputStateFocus
	InputStateError
	InputStateSuccess
	InputStateDisabled
)

// Input creates a styled input field
func (cs *ComponentStyler) Input(value, placeholder string, inputType InputType, state InputState, width int) string {
	if width <= 0 {
		width = 30 // Default width
	}
	
	baseStyle := lipgloss.NewStyle().
		Width(width).
		Padding(cs.theme.Spacing.ButtonPadding[0], cs.theme.Spacing.ButtonPadding[1]).
		Background(cs.theme.Colors.Surface).
		Foreground(cs.theme.Colors.Text).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cs.theme.Colors.Border)
	
	// Apply state styling
	switch state {
	case InputStateFocus:
		baseStyle = baseStyle.
			BorderForeground(cs.theme.Colors.Primary)
		
	case InputStateError:
		baseStyle = baseStyle.
			BorderForeground(cs.theme.Colors.Error).
			Background(cs.theme.Colors.BackgroundSubtle)
		
	case InputStateSuccess:
		baseStyle = baseStyle.
			BorderForeground(cs.theme.Colors.Success)
		
	case InputStateDisabled:
		baseStyle = baseStyle.
			Foreground(cs.theme.Colors.TextMuted).
			Background(cs.theme.Colors.BackgroundSubtle)
	}
	
	// Handle content
	content := value
	if content == "" && placeholder != "" {
		content = placeholder
		if state != InputStateDisabled {
			baseStyle = baseStyle.Foreground(cs.theme.Colors.TextMuted)
		}
	}
	
	// Handle password masking
	if inputType == InputTypePassword && value != "" {
		content = strings.Repeat("•", len(value))
	}
	
	// Handle textarea (multiline)
	if inputType == InputTypeTextarea {
		lines := strings.Split(content, "\n")
		height := len(lines)
		if height < 3 {
			height = 3
		}
		baseStyle = baseStyle.Height(height)
	}
	
	return baseStyle.Render(content)
}

// Label creates a styled label
func (cs *ComponentStyler) Label(text string, required bool) string {
	style := lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Text).
		Bold(true).
		MarginBottom(1)
	
	if required {
		text += lipgloss.NewStyle().
			Foreground(cs.theme.Colors.Error).
			Render(" *")
	}
	
	return style.Render(text)
}

// FormField creates a complete form field with label and input
func (cs *ComponentStyler) FormField(label, value, placeholder string, inputType InputType, state InputState, required bool, width int) string {
	var parts []string
	
	// Label
	if label != "" {
		parts = append(parts, cs.Label(label, required))
	}
	
	// Input
	parts = append(parts, cs.Input(value, placeholder, inputType, state, width))
	
	// Error message (if error state)
	if state == InputStateError {
		errorMsg := cs.ErrorMessage("Invalid input")
		parts = append(parts, errorMsg)
	}
	
	return strings.Join(parts, "\n")
}

// Chat Bubble Styles

// ChatBubbleType represents different chat bubble variants
type ChatBubbleType int

const (
	ChatBubbleUser ChatBubbleType = iota
	ChatBubbleAssistant
	ChatBubbleSystem
	ChatBubbleError
)

// ChatBubble creates a styled chat message bubble
func (cs *ComponentStyler) ChatBubble(content, author string, bubbleType ChatBubbleType, timestamp string) string {
	maxWidth := cs.width - 10
	if maxWidth < 30 {
		maxWidth = 30
	}
	
	var bubbleStyle lipgloss.Style
	var authorStyle lipgloss.Style
	
	// Configure styles based on bubble type
	switch bubbleType {
	case ChatBubbleUser:
		bubbleStyle = lipgloss.NewStyle().
			MaxWidth(maxWidth).
			Padding(1, 2).
			Background(cs.theme.Colors.Primary).
			Foreground(cs.theme.Colors.TextInverse).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cs.theme.Colors.Primary).
			Align(lipgloss.Right).
			MarginLeft(10)
		
		authorStyle = lipgloss.NewStyle().
			Foreground(cs.theme.Colors.Primary).
			Bold(true).
			Align(lipgloss.Right)
		
	case ChatBubbleAssistant:
		bubbleStyle = lipgloss.NewStyle().
			MaxWidth(maxWidth).
			Padding(1, 2).
			Background(cs.theme.Colors.Surface).
			Foreground(cs.theme.Colors.Text).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cs.theme.Colors.Border).
			MarginRight(10)
		
		authorStyle = lipgloss.NewStyle().
			Foreground(cs.theme.Colors.Secondary).
			Bold(true)
		
	case ChatBubbleSystem:
		bubbleStyle = lipgloss.NewStyle().
			MaxWidth(maxWidth).
			Padding(1, 2).
			Background(cs.theme.Colors.BackgroundSubtle).
			Foreground(cs.theme.Colors.TextMuted).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cs.theme.Colors.BorderSubtle).
			Align(lipgloss.Center)
		
		authorStyle = lipgloss.NewStyle().
			Foreground(cs.theme.Colors.TextMuted).
			Bold(true).
			Align(lipgloss.Center)
		
	case ChatBubbleError:
		bubbleStyle = lipgloss.NewStyle().
			MaxWidth(maxWidth).
			Padding(1, 2).
			Background(cs.theme.Colors.ErrorLight).
			Foreground(cs.theme.Colors.Error).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cs.theme.Colors.Error)
		
		authorStyle = lipgloss.NewStyle().
			Foreground(cs.theme.Colors.Error).
			Bold(true)
	}
	
	var parts []string
	
	// Author and timestamp header
	if author != "" {
		header := author
		if timestamp != "" {
			header += lipgloss.NewStyle().
				Foreground(cs.theme.Colors.TextMuted).
				Render(" • " + timestamp)
		}
		parts = append(parts, authorStyle.Render(header))
	}
	
	// Message content
	parts = append(parts, bubbleStyle.Render(content))
	
	return strings.Join(parts, "\n")
}

// List Styles

// ListType represents different list variants
type ListType int

const (
	ListTypeUnordered ListType = iota
	ListTypeOrdered
	ListTypeChecklist
	ListTypeMenu
)

// ListItemState represents list item states
type ListItemState int

const (
	ListItemStateNormal ListItemState = iota
	ListItemStateSelected
	ListItemStateDisabled
	ListItemStateChecked
)

// List creates a styled list
func (cs *ComponentStyler) List(items []string, listType ListType, selectedIndex int) string {
	if len(items) == 0 {
		return ""
	}
	
	var styledItems []string
	
	for i, item := range items {
		state := ListItemStateNormal
		if i == selectedIndex {
			state = ListItemStateSelected
		}
		
		styledItem := cs.ListItem(item, listType, state, i)
		styledItems = append(styledItems, styledItem)
	}
	
	return strings.Join(styledItems, "\n")
}

// ListItem creates a styled list item
func (cs *ComponentStyler) ListItem(text string, listType ListType, state ListItemState, index int) string {
	var prefix string
	var itemStyle lipgloss.Style
	
	// Base item style
	itemStyle = lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Text).
		Padding(0, 1)
	
	// Configure prefix and styling based on list type
	switch listType {
	case ListTypeUnordered:
		prefix = "• "
		
	case ListTypeOrdered:
		prefix = fmt.Sprintf("%d. ", index+1)
		
	case ListTypeChecklist:
		if state == ListItemStateChecked {
			prefix = "☑ "
		} else {
			prefix = "☐ "
		}
		
	case ListTypeMenu:
		if state == ListItemStateSelected {
			prefix = "▶ "
		} else {
			prefix = "  "
		}
	}
	
	// Apply state styling
	switch state {
	case ListItemStateSelected:
		itemStyle = itemStyle.
			Background(cs.theme.Colors.Selection).
			Foreground(cs.theme.Colors.Primary).
			Bold(true)
		
	case ListItemStateDisabled:
		itemStyle = itemStyle.
			Foreground(cs.theme.Colors.TextMuted)
		
	case ListItemStateChecked:
		itemStyle = itemStyle.
			Foreground(cs.theme.Colors.Success)
	}
	
	return itemStyle.Render(prefix + text)
}

// Card Styles

// CardType represents different card variants
type CardType int

const (
	CardTypeDefault CardType = iota
	CardTypeElevated
	CardTypeOutlined
	CardTypeSubtle
)

// Card creates a styled card container
func (cs *ComponentStyler) Card(title, content string, cardType CardType, width int) string {
	if width <= 0 {
		width = cs.width - 4
	}
	
	var cardStyle lipgloss.Style
	
	// Base card style
	cardStyle = lipgloss.NewStyle().
		Width(width).
		Padding(cs.theme.Spacing.CardPadding[0], cs.theme.Spacing.CardPadding[1]).
		Border(lipgloss.RoundedBorder())
	
	// Apply card type styling
	switch cardType {
	case CardTypeDefault:
		cardStyle = cardStyle.
			Background(cs.theme.Colors.Surface).
			BorderForeground(cs.theme.Colors.Border)
		
	case CardTypeElevated:
		cardStyle = cardStyle.
			Background(cs.theme.Colors.Surface).
			BorderForeground(cs.theme.Colors.BorderSubtle)
		// In a real implementation, you'd add shadow effects here
		
	case CardTypeOutlined:
		cardStyle = cardStyle.
			Background(lipgloss.Color("")).
			BorderForeground(cs.theme.Colors.Border)
		
	case CardTypeSubtle:
		cardStyle = cardStyle.
			Background(cs.theme.Colors.BackgroundSubtle).
			BorderForeground(cs.theme.Colors.BorderSubtle)
	}
	
	var parts []string
	
	// Card title
	if title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(cs.theme.Colors.Text).
			Bold(true).
			MarginBottom(1)
		parts = append(parts, titleStyle.Render(title))
	}
	
	// Card content
	if content != "" {
		contentStyle := lipgloss.NewStyle().
			Foreground(cs.theme.Colors.Text)
		parts = append(parts, contentStyle.Render(content))
	}
	
	cardContent := strings.Join(parts, "\n")
	return cardStyle.Render(cardContent)
}

// Progress and Status Styles

// ProgressBar creates a styled progress bar
func (cs *ComponentStyler) ProgressBar(progress float64, width int, showPercentage bool) string {
	if width <= 0 {
		width = 40
	}
	
	// Clamp progress between 0 and 1
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}
	
	filled := int(float64(width) * progress)
	empty := width - filled
	
	filledBar := strings.Repeat("█", filled)
	emptyBar := strings.Repeat("░", empty)
	
	bar := lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Primary).
		Render(filledBar) +
		lipgloss.NewStyle().
			Foreground(cs.theme.Colors.BorderSubtle).
			Render(emptyBar)
	
	if showPercentage {
		percentage := fmt.Sprintf(" %.0f%%", progress*100)
		bar += lipgloss.NewStyle().
			Foreground(cs.theme.Colors.TextMuted).
			Render(percentage)
	}
	
	return bar
}

// StatusIndicator creates a colored status indicator
func (cs *ComponentStyler) StatusIndicator(status, text string) string {
	var color lipgloss.Color
	var symbol string
	
	switch strings.ToLower(status) {
	case "success", "ok", "online", "connected":
		color = cs.theme.Colors.Success
		symbol = "●"
	case "error", "fail", "offline", "disconnected":
		color = cs.theme.Colors.Error
		symbol = "●"
	case "warning", "pending", "connecting":
		color = cs.theme.Colors.Warning
		symbol = "●"
	case "info", "loading":
		color = cs.theme.Colors.Info
		symbol = "●"
	default:
		color = cs.theme.Colors.TextMuted
		symbol = "○"
	}
	
	indicator := lipgloss.NewStyle().
		Foreground(color).
		Render(symbol)
	
	if text != "" {
		indicator += " " + lipgloss.NewStyle().
			Foreground(cs.theme.Colors.Text).
			Render(text)
	}
	
	return indicator
}

// Badge creates a styled badge
func (cs *ComponentStyler) Badge(text string, variant ButtonStyle) string {
	var style lipgloss.Style
	
	switch variant {
	case ButtonPrimary:
		style = lipgloss.NewStyle().
			Foreground(cs.theme.Colors.TextInverse).
			Background(cs.theme.Colors.Primary)
	case ButtonSuccess:
		style = lipgloss.NewStyle().
			Foreground(cs.theme.Colors.TextInverse).
			Background(cs.theme.Colors.Success)
	case ButtonError:
		style = lipgloss.NewStyle().
			Foreground(cs.theme.Colors.TextInverse).
			Background(cs.theme.Colors.Error)
	case ButtonWarning:
		style = lipgloss.NewStyle().
			Foreground(cs.theme.Colors.TextInverse).
			Background(cs.theme.Colors.Warning)
	case ButtonInfo:
		style = lipgloss.NewStyle().
			Foreground(cs.theme.Colors.TextInverse).
			Background(cs.theme.Colors.Info)
	default:
		style = lipgloss.NewStyle().
			Foreground(cs.theme.Colors.Text).
			Background(cs.theme.Colors.BackgroundSubtle)
	}
	
	return style.
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.GetBackground()).
		Render(text)
}

// Utility Components

// Divider creates a styled divider line
func (cs *ComponentStyler) Divider(width int, character string) string {
	if width <= 0 {
		width = cs.width
	}
	if character == "" {
		character = "─"
	}
	
	return lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Border).
		Render(strings.Repeat(character, width))
}

// Spinner creates an animated spinner (simplified)
func (cs *ComponentStyler) Spinner(size string) string {
	// In a real implementation, this would use the Harmonica library for animation
	// For now, return a static spinner character
	var spinner string
	switch size {
	case "small":
		spinner = "⠋"
	case "medium":
		spinner = "◐"
	case "large":
		spinner = "◯"
	default:
		spinner = "⠋"
	}
	
	return lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Primary).
		Render(spinner)
}

// LoadingDots creates animated loading dots
func (cs *ComponentStyler) LoadingDots() string {
	// In a real implementation, this would animate
	return lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Primary).
		Render("...")
}

// Tooltip creates a styled tooltip
func (cs *ComponentStyler) Tooltip(text, tooltip string) string {
	// For terminal UI, we'll show tooltip inline or as a note
	mainText := lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Text).
		Render(text)
	
	tooltipText := lipgloss.NewStyle().
		Foreground(cs.theme.Colors.TextMuted).
		Italic(true).
		Render(" (" + tooltip + ")")
	
	return mainText + tooltipText
}

// Message Types

// ErrorMessage creates a styled error message
func (cs *ComponentStyler) ErrorMessage(text string) string {
	return lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Error).
		Background(cs.theme.Colors.BackgroundSubtle).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cs.theme.Colors.Error).
		Render("❌ " + text)
}

// SuccessMessage creates a styled success message
func (cs *ComponentStyler) SuccessMessage(text string) string {
	return lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Success).
		Background(cs.theme.Colors.BackgroundSubtle).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cs.theme.Colors.Success).
		Render("✅ " + text)
}

// WarningMessage creates a styled warning message
func (cs *ComponentStyler) WarningMessage(text string) string {
	return lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Warning).
		Background(cs.theme.Colors.BackgroundSubtle).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cs.theme.Colors.Warning).
		Render("⚠️ " + text)
}

// InfoMessage creates a styled info message
func (cs *ComponentStyler) InfoMessage(text string) string {
	return lipgloss.NewStyle().
		Foreground(cs.theme.Colors.Info).
		Background(cs.theme.Colors.BackgroundSubtle).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cs.theme.Colors.Info).
		Render("ℹ️ " + text)
}