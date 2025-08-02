package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/john/klip/internal/api"
	"github.com/john/klip/internal/app"
)

// StatusMsg represents messages for status components
type StatusMsg struct {
	Type string
	Data interface{}
}

// NotificationType represents different types of notifications
type NotificationType int

const (
	NotificationInfo NotificationType = iota
	NotificationSuccess
	NotificationWarning
	NotificationError
)

// Notification represents a user notification
type Notification struct {
	ID       string
	Type     NotificationType
	Title    string
	Message  string
	Duration time.Duration
	ShowTime time.Time
	Actions  []NotificationAction
}

// NotificationAction represents an action button on a notification
type NotificationAction struct {
	Label   string
	Command string
	Style   lipgloss.Style
}

// ConnectionState represents the current connection status
type ConnectionState int

const (
	ConnectionDisconnected ConnectionState = iota
	ConnectionConnecting
	ConnectionConnected
	ConnectionError
)

// StatusBar provides a comprehensive status display
type StatusBar struct {
	// Connection status
	connectionState ConnectionState
	currentModel    string
	currentProvider string

	// Usage tracking
	tokenCount      int
	estimatedCost   float64
	requestCount    int
	sessionDuration time.Duration
	sessionStart    time.Time

	// Performance metrics
	avgLatency      time.Duration
	lastRequestTime time.Duration
	queuedRequests  int

	// System status
	memoryUsage    int64
	networkQuality int // 0-100
	apiHealth      map[string]bool

	width  int
	height int
}

// ProgressTracker manages multiple progress operations
type ProgressTracker struct {
	operations map[string]*ProgressOperation
	width      int
	height     int
}

// ProgressOperation represents a single progress operation
type ProgressOperation struct {
	ID           string
	Title        string
	Progress     float64
	Total        int64
	Current      int64
	Unit         string
	StartTime    time.Time
	EstimatedEnd time.Time
	Status       string
	Cancelable   bool
	progress     progress.Model
}

// LoadingSpinner provides various loading indicators
type LoadingSpinner struct {
	spinner    spinner.Model
	message    string
	subMessage string
	showTime   bool
	startTime  time.Time
	width      int
	height     int
}

// NotificationCenter manages user notifications
type NotificationCenter struct {
	notifications []Notification
	maxVisible    int
	width         int
	height        int
	position      NotificationPosition
}

// NotificationPosition represents where notifications appear
type NotificationPosition int

const (
	NotificationTopRight NotificationPosition = iota
	NotificationTopLeft
	NotificationBottomRight
	NotificationBottomLeft
	NotificationPositionCenter
)

// TokenUsageDisplay shows token usage and costs
type TokenUsageDisplay struct {
	currentTokens int
	sessionTokens int
	totalTokens   int64
	estimatedCost float64
	sessionCost   float64
	totalCost     float64
	currentModel  string
	rateLimit     int
	rateLimitUsed int
	width         int
	height        int
	showDetails   bool
}

// NewStatusBar creates a new status bar
func NewStatusBar(width, height int) *StatusBar {
	return &StatusBar{
		connectionState: ConnectionDisconnected,
		apiHealth:       make(map[string]bool),
		sessionStart:    time.Now(),
		width:           width,
		height:          height,
	}
}

