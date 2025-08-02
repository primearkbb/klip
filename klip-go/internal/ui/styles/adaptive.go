package styles

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// AdaptiveStyler handles dynamic styling based on terminal capabilities and environment
type AdaptiveStyler struct {
	capabilities *TerminalCapabilities
	profile      termenv.Profile
	theme        *Theme
	width        int
	height       int
	performance  *PerformanceSettings
}

// Note: TerminalCapabilities is defined in accessibility.go to avoid duplication

// PerformanceSettings controls rendering optimizations
type PerformanceSettings struct {
	EnableAnimations     bool
	EnableGradients      bool
	EnableComplexBorders bool
	EnableUnicodeChars   bool
	MaxFrameRate         int
	BatchUpdates         bool
	UseSimpleChars       bool
	ReduceColors         bool
}

// ResponsiveBreakpoints define different screen size categories
type ResponsiveBreakpoints struct {
	Tiny   ResponsiveConfig // < 30 cols
	Small  ResponsiveConfig // 30-59 cols
	Medium ResponsiveConfig // 60-99 cols
	Large  ResponsiveConfig // 100-139 cols
	Huge   ResponsiveConfig // >= 140 cols
}

// ResponsiveConfig holds configuration for each breakpoint
type ResponsiveConfig struct {
	MinWidth         int
	MaxWidth         int
	SidebarWidth     int
	PanelPadding     [4]int
	ComponentSpacing int
	FontScale        float64
	ShowSidebar      bool
	ShowDecorations  bool
	CompactMode      bool
}

// ColorDepthLevel represents different color depth options
type ColorDepthLevel int

const (
	ColorDepthMonochrome ColorDepthLevel = iota
	ColorDepthBasic                      // 16 colors
	ColorDepth256                        // 256 colors
	ColorDepthTrueColor                  // 16.7M colors
)

// NewAdaptiveStyler creates a new adaptive styler
func NewAdaptiveStyler(theme *Theme, width, height int) *AdaptiveStyler {
	as := &AdaptiveStyler{
		theme:  theme,
		width:  width,
		height: height,
	}

	// Detect terminal capabilities
	as.capabilities = as.detectTerminalCapabilities()
	as.profile = termenv.ColorProfile()

	// Configure performance settings based on capabilities
	as.performance = as.configurePerformanceSettings()

	return as
}

// detectTerminalCapabilities analyzes the terminal environment
func (as *AdaptiveStyler) detectTerminalCapabilities() *TerminalCapabilities {
	caps := &TerminalCapabilities{}

	// Detect color support
	profile := termenv.ColorProfile()
	caps.HasTrueColor = profile == termenv.TrueColor
	caps.Has256Color = profile >= termenv.ANSI256
	caps.HasBasicColor = profile >= termenv.ANSI
	caps.IsMonochrome = profile == termenv.Ascii

	// Detect terminal type
	term := strings.ToLower(os.Getenv("TERM"))
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))

	caps.TerminalType = term
	if termProgram != "" {
		caps.TerminalType = termProgram
	}

	// Platform detection
	caps.Platform = runtime.GOOS
	caps.Architecture = runtime.GOARCH

	// SSH detection
	caps.IsSSH = os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_TTY") != ""
	caps.IsRemote = caps.IsSSH || os.Getenv("REMOTE_HOST") != ""

	// Feature detection based on terminal type
	caps.SupportsUnicode = as.detectUnicodeSupport(term)
	caps.SupportsEmoji = as.detectEmojiSupport(term)
	caps.SupportsBold = as.detectStyleSupport(term, "bold")
	caps.SupportsItalic = as.detectStyleSupport(term, "italic")
	caps.SupportsUnderline = as.detectStyleSupport(term, "underline")
	caps.SupportsBlink = as.detectStyleSupport(term, "blink")
	caps.SupportsStrike = as.detectStyleSupport(term, "strike")
	caps.SupportsBoxDrawing = as.detectBoxDrawingSupport(term)

	// Performance characteristics
	caps.IsSlowTerminal = as.detectSlowTerminal(term)
	caps.HasLowBandwidth = caps.IsRemote || caps.IsSSH
	caps.LimitedBuffer = as.detectLimitedBuffer(term)

	return caps
}

