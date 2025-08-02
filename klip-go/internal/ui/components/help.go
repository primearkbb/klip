package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/john/klip/internal/app"
)

// HelpMsg represents messages for the help component
type HelpMsg struct {
	Type string
	Data interface{}
}

// HelpSection represents different help sections
type HelpSection int

const (
	HelpSectionOverview HelpSection = iota
	HelpSectionCommands
	HelpSectionKeybindings
	HelpSectionTutorial
	HelpSectionFAQ
	HelpSectionTroubleshooting
)

// HelpItem represents an item in the help system
type HelpItem struct {
	title       string
	description string
	content     string
	category    string
	keywords    []string
	highlighted bool
}

// Implement list.Item interface
func (hi HelpItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s",
		hi.title,
		hi.description,
		hi.category,
		strings.Join(hi.keywords, " "))
}

func (hi HelpItem) Title() string       { return hi.title }
func (hi HelpItem) Description() string { return hi.description }

// TutorialStep represents a step in the tutorial
type TutorialStep struct {
	Title       string
	Description string
	Command     string
	Expected    string
	Tips        []string
}

// InteractiveHelp manages the help system with search and navigation
type InteractiveHelp struct {
	width  int
	height int

	// UI components
	list        list.Model
	searchInput textinput.Model
	viewport    viewport.Model

	// State
	currentSection HelpSection
	selectedItem   *HelpItem
	items          []list.Item
	searchMode     bool
	tutorialStep   int

	// Help content
	helpItems    []HelpItem
	tutorialMode bool
}

// NewInteractiveHelp creates a new interactive help system
func NewInteractiveHelp(width, height int) *InteractiveHelp {
	ih := &InteractiveHelp{
		width:  width,
		height: height,
	}

	// Initialize help items
	ih.helpItems = ih.generateHelpItems()

	// Convert to list items
	for _, item := range ih.helpItems {
		ih.items = append(ih.items, item)
	}

	// Initialize list
	d := NewHelpDelegate()
	ih.list = list.New(ih.items, d, width/2, height-4)
	ih.list.Title = "Help Topics"
	ih.list.SetShowStatusBar(true)
	ih.list.SetFilteringEnabled(true)

	// Initialize search input
	ih.searchInput = textinput.New()
	ih.searchInput.Placeholder = "Search help topics..."
	ih.searchInput.Width = width/2 - 4

	// Initialize viewport
	ih.viewport = viewport.New(width/2-2, height-4)
	ih.viewport.SetContent("Select a help topic to view details")

	return ih
}

// generateHelpItems creates all help content
func (ih *InteractiveHelp) generateHelpItems() []HelpItem {
	return []HelpItem{
		{
			title:       "Getting Started",
			description: "Basic introduction to Klip",
			category:    "Overview",
			keywords:    []string{"basics", "intro", "start"},
			content:     ih.getGettingStartedContent(),
		},
		{
			title:       "Commands",
			description: "All available commands",
			category:    "Commands",
			keywords:    []string{"slash", "commands", "help"},
			content:     ih.getCommandsContent(),
		},
		{
			title:       "Key Bindings",
			description: "Keyboard shortcuts",
			category:    "Navigation",
			keywords:    []string{"keys", "shortcuts", "navigation"},
			content:     ih.getKeybindingsContent(),
		},
		{
			title:       "API Key Setup",
			description: "Configure API keys for providers",
			category:    "Configuration",
			keywords:    []string{"api", "keys", "setup", "config"},
			content:     ih.getAPIKeysContent(),
		},
		{
			title:       "Models",
			description: "Available AI models and switching",
			category:    "Models",
			keywords:    []string{"models", "switch", "ai", "providers"},
			content:     ih.getModelsContent(),
		},
		{
			title:       "Troubleshooting",
			description: "Common issues and solutions",
			category:    "Support",
			keywords:    []string{"problems", "issues", "errors", "fix"},
			content:     ih.getTroubleshootingContent(),
		},
	}
}