// Update handles status bar updates
func (sb *StatusBar) Update(msg tea.Msg) (*StatusBar, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		sb.width = msg.Width
		sb.height = msg.Height

	case StatusMsg:
		switch msg.Type {
		case "connection_state":
			if state, ok := msg.Data.(ConnectionState); ok {
				sb.connectionState = state
			}
		case "model_changed":
			if model, ok := msg.Data.(api.Model); ok {
				sb.currentModel = model.Name
				sb.currentProvider = model.Provider.String()
			}
		case "token_update":
			if tokens, ok := msg.Data.(int); ok {
				sb.tokenCount += tokens
			}
		case "request_completed":
			sb.requestCount++
			if latency, ok := msg.Data.(time.Duration); ok {
				sb.lastRequestTime = latency
				if sb.avgLatency == 0 {
					sb.avgLatency = latency
				} else {
					sb.avgLatency = (sb.avgLatency + latency) / 2
				}
			}
		case "cost_update":
			if cost, ok := msg.Data.(float64); ok {
				sb.estimatedCost += cost
			}
		case "network_quality":
			if quality, ok := msg.Data.(int); ok {
				sb.networkQuality = quality
			}
		case "api_health":
			if healthData, ok := msg.Data.(map[string]bool); ok {
				for provider, healthy := range healthData {
					sb.apiHealth[provider] = healthy
				}
			}
		}
	}

	// Update session duration
	sb.sessionDuration = time.Since(sb.sessionStart)

	return sb, nil
}

// View renders the status bar
func (sb *StatusBar) View() string {
	var sections []string

	// Connection status
	sections = append(sections, sb.renderConnectionStatus())

	// Model info
	if sb.currentModel != "" {
		sections = append(sections, sb.renderModelInfo())
	}

	// Usage stats
	sections = append(sections, sb.renderUsageStats())

	// Performance metrics
	sections = append(sections, sb.renderPerformanceMetrics())

	// System status
	sections = append(sections, sb.renderSystemStatus())

	content := strings.Join(sections, StatusSeparatorStyle.Render(" │ "))

	return StatusBarStyle.Render(content)
}

// renderConnectionStatus renders the connection status indicator
func (sb *StatusBar) renderConnectionStatus() string {
	var status string
	var style lipgloss.Style

	switch sb.connectionState {
	case ConnectionDisconnected:
		status = "●"
		style = StatusDisconnectedStyle
	case ConnectionConnecting:
		status = "◐"
		style = StatusConnectingStyle
	case ConnectionConnected:
		status = "●"
		style = StatusConnectedStyle
	case ConnectionError:
		status = "●"
		style = StatusErrorStyle
	}

	return style.Render(status)
}

// renderModelInfo renders current model information
func (sb *StatusBar) renderModelInfo() string {
	if sb.currentProvider != "" {
		return ModelInfoStyle.Render(fmt.Sprintf("%s:%s", sb.currentProvider, sb.currentModel))
	}
	return ModelInfoStyle.Render(sb.currentModel)
}

// renderUsageStats renders usage statistics
func (sb *StatusBar) renderUsageStats() string {
	var parts []string

	if sb.tokenCount > 0 {
		parts = append(parts, fmt.Sprintf("%s tokens", humanize.Comma(int64(sb.tokenCount))))
	}

	if sb.estimatedCost > 0 {
		parts = append(parts, fmt.Sprintf("$%.4f", sb.estimatedCost))
	}

	if sb.requestCount > 0 {
		parts = append(parts, fmt.Sprintf("%d reqs", sb.requestCount))
	}

	if len(parts) == 0 {
		return ""
	}

	return UsageStatsStyle.Render(strings.Join(parts, " • "))
}

// renderPerformanceMetrics renders performance information
func (sb *StatusBar) renderPerformanceMetrics() string {
	var parts []string

	if sb.avgLatency > 0 {
		parts = append(parts, fmt.Sprintf("~%dms", sb.avgLatency.Milliseconds()))
	}

	if sb.queuedRequests > 0 {
		parts = append(parts, fmt.Sprintf("%d queued", sb.queuedRequests))
	}

	if len(parts) == 0 {
		return ""
	}

	return PerformanceStyle.Render(strings.Join(parts, " • "))
}

