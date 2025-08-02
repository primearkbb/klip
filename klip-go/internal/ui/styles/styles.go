package styles

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// StyleManager is the centralized style management system for Klip
type StyleManager struct {
	// Core managers
	themeManager      *ThemeManager
	layoutManager     *LayoutManager
	componentStyler   *ComponentStyler
	interactiveStyler *InteractiveStyler
	adaptiveStyler    *AdaptiveStyler
	textFormatter     *TextFormatter
	accessibilityMgr  *AccessibilityManager

	// Current state
	currentTheme *Theme
	width        int
	height       int

	// Style cache for performance
	styleCache map[string]lipgloss.Style
	cacheMutex sync.RWMutex

	// Configuration
	config      *StyleConfig
	initialized bool

	// Performance metrics
	renderCount    int64
	cacheHits      int64
	cacheMisses    int64
	lastUpdateTime time.Time
}

// StyleConfig holds global style configuration
type StyleConfig struct {
	// Performance settings
	EnableCaching    bool
	CacheSize        int
	AutoResize       bool
	OptimizeForSpeed bool

	// Feature flags
	EnableAnimations    bool
	EnableGradients     bool
	EnableAccessibility bool
	EnableAdaptation    bool

	// Debug settings
	DebugMode      bool
	ShowMetrics    bool
	LogPerformance bool
}

// NewStyleManager creates a new centralized style manager
func NewStyleManager(width, height int) *StyleManager {
	sm := &StyleManager{
		width:      width,
		height:     height,
		styleCache: make(map[string]lipgloss.Style),
		config:     getDefaultStyleConfig(),
	}

	// Initialize all managers
	sm.initialize()

	return sm
}

// initialize sets up all the style managers
func (sm *StyleManager) initialize() {
	// Initialize theme manager first
	sm.themeManager = NewThemeManager()
	sm.currentTheme = sm.themeManager.GetCurrentTheme()

	// Initialize adaptive styler to detect terminal capabilities
	sm.adaptiveStyler = NewAdaptiveStyler(sm.currentTheme, sm.width, sm.height)
	capabilities := sm.adaptiveStyler.GetCapabilities()

	// Apply adaptive theme
	if sm.config.EnableAdaptation {
		sm.currentTheme = sm.adaptiveStyler.AdaptTheme(sm.currentTheme)
	}

	// Initialize accessibility manager
	if sm.config.EnableAccessibility {
		sm.accessibilityMgr = NewAccessibilityManager(sm.currentTheme, capabilities)

		// Apply accessibility adaptations
		if sm.accessibilityMgr.IsAccessibilityEnabled() {
			sm.currentTheme = sm.accessibilityMgr.CreateAccessibleTheme(sm.currentTheme, AccessibilityAA)
		}
	}

	// Initialize other managers with the final theme
	sm.layoutManager = NewLayoutManager(sm.width, sm.height, sm.currentTheme)
	sm.componentStyler = NewComponentStyler(sm.currentTheme, sm.width, sm.height)
	sm.textFormatter = NewTextFormatter(sm.currentTheme, sm.width, sm.height, capabilities)

	// Initialize interactive styler if animations are enabled
	if sm.config.EnableAnimations {
		sm.interactiveStyler = NewInteractiveStyler(sm.currentTheme, sm.width, sm.height)
	}

	sm.initialized = true
	sm.lastUpdateTime = time.Now()
}

// Theme Management

// SetTheme switches to a new theme
func (sm *StyleManager) SetTheme(themeName string) error {
	if !sm.initialized {
		return fmt.Errorf("style manager not initialized")
	}

	// Set theme through theme manager
	err := sm.themeManager.SetTheme(themeName)
	if err != nil {
		return err
	}

	// Update current theme
	sm.currentTheme = sm.themeManager.GetCurrentTheme()

	// Apply adaptations
	if sm.config.EnableAdaptation {
		sm.currentTheme = sm.adaptiveStyler.AdaptTheme(sm.currentTheme)
	}

	if sm.config.EnableAccessibility && sm.accessibilityMgr != nil && sm.accessibilityMgr.IsAccessibilityEnabled() {
		sm.currentTheme = sm.accessibilityMgr.CreateAccessibleTheme(sm.currentTheme, AccessibilityAA)
	}

	// Update all managers with new theme
	sm.updateManagersWithTheme()

	// Clear cache to force re-rendering with new theme
	sm.clearCache()

	return nil
}