// Content generation methods
func (ih *InteractiveHelp) getGettingStartedContent() string {
	return `# Getting Started with Klip

Klip is a terminal-based AI chat application that supports multiple AI providers.

## First Steps
1. Set up your API keys (see API Key Setup)
2. Choose your preferred model
3. Start chatting!

## Key Features
- Multiple AI providers (Anthropic, OpenAI, OpenRouter)
- Encrypted API key storage
- Chat history and logging
- Real-time streaming responses
- Web search integration
- Cross-platform support

## Quick Tips
- Use / to start commands
- Press F1 for help anytime
- Ctrl+C interrupts streaming responses
- All conversations are logged locally
`
}

func (ih *InteractiveHelp) getCommandsContent() string {
	return `# Available Commands

## Chat Commands
- **/help** - Show help system
- **/clear** - Clear current chat
- **/quit** - Exit application

## Model Management
- **/model** - Switch AI model
- **/models** - List all available models
- **/provider** - Switch provider

## Configuration
- **/settings** - Open settings
- **/keys** - Manage API keys
- **/config** - Show configuration

## History
- **/history** - View chat history
- **/save** - Save current chat
- **/load** - Load previous chat

## System
- **/status** - Show system status
- **/debug** - Toggle debug mode
- **/version** - Show version info
`
}

func (ih *InteractiveHelp) getKeybindingsContent() string {
	return `# Keyboard Shortcuts

## Global Keys
- **F1** - Toggle help
- **F2** - Model selection
- **F3** - Settings
- **F4** - History
- **F12** - Debug info
- **Ctrl+C** - Interrupt/Quit
- **Ctrl+D** - Quit (empty input)
- **Esc** - Return to chat

## Chat Input
- **Enter** - Send message
- **↑/↓** - Navigate input history
- **Ctrl+A** - Move to beginning
- **Ctrl+E** - Move to end
- **Ctrl+U** - Clear line
- **Ctrl+K** - Clear to end
- **Ctrl+W** - Delete word backward
- **Ctrl+L** - Clear screen

## Navigation
- **↑/↓** or **j/k** - Navigate lists
- **Enter** - Select item
- **/** - Start search
- **Esc** - Exit search/return
`
}

func (ih *InteractiveHelp) getAPIKeysContent() string {
	return `# API Key Setup

## Supported Providers

### Anthropic (Claude)
1. Visit https://console.anthropic.com/keys
2. Create a new API key
3. Copy the key (starts with 'sk-ant-')

### OpenAI
1. Go to https://platform.openai.com/api-keys
2. Create a new secret key
3. Copy the key (starts with 'sk-')

### OpenRouter
1. Visit https://openrouter.ai/keys
2. Sign up and create an API key
3. Copy the key (starts with 'sk-or-')

## Setting Keys in Klip

### Method 1: Settings Interface
1. Press **F3** or type **/settings**
2. Navigate to API Keys section
3. Enter your keys
4. Keys are automatically encrypted

### Method 2: Commands
- /set anthropic_key sk-ant-your-key
- /set openai_key sk-your-key
- /set openrouter_key sk-or-your-key

### Method 3: Environment Variables
- ANTHROPIC_API_KEY
- OPENAI_API_KEY
- OPENROUTER_API_KEY
`
}

func (ih *InteractiveHelp) getModelsContent() string {
	return `# AI Models

## Available Models

### Anthropic Claude
- Claude 3.5 Sonnet (Latest)
- Claude 3.5 Haiku
- Claude 3 Opus

### OpenAI
- GPT-4o
- GPT-4o Mini
- GPT-4 Turbo

### OpenRouter
- 200+ models including:
  - Open source models
  - Specialized models
  - Custom fine-tunes

## Switching Models
- Press **F2** for model selection
- Use **/model** command
- Search by typing in model selection

## Model Features
- Different context windows
- Varying capabilities
- Different pricing
- Some support web search
`
}

func (ih *InteractiveHelp) getTroubleshootingContent() string {
	return `# Troubleshooting

## Common Issues

### API Key Problems
- Verify key format and validity
- Check account balance/credits
- Ensure proper permissions

### Connection Issues
- Check internet connection
- Verify firewall settings
- Try different provider

### Performance Issues
- Check terminal compatibility
- Update to latest version
- Clear cache/config

### Chat Problems
- Restart application
- Clear chat history
- Check model availability

## Getting Help
- Check logs in ~/.klip/logs/
- Run with debug mode
- Report issues on GitHub

## Reset Configuration
Use **/reset** command to restore defaults
`
}

