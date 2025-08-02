package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/john/klip/internal/api"
	"github.com/john/klip/internal/app"
)

// ModelMsg represents messages for the model selector
type ModelMsg struct {
	Type string
	Data interface{}
}

// ModelItem represents a model in the list
type ModelItem struct {
	model     api.Model
	favorite  bool
	lastUsed  time.Time
	usageInfo *ModelUsageInfo
}

// ModelUsageInfo contains usage statistics for a model
type ModelUsageInfo struct {
	RequestCount  int
	TokensUsed    int
	AvgLatency    time.Duration
	LastUsed      time.Time
	EstimatedCost float64
}

// Implement list.Item interface
func (mi ModelItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s", mi.model.Name, mi.model.ID, mi.model.Provider.String())
}

func (mi ModelItem) Title() string { return mi.model.Name }
func (mi ModelItem) Description() string {
	desc := fmt.Sprintf("%s • %s • %dk context",
		mi.model.Provider.String(),
		humanize.Comma(int64(mi.model.MaxTokens)),
		mi.model.ContextWindow/1000)

	if mi.favorite {
		desc = "★ " + desc
	}

	if mi.usageInfo != nil && !mi.usageInfo.LastUsed.IsZero() {
		desc += fmt.Sprintf(" • Used %s", humanize.Time(mi.usageInfo.LastUsed))
	}

	return desc
}

// ModelSelector provides interactive model selection
type ModelSelector struct {
	list            list.Model
	searchInput     textinput.Model
	models          []api.Model
	filteredModels  []ModelItem
	favorites       map[string]bool
	usage           map[string]*ModelUsageInfo
	showDetails     bool
	selectedModel   *api.Model
	groupByProvider bool
	showFavorites   bool
	showRecent      bool
	searchActive    bool
	width           int
	height          int
	loading         bool
	errorMessage    string
}

// NewModelSelector creates a new model selector component
func NewModelSelector(width, height int) *ModelSelector {
	// Initialize list
	items := make([]list.Item, 0)
	l := list.New(items, NewModelDelegate(), width-4, height-8)
	l.Title = "Select AI Model"
	l.SetShowStatusBar(true)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = ModelListTitleStyle
	l.Styles.PaginationStyle = ModelListPaginationStyle
	// Note: StatusMessageStyle may not be available in current bubble tea version

	// Initialize search input
	search := textinput.New()
	search.Placeholder = "Search models..."
	search.Width = width - 6
	search.Blur()

	return &ModelSelector{
		list:            l,
		searchInput:     search,
		models:          make([]api.Model, 0),
		filteredModels:  make([]ModelItem, 0),
		favorites:       make(map[string]bool),
		usage:           make(map[string]*ModelUsageInfo),
		groupByProvider: true,
		width:           width,
		height:          height,
	}
}

// Init initializes the model selector
func (ms *ModelSelector) Init() tea.Cmd {
	// Bubble tea list and textinput models don't have Init() methods
	// They are initialized when created with New()
	return nil
}