// renderSystemStatus renders system status information
func (sb *StatusBar) renderSystemStatus() string {
	var parts []string

	// Session duration
	duration := sb.sessionDuration.Round(time.Second)
	parts = append(parts, fmt.Sprintf("⏱ %s", duration))

	// Network quality
	if sb.networkQuality > 0 {
		var qualityIcon string
		switch {
		case sb.networkQuality >= 90:
			qualityIcon = "📶"
		case sb.networkQuality >= 70:
			qualityIcon = "📶"
		case sb.networkQuality >= 50:
			qualityIcon = "📶"
		default:
			qualityIcon = "📶"
		}
		parts = append(parts, qualityIcon)
	}

	return SystemStatusStyle.Render(strings.Join(parts, " "))
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(width, height int) *ProgressTracker {
	return &ProgressTracker{
		operations: make(map[string]*ProgressOperation),
		width:      width,
		height:     height,
	}
}

// AddOperation adds a new progress operation
func (pt *ProgressTracker) AddOperation(id, title string, total int64, unit string) {
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = pt.width - 20 // Leave space for text

	pt.operations[id] = &ProgressOperation{
		ID:         id,
		Title:      title,
		Total:      total,
		Unit:       unit,
		StartTime:  time.Now(),
		Status:     "Starting...",
		Cancelable: true,
		progress:   prog,
	}
}

// UpdateOperation updates progress for an operation
func (pt *ProgressTracker) UpdateOperation(id string, current int64, status string) {
	if op, exists := pt.operations[id]; exists {
		op.Current = current
		op.Status = status

		if op.Total > 0 {
			op.Progress = float64(current) / float64(op.Total)

			// Estimate completion time
			elapsed := time.Since(op.StartTime)
			if op.Progress > 0 {
				totalEstimated := time.Duration(float64(elapsed) / op.Progress)
				op.EstimatedEnd = op.StartTime.Add(totalEstimated)
			}
		}
	}
}

// CompleteOperation marks an operation as complete
func (pt *ProgressTracker) CompleteOperation(id string) {
	if op, exists := pt.operations[id]; exists {
		op.Progress = 1.0
		op.Current = op.Total
		op.Status = "Complete"
	}
}

// RemoveOperation removes a completed operation
func (pt *ProgressTracker) RemoveOperation(id string) {
	delete(pt.operations, id)
}

// Update handles progress tracker updates
func (pt *ProgressTracker) Update(msg tea.Msg) (*ProgressTracker, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		pt.width = msg.Width
		pt.height = msg.Height

		// Update all progress bar widths
		for _, op := range pt.operations {
			op.progress.Width = pt.width - 20
		}

	case StatusMsg:
		switch msg.Type {
		case "progress_add":
			if data, ok := msg.Data.(map[string]interface{}); ok {
				id := data["id"].(string)
				title := data["title"].(string)
				total := data["total"].(int64)
				unit := data["unit"].(string)
				pt.AddOperation(id, title, total, unit)
			}
		case "progress_update":
			if data, ok := msg.Data.(map[string]interface{}); ok {
				id := data["id"].(string)
				current := data["current"].(int64)
				status := data["status"].(string)
				pt.UpdateOperation(id, current, status)
			}
		case "progress_complete":
			if id, ok := msg.Data.(string); ok {
				pt.CompleteOperation(id)
			}
		case "progress_remove":
			if id, ok := msg.Data.(string); ok {
				pt.RemoveOperation(id)
			}
		}
	}

	// Update all progress bars
	for id, op := range pt.operations {
		var cmd tea.Cmd
		model, cmd := op.progress.Update(msg)
		op.progress = model.(progress.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		pt.operations[id] = op
	}

	return pt, tea.Batch(cmds...)
}

// View renders all progress operations
func (pt *ProgressTracker) View() string {
	if len(pt.operations) == 0 {
		return ""
	}

	var content strings.Builder

	for _, op := range pt.operations {
		content.WriteString(pt.renderOperation(op))
		content.WriteString("\n")
	}

	return ProgressContainerStyle.Render(content.String())
}

// renderOperation renders a single progress operation
func (pt *ProgressTracker) renderOperation(op *ProgressOperation) string {
	var content strings.Builder

	// Title and status
	titleLine := fmt.Sprintf("%s - %s", op.Title, op.Status)
	content.WriteString(ProgressTitleStyle.Render(titleLine))
	content.WriteString("\n")

	// Progress bar
	content.WriteString(op.progress.ViewAs(op.Progress))
	content.WriteString("\n")

	// Details line
	var details []string

	if op.Total > 0 {
		details = append(details, fmt.Sprintf("%s/%s %s",
			humanize.Comma(op.Current),
			humanize.Comma(op.Total),
			op.Unit))

		percentage := int(op.Progress * 100)
		details = append(details, fmt.Sprintf("%d%%", percentage))
	}

	// Time estimates
	elapsed := time.Since(op.StartTime)
	details = append(details, fmt.Sprintf("elapsed: %s", elapsed.Round(time.Second)))

	if !op.EstimatedEnd.IsZero() && op.Progress > 0 && op.Progress < 1 {
		remaining := time.Until(op.EstimatedEnd)
		if remaining > 0 {
			details = append(details, fmt.Sprintf("remaining: ~%s", remaining.Round(time.Second)))
		}
	}

	if len(details) > 0 {
		content.WriteString(ProgressDetailsStyle.Render(strings.Join(details, " • ")))
	}

	return content.String()
}

// NewLoadingSpinner creates a new loading spinner
func NewLoadingSpinner(width, height int) *LoadingSpinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return &LoadingSpinner{
		spinner:   s,
		startTime: time.Now(),
		width:     width,
		height:    height,
	}
}