// configurePerformanceSettings sets optimal performance configuration
func (as *AdaptiveStyler) configurePerformanceSettings() *PerformanceSettings {
	settings := &PerformanceSettings{
		EnableAnimations:     true,
		EnableGradients:      true,
		EnableComplexBorders: true,
		EnableUnicodeChars:   true,
		MaxFrameRate:         60,
		BatchUpdates:         false,
		UseSimpleChars:       false,
		ReduceColors:         false,
	}

	// Adjust based on terminal capabilities
	if as.capabilities.IsSlowTerminal {
		settings.EnableAnimations = false
		settings.MaxFrameRate = 15
		settings.BatchUpdates = true
	}

	if as.capabilities.HasLowBandwidth {
		settings.EnableGradients = false
		settings.EnableComplexBorders = false
		settings.UseSimpleChars = true
		settings.BatchUpdates = true
	}

	if as.capabilities.LimitedBuffer {
		settings.UseSimpleChars = true
		settings.ReduceColors = true
	}

	if as.capabilities.IsMonochrome {
		settings.EnableGradients = false
		settings.ReduceColors = true
	}

	if !as.capabilities.SupportsUnicode {
		settings.EnableUnicodeChars = false
		settings.UseSimpleChars = true
	}

	// Check environment overrides
	if os.Getenv("KLIP_NO_ANIMATIONS") == "true" {
		settings.EnableAnimations = false
	}

	if os.Getenv("KLIP_SIMPLE_MODE") == "true" {
		settings.UseSimpleChars = true
		settings.EnableGradients = false
		settings.EnableComplexBorders = false
	}

	return settings
}

// GetResponsiveConfig returns configuration for current screen size
func (as *AdaptiveStyler) GetResponsiveConfig() ResponsiveConfig {
	breakpoints := as.getResponsiveBreakpoints()

	switch {
	case as.width < 30:
		return breakpoints.Tiny
	case as.width < 60:
		return breakpoints.Small
	case as.width < 100:
		return breakpoints.Medium
	case as.width < 140:
		return breakpoints.Large
	default:
		return breakpoints.Huge
	}
}

// getResponsiveBreakpoints defines breakpoint configurations
func (as *AdaptiveStyler) getResponsiveBreakpoints() ResponsiveBreakpoints {
	return ResponsiveBreakpoints{
		Tiny: ResponsiveConfig{
			MinWidth:         0,
			MaxWidth:         29,
			SidebarWidth:     0,
			PanelPadding:     [4]int{0, 1, 0, 1},
			ComponentSpacing: 1,
			FontScale:        0.8,
			ShowSidebar:      false,
			ShowDecorations:  false,
			CompactMode:      true,
		},
		Small: ResponsiveConfig{
			MinWidth:         30,
			MaxWidth:         59,
			SidebarWidth:     0,
			PanelPadding:     [4]int{1, 1, 1, 1},
			ComponentSpacing: 1,
			FontScale:        0.9,
			ShowSidebar:      false,
			ShowDecorations:  true,
			CompactMode:      true,
		},
		Medium: ResponsiveConfig{
			MinWidth:         60,
			MaxWidth:         99,
			SidebarWidth:     20,
			PanelPadding:     [4]int{1, 2, 1, 2},
			ComponentSpacing: 2,
			FontScale:        1.0,
			ShowSidebar:      true,
			ShowDecorations:  true,
			CompactMode:      false,
		},
		Large: ResponsiveConfig{
			MinWidth:         100,
			MaxWidth:         139,
			SidebarWidth:     25,
			PanelPadding:     [4]int{2, 3, 2, 3},
			ComponentSpacing: 3,
			FontScale:        1.0,
			ShowSidebar:      true,
			ShowDecorations:  true,
			CompactMode:      false,
		},
		Huge: ResponsiveConfig{
			MinWidth:         140,
			MaxWidth:         999,
			SidebarWidth:     30,
			PanelPadding:     [4]int{2, 4, 2, 4},
			ComponentSpacing: 4,
			FontScale:        1.1,
			ShowSidebar:      true,
			ShowDecorations:  true,
			CompactMode:      false,
		},
	}
}

