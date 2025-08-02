package components

import (
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/john/klip/internal/app"
)

// ComponentRegistry manages all UI components for the Klip application
type ComponentRegistry struct {
	chat         *ChatView
	input        *EnhancedInput
	models       *ModelSelector
	settings     *SettingsForm
	history      *HistoryBrowser
	help         *InteractiveHelp
	statusBar    *StatusBar
	progress     *ProgressTracker
	spinner      *LoadingSpinner
	notifications *NotificationCenter
	tokenUsage   *TokenUsageDisplay
	
	width  int
	height int
	mu     sync.RWMutex
}

// ComponentManager handles component lifecycle and communication
type ComponentManager struct {
	registry    *ComponentRegistry
	eventBus    *EventBus
	initialized bool
}

// EventBus handles inter-component communication
type EventBus struct {
	subscribers map[string][]chan ComponentEvent
	mu          sync.RWMutex
}

// ComponentEvent represents an event that can be sent between components
type ComponentEvent struct {
	Type      string
	Source    string
	Target    string
	Data      interface{}
	Timestamp time.Time
}

// NewComponentRegistry creates a new component registry
func NewComponentRegistry(width, height int) *ComponentRegistry {
	return &ComponentRegistry{
		width:  width,
		height: height,
	}
}

// Initialize initializes all components
func (cr *ComponentRegistry) Initialize() {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	
	// Initialize core components
	cr.chat = NewChatView(cr.width-20, cr.height-10)
	cr.input = NewEnhancedInput(InputTypeText, cr.width-20, 3)
	cr.statusBar = NewStatusBar(cr.width, 1)
	
	// Initialize secondary components
	cr.models = NewModelSelector(cr.width-10, cr.height-5)
	cr.settings = NewSettingsForm(nil, cr.width-10, cr.height-5)
	cr.history = NewHistoryBrowser(cr.width-10, cr.height-5)
	cr.help = NewInteractiveHelp(cr.width-10, cr.height-5)
	
	// Initialize utility components
	cr.progress = NewProgressTracker(cr.width-10, cr.height-15)
	cr.spinner = NewLoadingSpinner(cr.width-20, cr.height-20)
	cr.notifications = NewNotificationCenter(cr.width, cr.height)
	cr.tokenUsage = NewTokenUsageDisplay(cr.width-30, cr.height-25)
}