// Update method for Bubble Tea
func (ih *InteractiveHelp) Update(msg tea.Msg) (*InteractiveHelp, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if ih.searchMode {
				ih.searchMode = false
				ih.searchInput.Blur()
				return ih, nil
			}
			return ih, tea.Quit

		case "/":
			if !ih.searchMode {
				ih.searchMode = true
				ih.searchInput.Focus()
				return ih, nil
			}

		case "enter":
			if ih.searchMode {
				// Perform search
				ih.searchMode = false
				ih.searchInput.Blur()
				ih.filterItems(ih.searchInput.Value())
				return ih, nil
			} else {
				// Select item
				if selected := ih.list.SelectedItem(); selected != nil {
					if helpItem, ok := selected.(HelpItem); ok {
						ih.selectedItem = &helpItem
						ih.viewport.SetContent(helpItem.content)
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		ih.width = msg.Width
		ih.height = msg.Height
		ih.list.SetSize(msg.Width/2, msg.Height-4)
		ih.viewport = viewport.New(msg.Width/2-2, msg.Height-4)
		ih.searchInput.Width = msg.Width/2 - 4
	}

	// Update components
	if ih.searchMode {
		ih.searchInput, cmd = ih.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		ih.list, cmd = ih.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	ih.viewport, cmd = ih.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return ih, tea.Batch(cmds...)
}

// View renders the help interface
func (ih *InteractiveHelp) View() string {
	leftPanel := ih.renderLeftPanel()
	rightPanel := ih.renderRightPanel()

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func (ih *InteractiveHelp) renderLeftPanel() string {
	var content string

	if ih.searchMode {
		content = fmt.Sprintf("Search: %s\n\n%s",
			ih.searchInput.View(),
			ih.list.View())
	} else {
		content = ih.list.View()
	}

	style := lipgloss.NewStyle().
		Width(ih.width / 2).
		Height(ih.height).
		Padding(1).
		Border(lipgloss.RoundedBorder())

	return style.Render(content)
}

func (ih *InteractiveHelp) renderRightPanel() string {
	style := lipgloss.NewStyle().
		Width(ih.width / 2).
		Height(ih.height).
		Padding(1).
		Border(lipgloss.RoundedBorder())

	return style.Render(ih.viewport.View())
}

// filterItems filters help items based on search query
func (ih *InteractiveHelp) filterItems(query string) {
	if query == "" {
		// Show all items
		ih.items = ih.items[:0]
		for _, item := range ih.helpItems {
			ih.items = append(ih.items, item)
		}
	} else {
		// Filter items
		query = strings.ToLower(query)
		var filtered []list.Item

		for _, item := range ih.helpItems {
			if ih.matchesQuery(item, query) {
				filtered = append(filtered, item)
			}
		}
		ih.items = filtered
	}

	ih.list.SetItems(ih.items)
}

func (ih *InteractiveHelp) matchesQuery(item HelpItem, query string) bool {
	searchText := fmt.Sprintf("%s %s %s %s",
		strings.ToLower(item.title),
		strings.ToLower(item.description),
		strings.ToLower(item.category),
		strings.ToLower(strings.Join(item.keywords, " ")))

	return strings.Contains(searchText, query)
}

// NewHelpDelegate creates a custom list delegate for help items
func NewHelpDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA"))

	return d
}

// Helper functions for integration with app state
func NewInteractiveHelpFromState(state *app.HelpState, width, height int) *InteractiveHelp {
	ih := NewInteractiveHelp(width, height)

	if state.CurrentSection != "" {
		// Map string to HelpSection enum if needed
		// This would depend on how the app state stores the section
	}

	return ih
}

// UpdateFromState updates the help system from app state
func (ih *InteractiveHelp) UpdateFromState(state *app.HelpState) {
	// Update based on app state if needed
	// This would depend on how the app manages help state
}
