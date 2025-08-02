package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/john/klip/internal/app"
	"github.com/john/klip/internal/storage"
)

// HistoryMsg represents messages for the history component
type HistoryMsg struct {
	Type string
	Data interface{}
}

// HistoryViewMode represents different view modes for history
type HistoryViewMode int

const (
	HistoryViewList HistoryViewMode = iota
	HistoryViewTable
	HistoryViewPreview
	HistoryViewExport
)

// SessionItem represents a chat session in the list
type SessionItem struct {
	session     storage.ChatSession
	metadata    *SessionMetadata
	highlighted bool
}

// SessionMetadata contains calculated metadata for a session
type SessionMetadata struct {
	MessageCount int
	TokenCount   int
	Duration     time.Duration
	Models       []string
	LastMessage  string
	SearchMatch  bool
}

// Implement list.Item interface
func (si SessionItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s",
		si.session.ID,
		si.session.Title,
		si.session.CreatedAt.Format("2006-01-02"),
		si.metadata.LastMessage)
}

func (si SessionItem) Title() string {
	title := si.session.Title
	if title == "" {
		title = fmt.Sprintf("Session %s", si.session.ID[:8])
	}

	if si.highlighted {
		title = "🔍 " + title
	}

	return title
}

func (si SessionItem) Description() string {
	desc := fmt.Sprintf("%s • %d messages • %s tokens • %s",
		humanize.Time(si.session.CreatedAt),
		si.metadata.MessageCount,
		humanize.Comma(int64(si.metadata.TokenCount)),
		strings.Join(si.metadata.Models, ", "))

	if si.metadata.LastMessage != "" {
		lastMsg := si.metadata.LastMessage
		if len(lastMsg) > 50 {
			lastMsg = lastMsg[:47] + "..."
		}
		desc += fmt.Sprintf(" • \"%s\"", lastMsg)
	}

	return desc
}

// HistoryBrowser provides comprehensive chat history browsing capabilities
type HistoryBrowser struct {
	list             list.Model
	table            table.Model
	preview          viewport.Model
	searchInput      textinput.Model
	sessions         []storage.ChatSession
	filteredSessions []SessionItem
	selectedSession  *storage.ChatSession
	viewMode         HistoryViewMode
	searchActive     bool
	searchQuery      string
	exportFormats    []string
	selectedFormat   int
	width            int
	height           int
	loading          bool
	errorMessage     string
	analytics        *HistoryAnalytics
	sortBy           string
	sortDesc         bool
}

// HistoryAnalytics contains analytics about chat history
type HistoryAnalytics struct {
	TotalSessions    int
	TotalMessages    int
	TotalTokens      int
	AvgSessionLength time.Duration
	TopModels        []ModelUsage
	DailyActivity    []DayActivity
	SearchResults    int
}

// ModelUsage represents usage statistics for a model
type ModelUsage struct {
	Model  string
	Count  int
	Tokens int
}

// DayActivity represents activity for a single day
type DayActivity struct {
	Date     time.Time
	Sessions int
	Messages int
}