// AdaptTheme adapts a theme to terminal capabilities
func (as *AdaptiveStyler) AdaptTheme(theme *Theme) *Theme {
	adaptedTheme := *theme // Copy theme

	// Adapt colors based on color depth
	colorDepth := as.getColorDepth()
	adaptedTheme.Colors = as.adaptColorPalette(theme.Colors, colorDepth)

	// Adapt components based on performance settings
	adaptedTheme.Components = as.adaptComponentStyles(theme.Components)

	// Adapt spacing based on screen size
	adaptedTheme.Spacing = as.adaptSpacing(theme.Spacing)

	return &adaptedTheme
}

// getColorDepth determines the optimal color depth to use
func (as *AdaptiveStyler) getColorDepth() ColorDepthLevel {
	if as.capabilities.IsMonochrome || as.performance.ReduceColors {
		return ColorDepthMonochrome
	} else if as.capabilities.HasTrueColor && !as.performance.ReduceColors {
		return ColorDepthTrueColor
	} else if as.capabilities.Has256Color {
		return ColorDepth256
	} else if as.capabilities.HasBasicColor {
		return ColorDepthBasic
	} else {
		return ColorDepthMonochrome
	}
}

// adaptColorPalette adapts colors to the target color depth
func (as *AdaptiveStyler) adaptColorPalette(colors ColorPalette, depth ColorDepthLevel) ColorPalette {
	switch depth {
	case ColorDepthMonochrome:
		return as.convertToMonochrome(colors)
	case ColorDepthBasic:
		return as.convertToBasicColors(colors)
	case ColorDepth256:
		return as.convertTo256Colors(colors)
	default:
		return colors // Keep original true colors
	}
}

// convertToMonochrome converts colors to black/white/gray
func (as *AdaptiveStyler) convertToMonochrome(colors ColorPalette) ColorPalette {
	white := lipgloss.Color("#FFFFFF")
	black := lipgloss.Color("#000000")
	gray := lipgloss.Color("#808080")
	lightGray := lipgloss.Color("#C0C0C0")
	darkGray := lipgloss.Color("#404040")

	return ColorPalette{
		Primary:          black,
		PrimaryLight:     gray,
		PrimaryDark:      black,
		Secondary:        gray,
		SecondaryLight:   lightGray,
		SecondaryDark:    darkGray,
		Accent:           black,
		AccentLight:      gray,
		AccentDark:       black,
		Background:       white,
		BackgroundAlt:    lightGray,
		BackgroundSubtle: lightGray,
		Surface:          white,
		SurfaceAlt:       lightGray,
		SurfaceSubtle:    lightGray,
		Text:             black,
		TextSubtle:       gray,
		TextMuted:        gray,
		TextInverse:      white,
		Border:           gray,
		BorderSubtle:     lightGray,
		BorderFocus:      black,
		Success:          black,
		SuccessLight:     gray,
		SuccessDark:      black,
		Error:            black,
		ErrorLight:       gray,
		ErrorDark:        black,
		Warning:          gray,
		WarningLight:     lightGray,
		WarningDark:      darkGray,
		Info:             gray,
		InfoLight:        lightGray,
		InfoDark:         darkGray,
		Highlight:        lightGray,
		Selection:        gray,
		Shadow:           black,
	}
}