// SetMessage sets the main loading message
func (ls *LoadingSpinner) SetMessage(message string) {
	ls.message = message
}

// SetSubMessage sets the sub-message
func (ls *LoadingSpinner) SetSubMessage(subMessage string) {
	ls.subMessage = subMessage
}

// SetShowTime enables/disables time display
func (ls *LoadingSpinner) SetShowTime(show bool) {
	ls.showTime = show
}

// Update handles spinner updates
func (ls *LoadingSpinner) Update(msg tea.Msg) (*LoadingSpinner, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ls.width = msg.Width
		ls.height = msg.Height

	case StatusMsg:
		switch msg.Type {
		case "spinner_message":
			if message, ok := msg.Data.(string); ok {
				ls.SetMessage(message)
			}
		case "spinner_sub_message":
			if subMessage, ok := msg.Data.(string); ok {
				ls.SetSubMessage(subMessage)
			}
		}
	}

	var cmd tea.Cmd
	ls.spinner, cmd = ls.spinner.Update(msg)

	return ls, cmd
}

// View renders the loading spinner
func (ls *LoadingSpinner) View() string {
	var content strings.Builder

	// Spinner and main message
	line := fmt.Sprintf("%s %s", ls.spinner.View(), ls.message)
	content.WriteString(SpinnerMessageStyle.Render(line))

	// Sub-message
	if ls.subMessage != "" {
		content.WriteString("\n")
		content.WriteString(SpinnerSubMessageStyle.Render(ls.subMessage))
	}

	// Time display
	if ls.showTime {
		elapsed := time.Since(ls.startTime).Round(time.Second)
		content.WriteString("\n")
		content.WriteString(SpinnerTimeStyle.Render(fmt.Sprintf("(%s)", elapsed)))
	}

	return SpinnerContainerStyle.Render(content.String())
}

// NewNotificationCenter creates a new notification center
func NewNotificationCenter(width, height int) *NotificationCenter {
	return &NotificationCenter{
		notifications: make([]Notification, 0),
		maxVisible:    5,
		width:         width,
		height:        height,
		position:      NotificationTopRight,
	}
}

// AddNotification adds a new notification
func (nc *NotificationCenter) AddNotification(notification Notification) {
	notification.ShowTime = time.Now()
	nc.notifications = append(nc.notifications, notification)

	// Remove old notifications if we exceed the limit
	if len(nc.notifications) > 20 {
		nc.notifications = nc.notifications[len(nc.notifications)-20:]
	}
}