// Update handles model selector updates
func (ms *ModelSelector) Update(msg tea.Msg) (*ModelSelector, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ms.width = msg.Width
		ms.height = msg.Height
		ms.list.SetWidth(msg.Width - 4)
		ms.list.SetHeight(msg.Height - 8)
		ms.searchInput.Width = msg.Width - 6

	case ModelMsg:
		switch msg.Type {
		case "set_models":
			if models, ok := msg.Data.([]api.Model); ok {
				ms.SetModels(models)
			}
		case "set_selected":
			if model, ok := msg.Data.(api.Model); ok {
				ms.SetSelectedModel(model)
			}
		case "toggle_favorite":
			if modelID, ok := msg.Data.(string); ok {
				ms.ToggleFavorite(modelID)
			}
		case "set_loading":
			if loading, ok := msg.Data.(bool); ok {
				ms.loading = loading
			}
		case "set_error":
			if err, ok := msg.Data.(string); ok {
				ms.errorMessage = err
			}
		case "update_usage":
			if usage, ok := msg.Data.(map[string]*ModelUsageInfo); ok {
				ms.usage = usage
				ms.updateList()
			}
		}

	case tea.KeyMsg:
		// Handle global shortcuts
		switch msg.String() {
		case "ctrl+f", "/":
			if !ms.searchActive {
				ms.searchActive = true
				ms.list.SetFilteringEnabled(false)
				return ms, ms.searchInput.Focus()
			}
		case "esc":
			if ms.searchActive {
				ms.searchActive = false
				ms.searchInput.Blur()
				ms.searchInput.SetValue("")
				ms.list.SetFilteringEnabled(true)
				ms.filterModels("")
				return ms, nil
			}
			if ms.showDetails {
				ms.showDetails = false
				return ms, nil
			}
		case "enter":
			if ms.searchActive {
				query := ms.searchInput.Value()
				ms.searchActive = false
				ms.searchInput.Blur()
				ms.filterModels(query)
				return ms, nil
			}
			// Select current model
			if item, ok := ms.list.SelectedItem().(ModelItem); ok {
				ms.selectedModel = &item.model
				return ms, ms.selectModel(item.model)
			}
		case "f":
			if !ms.searchActive {
				if item, ok := ms.list.SelectedItem().(ModelItem); ok {
					ms.ToggleFavorite(item.model.ID)
				}
			}
		case "d":
			if !ms.searchActive {
				ms.showDetails = !ms.showDetails
			}
		case "g":
			if !ms.searchActive {
				ms.groupByProvider = !ms.groupByProvider
				ms.updateList()
			}
		case "ctrl+r":
			ms.showRecent = !ms.showRecent
			ms.updateList()
		case "ctrl+s":
			ms.showFavorites = !ms.showFavorites
			ms.updateList()
		}

		// Handle search input
		if ms.searchActive {
			ms.searchInput, cmd = ms.searchInput.Update(msg)
			cmds = append(cmds, cmd)

			// Live search
			query := ms.searchInput.Value()
			ms.filterModels(query)
		} else {
			// Handle list navigation
			ms.list, cmd = ms.list.Update(msg)
			cmds = append(cmds, cmd)
		}

	default:
		if ms.searchActive {
			ms.searchInput, cmd = ms.searchInput.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			ms.list, cmd = ms.list.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return ms, tea.Batch(cmds...)
}

// View renders the model selector
func (ms *ModelSelector) View() string {
	var content strings.Builder

	// Header
	content.WriteString(ms.renderHeader())
	content.WriteString("\n")

	// Search input if active
	if ms.searchActive {
		content.WriteString(ModelSearchStyle.Render(ms.searchInput.View()))
		content.WriteString("\n")
	}

	// Error message
	if ms.errorMessage != "" {
		content.WriteString(ErrorStyle.Render("Error: " + ms.errorMessage))
		content.WriteString("\n")
	}

	// Loading indicator
	if ms.loading {
		content.WriteString(LoadingStyle.Render("Loading models..."))
		content.WriteString("\n")
	}

	// Main content
	if ms.showDetails && ms.selectedModel != nil {
		content.WriteString(ms.renderModelDetails())
	} else {
		content.WriteString(ms.list.View())
	}

	// Footer
	content.WriteString("\n")
	content.WriteString(ms.renderFooter())

	return ModelContainerStyle.Render(content.String())
}

// SetModels sets the available models
func (ms *ModelSelector) SetModels(models []api.Model) {
	ms.models = models
	ms.updateList()
}

// SetSelectedModel sets the currently selected model
func (ms *ModelSelector) SetSelectedModel(model api.Model) {
	ms.selectedModel = &model

	// Find and select the model in the list
	for i, item := range ms.list.Items() {
		if modelItem, ok := item.(ModelItem); ok {
			if modelItem.model.ID == model.ID {
				ms.list.Select(i)
				break
			}
		}
	}
}

// GetSelectedModel returns the currently selected model
func (ms *ModelSelector) GetSelectedModel() *api.Model {
	if item, ok := ms.list.SelectedItem().(ModelItem); ok {
		return &item.model
	}
	return ms.selectedModel
}

// ToggleFavorite toggles the favorite status of a model
func (ms *ModelSelector) ToggleFavorite(modelID string) {
	ms.favorites[modelID] = !ms.favorites[modelID]
	ms.updateList()
}

// filterModels filters models based on search query
func (ms *ModelSelector) filterModels(query string) {
	if query == "" {
		ms.updateList()
		return
	}

	query = strings.ToLower(query)
	filtered := make([]api.Model, 0)

	for _, model := range ms.models {
		if ms.matchesQuery(model, query) {
			filtered = append(filtered, model)
		}
	}

	ms.buildModelItems(filtered)
}

// matchesQuery checks if a model matches the search query
func (ms *ModelSelector) matchesQuery(model api.Model, query string) bool {
	searchFields := []string{
		strings.ToLower(model.Name),
		strings.ToLower(model.ID),
		strings.ToLower(model.Provider.String()),
	}

	for _, field := range searchFields {
		if strings.Contains(field, query) {
			return true
		}
	}

	return false
}

// updateList updates the list with current models and filters
func (ms *ModelSelector) updateList() {
	models := ms.models

	// Apply filters
	if ms.showFavorites {
		filtered := make([]api.Model, 0)
		for _, model := range models {
			if ms.favorites[model.ID] {
				filtered = append(filtered, model)
			}
		}
		models = filtered
	}

	if ms.showRecent {
		// Sort by last used
		sort.Slice(models, func(i, j int) bool {
			usageI := ms.usage[models[i].ID]
			usageJ := ms.usage[models[j].ID]

			if usageI == nil {
				return false
			}
			if usageJ == nil {
				return true
			}

			return usageI.LastUsed.After(usageJ.LastUsed)
		})

		// Take only recent models
		if len(models) > 10 {
			models = models[:10]
		}
	}

	ms.buildModelItems(models)
}

// buildModelItems builds the list items from models
func (ms *ModelSelector) buildModelItems(models []api.Model) {
	items := make([]list.Item, 0)

	if ms.groupByProvider {
		// Group by provider
		providers := make(map[api.Provider][]api.Model)
		for _, model := range models {
			providers[model.Provider] = append(providers[model.Provider], model)
		}

		// Sort providers
		providerOrder := []api.Provider{api.ProviderAnthropic, api.ProviderOpenAI, api.ProviderOpenRouter}

		for _, provider := range providerOrder {
			if providerModels, exists := providers[provider]; exists {
				// Sort models within provider
				sort.Slice(providerModels, func(i, j int) bool {
					return providerModels[i].Name < providerModels[j].Name
				})

				for _, model := range providerModels {
					item := ModelItem{
						model:     model,
						favorite:  ms.favorites[model.ID],
						usageInfo: ms.usage[model.ID],
					}
					if item.usageInfo != nil {
						item.lastUsed = item.usageInfo.LastUsed
					}
					items = append(items, item)
				}
			}
		}
	} else {
		// Flat list, sorted by name
		sort.Slice(models, func(i, j int) bool {
			return models[i].Name < models[j].Name
		})

		for _, model := range models {
			item := ModelItem{
				model:     model,
				favorite:  ms.favorites[model.ID],
				usageInfo: ms.usage[model.ID],
			}
			if item.usageInfo != nil {
				item.lastUsed = item.usageInfo.LastUsed
			}
			items = append(items, item)
		}
	}

	ms.list.SetItems(items)
}

// renderHeader renders the header with title and filters
func (ms *ModelSelector) renderHeader() string {
	var parts []string

	title := "AI Models"
	if ms.showFavorites {
		title += " (Favorites)"
	} else if ms.showRecent {
		title += " (Recent)"
	}

	parts = append(parts, ModelTitleStyle.Render(title))

	// Filter indicators
	var filters []string
	if ms.groupByProvider {
		filters = append(filters, "Grouped")
	}
	if len(ms.models) > 0 {
		filters = append(filters, fmt.Sprintf("%d models", len(ms.models)))
	}

	if len(filters) > 0 {
		parts = append(parts, ModelFilterStyle.Render(strings.Join(filters, " • ")))
	}

	return strings.Join(parts, " ")
}

// renderModelDetails renders detailed information about the selected model
func (ms *ModelSelector) renderModelDetails() string {
	if ms.selectedModel == nil {
		return "No model selected"
	}

	model := *ms.selectedModel
	var content strings.Builder

	// Model name and provider
	content.WriteString(ModelDetailTitleStyle.Render(model.Name))
	content.WriteString("\n")
	content.WriteString(ModelDetailProviderStyle.Render(fmt.Sprintf("Provider: %s", model.Provider.String())))
	content.WriteString("\n\n")

	// Basic information
	content.WriteString(ModelDetailSectionStyle.Render("Specifications"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("Model ID: %s\n", model.ID))
	content.WriteString(fmt.Sprintf("Max Tokens: %s\n", humanize.Comma(int64(model.MaxTokens))))
	content.WriteString(fmt.Sprintf("Context Window: %s tokens\n", humanize.Comma(int64(model.ContextWindow))))

	// Favorite status
	if ms.favorites[model.ID] {
		content.WriteString("Status: ★ Favorite\n")
	}
	content.WriteString("\n")

	// Usage information
	if usage, exists := ms.usage[model.ID]; exists && usage != nil {
		content.WriteString(ModelDetailSectionStyle.Render("Usage Statistics"))
		content.WriteString("\n")
		content.WriteString(fmt.Sprintf("Requests: %s\n", humanize.Comma(int64(usage.RequestCount))))
		content.WriteString(fmt.Sprintf("Tokens Used: %s\n", humanize.Comma(int64(usage.TokensUsed))))
		if usage.AvgLatency > 0 {
			content.WriteString(fmt.Sprintf("Avg Latency: %v\n", usage.AvgLatency.Round(time.Millisecond)))
		}
		if !usage.LastUsed.IsZero() {
			content.WriteString(fmt.Sprintf("Last Used: %s\n", humanize.Time(usage.LastUsed)))
		}
		if usage.EstimatedCost > 0 {
			content.WriteString(fmt.Sprintf("Est. Cost: $%.4f\n", usage.EstimatedCost))
		}
		content.WriteString("\n")
	}

	// Capabilities (based on provider and model)
	content.WriteString(ModelDetailSectionStyle.Render("Capabilities"))
	content.WriteString("\n")

	capabilities := ms.getModelCapabilities(model)
	for _, capability := range capabilities {
		content.WriteString(fmt.Sprintf("• %s\n", capability))
	}

	return ModelDetailContainerStyle.Render(content.String())
}

// getModelCapabilities returns the capabilities of a model
func (ms *ModelSelector) getModelCapabilities(model api.Model) []string {
	capabilities := []string{"Text Generation", "Conversation"}

	switch model.Provider {
	case api.ProviderAnthropic:
		capabilities = append(capabilities, "Web Search", "Function Calling", "JSON Mode")
		if strings.Contains(strings.ToLower(model.ID), "opus") {
			capabilities = append(capabilities, "Advanced Reasoning", "Long Context")
		}
	case api.ProviderOpenAI:
		capabilities = append(capabilities, "Function Calling", "JSON Mode")
		if strings.Contains(strings.ToLower(model.ID), "o1") || strings.Contains(strings.ToLower(model.ID), "o3") {
			capabilities = append(capabilities, "Advanced Reasoning", "Chain of Thought")
		}
		if strings.Contains(strings.ToLower(model.ID), "4o") {
			capabilities = append(capabilities, "Vision", "Multimodal")
		}
	case api.ProviderOpenRouter:
		capabilities = append(capabilities, "Open Source Models", "Flexible Pricing")
	}

	return capabilities
}

// renderFooter renders the footer with keyboard shortcuts
func (ms *ModelSelector) renderFooter() string {
	shortcuts := []string{
		"enter: select",
		"f: favorite",
		"d: details",
		"/: search",
		"g: group",
		"esc: back",
	}

	if ms.searchActive {
		shortcuts = []string{
			"enter: search",
			"esc: cancel",
		}
	}

	return ModelFooterStyle.Render(strings.Join(shortcuts, " • "))
}

// selectModel sends a model selection message
func (ms *ModelSelector) selectModel(model api.Model) tea.Cmd {
	return func() tea.Msg {
		return ModelMsg{Type: "model_selected", Data: model}
	}
}

// ModelDelegate handles rendering of individual model items
func NewModelDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = SelectedModelStyle
	d.Styles.SelectedDesc = SelectedModelDescStyle
	d.Styles.NormalTitle = ModelItemStyle
	d.Styles.NormalDesc = ModelItemDescStyle

	d.SetHeight(2)
	d.SetSpacing(1)

	return d
}

// Model selector styles
var (
	// Container styles
	ModelContainerStyle = lipgloss.NewStyle().
				Padding(1)

	ModelTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

	ModelFilterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				Italic(true)

	// List styles
	ModelListTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true).
				Padding(0, 1)

	ModelListPaginationStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#6B7280"))

	ModelListStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#059669"))

	// Item styles
	ModelItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1F2937"))

	ModelItemDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	SelectedModelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true).
				Background(lipgloss.Color("#F3F4F6"))

	SelectedModelDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Background(lipgloss.Color("#F3F4F6"))

	// Search styles
	ModelSearchStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED")).
				Padding(0, 1).
				MarginBottom(1)

	// Detail styles
	ModelDetailContainerStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#D1D5DB")).
					Padding(2)

	ModelDetailTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	ModelDetailProviderStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#059669")).
					Bold(true)

	ModelDetailSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#374151")).
				Bold(true).
				Underline(true)

	// Footer styles
	ModelFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				BorderTop(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#E5E7EB")).
				PaddingTop(1)

	// Loading and error styles (LoadingStyle is defined in components.go)
)

// Helper functions for integration with app state
func NewModelSelectorFromState(state *app.ModelsState, width, height int) *ModelSelector {
	ms := NewModelSelector(width, height)
	ms.SetModels(state.AvailableModels)
	ms.loading = state.Loading

	if state.Error != nil {
		ms.errorMessage = state.Error.Error()
	}

	if state.CurrentModel.ID != "" {
		ms.SetSelectedModel(state.CurrentModel)
	}

	return ms
}

// UpdateFromState updates the model selector from app state
func (ms *ModelSelector) UpdateFromState(state *app.ModelsState) {
	ms.SetModels(state.FilteredModels)
	ms.loading = state.Loading

	if state.Error != nil {
		ms.errorMessage = state.Error.Error()
	} else {
		ms.errorMessage = ""
	}

	if state.CurrentModel.ID != "" {
		ms.SetSelectedModel(state.CurrentModel)
	}
}