// convertToBasicColors converts to 16 basic ANSI colors
func (as *AdaptiveStyler) convertToBasicColors(colors ColorPalette) ColorPalette {
	return ColorPalette{
		Primary:          lipgloss.Color("5"),  // Magenta
		PrimaryLight:     lipgloss.Color("13"), // Bright Magenta
		PrimaryDark:      lipgloss.Color("5"),
		Secondary:        lipgloss.Color("4"),  // Blue
		SecondaryLight:   lipgloss.Color("12"), // Bright Blue
		SecondaryDark:    lipgloss.Color("4"),
		Accent:           lipgloss.Color("6"),  // Cyan
		AccentLight:      lipgloss.Color("14"), // Bright Cyan
		AccentDark:       lipgloss.Color("6"),
		Background:       lipgloss.Color("0"), // Black/White based on theme
		BackgroundAlt:    lipgloss.Color("8"), // Gray
		BackgroundSubtle: lipgloss.Color("8"),
		Surface:          lipgloss.Color("0"),
		SurfaceAlt:       lipgloss.Color("8"),
		SurfaceSubtle:    lipgloss.Color("8"),
		Text:             lipgloss.Color("7"), // White/Black based on theme
		TextSubtle:       lipgloss.Color("8"),
		TextMuted:        lipgloss.Color("8"),
		TextInverse:      lipgloss.Color("0"),
		Border:           lipgloss.Color("8"),
		BorderSubtle:     lipgloss.Color("8"),
		BorderFocus:      lipgloss.Color("5"),
		Success:          lipgloss.Color("2"),  // Green
		SuccessLight:     lipgloss.Color("10"), // Bright Green
		SuccessDark:      lipgloss.Color("2"),
		Error:            lipgloss.Color("1"), // Red
		ErrorLight:       lipgloss.Color("9"), // Bright Red
		ErrorDark:        lipgloss.Color("1"),
		Warning:          lipgloss.Color("3"),  // Yellow
		WarningLight:     lipgloss.Color("11"), // Bright Yellow
		WarningDark:      lipgloss.Color("3"),
		Info:             lipgloss.Color("4"),  // Blue
		InfoLight:        lipgloss.Color("12"), // Bright Blue
		InfoDark:         lipgloss.Color("4"),
		Highlight:        lipgloss.Color("11"),
		Selection:        lipgloss.Color("4"),
		Shadow:           lipgloss.Color("0"),
	}
}

// convertTo256Colors adapts colors for 256-color terminals
func (as *AdaptiveStyler) convertTo256Colors(colors ColorPalette) ColorPalette {
	// For 256-color terminals, we can keep most colors but may need
	// to map some custom colors to the closest 256-color equivalent
	// This is a simplified implementation
	return colors
}

// adaptComponentStyles adapts component styling based on capabilities
func (as *AdaptiveStyler) adaptComponentStyles(components ComponentStyles) ComponentStyles {
	adapted := components

	// Simplify borders for limited terminals
	if !as.capabilities.SupportsBoxDrawing || as.performance.UseSimpleChars {
		adapted.BorderStyle = lipgloss.NormalBorder()
		adapted.BorderRadius = 0
	}

	// Disable shadows for performance
	if as.performance.UseSimpleChars || as.capabilities.IsSlowTerminal {
		adapted.ShadowEnabled = false
	}

	// Reduce animation speed for slow terminals
	if as.capabilities.IsSlowTerminal {
		adapted.AnimationSpeed = "slow"
		adapted.TransitionDuration = "500ms"
	}

	return adapted
}

// adaptSpacing adapts spacing based on screen size
func (as *AdaptiveStyler) adaptSpacing(spacing Spacing) Spacing {
	config := as.GetResponsiveConfig()
	adapted := spacing

	// Scale spacing based on screen size
	if config.CompactMode {
		adapted.Base = max(1, spacing.Base/2)
		adapted.XSmall = max(0, spacing.XSmall/2)
		adapted.Small = max(1, spacing.Small/2)
		adapted.Medium = max(1, spacing.Medium/2)
		adapted.Large = max(2, spacing.Large/2)
		adapted.XLarge = max(2, spacing.XLarge/2)
		adapted.XXLarge = max(3, spacing.XXLarge/2)
	}

	// Update component-specific spacing
	copy(adapted.ButtonPadding[:], config.PanelPadding[:2])
	copy(adapted.InputPadding[:], config.PanelPadding[:2])
	copy(adapted.PanelPadding[:], config.PanelPadding[:])
	adapted.ComponentSpacing = config.ComponentSpacing

	return adapted
}

// GetOptimalBorder returns the best border style for current capabilities
func (as *AdaptiveStyler) GetOptimalBorder() lipgloss.Border {
	if as.capabilities.SupportsBoxDrawing && as.performance.EnableComplexBorders {
		return lipgloss.RoundedBorder()
	} else if as.capabilities.SupportsUnicode && !as.performance.UseSimpleChars {
		return lipgloss.NormalBorder()
	} else {
		return lipgloss.Border{
			Top:         "-",
			Bottom:      "-",
			Left:        "|",
			Right:       "|",
			TopLeft:     "+",
			TopRight:    "+",
			BottomLeft:  "+",
			BottomRight: "+",
		}
	}
}

