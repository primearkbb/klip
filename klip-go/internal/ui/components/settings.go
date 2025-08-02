package components

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/john/klip/internal/app"
	"github.com/john/klip/internal/storage"
)

// SettingsMsg represents messages for the settings component
type SettingsMsg struct {
	Type string
	Data interface{}
}

// SettingsSection represents different settings sections
type SettingsSection int

const (
	SectionGeneral SettingsSection = iota
	SectionProviders
	SectionDisplay
	SectionAdvanced
	SectionAbout
)

// SettingsForm manages the settings form using huh
type SettingsForm struct {
	form            *huh.Form
	config          *storage.Config
	tempConfig      *storage.Config
	currentSection  SettingsSection
	sections        []SettingsSection
	width           int
	height          int
	unsavedChanges  bool
	validationError string
	saveCallback    func(*storage.Config) error
	resetCallback   func() error
}

// NewSettingsForm creates a new settings form
func NewSettingsForm(config *storage.Config, width, height int) *SettingsForm {
	sf := &SettingsForm{
		config:         config,
		currentSection: SectionGeneral,
		sections: []SettingsSection{
			SectionGeneral,
			SectionProviders,
			SectionDisplay,
			SectionAdvanced,
			SectionAbout,
		},
		width:  width,
		height: height,
	}

	sf.tempConfig = sf.copyConfig(config)
	sf.buildForm()
	return sf
}

// Init initializes the settings form
func (sf *SettingsForm) Init() tea.Cmd {
	return sf.form.Init()
}

// Update handles settings form updates
func (sf *SettingsForm) Update(msg tea.Msg) (*SettingsForm, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		sf.width = msg.Width
		sf.height = msg.Height
		sf.buildForm() // Rebuild form with new dimensions

	case SettingsMsg:
		switch msg.Type {
		case "save":
			return sf, sf.save()
		case "reset":
			return sf, sf.reset()
		case "cancel":
			sf.tempConfig = sf.copyConfig(sf.config)
			sf.unsavedChanges = false
			sf.buildForm()
		case "set_config":
			if config, ok := msg.Data.(*storage.Config); ok {
				sf.config = config
				sf.tempConfig = sf.copyConfig(config)
				sf.buildForm()
			}
		case "next_section":
			sf.nextSection()
			sf.buildForm()
		case "prev_section":
			sf.prevSection()
			sf.buildForm()
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			return sf, sf.save()
		case "ctrl+r":
			return sf, sf.reset()
		case "ctrl+z":
			sf.tempConfig = sf.copyConfig(sf.config)
			sf.unsavedChanges = false
			sf.buildForm()
		case "tab":
			sf.nextSection()
			sf.buildForm()
		case "shift+tab":
			sf.prevSection()
			sf.buildForm()
		case "f1":
			sf.currentSection = SectionGeneral
			sf.buildForm()
		case "f2":
			sf.currentSection = SectionProviders
			sf.buildForm()
		case "f3":
			sf.currentSection = SectionDisplay
			sf.buildForm()
		case "f4":
			sf.currentSection = SectionAdvanced
			sf.buildForm()
		case "f5":
			sf.currentSection = SectionAbout
			sf.buildForm()
		}
	}

	form, cmd := sf.form.Update(msg)
	sf.form = form.(*huh.Form)

	// Check for changes
	sf.checkForChanges()

	return sf, cmd
}

// View renders the settings form
func (sf *SettingsForm) View() string {
	var content strings.Builder

	// Header
	content.WriteString(sf.renderHeader())
	content.WriteString("\n")

	// Section tabs
	content.WriteString(sf.renderSectionTabs())
	content.WriteString("\n")

	// Form content
	content.WriteString(sf.form.View())
	content.WriteString("\n")

	// Footer
	content.WriteString(sf.renderFooter())

	return SettingsContainerStyle.Render(content.String())
}

// SetSaveCallback sets the callback for saving settings
func (sf *SettingsForm) SetSaveCallback(callback func(*storage.Config) error) {
	sf.saveCallback = callback
}

// SetResetCallback sets the callback for resetting settings
func (sf *SettingsForm) SetResetCallback(callback func() error) {
	sf.resetCallback = callback
}

// GetConfig returns the current configuration
func (sf *SettingsForm) GetConfig() *storage.Config {
	return sf.tempConfig
}

// HasUnsavedChanges returns whether there are unsaved changes
func (sf *SettingsForm) HasUnsavedChanges() bool {
	return sf.unsavedChanges
}