// GetCurrentTheme returns the currently active theme
func (sm *StyleManager) GetCurrentTheme() *Theme {
	return sm.currentTheme
}

// GetAvailableThemes returns all available themes
func (sm *StyleManager) GetAvailableThemes() map[string]*Theme {
	if sm.themeManager == nil {
		return make(map[string]*Theme)
	}
	return sm.themeManager.GetAvailableThemes()
}

// Layout Management

// CreateLayout creates a layout of the specified type
func (sm *StyleManager) CreateLayout(layoutType string) interface{} {
	switch layoutType {
	case "application":
		return sm.layoutManager.CreateApplicationLayout()
	case "grid":
		return sm.layoutManager.CreateResponsiveGrid(3) // Default 3 columns
	case "flex":
		return sm.layoutManager.CreateFlexLayout(Row)
	default:
		return sm.layoutManager.CreateApplicationLayout()
	}
}

// RenderLayout renders content within a layout
func (sm *StyleManager) RenderLayout(layout interface{}, content ...string) string {
	switch l := layout.(type) {
	case *PanelLayout:
		if len(content) >= 4 {
			return sm.layoutManager.RenderPanelLayout(l, content[0], content[1], content[2], content[3])
		} else if len(content) >= 3 {
			return sm.layoutManager.RenderPanelLayout(l, content[0], "", content[1], content[2])
		} else if len(content) >= 2 {
			return sm.layoutManager.RenderPanelLayout(l, content[0], "", content[1], "")
		} else if len(content) >= 1 {
			return sm.layoutManager.RenderPanelLayout(l, "", "", content[0], "")
		}
		return ""
	case *Grid:
		return sm.layoutManager.RenderGrid(l, content)
	case *Layout:
		return sm.layoutManager.RenderFlex(l, content)
	default:
		if len(content) > 0 {
			return content[0]
		}
		return ""
	}
}

// Component Styling

// StyleButton creates a styled button
func (sm *StyleManager) StyleButton(text string, variant ButtonStyle, size ButtonSize, state ButtonState) string {
	cacheKey := fmt.Sprintf("button_%s_%d_%d_%d", text, variant, size, state)

	if cached, exists := sm.getCached(cacheKey); exists {
		return cached.Render(text)
	}

	result := sm.componentStyler.Button(text, variant, size, state)

	// Cache the style (not the rendered result since text varies)
	sm.setCached(cacheKey, lipgloss.NewStyle())

	return result
}

// StyleInput creates a styled input field
func (sm *StyleManager) StyleInput(value, placeholder string, inputType InputType, state InputState, width int) string {
	return sm.componentStyler.Input(value, placeholder, inputType, state, width)
}

// StyleChatBubble creates a styled chat bubble
func (sm *StyleManager) StyleChatBubble(content, author string, bubbleType ChatBubbleType, timestamp string) string {
	return sm.componentStyler.ChatBubble(content, author, bubbleType, timestamp)
}

// StyleList creates a styled list
func (sm *StyleManager) StyleList(items []string, listType ListType, selectedIndex int) string {
	return sm.componentStyler.List(items, listType, selectedIndex)
}

// StyleCard creates a styled card
func (sm *StyleManager) StyleCard(title, content string, cardType CardType, width int) string {
	return sm.componentStyler.Card(title, content, cardType, width)
}

// StyleProgressBar creates a styled progress bar
func (sm *StyleManager) StyleProgressBar(progress float64, width int, showPercentage bool) string {
	return sm.componentStyler.ProgressBar(progress, width, showPercentage)
}

// Interactive Styling (if animations enabled)