// GetOptimalCharset returns appropriate character set for current capabilities
func (as *AdaptiveStyler) GetOptimalCharset() CharacterSet {
	if as.capabilities.SupportsUnicode && as.performance.EnableUnicodeChars {
		return UnicodeCharacterSet
	} else {
		return ASCIICharacterSet
	}
}

// CharacterSet defines different character sets for UI elements
type CharacterSet struct {
	CheckMark     string
	CrossMark     string
	Arrow         string
	Bullet        string
	Spinner       []string
	ProgressFull  string
	ProgressEmpty string
	Ellipsis      string
}

var (
	UnicodeCharacterSet = CharacterSet{
		CheckMark:     "✓",
		CrossMark:     "✗",
		Arrow:         "→",
		Bullet:        "•",
		Spinner:       []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		ProgressFull:  "█",
		ProgressEmpty: "░",
		Ellipsis:      "…",
	}

	ASCIICharacterSet = CharacterSet{
		CheckMark:     "x",
		CrossMark:     "X",
		Arrow:         ">",
		Bullet:        "*",
		Spinner:       []string{"|", "/", "-", "\\"},
		ProgressFull:  "#",
		ProgressEmpty: "-",
		Ellipsis:      "...",
	}
)

// Performance optimization methods

// ShouldSkipFrame determines if a frame should be skipped for performance
func (as *AdaptiveStyler) ShouldSkipFrame(frameCount int) bool {
	if as.capabilities.IsSlowTerminal {
		// Skip every other frame for slow terminals
		return frameCount%2 == 1
	}

	if as.capabilities.HasLowBandwidth {
		// Skip 2 out of 3 frames for low bandwidth
		return frameCount%3 != 0
	}

	return false
}

// OptimizeContent reduces content complexity for performance
func (as *AdaptiveStyler) OptimizeContent(content string) string {
	if as.performance.UseSimpleChars {
		// Replace complex unicode characters with simpler alternatives
		content = strings.ReplaceAll(content, "█", "#")
		content = strings.ReplaceAll(content, "░", "-")
		content = strings.ReplaceAll(content, "▓", "=")
		content = strings.ReplaceAll(content, "●", "o")
		content = strings.ReplaceAll(content, "○", "O")
		content = strings.ReplaceAll(content, "◐", "Q")
		content = strings.ReplaceAll(content, "◑", "q")
		content = strings.ReplaceAll(content, "◒", "p")
		content = strings.ReplaceAll(content, "◓", "b")
	}

	return content
}

// GetFrameRate returns the optimal frame rate for current conditions
func (as *AdaptiveStyler) GetFrameRate() int {
	if as.capabilities.IsSlowTerminal {
		return 10
	} else if as.capabilities.HasLowBandwidth {
		return 20
	} else {
		return as.performance.MaxFrameRate
	}
}

// Detection helper methods

func (as *AdaptiveStyler) detectUnicodeSupport(term string) bool {
	switch {
	case strings.Contains(term, "xterm"):
		return true
	case strings.Contains(term, "screen"):
		return true
	case strings.Contains(term, "tmux"):
		return true
	case strings.Contains(term, "alacritty"):
		return true
	case strings.Contains(term, "kitty"):
		return true
	case strings.Contains(term, "wezterm"):
		return true
	case term == "dumb":
		return false
	default:
		// Check for UTF-8 support via locale
		lang := os.Getenv("LANG")
		return strings.Contains(strings.ToUpper(lang), "UTF-8") ||
			strings.Contains(strings.ToUpper(lang), "UTF8")
	}
}

func (as *AdaptiveStyler) detectEmojiSupport(term string) bool {
	// Most modern terminals support emoji
	switch {
	case strings.Contains(term, "iterm"):
		return true
	case strings.Contains(term, "alacritty"):
		return true
	case strings.Contains(term, "kitty"):
		return true
	case strings.Contains(term, "wezterm"):
		return true
	case strings.Contains(term, "vscode"):
		return true
	case term == "dumb":
		return false
	default:
		return as.detectUnicodeSupport(term)
	}
}