// buildForm builds the huh form based on current section
func (sf *SettingsForm) buildForm() {
	var groups []*huh.Group

	switch sf.currentSection {
	case SectionGeneral:
		groups = sf.buildGeneralSection()
	case SectionProviders:
		groups = sf.buildProvidersSection()
	case SectionDisplay:
		groups = sf.buildDisplaySection()
	case SectionAdvanced:
		groups = sf.buildAdvancedSection()
	case SectionAbout:
		groups = sf.buildAboutSection()
	}

	sf.form = huh.NewForm(groups...).
		WithWidth(sf.width - 6).
		WithHeight(sf.height - 8).
		WithTheme(huh.ThemeCharm())
}

// buildGeneralSection builds the general settings section
func (sf *SettingsForm) buildGeneralSection() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewInput().
				Title("Default Model").
				Description("The default AI model to use").
				Value(&sf.tempConfig.DefaultModel).
				Placeholder("claude-sonnet-4-20250514"),

			huh.NewConfirm().
				Title("Enable Logging").
				Description("Save chat sessions to log files").
				Value(&sf.tempConfig.EnableLogging),

			huh.NewConfirm().
				Title("Enable Analytics").
				Description("Collect usage analytics (anonymous)").
				Value(&sf.tempConfig.EnableAnalytics),

			huh.NewInput().
				Title("Log Directory").
				Description("Directory to store chat logs").
				Value(&sf.tempConfig.LogDirectory).
				Placeholder("~/.klip/logs"),

			huh.NewSelect[int]().
				Title("Max History").
				Description("Maximum number of messages to keep in memory").
				Options(
					huh.NewOption("50 messages", 50),
					huh.NewOption("100 messages", 100),
					huh.NewOption("200 messages", 200),
					huh.NewOption("500 messages", 500),
					huh.NewOption("Unlimited", -1),
				).
				Value(&sf.tempConfig.MaxHistory),

			huh.NewSelect[time.Duration]().
				Title("Request Timeout").
				Description("Maximum time to wait for API responses").
				Options(
					huh.NewOption("30 seconds", 30*time.Second),
					huh.NewOption("60 seconds", 60*time.Second),
					huh.NewOption("120 seconds", 120*time.Second),
					huh.NewOption("300 seconds", 300*time.Second),
				).
				Value(&sf.tempConfig.RequestTimeout),
		),
	}
}

// buildProvidersSection builds the providers settings section
func (sf *SettingsForm) buildProvidersSection() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewNote().
				Title("API Keys").
				Description("Configure API keys for different providers. Keys are encrypted and stored securely."),

			huh.NewInput().
				Title("Anthropic API Key").
				Description("Your Anthropic Claude API key").
				Value(&sf.tempConfig.AnthropicAPIKey).
				Password(true).
				Placeholder("sk-ant-..."),

			huh.NewInput().
				Title("OpenAI API Key").
				Description("Your OpenAI API key").
				Value(&sf.tempConfig.OpenAIAPIKey).
				Password(true).
				Placeholder("sk-..."),

			huh.NewInput().
				Title("OpenRouter API Key").
				Description("Your OpenRouter API key").
				Value(&sf.tempConfig.OpenRouterAPIKey).
				Password(true).
				Placeholder("sk-or-..."),
		),

		huh.NewGroup(
			huh.NewNote().
				Title("Provider Settings").
				Description("Configure provider-specific settings and preferences."),

			huh.NewSelect[string]().
				Title("Default Provider").
				Description("The preferred provider when multiple options are available").
				Options(
					huh.NewOption("Anthropic", "anthropic"),
					huh.NewOption("OpenAI", "openai"),
					huh.NewOption("OpenRouter", "openrouter"),
				).
				Value(&sf.tempConfig.DefaultProvider),

			huh.NewConfirm().
				Title("Enable Web Search").
				Description("Allow models to search the web (Anthropic only)").
				Value(&sf.tempConfig.EnableWebSearch),

			huh.NewInput().
				Title("Base URL Override").
				Description("Custom base URL for API calls (advanced)").
				Value(&sf.tempConfig.BaseURL).
				Placeholder("Leave empty for default"),

			huh.NewSelect[int]().
				Title("Max Retries").
				Description("Maximum number of retries for failed requests").
				Options(
					huh.NewOption("1", 1),
					huh.NewOption("3", 3),
					huh.NewOption("5", 5),
					huh.NewOption("10", 10),
				).
				Value(&sf.tempConfig.MaxRetries),
		),
	}
}