// RemoveNotification removes a notification by ID
func (nc *NotificationCenter) RemoveNotification(id string) {
	for i, notification := range nc.notifications {
		if notification.ID == id {
			nc.notifications = append(nc.notifications[:i], nc.notifications[i+1:]...)
			break
		}
	}
}

// Update handles notification center updates
func (nc *NotificationCenter) Update(msg tea.Msg) (*NotificationCenter, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		nc.width = msg.Width
		nc.height = msg.Height

	case StatusMsg:
		switch msg.Type {
		case "notification_add":
			if notification, ok := msg.Data.(Notification); ok {
				nc.AddNotification(notification)
			}
		case "notification_remove":
			if id, ok := msg.Data.(string); ok {
				nc.RemoveNotification(id)
			}
		}
	}

	// Auto-remove expired notifications
	now := time.Now()
	var active []Notification
	for _, notification := range nc.notifications {
		if notification.Duration > 0 && now.Sub(notification.ShowTime) < notification.Duration {
			active = append(active, notification)
		} else if notification.Duration == 0 {
			active = append(active, notification) // Permanent notification
		}
	}
	nc.notifications = active

	return nc, tea.Batch(cmds...)
}

// View renders visible notifications
func (nc *NotificationCenter) View() string {
	if len(nc.notifications) == 0 {
		return ""
	}

	// Show only the most recent notifications
	visible := nc.notifications
	if len(visible) > nc.maxVisible {
		visible = visible[len(visible)-nc.maxVisible:]
	}

	var content strings.Builder

	for i, notification := range visible {
		content.WriteString(nc.renderNotification(notification))
		if i < len(visible)-1 {
			content.WriteString("\n")
		}
	}

	return nc.positionContent(content.String())
}

// renderNotification renders a single notification
func (nc *NotificationCenter) renderNotification(notification Notification) string {
	var style lipgloss.Style
	var icon string

	switch notification.Type {
	case NotificationInfo:
		style = NotificationInfoStyle
		icon = "ℹ"
	case NotificationSuccess:
		style = NotificationSuccessStyle
		icon = "✓"
	case NotificationWarning:
		style = NotificationWarningStyle
		icon = "⚠"
	case NotificationError:
		style = NotificationErrorStyle
		icon = "✗"
	}

	var content strings.Builder

	// Title with icon
	title := fmt.Sprintf("%s %s", icon, notification.Title)
	content.WriteString(NotificationTitleStyle.Render(title))

	// Message
	if notification.Message != "" {
		content.WriteString("\n")
		content.WriteString(NotificationMessageStyle.Render(notification.Message))
	}

	// Actions
	if len(notification.Actions) > 0 {
		content.WriteString("\n")
		var actions []string
		for _, action := range notification.Actions {
			actions = append(actions, action.Style.Render(action.Label))
		}
		content.WriteString(strings.Join(actions, " "))
	}

	return style.Render(content.String())
}

// positionContent positions the notifications based on the position setting
func (nc *NotificationCenter) positionContent(content string) string {
	switch nc.position {
	case NotificationTopRight:
		return lipgloss.NewStyle().
			Align(lipgloss.Right).
			Render(content)
	case NotificationTopLeft:
		return lipgloss.NewStyle().
			Align(lipgloss.Left).
			Render(content)
	case NotificationBottomRight:
		return lipgloss.NewStyle().
			Align(lipgloss.Right).
			Render(content)
	case NotificationBottomLeft:
		return lipgloss.NewStyle().
			Align(lipgloss.Left).
			Render(content)
	case NotificationPositionCenter:
		return lipgloss.NewStyle().
			Align(lipgloss.Center).
			Render(content)
	default:
		return content
	}
}

// NewTokenUsageDisplay creates a new token usage display
func NewTokenUsageDisplay(width, height int) *TokenUsageDisplay {
	return &TokenUsageDisplay{
		width:  width,
		height: height,
	}
}