// StyleInteractiveButton creates an animated interactive button
func (sm *StyleManager) StyleInteractiveButton(text string, variant ButtonStyle, size ButtonSize, state InteractionState, progress float64) string {
	if sm.interactiveStyler == nil || !sm.config.EnableAnimations {
		// Fall back to static button
		buttonState := ButtonStateNormal
		switch state {
		case StateHover:
			buttonState = ButtonStateNormal // No visual change without animations
		case StateFocus:
			buttonState = ButtonStateNormal
		case StateActive:
			buttonState = ButtonStateActive
		case StateLoading:
			buttonState = ButtonStateLoading
		}
		return sm.componentStyler.Button(text, variant, size, buttonState)
	}

	return sm.interactiveStyler.InteractiveButton(text, variant, size, state, progress)
}

// StyleAnimatedProgress creates an animated progress bar
func (sm *StyleManager) StyleAnimatedProgress(targetProgress, currentProgress float64, width int, showPercentage bool, animationType AnimationType) string {
	if sm.interactiveStyler == nil || !sm.config.EnableAnimations {
		return sm.componentStyler.ProgressBar(currentProgress, width, showPercentage)
	}

	return sm.interactiveStyler.AnimatedProgressBar(targetProgress, currentProgress, width, showPercentage, animationType)
}

// Text Formatting

// RenderMarkdown renders markdown text
func (sm *StyleManager) RenderMarkdown(markdown string) (string, error) {
	if sm.textFormatter == nil {
		return markdown, fmt.Errorf("text formatter not available")
	}
	return sm.textFormatter.RenderMarkdown(markdown)
}

// HighlightCode applies syntax highlighting to code
func (sm *StyleManager) HighlightCode(code string, language CodeLanguage) (string, error) {
	if sm.textFormatter == nil {
		return code, fmt.Errorf("text formatter not available")
	}
	return sm.textFormatter.HighlightCode(code, language)
}

// StyleText applies text styling
func (sm *StyleManager) StyleText(text string, style TextStyle) string {
	if sm.textFormatter == nil {
		return text
	}
	return sm.textFormatter.ApplyTextStyle(text, style)
}

// Accessibility

// GetAccessibilityStatus returns accessibility status
func (sm *StyleManager) GetAccessibilityStatus() string {
	if sm.accessibilityMgr == nil {
		return "Accessibility features disabled"
	}
	return sm.accessibilityMgr.GetAccessibilityStatus()
}

// IsAccessibilityEnabled returns if accessibility features are active
func (sm *StyleManager) IsAccessibilityEnabled() bool {
	return sm.accessibilityMgr != nil && sm.accessibilityMgr.IsAccessibilityEnabled()
}

// Responsive Design

// Resize updates all managers with new dimensions
func (sm *StyleManager) Resize(width, height int) {
	sm.width = width
	sm.height = height

	if sm.layoutManager != nil {
		sm.layoutManager.Resize(width, height)
	}

	if sm.adaptiveStyler != nil {
		sm.adaptiveStyler.Resize(width, height)
	}

	if sm.textFormatter != nil {
		sm.textFormatter.Resize(width, height)
	}

	// Clear cache as dimensions affect rendering
	sm.clearCache()
}

// GetScreenSize returns current screen size category
func (sm *StyleManager) GetScreenSize() ScreenSize {
	if sm.layoutManager == nil {
		return ScreenMedium
	}
	return sm.layoutManager.GetScreenSize()
}

// Performance and Caching

// getCached retrieves a cached style
func (sm *StyleManager) getCached(key string) (lipgloss.Style, bool) {
	if !sm.config.EnableCaching {
		return lipgloss.NewStyle(), false
	}

	sm.cacheMutex.RLock()
	defer sm.cacheMutex.RUnlock()

	style, exists := sm.styleCache[key]
	if exists {
		sm.cacheHits++
	} else {
		sm.cacheMisses++
	}

	return style, exists
}

// setCached stores a style in cache
func (sm *StyleManager) setCached(key string, style lipgloss.Style) {
	if !sm.config.EnableCaching {
		return
	}

	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()

	// Simple cache size management
	if len(sm.styleCache) >= sm.config.CacheSize {
		// Remove oldest entries (simplified LRU)
		for k := range sm.styleCache {
			delete(sm.styleCache, k)
			break
		}
	}

	sm.styleCache[key] = style
}