// buildDisplaySection builds the display settings section
func (sf *SettingsForm) buildDisplaySection() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Theme").
				Description("Color theme for the application").
				Options(
					huh.NewOption("Charm (Purple)", "charm"),
					huh.NewOption("Dark", "dark"),
					huh.NewOption("Light", "light"),
					huh.NewOption("Catppuccin", "catppuccin"),
				).
				Value(&sf.tempConfig.Theme),

			huh.NewConfirm().
				Title("Show Timestamps").
				Description("Display timestamps for messages").
				Value(&sf.tempConfig.ShowTimestamps),

			huh.NewConfirm().
				Title("Syntax Highlighting").
				Description("Enable syntax highlighting for code blocks").
				Value(&sf.tempConfig.SyntaxHighlighting),

			huh.NewConfirm().
				Title("Show Token Count").
				Description("Display estimated token count for inputs").
				Value(&sf.tempConfig.ShowTokenCount),

			huh.NewConfirm().
				Title("Auto Scroll").
				Description("Automatically scroll to new messages").
				Value(&sf.tempConfig.AutoScroll),

			huh.NewSelect[int]().
				Title("Max Line Length").
				Description("Maximum line length for word wrapping").
				Options(
					huh.NewOption("80 characters", 80),
					huh.NewOption("100 characters", 100),
					huh.NewOption("120 characters", 120),
					huh.NewOption("No limit", -1),
				).
				Value(&sf.tempConfig.MaxLineLength),
		),

		huh.NewGroup(
			huh.NewNote().
				Title("Animation Settings").
				Description("Configure animations and visual effects."),

			huh.NewConfirm().
				Title("Enable Animations").
				Description("Enable smooth animations and transitions").
				Value(&sf.tempConfig.EnableAnimations),

			huh.NewConfirm().
				Title("Typing Indicator").
				Description("Show typing indicator during streaming responses").
				Value(&sf.tempConfig.ShowTypingIndicator),

			huh.NewSelect[time.Duration]().
				Title("Animation Speed").
				Description("Speed of animations and transitions").
				Options(
					huh.NewOption("Slow", 200*time.Millisecond),
					huh.NewOption("Normal", 100*time.Millisecond),
					huh.NewOption("Fast", 50*time.Millisecond),
					huh.NewOption("Instant", 0*time.Millisecond),
				).
				Value(&sf.tempConfig.AnimationSpeed),
		),
	}
}

// buildAdvancedSection builds the advanced settings section
func (sf *SettingsForm) buildAdvancedSection() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewNote().
				Title("Advanced Configuration").
				Description("Advanced settings for power users. Change with caution."),

			huh.NewConfirm().
				Title("Debug Mode").
				Description("Enable debug logging and verbose output").
				Value(&sf.tempConfig.DebugMode),

			huh.NewInput().
				Title("Config Directory").
				Description("Directory to store configuration files").
				Value(&sf.tempConfig.ConfigDir).
				Placeholder("~/.klip"),

			huh.NewSelect[string]().
				Title("Log Level").
				Description("Logging verbosity level").
				Options(
					huh.NewOption("Error", "error"),
					huh.NewOption("Warn", "warn"),
					huh.NewOption("Info", "info"),
					huh.NewOption("Debug", "debug"),
				).
				Value(&sf.tempConfig.LogLevel),

			huh.NewInput().
				Title("User Agent").
				Description("Custom user agent string for API requests").
				Value(&sf.tempConfig.UserAgent).
				Placeholder("klip/1.0.0"),
		),

		huh.NewGroup(
			huh.NewNote().
				Title("Performance Tuning").
				Description("Settings to optimize performance and resource usage."),

			huh.NewSelect[int]().
				Title("Stream Buffer Size").
				Description("Buffer size for streaming responses").
				Options(
					huh.NewOption("1KB", 1024),
					huh.NewOption("4KB", 4096),
					huh.NewOption("8KB", 8192),
					huh.NewOption("16KB", 16384),
				).
				Value(&sf.tempConfig.StreamBufferSize),

			huh.NewSelect[int]().
				Title("Concurrent Requests").
				Description("Maximum number of concurrent API requests").
				Options(
					huh.NewOption("1", 1),
					huh.NewOption("2", 2),
					huh.NewOption("4", 4),
					huh.NewOption("8", 8),
				).
				Value(&sf.tempConfig.MaxConcurrentRequests),

			huh.NewConfirm().
				Title("Cache Models").
				Description("Cache model information to reduce API calls").
				Value(&sf.tempConfig.CacheModels),

			huh.NewSelect[time.Duration]().
				Title("Cache Duration").
				Description("How long to cache model information").
				Options(
					huh.NewOption("5 minutes", 5*time.Minute),
					huh.NewOption("15 minutes", 15*time.Minute),
					huh.NewOption("1 hour", time.Hour),
					huh.NewOption("24 hours", 24*time.Hour),
				).
				Value(&sf.tempConfig.CacheDuration),
		),
	}
}