func (as *AdaptiveStyler) detectStyleSupport(term, style string) bool {
	switch term {
	case "dumb":
		return false
	case "linux":
		return style != "italic" // Linux console doesn't support italic
	default:
		return true // Most terminals support basic styles
	}
}

func (as *AdaptiveStyler) detectBoxDrawingSupport(term string) bool {
	switch {
	case strings.Contains(term, "xterm"):
		return true
	case strings.Contains(term, "screen"):
		return true
	case strings.Contains(term, "tmux"):
		return true
	case term == "dumb":
		return false
	default:
		return as.detectUnicodeSupport(term)
	}
}

func (as *AdaptiveStyler) detectSlowTerminal(term string) bool {
	switch {
	case term == "dumb":
		return true
	case strings.Contains(term, "vt100"):
		return true
	case strings.Contains(term, "vt102"):
		return true
	case os.Getenv("TERM_PROGRAM") == "Apple_Terminal":
		return true // Apple Terminal can be slow with heavy content
	default:
		// Check if we're on a slow connection
		return as.capabilities.IsRemote && as.detectLowBandwidth()
	}
}

func (as *AdaptiveStyler) detectLimitedBuffer(term string) bool {
	switch {
	case term == "dumb":
		return true
	case strings.Contains(term, "vt"):
		return true
	default:
		return false
	}
}

func (as *AdaptiveStyler) detectLowBandwidth() bool {
	// Simple heuristic: assume SSH connections might have lower bandwidth
	return as.capabilities.IsSSH
}

// Utility methods

// UpdateCapabilities re-detects terminal capabilities (useful for dynamic changes)
func (as *AdaptiveStyler) UpdateCapabilities() {
	as.capabilities = as.detectTerminalCapabilities()
	as.performance = as.configurePerformanceSettings()
}

// GetCapabilities returns current terminal capabilities
func (as *AdaptiveStyler) GetCapabilities() *TerminalCapabilities {
	return as.capabilities
}

// GetPerformanceSettings returns current performance settings
func (as *AdaptiveStyler) GetPerformanceSettings() *PerformanceSettings {
	return as.performance
}

// Resize updates the adaptive styler dimensions
func (as *AdaptiveStyler) Resize(width, height int) {
	as.width = width
	as.height = height
}

// PrintCapabilities outputs detected capabilities for debugging
func (as *AdaptiveStyler) PrintCapabilities() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Terminal Capabilities:\n")
	fmt.Fprintf(&b, "  Type: %s\n", as.capabilities.TerminalType)
	fmt.Fprintf(&b, "  Platform: %s/%s\n", as.capabilities.Platform, as.capabilities.Architecture)
	fmt.Fprintf(&b, "  Remote: %v (SSH: %v)\n", as.capabilities.IsRemote, as.capabilities.IsSSH)
	fmt.Fprintf(&b, "  Color Support:\n")
	fmt.Fprintf(&b, "    True Color: %v\n", as.capabilities.HasTrueColor)
	fmt.Fprintf(&b, "    256 Color: %v\n", as.capabilities.Has256Color)
	fmt.Fprintf(&b, "    Basic Color: %v\n", as.capabilities.HasBasicColor)
	fmt.Fprintf(&b, "    Monochrome: %v\n", as.capabilities.IsMonochrome)
	fmt.Fprintf(&b, "  Features:\n")
	fmt.Fprintf(&b, "    Unicode: %v\n", as.capabilities.SupportsUnicode)
	fmt.Fprintf(&b, "    Emoji: %v\n", as.capabilities.SupportsEmoji)
	fmt.Fprintf(&b, "    Box Drawing: %v\n", as.capabilities.SupportsBoxDrawing)
	fmt.Fprintf(&b, "  Performance:\n")
	fmt.Fprintf(&b, "    Slow Terminal: %v\n", as.capabilities.IsSlowTerminal)
	fmt.Fprintf(&b, "    Low Bandwidth: %v\n", as.capabilities.HasLowBandwidth)
	fmt.Fprintf(&b, "    Animations: %v\n", as.performance.EnableAnimations)
	fmt.Fprintf(&b, "    Max FPS: %d\n", as.performance.MaxFrameRate)

	return b.String()
}