// clearCache clears the style cache
func (sm *StyleManager) clearCache() {
	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()

	sm.styleCache = make(map[string]lipgloss.Style)
}

// GetPerformanceMetrics returns performance statistics
func (sm *StyleManager) GetPerformanceMetrics() map[string]interface{} {
	return map[string]interface{}{
		"render_count":   sm.renderCount,
		"cache_hits":     sm.cacheHits,
		"cache_misses":   sm.cacheMisses,
		"cache_hit_rate": float64(sm.cacheHits) / float64(sm.cacheHits+sm.cacheMisses),
		"cache_size":     len(sm.styleCache),
		"last_update":    sm.lastUpdateTime,
	}
}

// Configuration

// UpdateConfig updates the style manager configuration
func (sm *StyleManager) UpdateConfig(config *StyleConfig) {
	sm.config = config

	// Reinitialize if needed
	if sm.initialized {
		sm.initialize()
	}
}

// GetConfig returns current configuration
func (sm *StyleManager) GetConfig() *StyleConfig {
	return sm.config
}

// Advanced Features

// CreateCustomStyle creates a custom style using the current theme
func (sm *StyleManager) CreateCustomStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(sm.currentTheme.Colors.Text).
		Background(sm.currentTheme.Colors.Background)
}

// GetThemeColor returns a color from the current theme
func (sm *StyleManager) GetThemeColor(colorName string) lipgloss.Color {
	switch colorName {
	case "primary":
		return sm.currentTheme.Colors.Primary
	case "secondary":
		return sm.currentTheme.Colors.Secondary
	case "accent":
		return sm.currentTheme.Colors.Accent
	case "success":
		return sm.currentTheme.Colors.Success
	case "error":
		return sm.currentTheme.Colors.Error
	case "warning":
		return sm.currentTheme.Colors.Warning
	case "info":
		return sm.currentTheme.Colors.Info
	case "text":
		return sm.currentTheme.Colors.Text
	case "background":
		return sm.currentTheme.Colors.Background
	default:
		return sm.currentTheme.Colors.Text
	}
}

// ApplyThemeToStyle applies theme colors to a custom style
func (sm *StyleManager) ApplyThemeToStyle(style lipgloss.Style) lipgloss.Style {
	return style.
		Foreground(sm.currentTheme.Colors.Text).
		Background(sm.currentTheme.Colors.Background).
		BorderForeground(sm.currentTheme.Colors.Border)
}

// Helper methods

// updateManagersWithTheme updates all managers with the current theme
func (sm *StyleManager) updateManagersWithTheme() {
	if sm.layoutManager != nil {
		sm.layoutManager.theme = sm.currentTheme
	}

	if sm.componentStyler != nil {
		sm.componentStyler.theme = sm.currentTheme
	}

	if sm.interactiveStyler != nil {
		sm.interactiveStyler.theme = sm.currentTheme
	}

	if sm.textFormatter != nil {
		sm.textFormatter.theme = sm.currentTheme
	}
}

// getDefaultStyleConfig returns default configuration
func getDefaultStyleConfig() *StyleConfig {
	return &StyleConfig{
		EnableCaching:       true,
		CacheSize:           1000,
		AutoResize:          true,
		OptimizeForSpeed:    false,
		EnableAnimations:    true,
		EnableGradients:     true,
		EnableAccessibility: true,
		EnableAdaptation:    true,
		DebugMode:           false,
		ShowMetrics:         false,
		LogPerformance:      false,
	}
}

// Debugging and Diagnostics