// buildAboutSection builds the about section
func (sf *SettingsForm) buildAboutSection() []*huh.Group {
	return []*huh.Group{
		huh.NewGroup(
			huh.NewNote().
				Title("Klip - Terminal AI Chat").
				Description(fmt.Sprintf("Version: %s\nA beautiful terminal-based AI chat application built with Go and Bubble Tea.\n\nDeveloped with ❤️ using Charm libraries.", sf.getVersion())),

			huh.NewNote().
				Title("Configuration").
				Description(fmt.Sprintf("Config file: %s\nLog directory: %s\nData directory: %s",
					sf.getConfigPath(),
					sf.getLogPath(),
					sf.getDataPath())),

			huh.NewNote().
				Title("System Information").
				Description(sf.getSystemInfo()),
		),

		huh.NewGroup(
			huh.NewNote().
				Title("Actions").
				Description("Management and maintenance actions."),

			huh.NewConfirm().
				Title("Export Configuration").
				Description("Export current settings to a file").
				Value(new(bool)),

			huh.NewConfirm().
				Title("Clear Cache").
				Description("Clear all cached data").
				Value(new(bool)),

			huh.NewConfirm().
				Title("Reset to Defaults").
				Description("⚠️  Reset all settings to default values").
				Value(new(bool)),
		),
	}
}

// nextSection moves to the next settings section
func (sf *SettingsForm) nextSection() {
	if int(sf.currentSection) < len(sf.sections)-1 {
		sf.currentSection = sf.sections[int(sf.currentSection)+1]
	} else {
		sf.currentSection = sf.sections[0]
	}
}

// prevSection moves to the previous settings section
func (sf *SettingsForm) prevSection() {
	if int(sf.currentSection) > 0 {
		sf.currentSection = sf.sections[int(sf.currentSection)-1]
	} else {
		sf.currentSection = sf.sections[len(sf.sections)-1]
	}
}

// checkForChanges checks if there are unsaved changes
func (sf *SettingsForm) checkForChanges() {
	sf.unsavedChanges = !sf.configsEqual(sf.config, sf.tempConfig)
}

// configsEqual compares two configurations for equality
func (sf *SettingsForm) configsEqual(a, b *storage.Config) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare key fields (simplified)
	return a.DefaultModel == b.DefaultModel &&
		a.EnableLogging == b.EnableLogging &&
		a.EnableAnalytics == b.EnableAnalytics &&
		a.Theme == b.Theme &&
		a.ShowTimestamps == b.ShowTimestamps
}

// copyConfig creates a deep copy of a configuration
func (sf *SettingsForm) copyConfig(config *storage.Config) *storage.Config {
	if config == nil {
		return &storage.Config{}
	}

	// Create a new config with copied values
	return &storage.Config{
		DefaultModel:          config.DefaultModel,
		EnableLogging:         config.EnableLogging,
		EnableAnalytics:       config.EnableAnalytics,
		LogDirectory:          config.LogDirectory,
		MaxHistory:            config.MaxHistory,
		RequestTimeout:        config.RequestTimeout,
		AnthropicAPIKey:       config.AnthropicAPIKey,
		OpenAIAPIKey:          config.OpenAIAPIKey,
		OpenRouterAPIKey:      config.OpenRouterAPIKey,
		DefaultProvider:       config.DefaultProvider,
		EnableWebSearch:       config.EnableWebSearch,
		BaseURL:               config.BaseURL,
		MaxRetries:            config.MaxRetries,
		Theme:                 config.Theme,
		ShowTimestamps:        config.ShowTimestamps,
		SyntaxHighlighting:    config.SyntaxHighlighting,
		ShowTokenCount:        config.ShowTokenCount,
		AutoScroll:            config.AutoScroll,
		MaxLineLength:         config.MaxLineLength,
		EnableAnimations:      config.EnableAnimations,
		ShowTypingIndicator:   config.ShowTypingIndicator,
		AnimationSpeed:        config.AnimationSpeed,
		DebugMode:             config.DebugMode,
		ConfigDir:             config.ConfigDir,
		LogLevel:              config.LogLevel,
		UserAgent:             config.UserAgent,
		StreamBufferSize:      config.StreamBufferSize,
		MaxConcurrentRequests: config.MaxConcurrentRequests,
		CacheModels:           config.CacheModels,
		CacheDuration:         config.CacheDuration,
	}
}