// Update updates all components with a message
func (cr *ComponentRegistry) Update(msg tea.Msg) tea.Cmd {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	
	var cmds []tea.Cmd
	
	// Update components if they exist
	if cr.chat != nil {
		var cmd tea.Cmd
		cr.chat, cmd = cr.chat.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	if cr.input != nil {
		var cmd tea.Cmd
		cr.input, cmd = cr.input.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	if cr.statusBar != nil {
		var cmd tea.Cmd
		cr.statusBar, cmd = cr.statusBar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	if cr.models != nil {
		var cmd tea.Cmd
		cr.models, cmd = cr.models.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	if cr.settings != nil {
		var cmd tea.Cmd
		cr.settings, cmd = cr.settings.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	if cr.history != nil {
		var cmd tea.Cmd
		cr.history, cmd = cr.history.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	if cr.help != nil {
		var cmd tea.Cmd
		cr.help, cmd = cr.help.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	if cr.progress != nil {
		var cmd tea.Cmd
		cr.progress, cmd = cr.progress.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	if cr.spinner != nil {
		var cmd tea.Cmd
		cr.spinner, cmd = cr.spinner.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	if cr.notifications != nil {
		var cmd tea.Cmd
		cr.notifications, cmd = cr.notifications.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	if cr.tokenUsage != nil {
		var cmd tea.Cmd
		cr.tokenUsage, cmd = cr.tokenUsage.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	
	return tea.Batch(cmds...)
}

// Resize updates component dimensions
func (cr *ComponentRegistry) Resize(width, height int) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	
	cr.width = width
	cr.height = height
	
	// Resize all components
	if cr.chat != nil {
		cr.chat.width = width - 20
		cr.chat.height = height - 10
	}
	
	if cr.input != nil {
		cr.input.width = width - 20
		cr.input.height = 3
	}
	
	if cr.statusBar != nil {
		cr.statusBar.width = width
		cr.statusBar.height = 1
	}
	
	// Resize secondary components
	secondaryWidth := width - 10
	secondaryHeight := height - 5
	
	if cr.models != nil {
		cr.models.width = secondaryWidth
		cr.models.height = secondaryHeight
	}
	
	if cr.settings != nil {
		cr.settings.width = secondaryWidth
		cr.settings.height = secondaryHeight
	}
	
	if cr.history != nil {
		cr.history.width = secondaryWidth
		cr.history.height = secondaryHeight
	}
	
	if cr.help != nil {
		cr.help.width = secondaryWidth
		cr.help.height = secondaryHeight
	}
	
	// Resize utility components
	if cr.progress != nil {
		cr.progress.width = width - 10
		cr.progress.height = height - 15
	}
	
	if cr.spinner != nil {
		cr.spinner.width = width - 20
		cr.spinner.height = height - 20
	}
	
	if cr.notifications != nil {
		cr.notifications.width = width
		cr.notifications.height = height
	}
	
	if cr.tokenUsage != nil {
		cr.tokenUsage.width = width - 30
		cr.tokenUsage.height = height - 25
	}
}

// Component accessors with thread safety
func (cr *ComponentRegistry) Chat() *ChatView {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.chat
}

func (cr *ComponentRegistry) Input() *EnhancedInput {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.input
}

func (cr *ComponentRegistry) Models() *ModelSelector {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.models
}

func (cr *ComponentRegistry) Settings() *SettingsForm {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.settings
}

func (cr *ComponentRegistry) History() *HistoryBrowser {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.history
}

func (cr *ComponentRegistry) Help() *InteractiveHelp {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.help
}

func (cr *ComponentRegistry) StatusBar() *StatusBar {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.statusBar
}

func (cr *ComponentRegistry) Progress() *ProgressTracker {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.progress
}

func (cr *ComponentRegistry) Spinner() *LoadingSpinner {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.spinner
}

func (cr *ComponentRegistry) Notifications() *NotificationCenter {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.notifications
}

func (cr *ComponentRegistry) TokenUsage() *TokenUsageDisplay {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.tokenUsage
}

// NewComponentManager creates a new component manager
func NewComponentManager(width, height int) *ComponentManager {
	return &ComponentManager{
		registry: NewComponentRegistry(width, height),
		eventBus: NewEventBus(),
	}
}

// Initialize initializes the component manager
func (cm *ComponentManager) Initialize() error {
	if cm.initialized {
		return nil
	}
	
	cm.registry.Initialize()
	cm.initialized = true
	
	return nil
}

// Registry returns the component registry
func (cm *ComponentManager) Registry() *ComponentRegistry {
	return cm.registry
}

// EventBus returns the event bus
func (cm *ComponentManager) EventBus() *EventBus {
	return cm.eventBus
}

// UpdateFromAppState updates all components from application state
func (cm *ComponentManager) UpdateFromAppState(state *app.State) {
	if !cm.initialized {
		return
	}
	
	// Update chat component from chat state
	if chat := cm.registry.Chat(); chat != nil && state.Chat != nil {
		chat.UpdateFromState(state.Chat)
	}
	
	// Update input component from chat state
	if input := cm.registry.Input(); input != nil && state.Chat != nil {
		input.UpdateFromState(state.Chat)
	}
	
	// Update models component from models state
	if models := cm.registry.Models(); models != nil && state.Models != nil {
		models.UpdateFromState(state.Models)
	}
	
	// Update settings component from settings state
	if settings := cm.registry.Settings(); settings != nil && state.Settings != nil {
		settings.UpdateFromState(state.Settings)
	}
	
	// Update history component from history state
	if history := cm.registry.History(); history != nil && state.History != nil {
		history.UpdateFromState(state.History)
	}
	
	// Update status bar from status state
	if statusBar := cm.registry.StatusBar(); statusBar != nil && state.Status != nil {
		statusBar.UpdateFromState(state.Status)
	}
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan ComponentEvent),
	}
}

// Subscribe subscribes to events of a given type
func (eb *EventBus) Subscribe(eventType string) <-chan ComponentEvent {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	ch := make(chan ComponentEvent, 10) // Buffered channel
	eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)
	
	return ch
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(event ComponentEvent) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	event.Timestamp = time.Now()
	
	if subscribers, exists := eb.subscribers[event.Type]; exists {
		for _, ch := range subscribers {
			select {
			case ch <- event:
			default:
				// Channel is full, skip this subscriber
			}
		}
	}
}

// Unsubscribe removes a subscriber (simplified - would need channel tracking in real implementation)
func (eb *EventBus) Unsubscribe(eventType string, ch <-chan ComponentEvent) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	if subscribers, exists := eb.subscribers[eventType]; exists {
		// In a real implementation, we'd remove the specific channel
		// This is a simplified version
		for i, subscriber := range subscribers {
			if subscriber == ch {
				eb.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
				close(subscriber)
				break
			}
		}
	}
}

// Component integration helpers

// CreateChatViewFromState creates a chat view from app state
func CreateChatViewFromState(state *app.ChatState, width, height int) *ChatView {
	return NewChatViewFromState(state, width, height)
}

// CreateInputFromState creates an enhanced input from app state
func CreateInputFromState(state *app.ChatState, width, height int) *EnhancedInput {
	return NewEnhancedInputFromState(state, width, height)
}

// CreateModelSelectorFromState creates a model selector from app state
func CreateModelSelectorFromState(state *app.ModelsState, width, height int) *ModelSelector {
	return NewModelSelectorFromState(state, width, height)
}

// CreateSettingsFormFromState creates a settings form from app state
func CreateSettingsFormFromState(state *app.SettingsState, width, height int) *SettingsForm {
	return NewSettingsFormFromState(state, width, height)
}

// CreateHistoryBrowserFromState creates a history browser from app state
func CreateHistoryBrowserFromState(state *app.HistoryState, width, height int) *HistoryBrowser {
	return NewHistoryBrowserFromState(state, width, height)
}

// CreateStatusBarFromState creates a status bar from app state
func CreateStatusBarFromState(state *app.StatusState, width, height int) *StatusBar {
	return NewStatusBarFromState(state, width, height)
}

// Common UI styles and constants
var (
	// Base styles
	BaseStyle = lipgloss.NewStyle().
		Padding(1, 2)

	// Primary button style
	PrimaryButtonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C3AED")).
		Padding(0, 2).
		Bold(true)

	// Secondary button style  
	SecondaryButtonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(0, 2)

	// Error message style
	ErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444")).
		Bold(true)

	// Success message style  
	SuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true)

	// Loading style
	LoadingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	// Warning style
	WarningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true)

	// Info style
	InfoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Bold(true)
)

// Theme management
type Theme struct {
	Name           string
	Primary        lipgloss.Color
	Secondary      lipgloss.Color
	Background     lipgloss.Color
	Foreground     lipgloss.Color
	Border         lipgloss.Color
	Success        lipgloss.Color
	Error          lipgloss.Color
	Warning        lipgloss.Color
	Info           lipgloss.Color
}

// Predefined themes
var (
	CharmTheme = Theme{
		Name:       "charm",
		Primary:    lipgloss.Color("#7C3AED"),
		Secondary:  lipgloss.Color("#A855F7"),
		Background: lipgloss.Color("#FFFFFF"),
		Foreground: lipgloss.Color("#1F2937"),
		Border:     lipgloss.Color("#E5E7EB"),
		Success:    lipgloss.Color("#10B981"),
		Error:      lipgloss.Color("#EF4444"),
		Warning:    lipgloss.Color("#F59E0B"),
		Info:       lipgloss.Color("#3B82F6"),
	}
	
	DarkTheme = Theme{
		Name:       "dark",
		Primary:    lipgloss.Color("#8B5CF6"),
		Secondary:  lipgloss.Color("#A78BFA"),
		Background: lipgloss.Color("#111827"),
		Foreground: lipgloss.Color("#F9FAFB"),
		Border:     lipgloss.Color("#374151"),
		Success:    lipgloss.Color("#34D399"),
		Error:      lipgloss.Color("#F87171"),
		Warning:    lipgloss.Color("#FBBF24"),
		Info:       lipgloss.Color("#60A5FA"),
	}
)

// ApplyTheme applies a theme to the component styles
func ApplyTheme(theme Theme) {
	PrimaryButtonStyle = PrimaryButtonStyle.
		Background(theme.Primary).
		Foreground(theme.Background)
	
	SecondaryButtonStyle = SecondaryButtonStyle.
		Foreground(theme.Primary).
		BorderForeground(theme.Primary)
	
	ErrorStyle = ErrorStyle.Foreground(theme.Error)
	SuccessStyle = SuccessStyle.Foreground(theme.Success)
	LoadingStyle = LoadingStyle.Foreground(theme.Primary)
	WarningStyle = WarningStyle.Foreground(theme.Warning)
	InfoStyle = InfoStyle.Foreground(theme.Info)
}