// GetSystemInfo returns comprehensive system information
func (sm *StyleManager) GetSystemInfo() map[string]interface{} {
	info := map[string]interface{}{
		"initialized":   sm.initialized,
		"current_theme": sm.currentTheme.Name,
		"width":         sm.width,
		"height":        sm.height,
		"screen_size":   sm.GetScreenSize(),
		"config":        sm.config,
		"performance":   sm.GetPerformanceMetrics(),
	}

	if sm.adaptiveStyler != nil {
		info["terminal_capabilities"] = sm.adaptiveStyler.GetCapabilities()
		info["performance_settings"] = sm.adaptiveStyler.GetPerformanceSettings()
	}

	if sm.accessibilityMgr != nil {
		info["accessibility_status"] = sm.accessibilityMgr.GetAccessibilityStatus()
		info["accessibility_preferences"] = sm.accessibilityMgr.GetPreferences()
	}

	return info
}

// PrintDiagnostics outputs diagnostic information
func (sm *StyleManager) PrintDiagnostics() string {
	info := sm.GetSystemInfo()

	var result strings.Builder
	result.WriteString("Klip Style Manager Diagnostics\n")
	result.WriteString("==============================\n\n")

	result.WriteString(fmt.Sprintf("Status: %s\n", map[bool]string{true: "Initialized", false: "Not Initialized"}[sm.initialized]))
	result.WriteString(fmt.Sprintf("Theme: %s\n", sm.currentTheme.Name))
	result.WriteString(fmt.Sprintf("Dimensions: %dx%d\n", sm.width, sm.height))
	result.WriteString(fmt.Sprintf("Screen Size: %v\n", info["screen_size"]))

	result.WriteString("\nFeatures:\n")
	result.WriteString(fmt.Sprintf("  Caching: %v\n", sm.config.EnableCaching))
	result.WriteString(fmt.Sprintf("  Animations: %v\n", sm.config.EnableAnimations))
	result.WriteString(fmt.Sprintf("  Accessibility: %v\n", sm.config.EnableAccessibility))
	result.WriteString(fmt.Sprintf("  Adaptation: %v\n", sm.config.EnableAdaptation))

	if sm.config.ShowMetrics {
		metrics := sm.GetPerformanceMetrics()
		result.WriteString(fmt.Sprintf("\nPerformance:\n"))
		result.WriteString(fmt.Sprintf("  Render Count: %v\n", metrics["render_count"]))
		result.WriteString(fmt.Sprintf("  Cache Hit Rate: %.2f%%\n", metrics["cache_hit_rate"].(float64)*100))
		result.WriteString(fmt.Sprintf("  Cache Size: %v\n", metrics["cache_size"]))
	}

	if sm.accessibilityMgr != nil {
		result.WriteString(fmt.Sprintf("\nAccessibility: %s\n", sm.accessibilityMgr.GetAccessibilityStatus()))
	}

	if sm.adaptiveStyler != nil {
		result.WriteString(fmt.Sprintf("\nTerminal Capabilities:\n"))
		caps := sm.adaptiveStyler.GetCapabilities()
		result.WriteString(fmt.Sprintf("  Terminal: %s\n", caps.TerminalType))
		result.WriteString(fmt.Sprintf("  True Color: %v\n", caps.HasTrueColor))
		result.WriteString(fmt.Sprintf("  256 Color: %v\n", caps.Has256Color))
		result.WriteString(fmt.Sprintf("  Unicode: %v\n", caps.SupportsUnicode))
	}

	return result.String()
}

// Global style manager instance
var DefaultStyleManager *StyleManager

// Initialize global style manager
func init() {
	// Will be properly initialized when first used
	DefaultStyleManager = nil
}

// GetGlobalStyleManager returns the global style manager, initializing if needed
func GetGlobalStyleManager(width, height int) *StyleManager {
	if DefaultStyleManager == nil {
		DefaultStyleManager = NewStyleManager(width, height)
	}
	return DefaultStyleManager
}

// Convenience functions for global access

// SetGlobalTheme sets the theme on the global style manager
func SetGlobalTheme(themeName string) error {
	if DefaultStyleManager == nil {
		return fmt.Errorf("global style manager not initialized")
	}
	return DefaultStyleManager.SetTheme(themeName)
}

// GetGlobalTheme returns the current global theme
func GetGlobalTheme() *Theme {
	if DefaultStyleManager == nil {
		return &CharmDark // Fallback theme
	}
	return DefaultStyleManager.GetCurrentTheme()
}