// save saves the current configuration
func (sf *SettingsForm) save() tea.Cmd {
	return func() tea.Msg {
		if sf.saveCallback != nil {
			if err := sf.saveCallback(sf.tempConfig); err != nil {
				return SettingsMsg{Type: "save_error", Data: err}
			}
		}
		sf.config = sf.copyConfig(sf.tempConfig)
		sf.unsavedChanges = false
		return SettingsMsg{Type: "save_success"}
	}
}

// reset resets settings to defaults
func (sf *SettingsForm) reset() tea.Cmd {
	return func() tea.Msg {
		if sf.resetCallback != nil {
			if err := sf.resetCallback(); err != nil {
				return SettingsMsg{Type: "reset_error", Data: err}
			}
		}
		return SettingsMsg{Type: "reset_success"}
	}
}

// renderHeader renders the settings header
func (sf *SettingsForm) renderHeader() string {
	var title strings.Builder
	title.WriteString(SettingsTitleStyle.Render("Settings"))

	if sf.unsavedChanges {
		title.WriteString(" ")
		title.WriteString(UnsavedChangesStyle.Render("●"))
	}

	return title.String()
}

// renderSectionTabs renders the section navigation tabs
func (sf *SettingsForm) renderSectionTabs() string {
	sections := map[SettingsSection]string{
		SectionGeneral:   "General",
		SectionProviders: "Providers",
		SectionDisplay:   "Display",
		SectionAdvanced:  "Advanced",
		SectionAbout:     "About",
	}

	var tabs []string
	for _, section := range sf.sections {
		name := sections[section]
		if section == sf.currentSection {
			tabs = append(tabs, ActiveTabStyle.Render(name))
		} else {
			tabs = append(tabs, InactiveTabStyle.Render(name))
		}
	}

	return TabContainerStyle.Render(strings.Join(tabs, ""))
}

// renderFooter renders the settings footer
func (sf *SettingsForm) renderFooter() string {
	var parts []string

	// Save indicator
	if sf.unsavedChanges {
		parts = append(parts, UnsavedChangesFooterStyle.Render("● Unsaved changes"))
	} else {
		parts = append(parts, SavedStyle.Render("✓ Saved"))
	}

	// Keyboard shortcuts
	shortcuts := []string{
		"Ctrl+S: save",
		"Ctrl+R: reset",
		"Tab: next section",
		"F1-F5: jump to section",
	}
	parts = append(parts, strings.Join(shortcuts, " • "))

	return SettingsFooterStyle.Render(strings.Join(parts, " │ "))
}

// Helper methods for about section
func (sf *SettingsForm) getVersion() string {
	return "1.0.0" // TODO: Get from build info
}

func (sf *SettingsForm) getConfigPath() string {
	if sf.config != nil && sf.config.ConfigDir != "" {
		return filepath.Join(sf.config.ConfigDir, "config.json")
	}
	return "~/.klip/config.json"
}

func (sf *SettingsForm) getLogPath() string {
	if sf.config != nil && sf.config.LogDirectory != "" {
		return sf.config.LogDirectory
	}
	return "~/.klip/logs"
}

func (sf *SettingsForm) getDataPath() string {
	if sf.config != nil && sf.config.ConfigDir != "" {
		return sf.config.ConfigDir
	}
	return "~/.klip"
}

func (sf *SettingsForm) getSystemInfo() string {
	return "Go runtime information and system details would go here"
}

// Settings component styles
var (
	// Container styles
	SettingsContainerStyle = lipgloss.NewStyle().
				Padding(1)

	SettingsTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	UnsavedChangesStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B"))

	// Tab styles
	TabContainerStyle = lipgloss.NewStyle().
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#E5E7EB")).
				MarginBottom(1)

	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Background(lipgloss.Color("#F3F4F6")).
			Bold(true).
			Padding(0, 2).
			MarginRight(1)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				Padding(0, 2).
				MarginRight(1)

	// Footer styles
	SettingsFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				BorderTop(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#E5E7EB")).
				PaddingTop(1)

	UnsavedChangesFooterStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#F59E0B")).
					Bold(true)

	SavedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))
)

// Helper functions for integration with app state
func NewSettingsFormFromState(state *app.SettingsState, width, height int) *SettingsForm {
	sf := NewSettingsForm(state.Config, width, height)
	sf.unsavedChanges = state.UnsavedChanges

	return sf
}

// UpdateFromState updates the settings form from app state
func (sf *SettingsForm) UpdateFromState(state *app.SettingsState) {
	if state.Config != nil {
		sf.config = state.Config
		sf.tempConfig = sf.copyConfig(state.Config)
	}
	sf.unsavedChanges = state.UnsavedChanges
	sf.buildForm()
}