// Update handles token usage display updates
func (tud *TokenUsageDisplay) Update(msg tea.Msg) (*TokenUsageDisplay, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		tud.width = msg.Width
		tud.height = msg.Height

	case StatusMsg:
		switch msg.Type {
		case "token_current":
			if tokens, ok := msg.Data.(int); ok {
				tud.currentTokens = tokens
			}
		case "token_session":
			if tokens, ok := msg.Data.(int); ok {
				tud.sessionTokens = tokens
			}
		case "token_total":
			if tokens, ok := msg.Data.(int64); ok {
				tud.totalTokens = tokens
			}
		case "cost_estimated":
			if cost, ok := msg.Data.(float64); ok {
				tud.estimatedCost = cost
			}
		case "cost_session":
			if cost, ok := msg.Data.(float64); ok {
				tud.sessionCost = cost
			}
		case "cost_total":
			if cost, ok := msg.Data.(float64); ok {
				tud.totalCost = cost
			}
		case "model_current":
			if model, ok := msg.Data.(string); ok {
				tud.currentModel = model
			}
		case "rate_limit":
			if data, ok := msg.Data.(map[string]int); ok {
				if limit, exists := data["limit"]; exists {
					tud.rateLimit = limit
				}
				if used, exists := data["used"]; exists {
					tud.rateLimitUsed = used
				}
			}
		case "toggle_details":
			tud.showDetails = !tud.showDetails
		}
	}

	return tud, nil
}

// View renders the token usage display
func (tud *TokenUsageDisplay) View() string {
	if !tud.showDetails {
		return tud.renderCompact()
	}
	return tud.renderDetailed()
}

// renderCompact renders a compact token usage view
func (tud *TokenUsageDisplay) renderCompact() string {
	var parts []string

	if tud.currentTokens > 0 {
		parts = append(parts, fmt.Sprintf("%s tokens", humanize.Comma(int64(tud.currentTokens))))
	}

	if tud.estimatedCost > 0 {
		parts = append(parts, fmt.Sprintf("~$%.4f", tud.estimatedCost))
	}

	if len(parts) == 0 {
		return ""
	}

	return TokenUsageCompactStyle.Render(strings.Join(parts, " • "))
}

// renderDetailed renders a detailed token usage view
func (tud *TokenUsageDisplay) renderDetailed() string {
	var content strings.Builder

	content.WriteString(TokenUsageTitleStyle.Render("Token Usage"))
	content.WriteString("\n")

	// Current request
	if tud.currentTokens > 0 {
		content.WriteString(fmt.Sprintf("Current: %s tokens", humanize.Comma(int64(tud.currentTokens))))
		if tud.estimatedCost > 0 {
			content.WriteString(fmt.Sprintf(" (~$%.4f)", tud.estimatedCost))
		}
		content.WriteString("\n")
	}

	// Session totals
	if tud.sessionTokens > 0 {
		content.WriteString(fmt.Sprintf("Session: %s tokens", humanize.Comma(int64(tud.sessionTokens))))
		if tud.sessionCost > 0 {
			content.WriteString(fmt.Sprintf(" ($%.4f)", tud.sessionCost))
		}
		content.WriteString("\n")
	}

	// All-time totals
	if tud.totalTokens > 0 {
		content.WriteString(fmt.Sprintf("Total: %s tokens", humanize.Comma(tud.totalTokens)))
		if tud.totalCost > 0 {
			content.WriteString(fmt.Sprintf(" ($%.2f)", tud.totalCost))
		}
		content.WriteString("\n")
	}

	// Rate limits
	if tud.rateLimit > 0 {
		content.WriteString("\n")
		content.WriteString(TokenUsageRateLimitStyle.Render("Rate Limit"))
		content.WriteString("\n")

		percentage := float64(tud.rateLimitUsed) / float64(tud.rateLimit) * 100
		content.WriteString(fmt.Sprintf("%d/%d requests (%.1f%%)",
			tud.rateLimitUsed, tud.rateLimit, percentage))

		// Rate limit bar
		prog := progress.New(progress.WithDefaultGradient())
		prog.Width = tud.width - 10
		rateLimitProgress := float64(tud.rateLimitUsed) / float64(tud.rateLimit)
		content.WriteString("\n")
		content.WriteString(prog.ViewAs(rateLimitProgress))
	}

	// Current model
	if tud.currentModel != "" {
		content.WriteString("\n")
		content.WriteString(TokenUsageModelStyle.Render(fmt.Sprintf("Model: %s", tud.currentModel)))
	}

	return TokenUsageContainerStyle.Render(content.String())
}