// NewHistoryBrowser creates a new comprehensive history browser
func NewHistoryBrowser(width, height int) *HistoryBrowser {
	// Initialize list
	items := make([]list.Item, 0)
	l := list.New(items, NewHistoryDelegate(), width-4, height-8)
	l.Title = "Chat History"
	l.SetShowStatusBar(true)
	l.SetShowPagination(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = HistoryListTitleStyle
	l.Styles.PaginationStyle = HistoryListPaginationStyle

	// Initialize table
	columns := []table.Column{
		{Title: "Title", Width: 30},
		{Title: "Date", Width: 12},
		{Title: "Messages", Width: 10},
		{Title: "Tokens", Width: 10},
		{Title: "Model", Width: 15},
		{Title: "Duration", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(height-10),
	)
	t.SetStyles(HistoryTableStyles())

	// Initialize preview viewport
	vp := viewport.New(width-6, height-10)
	vp.Style = HistoryPreviewStyle

	// Initialize search input
	search := textinput.New()
	search.Placeholder = "Search sessions, messages, or models..."
	search.Width = width - 6
	search.Blur()

	return &HistoryBrowser{
		list:          l,
		table:         t,
		preview:       vp,
		searchInput:   search,
		sessions:      make([]storage.ChatSession, 0),
		exportFormats: []string{"JSON", "Text", "Markdown", "CSV"},
		width:         width,
		height:        height,
		sortBy:        "date",
		sortDesc:      true,
		analytics:     &HistoryAnalytics{},
	}
}

// Init initializes the history browser
func (hb *HistoryBrowser) Init() tea.Cmd {
	// Bubble tea list and textinput models don't have Init() methods
	// They are initialized when created with New()
	return nil
}

// Update handles history browser updates
func (hb *HistoryBrowser) Update(msg tea.Msg) (*HistoryBrowser, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		hb.width = msg.Width
		hb.height = msg.Height
		hb.list.SetWidth(msg.Width - 4)
		hb.list.SetHeight(msg.Height - 8)
		hb.table.SetWidth(msg.Width - 4)
		hb.table.SetHeight(msg.Height - 10)
		hb.preview.Width = msg.Width - 6
		hb.preview.Height = msg.Height - 10
		hb.searchInput.Width = msg.Width - 6

	case HistoryMsg:
		switch msg.Type {
		case "set_sessions":
			if sessions, ok := msg.Data.([]storage.ChatSession); ok {
				hb.SetSessions(sessions)
			}
		case "set_loading":
			if loading, ok := msg.Data.(bool); ok {
				hb.loading = loading
			}
		case "set_error":
			if err, ok := msg.Data.(string); ok {
				hb.errorMessage = err
			}
		case "delete_session":
			if sessionID, ok := msg.Data.(string); ok {
				return hb, hb.deleteSession(sessionID)
			}
		case "export_session":
			if data, ok := msg.Data.(map[string]interface{}); ok {
				sessionID := data["session_id"].(string)
				format := data["format"].(string)
				return hb, hb.exportSession(sessionID, format)
			}
		case "refresh":
			return hb, hb.refresh()
		}

	case tea.KeyMsg:
		// Handle global shortcuts
		switch msg.String() {
		case "ctrl+f", "/":
			if !hb.searchActive {
				hb.searchActive = true
				return hb, hb.searchInput.Focus()
			}
		case "esc":
			if hb.searchActive {
				hb.searchActive = false
				hb.searchInput.Blur()
				hb.searchInput.SetValue("")
				hb.searchQuery = ""
				hb.filterSessions()
				return hb, nil
			}
			if hb.viewMode != HistoryViewList {
				hb.viewMode = HistoryViewList
				return hb, nil
			}
		case "enter":
			if hb.searchActive {
				hb.searchQuery = hb.searchInput.Value()
				hb.searchActive = false
				hb.searchInput.Blur()
				hb.filterSessions()
				return hb, nil
			}
			// Open selected session
			if hb.viewMode == HistoryViewList {
				if item, ok := hb.list.SelectedItem().(SessionItem); ok {
					hb.selectedSession = &item.session
					hb.viewMode = HistoryViewPreview
					hb.updatePreview()
				}
			}
		case "1":
			hb.viewMode = HistoryViewList
		case "2":
			hb.viewMode = HistoryViewTable
			hb.updateTable()
		case "3":
			if hb.selectedSession != nil {
				hb.viewMode = HistoryViewPreview
				hb.updatePreview()
			}
		case "4":
			hb.viewMode = HistoryViewExport
		case "d":
			if !hb.searchActive && hb.viewMode == HistoryViewList {
				if item, ok := hb.list.SelectedItem().(SessionItem); ok {
					return hb, hb.deleteSession(item.session.ID)
				}
			}
		case "e":
			if !hb.searchActive && hb.selectedSession != nil {
				hb.viewMode = HistoryViewExport
			}
		case "r":
			if !hb.searchActive {
				return hb, hb.refresh()
			}
		case "s":
			if !hb.searchActive {
				hb.cycleSortOrder()
				hb.sortSessions()
			}
		case "ctrl+a":
			return hb, hb.exportAll()
		}

		// Handle view-specific input
		if hb.searchActive {
			hb.searchInput, cmd = hb.searchInput.Update(msg)
			cmds = append(cmds, cmd)

			// Live search
			query := hb.searchInput.Value()
			if query != hb.searchQuery {
				hb.searchQuery = query
				hb.filterSessions()
			}
		} else {
			switch hb.viewMode {
			case HistoryViewList:
				hb.list, cmd = hb.list.Update(msg)
				cmds = append(cmds, cmd)
			case HistoryViewTable:
				hb.table, cmd = hb.table.Update(msg)
				cmds = append(cmds, cmd)
			case HistoryViewPreview:
				hb.preview, cmd = hb.preview.Update(msg)
				cmds = append(cmds, cmd)
			case HistoryViewExport:
				// Handle export selection
				switch msg.String() {
				case "up", "k":
					if hb.selectedFormat > 0 {
						hb.selectedFormat--
					}
				case "down", "j":
					if hb.selectedFormat < len(hb.exportFormats)-1 {
						hb.selectedFormat++
					}
				case "enter":
					if hb.selectedSession != nil {
						format := strings.ToLower(hb.exportFormats[hb.selectedFormat])
						return hb, hb.exportSession(hb.selectedSession.ID, format)
					}
				}
			}
		}

	default:
		if hb.searchActive {
			hb.searchInput, cmd = hb.searchInput.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			switch hb.viewMode {
			case HistoryViewList:
				hb.list, cmd = hb.list.Update(msg)
				cmds = append(cmds, cmd)
			case HistoryViewTable:
				hb.table, cmd = hb.table.Update(msg)
				cmds = append(cmds, cmd)
			case HistoryViewPreview:
				hb.preview, cmd = hb.preview.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return hb, tea.Batch(cmds...)
}

// View renders the history browser
func (hb *HistoryBrowser) View() string {
	var content strings.Builder

	// Header
	content.WriteString(hb.renderHeader())
	content.WriteString("\n")

	// Search input if active
	if hb.searchActive {
		content.WriteString(HistorySearchStyle.Render(hb.searchInput.View()))
		content.WriteString("\n")
	}

	// Error message
	if hb.errorMessage != "" {
		content.WriteString(ErrorStyle.Render("Error: " + hb.errorMessage))
		content.WriteString("\n")
	}

	// Loading indicator
	if hb.loading {
		content.WriteString(LoadingStyle.Render("Loading history..."))
		content.WriteString("\n")
	}

	// Main content based on view mode
	switch hb.viewMode {
	case HistoryViewList:
		content.WriteString(hb.list.View())
	case HistoryViewTable:
		content.WriteString(hb.table.View())
	case HistoryViewPreview:
		content.WriteString(hb.renderPreview())
	case HistoryViewExport:
		content.WriteString(hb.renderExportView())
	}

	// Footer
	content.WriteString("\n")
	content.WriteString(hb.renderFooter())

	return HistoryContainerStyle.Render(content.String())
}

// SetSessions sets the chat sessions
func (hb *HistoryBrowser) SetSessions(sessions []storage.ChatSession) {
	hb.sessions = sessions
	hb.calculateAnalytics()
	hb.sortSessions()
	hb.filterSessions()
}

// filterSessions filters sessions based on search query
func (hb *HistoryBrowser) filterSessions() {
	filtered := make([]SessionItem, 0)

	for _, session := range hb.sessions {
		item := hb.createSessionItem(session)

		if hb.searchQuery == "" || hb.matchesSearch(session, hb.searchQuery) {
			if hb.searchQuery != "" {
				item.highlighted = true
			}
			filtered = append(filtered, item)
		}
	}

	hb.filteredSessions = filtered
	hb.analytics.SearchResults = len(filtered)

	// Update list items
	listItems := make([]list.Item, len(filtered))
	for i, item := range filtered {
		listItems[i] = item
	}
	hb.list.SetItems(listItems)

	// Update table rows
	hb.updateTable()
}

// matchesSearch checks if a session matches the search query
func (hb *HistoryBrowser) matchesSearch(session storage.ChatSession, query string) bool {
	query = strings.ToLower(query)

	// Search in title
	if strings.Contains(strings.ToLower(session.Title), query) {
		return true
	}

	// Search in session ID
	if strings.Contains(strings.ToLower(session.ID), query) {
		return true
	}

	// Search in messages
	for _, msg := range session.Messages {
		if strings.Contains(strings.ToLower(msg.Content), query) {
			return true
		}
	}

	// Search in model names
	models := hb.extractModels(session)
	for _, model := range models {
		if strings.Contains(strings.ToLower(model), query) {
			return true
		}
	}

	return false
}

// sortSessions sorts sessions based on current sort criteria
func (hb *HistoryBrowser) sortSessions() {
	sort.Slice(hb.sessions, func(i, j int) bool {
		var result bool

		switch hb.sortBy {
		case "date":
			result = hb.sessions[i].CreatedAt.After(hb.sessions[j].CreatedAt)
		case "title":
			result = strings.ToLower(hb.sessions[i].Title) < strings.ToLower(hb.sessions[j].Title)
		case "messages":
			result = len(hb.sessions[i].Messages) > len(hb.sessions[j].Messages)
		case "tokens":
			tokensI := hb.calculateTokens(hb.sessions[i])
			tokensJ := hb.calculateTokens(hb.sessions[j])
			result = tokensI > tokensJ
		default:
			result = hb.sessions[i].CreatedAt.After(hb.sessions[j].CreatedAt)
		}

		if hb.sortDesc {
			return result
		}
		return !result
	})
}

// cycleSortOrder cycles through sort orders
func (hb *HistoryBrowser) cycleSortOrder() {
	switch hb.sortBy {
	case "date":
		hb.sortBy = "title"
	case "title":
		hb.sortBy = "messages"
	case "messages":
		hb.sortBy = "tokens"
	case "tokens":
		hb.sortBy = "date"
		hb.sortDesc = !hb.sortDesc
	}
}

// createSessionItem creates a SessionItem with metadata
func (hb *HistoryBrowser) createSessionItem(session storage.ChatSession) SessionItem {
	metadata := &SessionMetadata{
		MessageCount: len(session.Messages),
		TokenCount:   hb.calculateTokens(session),
		Models:       hb.extractModels(session),
	}

	if len(session.Messages) > 0 {
		lastMsg := session.Messages[len(session.Messages)-1]
		metadata.LastMessage = lastMsg.Content
		metadata.Duration = session.UpdatedAt.Sub(session.CreatedAt)
	}

	return SessionItem{
		session:  session,
		metadata: metadata,
	}
}

// calculateTokens estimates token count for a session
func (hb *HistoryBrowser) calculateTokens(session storage.ChatSession) int {
	total := 0
	for _, msg := range session.Messages {
		// Rough estimation: 1 token ≈ 4 characters
		total += len(msg.Content) / 4
	}
	return total
}

// extractModels extracts unique models used in a session
func (hb *HistoryBrowser) extractModels(session storage.ChatSession) []string {
	modelSet := make(map[string]bool)
	for _, msg := range session.Messages {
		if msg.Model != "" {
			modelSet[msg.Model] = true
		}
	}

	models := make([]string, 0, len(modelSet))
	for model := range modelSet {
		models = append(models, model)
	}

	sort.Strings(models)
	return models
}

// calculateAnalytics calculates analytics for the session history
func (hb *HistoryBrowser) calculateAnalytics() {
	analytics := &HistoryAnalytics{
		TotalSessions: len(hb.sessions),
		TopModels:     make([]ModelUsage, 0),
		DailyActivity: make([]DayActivity, 0),
	}

	modelUsage := make(map[string]*ModelUsage)
	dailyActivity := make(map[string]*DayActivity)

	totalDuration := time.Duration(0)

	for _, session := range hb.sessions {
		analytics.TotalMessages += len(session.Messages)
		analytics.TotalTokens += hb.calculateTokens(session)

		duration := session.UpdatedAt.Sub(session.CreatedAt)
		totalDuration += duration

		// Track model usage
		models := hb.extractModels(session)
		for _, model := range models {
			if usage, exists := modelUsage[model]; exists {
				usage.Count++
				usage.Tokens += hb.calculateTokens(session)
			} else {
				modelUsage[model] = &ModelUsage{
					Model:  model,
					Count:  1,
					Tokens: hb.calculateTokens(session),
				}
			}
		}

		// Track daily activity
		dateKey := session.CreatedAt.Format("2006-01-02")
		if activity, exists := dailyActivity[dateKey]; exists {
			activity.Sessions++
			activity.Messages += len(session.Messages)
		} else {
			dailyActivity[dateKey] = &DayActivity{
				Date:     session.CreatedAt,
				Sessions: 1,
				Messages: len(session.Messages),
			}
		}
	}

	if analytics.TotalSessions > 0 {
		analytics.AvgSessionLength = totalDuration / time.Duration(analytics.TotalSessions)
	}

	// Convert maps to slices and sort
	for _, usage := range modelUsage {
		analytics.TopModels = append(analytics.TopModels, *usage)
	}
	sort.Slice(analytics.TopModels, func(i, j int) bool {
		return analytics.TopModels[i].Count > analytics.TopModels[j].Count
	})

	for _, activity := range dailyActivity {
		analytics.DailyActivity = append(analytics.DailyActivity, *activity)
	}
	sort.Slice(analytics.DailyActivity, func(i, j int) bool {
		return analytics.DailyActivity[i].Date.After(analytics.DailyActivity[j].Date)
	})

	hb.analytics = analytics
}

// updateTable updates the table view with current sessions
func (hb *HistoryBrowser) updateTable() {
	rows := make([]table.Row, 0, len(hb.filteredSessions))

	for _, item := range hb.filteredSessions {
		session := item.session
		title := session.Title
		if title == "" {
			title = fmt.Sprintf("Session %s", session.ID[:8])
		}

		models := strings.Join(item.metadata.Models, ", ")
		if len(models) > 13 {
			models = models[:10] + "..."
		}

		duration := item.metadata.Duration.Round(time.Minute).String()

		row := table.Row{
			title,
			session.CreatedAt.Format("2006-01-02"),
			fmt.Sprintf("%d", item.metadata.MessageCount),
			humanize.Comma(int64(item.metadata.TokenCount)),
			models,
			duration,
		}
		rows = append(rows, row)
	}

	hb.table.SetRows(rows)
}

// updatePreview updates the preview content
func (hb *HistoryBrowser) updatePreview() {
	if hb.selectedSession == nil {
		hb.preview.SetContent("No session selected")
		return
	}

	var content strings.Builder
	session := *hb.selectedSession

	// Session header
	content.WriteString(HistoryPreviewTitleStyle.Render(session.Title))
	content.WriteString("\n")
	content.WriteString(HistoryPreviewMetaStyle.Render(fmt.Sprintf("Created: %s | Updated: %s | Messages: %d",
		session.CreatedAt.Format("2006-01-02 15:04:05"),
		session.UpdatedAt.Format("2006-01-02 15:04:05"),
		len(session.Messages))))
	content.WriteString("\n\n")

	// Messages
	for i, msg := range session.Messages {
		// Message header
		role := strings.Title(msg.Role)
		timestamp := msg.Timestamp.Format("15:04:05")
		header := fmt.Sprintf("%s (%s)", role, timestamp)

		var headerStyle lipgloss.Style
		switch msg.Role {
		case "user":
			headerStyle = HistoryPreviewUserHeaderStyle
		case "assistant":
			headerStyle = HistoryPreviewAssistantHeaderStyle
		default:
			headerStyle = HistoryPreviewSystemHeaderStyle
		}

		content.WriteString(headerStyle.Render(header))
		content.WriteString("\n")

		// Message content
		msgContent := msg.Content
		if len(msgContent) > 1000 {
			msgContent = msgContent[:997] + "..."
		}

		content.WriteString(HistoryPreviewContentStyle.Render(msgContent))

		if i < len(session.Messages)-1 {
			content.WriteString("\n\n")
		}
	}

	hb.preview.SetContent(content.String())
}

// renderHeader renders the header with view mode tabs and analytics
func (hb *HistoryBrowser) renderHeader() string {
	var content strings.Builder

	// Title and analytics
	title := "Chat History"
	if hb.analytics.TotalSessions > 0 {
		title += fmt.Sprintf(" (%d sessions, %d messages)",
			hb.analytics.TotalSessions,
			hb.analytics.TotalMessages)
	}

	if hb.searchQuery != "" {
		title += fmt.Sprintf(" - Search: \"%s\" (%d results)",
			hb.searchQuery,
			hb.analytics.SearchResults)
	}

	content.WriteString(HistoryTitleStyle.Render(title))
	content.WriteString("\n")

	// View mode tabs
	tabs := []string{"List", "Table", "Preview", "Export"}
	var tabRendered []string

	for i, tab := range tabs {
		if HistoryViewMode(i) == hb.viewMode {
			tabRendered = append(tabRendered, HistoryActiveTabStyle.Render(fmt.Sprintf("%d:%s", i+1, tab)))
		} else {
			tabRendered = append(tabRendered, HistoryInactiveTabStyle.Render(fmt.Sprintf("%d:%s", i+1, tab)))
		}
	}

	content.WriteString(HistoryTabContainerStyle.Render(strings.Join(tabRendered, "")))

	return content.String()
}

// renderPreview renders the preview view
func (hb *HistoryBrowser) renderPreview() string {
	if hb.selectedSession == nil {
		return HistoryPreviewStyle.Render("No session selected. Press Esc to return to list.")
	}

	return HistoryPreviewContainerStyle.Render(hb.preview.View())
}

// renderExportView renders the export selection view
func (hb *HistoryBrowser) renderExportView() string {
	var content strings.Builder

	content.WriteString(HistoryExportTitleStyle.Render("Export Options"))
	content.WriteString("\n\n")

	if hb.selectedSession != nil {
		content.WriteString(fmt.Sprintf("Selected Session: %s\n", hb.selectedSession.Title))
		content.WriteString("\n")
	}

	content.WriteString("Choose format:\n")
	for i, format := range hb.exportFormats {
		prefix := "  "
		style := HistoryExportOptionStyle

		if i == hb.selectedFormat {
			prefix = "→ "
			style = HistoryExportSelectedStyle
		}

		content.WriteString(style.Render(fmt.Sprintf("%s%s", prefix, format)))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString("Ctrl+A: Export all sessions")

	return HistoryExportContainerStyle.Render(content.String())
}

// renderFooter renders the footer with keyboard shortcuts
func (hb *HistoryBrowser) renderFooter() string {
	var shortcuts []string

	if hb.searchActive {
		shortcuts = []string{"enter: search", "esc: cancel"}
	} else {
		switch hb.viewMode {
		case HistoryViewList:
			shortcuts = []string{
				"enter: preview", "d: delete", "e: export", "/: search", "s: sort", "r: refresh",
			}
		case HistoryViewTable:
			shortcuts = []string{
				"1: list", "3: preview", "s: sort", "/: search", "r: refresh",
			}
		case HistoryViewPreview:
			shortcuts = []string{
				"esc: back", "e: export", "d: delete",
			}
		case HistoryViewExport:
			shortcuts = []string{
				"enter: export", "esc: back", "ctrl+a: export all",
			}
		}
	}

	return HistoryFooterStyle.Render(strings.Join(shortcuts, " • "))
}

// Command functions
func (hb *HistoryBrowser) deleteSession(sessionID string) tea.Cmd {
	return func() tea.Msg {
		return HistoryMsg{Type: "delete_requested", Data: sessionID}
	}
}

func (hb *HistoryBrowser) exportSession(sessionID, format string) tea.Cmd {
	return func() tea.Msg {
		return HistoryMsg{Type: "export_requested", Data: map[string]interface{}{
			"session_id": sessionID,
			"format":     format,
		}}
	}
}

func (hb *HistoryBrowser) exportAll() tea.Cmd {
	return func() tea.Msg {
		return HistoryMsg{Type: "export_all_requested"}
	}
}

func (hb *HistoryBrowser) refresh() tea.Cmd {
	return func() tea.Msg {
		return HistoryMsg{Type: "refresh_requested"}
	}
}

// NewHistoryDelegate creates a delegate for history list items
func NewHistoryDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = HistorySelectedItemStyle
	d.Styles.SelectedDesc = HistorySelectedItemDescStyle
	d.Styles.NormalTitle = HistoryItemStyle
	d.Styles.NormalDesc = HistoryItemDescStyle

	d.SetHeight(2)
	d.SetSpacing(1)

	return d
}

// HistoryTableStyles returns table styles for history
func HistoryTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#7C3AED")).
		Background(lipgloss.Color("#F3F4F6")).
		Bold(false)
	return s
}

// History component styles
var (
	// Container styles
	HistoryContainerStyle = lipgloss.NewStyle().
				Padding(1)

	HistoryTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	// Tab styles
	HistoryTabContainerStyle = lipgloss.NewStyle().
					BorderBottom(true).
					BorderStyle(lipgloss.NormalBorder()).
					BorderForeground(lipgloss.Color("#E5E7EB")).
					MarginBottom(1)

	HistoryActiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Background(lipgloss.Color("#F3F4F6")).
				Bold(true).
				Padding(0, 2).
				MarginRight(1)

	HistoryInactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				Padding(0, 2).
				MarginRight(1)

	// List styles
	HistoryListTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	HistoryListPaginationStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#6B7280"))

	HistoryItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#1F2937"))

	HistoryItemDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	HistorySelectedItemStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#7C3AED")).
					Bold(true).
					Background(lipgloss.Color("#F3F4F6"))

	HistorySelectedItemDescStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#7C3AED")).
					Background(lipgloss.Color("#F3F4F6"))

	// Search styles
	HistorySearchStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED")).
				Padding(0, 1).
				MarginBottom(1)

	// Preview styles
	HistoryPreviewContainerStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#D1D5DB")).
					Padding(1)

	HistoryPreviewStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				PaddingRight(1)

	HistoryPreviewTitleStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#7C3AED")).
					Bold(true)

	HistoryPreviewMetaStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				Italic(true)

	HistoryPreviewUserHeaderStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#3B82F6")).
					Bold(true)

	HistoryPreviewAssistantHeaderStyle = lipgloss.NewStyle().
						Foreground(lipgloss.Color("#7C3AED")).
						Bold(true)

	HistoryPreviewSystemHeaderStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#6B7280")).
					Bold(true)

	HistoryPreviewContentStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#1F2937")).
					PaddingLeft(2)

	// Export styles
	HistoryExportContainerStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#D1D5DB")).
					Padding(2)

	HistoryExportTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)

	HistoryExportOptionStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#374151"))

	HistoryExportSelectedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#7C3AED")).
					Bold(true).
					Background(lipgloss.Color("#F3F4F6"))

	// Footer styles
	HistoryFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				BorderTop(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#E5E7EB")).
				PaddingTop(1)
)

// Helper functions for integration with app state
func NewHistoryBrowserFromState(state *app.HistoryState, width, height int) *HistoryBrowser {
	hb := NewHistoryBrowser(width, height)
	hb.SetSessions(state.Sessions)
	hb.loading = state.Loading

	if state.Error != nil {
		hb.errorMessage = state.Error.Error()
	}

	return hb
}

// UpdateFromState updates the history browser from app state
func (hb *HistoryBrowser) UpdateFromState(state *app.HistoryState) {
	hb.SetSessions(state.Sessions)
	hb.loading = state.Loading

	if state.Error != nil {
		hb.errorMessage = state.Error.Error()
	} else {
		hb.errorMessage = ""
	}
}