// Status and progress component styles
var (
	// Status bar styles
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#F3F4F6")).
			Foreground(lipgloss.Color("#374151")).
			Padding(0, 1).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#E5E7EB"))

	StatusSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#D1D5DB"))

	StatusConnectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#10B981"))

	StatusConnectingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F59E0B"))

	StatusDisconnectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EF4444"))

	ModelInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

	UsageStatsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#059669"))

	PerformanceStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3B82F6"))

	SystemStatusStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	// Progress tracker styles
	ProgressContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#D1D5DB")).
				Padding(1).
				MarginBottom(1)

	ProgressTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#374151")).
				Bold(true)

	ProgressDetailsStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280"))

	// Spinner styles
	SpinnerContainerStyle = lipgloss.NewStyle().
				Padding(1).
				MarginBottom(1)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED"))

	SpinnerMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#374151")).
				Bold(true)

	SpinnerSubMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				PaddingLeft(2)

	SpinnerTimeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#9CA3AF")).
				Italic(true)

	// Notification styles
	NotificationInfoStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#3B82F6")).
				Background(lipgloss.Color("#EFF6FF")).
				Foreground(lipgloss.Color("#1E40AF")).
				Padding(1).
				MarginBottom(1)

	NotificationSuccessStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#10B981")).
					Background(lipgloss.Color("#ECFDF5")).
					Foreground(lipgloss.Color("#065F46")).
					Padding(1).
					MarginBottom(1)

	NotificationWarningStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#F59E0B")).
					Background(lipgloss.Color("#FFFBEB")).
					Foreground(lipgloss.Color("#92400E")).
					Padding(1).
					MarginBottom(1)

	NotificationErrorStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#EF4444")).
				Background(lipgloss.Color("#FEF2F2")).
				Foreground(lipgloss.Color("#991B1B")).
				Padding(1).
				MarginBottom(1)

	NotificationTitleStyle = lipgloss.NewStyle().
				Bold(true)

	NotificationMessageStyle = lipgloss.NewStyle().
					PaddingTop(1)

	// Token usage styles
	TokenUsageContainerStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("#D1D5DB")).
					Padding(1)

	TokenUsageCompactStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#059669"))

	TokenUsageTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#374151")).
				Bold(true)

	TokenUsageRateLimitStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#F59E0B")).
					Bold(true)

	TokenUsageModelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Italic(true)
)

// Helper functions for integration with app state
func NewStatusBarFromState(state *app.StatusState, width, height int) *StatusBar {
	sb := NewStatusBar(width, height)
	sb.connectionState = ConnectionState(state.ConnectionState)
	sb.currentModel = state.CurrentModel
	sb.currentProvider = state.CurrentProvider
	sb.tokenCount = state.TokenCount
	sb.estimatedCost = state.EstimatedCost
	sb.requestCount = state.RequestCount

	return sb
}

// UpdateFromState updates status components from app state
func (sb *StatusBar) UpdateFromState(state *app.StatusState) {
	sb.connectionState = ConnectionState(state.ConnectionState)
	sb.currentModel = state.CurrentModel
	sb.currentProvider = state.CurrentProvider
	sb.tokenCount = state.TokenCount
	sb.estimatedCost = state.EstimatedCost
	sb.requestCount = state.RequestCount
}
