package styles

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// TerminalCapabilities represents the capabilities of the current terminal
type TerminalCapabilities struct {
	// Color support - using both naming conventions for compatibility
	SupportsTrueColor bool
	HasTrueColor      bool // alias for SupportsTrueColor
	Has256Color       bool
	HasBasicColor     bool
	IsMonochrome      bool
	
	// Text formatting support
	SupportsBold      bool
	SupportsItalic    bool
	SupportsUnderline bool
	SupportsStrike    bool
	SupportsBlink     bool
	
	// Unicode and character support
	SupportsUnicode   bool
	SupportsEmoji     bool
	
	// Layout capabilities
	SupportsBoxDrawing      bool
	SupportsMouseInput      bool
	SupportsResize          bool
	SupportsAlternateScreen bool
	SupportsAltScreen       bool // alias
	SupportsMouse           bool // alias
	SupportsHyperlinks      bool
	
	// Performance characteristics
	IsSlowTerminal   bool
	HasLowBandwidth  bool
	LimitedBuffer    bool
	
	// Terminal identification
	TerminalType     string
	TerminalVersion  string
	IsSSH            bool
	IsRemote         bool
	
	// Platform info
	Platform         string
	Architecture     string
}

// AccessibilityManager handles accessibility features and compliance
type AccessibilityManager struct {
	theme              *Theme
	capabilities       *TerminalCapabilities
	preferences        *AccessibilityPreferences
	contrastChecker    *ContrastChecker
	screenReaderMode   bool
	highContrastMode   bool
	reducedMotionMode  bool
	colorBlindSupport  bool
}

// AccessibilityPreferences stores user accessibility preferences
type AccessibilityPreferences struct {
	// Visual preferences
	HighContrast        bool
	ReducedMotion       bool
	LargeText           bool
	ReduceTransparency  bool
	PreferSimpleColors  bool
	
	// Color vision preferences
	ColorBlindType      ColorBlindType
	UseColorBlindPalette bool
	
	// Screen reader preferences
	ScreenReaderMode    bool
	VerboseDescriptions bool
	StructuralNavigation bool
	
	// Motor accessibility
	StickyKeys          bool
	SlowKeys            bool
	ReduceInteraction   bool
	
	// Cognitive accessibility
	ReduceComplexity    bool
	SimplifyLanguage    bool
	ShowHelpText        bool
}

// ColorBlindType represents different types of color vision deficiency
type ColorBlindType int

const (
	ColorBlindNone ColorBlindType = iota
	ColorBlindProtanopia    // Red-blind
	ColorBlindDeuteranopia  // Green-blind
	ColorBlindTritanopia    // Blue-blind
	ColorBlindProtanomaly   // Red-weak
	ColorBlindDeuteranomaly // Green-weak
	ColorBlindTritanomaly   // Blue-weak
	ColorBlindMonochromacy  // Complete color blindness
)

// ContrastChecker validates color contrast ratios for accessibility
type ContrastChecker struct {
	minNormalContrast float64 // WCAG AA: 4.5:1
	minLargeContrast  float64 // WCAG AA Large: 3:1
	minEnhancedContrast float64 // WCAG AAA: 7:1
}

// AccessibilityLevel represents different accessibility compliance levels
type AccessibilityLevel int

const (
	AccessibilityA AccessibilityLevel = iota
	AccessibilityAA
	AccessibilityAAA
)

// NewAccessibilityManager creates a new accessibility manager
func NewAccessibilityManager(theme *Theme, capabilities *TerminalCapabilities) *AccessibilityManager {
	am := &AccessibilityManager{
		theme:        theme,
		capabilities: capabilities,
		preferences:  detectAccessibilityPreferences(),
		contrastChecker: &ContrastChecker{
			minNormalContrast:   4.5,
			minLargeContrast:    3.0,
			minEnhancedContrast: 7.0,
		},
	}
	
	// Apply detected preferences
	am.applyPreferences()
	
	return am
}

// detectAccessibilityPreferences detects accessibility preferences from environment
func detectAccessibilityPreferences() *AccessibilityPreferences {
	prefs := &AccessibilityPreferences{}
	
	// Check environment variables
	prefs.HighContrast = isEnvTrue("ACCESSIBILITY_HIGH_CONTRAST") ||
		isEnvTrue("FORCE_HIGH_CONTRAST") ||
		isEnvTrue("HIGH_CONTRAST")
	
	prefs.ReducedMotion = isEnvTrue("ACCESSIBILITY_REDUCE_MOTION") ||
		isEnvTrue("REDUCE_MOTION")
	
	prefs.LargeText = isEnvTrue("ACCESSIBILITY_LARGE_TEXT") ||
		isEnvTrue("LARGE_TEXT")
	
	prefs.ScreenReaderMode = isEnvTrue("ACCESSIBILITY_SCREEN_READER") ||
		isEnvTrue("SCREEN_READER") ||
		os.Getenv("NVDA") != "" ||
		os.Getenv("JAWS") != "" ||
		os.Getenv("VOICEOVER") != ""
	
	prefs.VerboseDescriptions = prefs.ScreenReaderMode ||
		isEnvTrue("ACCESSIBILITY_VERBOSE")
	
	// Detect color blind preferences
	colorBlindEnv := strings.ToLower(os.Getenv("COLOR_BLIND_TYPE"))
	switch colorBlindEnv {
	case "protanopia":
		prefs.ColorBlindType = ColorBlindProtanopia
		prefs.UseColorBlindPalette = true
	case "deuteranopia":
		prefs.ColorBlindType = ColorBlindDeuteranopia
		prefs.UseColorBlindPalette = true
	case "tritanopia":
		prefs.ColorBlindType = ColorBlindTritanopia
		prefs.UseColorBlindPalette = true
	case "protanomaly":
		prefs.ColorBlindType = ColorBlindProtanomaly
		prefs.UseColorBlindPalette = true
	case "deuteranomaly":
		prefs.ColorBlindType = ColorBlindDeuteranomaly
		prefs.UseColorBlindPalette = true
	case "tritanomaly":
		prefs.ColorBlindType = ColorBlindTritanomaly
		prefs.UseColorBlindPalette = true
	case "monochromacy":
		prefs.ColorBlindType = ColorBlindMonochromacy
		prefs.UseColorBlindPalette = true
	}
	
	// Cognitive preferences
	prefs.ReduceComplexity = isEnvTrue("ACCESSIBILITY_SIMPLE_MODE") ||
		isEnvTrue("SIMPLE_MODE")
	
	prefs.ShowHelpText = prefs.ScreenReaderMode ||
		isEnvTrue("ACCESSIBILITY_SHOW_HELP")
	
	return prefs
}

// applyPreferences applies accessibility preferences to the manager
func (am *AccessibilityManager) applyPreferences() {
	am.highContrastMode = am.preferences.HighContrast
	am.screenReaderMode = am.preferences.ScreenReaderMode
	am.reducedMotionMode = am.preferences.ReducedMotion
	am.colorBlindSupport = am.preferences.UseColorBlindPalette
}

// CreateAccessibleTheme creates a theme optimized for accessibility
func (am *AccessibilityManager) CreateAccessibleTheme(baseTheme *Theme, level AccessibilityLevel) *Theme {
	accessibleTheme := *baseTheme // Copy theme
	
	// Apply high contrast if needed
	if am.highContrastMode {
		accessibleTheme = *am.createHighContrastTheme(baseTheme)
	}
	
	// Apply color blind friendly palette if needed
	if am.colorBlindSupport {
		accessibleTheme.Colors = am.adaptColorsForColorBlindness(accessibleTheme.Colors, am.preferences.ColorBlindType)
	}
	
	// Ensure color contrast compliance
	accessibleTheme.Colors = am.ensureContrastCompliance(accessibleTheme.Colors, level)
	
	// Adjust component styles for accessibility
	accessibleTheme.Components = am.adaptComponentsForAccessibility(accessibleTheme.Components)
	
	// Adjust spacing for better usability
	if am.preferences.LargeText {
		accessibleTheme.Spacing = am.increasedSpacing(accessibleTheme.Spacing)
	}
	
	return &accessibleTheme
}

// createHighContrastTheme creates a high contrast version of the theme
func (am *AccessibilityManager) createHighContrastTheme(baseTheme *Theme) *Theme {
	highContrastTheme := *baseTheme
	
	if baseTheme.IsDark {
		// Dark high contrast theme
		highContrastTheme.Colors = ColorPalette{
			Primary:          lipgloss.Color("#FFFFFF"),
			PrimaryLight:     lipgloss.Color("#FFFFFF"),
			PrimaryDark:      lipgloss.Color("#CCCCCC"),
			Secondary:        lipgloss.Color("#FFFF00"),
			SecondaryLight:   lipgloss.Color("#FFFF99"),
			SecondaryDark:    lipgloss.Color("#CCCC00"),
			Accent:           lipgloss.Color("#00FFFF"),
			AccentLight:      lipgloss.Color("#99FFFF"),
			AccentDark:       lipgloss.Color("#00CCCC"),
			Background:       lipgloss.Color("#000000"),
			BackgroundAlt:    lipgloss.Color("#111111"),
			BackgroundSubtle: lipgloss.Color("#222222"),
			Surface:          lipgloss.Color("#000000"),
			SurfaceAlt:       lipgloss.Color("#111111"),
			SurfaceSubtle:    lipgloss.Color("#222222"),
			Text:             lipgloss.Color("#FFFFFF"),
			TextSubtle:       lipgloss.Color("#CCCCCC"),
			TextMuted:        lipgloss.Color("#AAAAAA"),
			TextInverse:      lipgloss.Color("#000000"),
			Border:           lipgloss.Color("#FFFFFF"),
			BorderSubtle:     lipgloss.Color("#CCCCCC"),
			BorderFocus:      lipgloss.Color("#FFFF00"),
			Success:          lipgloss.Color("#00FF00"),
			SuccessLight:     lipgloss.Color("#99FF99"),
			SuccessDark:      lipgloss.Color("#00CC00"),
			Error:            lipgloss.Color("#FF0000"),
			ErrorLight:       lipgloss.Color("#FF9999"),
			ErrorDark:        lipgloss.Color("#CC0000"),
			Warning:          lipgloss.Color("#FFFF00"),
			WarningLight:     lipgloss.Color("#FFFF99"),
			WarningDark:      lipgloss.Color("#CCCC00"),
			Info:             lipgloss.Color("#00FFFF"),
			InfoLight:        lipgloss.Color("#99FFFF"),
			InfoDark:         lipgloss.Color("#00CCCC"),
			Highlight:        lipgloss.Color("#FFFF00"),
			Selection:        lipgloss.Color("#0000FF"),
			Shadow:           lipgloss.Color("#000000"),
			CodeForeground:   lipgloss.Color("#FFFFFF"),
			CodeBackground:   lipgloss.Color("#000000"),
		}
	} else {
		// Light high contrast theme
		highContrastTheme.Colors = ColorPalette{
			Primary:          lipgloss.Color("#000000"),
			PrimaryLight:     lipgloss.Color("#333333"),
			PrimaryDark:      lipgloss.Color("#000000"),
			Secondary:        lipgloss.Color("#0000FF"),
			SecondaryLight:   lipgloss.Color("#6666FF"),
			SecondaryDark:    lipgloss.Color("#0000CC"),
			Accent:           lipgloss.Color("#800080"),
			AccentLight:      lipgloss.Color("#CC66CC"),
			AccentDark:       lipgloss.Color("#660066"),
			Background:       lipgloss.Color("#FFFFFF"),
			BackgroundAlt:    lipgloss.Color("#F5F5F5"),
			BackgroundSubtle: lipgloss.Color("#EEEEEE"),
			Surface:          lipgloss.Color("#FFFFFF"),
			SurfaceAlt:       lipgloss.Color("#F5F5F5"),
			SurfaceSubtle:    lipgloss.Color("#EEEEEE"),
			Text:             lipgloss.Color("#000000"),
			TextSubtle:       lipgloss.Color("#333333"),
			TextMuted:        lipgloss.Color("#666666"),
			TextInverse:      lipgloss.Color("#FFFFFF"),
			Border:           lipgloss.Color("#000000"),
			BorderSubtle:     lipgloss.Color("#666666"),
			BorderFocus:      lipgloss.Color("#0000FF"),
			Success:          lipgloss.Color("#008000"),
			SuccessLight:     lipgloss.Color("#66CC66"),
			SuccessDark:      lipgloss.Color("#006600"),
			Error:            lipgloss.Color("#FF0000"),
			ErrorLight:       lipgloss.Color("#FF6666"),
			ErrorDark:        lipgloss.Color("#CC0000"),
			Warning:          lipgloss.Color("#FF8000"),
			WarningLight:     lipgloss.Color("#FFB366"),
			WarningDark:      lipgloss.Color("#CC6600"),
			Info:             lipgloss.Color("#0000FF"),
			InfoLight:        lipgloss.Color("#6666FF"),
			InfoDark:         lipgloss.Color("#0000CC"),
			Highlight:        lipgloss.Color("#FFFF00"),
			Selection:        lipgloss.Color("#0080FF"),
			Shadow:           lipgloss.Color("#000000"),
			CodeForeground:   lipgloss.Color("#000000"),
			CodeBackground:   lipgloss.Color("#FFFFFF"),
		}
	}
	
	highContrastTheme.IsHighContrast = true
	return &highContrastTheme
}

// adaptColorsForColorBlindness adapts colors for different types of color blindness
func (am *AccessibilityManager) adaptColorsForColorBlindness(colors ColorPalette, colorBlindType ColorBlindType) ColorPalette {
	adapted := colors
	
	switch colorBlindType {
	case ColorBlindProtanopia, ColorBlindProtanomaly:
		// Red-blind: Replace reds with distinguishable alternatives
		adapted.Primary = lipgloss.Color("#0066CC")      // Blue
		adapted.Error = lipgloss.Color("#FF8800")        // Orange
		adapted.Success = lipgloss.Color("#0088CC")      // Blue-green
		adapted.Warning = lipgloss.Color("#FFAA00")      // Amber
		
	case ColorBlindDeuteranopia, ColorBlindDeuteranomaly:
		// Green-blind: Replace greens with distinguishable alternatives
		adapted.Success = lipgloss.Color("#0066FF")      // Blue
		adapted.Primary = lipgloss.Color("#8800CC")      // Purple
		adapted.Info = lipgloss.Color("#0088FF")         // Light blue
		adapted.Accent = lipgloss.Color("#CC6600")       // Orange
		
	case ColorBlindTritanopia, ColorBlindTritanomaly:
		// Blue-blind: Replace blues with distinguishable alternatives
		adapted.Info = lipgloss.Color("#CC0066")         // Magenta
		adapted.Primary = lipgloss.Color("#CC6600")      // Orange
		adapted.Secondary = lipgloss.Color("#009900")    // Green
		adapted.Accent = lipgloss.Color("#990099")       // Purple
		
	case ColorBlindMonochromacy:
		// Complete color blindness: Use only grayscale
		return am.convertToGrayscale(colors)
	}
	
	return adapted
}

// convertToGrayscale converts all colors to grayscale
func (am *AccessibilityManager) convertToGrayscale(colors ColorPalette) ColorPalette {
	return ColorPalette{
		Primary:          lipgloss.Color("#000000"),
		PrimaryLight:     lipgloss.Color("#444444"),
		PrimaryDark:      lipgloss.Color("#000000"),
		Secondary:        lipgloss.Color("#666666"),
		SecondaryLight:   lipgloss.Color("#888888"),
		SecondaryDark:    lipgloss.Color("#444444"),
		Accent:           lipgloss.Color("#333333"),
		AccentLight:      lipgloss.Color("#777777"),
		AccentDark:       lipgloss.Color("#111111"),
		Background:       lipgloss.Color("#FFFFFF"),
		BackgroundAlt:    lipgloss.Color("#F5F5F5"),
		BackgroundSubtle: lipgloss.Color("#EEEEEE"),
		Surface:          lipgloss.Color("#FFFFFF"),
		SurfaceAlt:       lipgloss.Color("#F8F8F8"),
		SurfaceSubtle:    lipgloss.Color("#F0F0F0"),
		Text:             lipgloss.Color("#000000"),
		TextSubtle:       lipgloss.Color("#444444"),
		TextMuted:        lipgloss.Color("#888888"),
		TextInverse:      lipgloss.Color("#FFFFFF"),
		Border:           lipgloss.Color("#CCCCCC"),
		BorderSubtle:     lipgloss.Color("#EEEEEE"),
		BorderFocus:      lipgloss.Color("#000000"),
		Success:          lipgloss.Color("#000000"),
		SuccessLight:     lipgloss.Color("#666666"),
		SuccessDark:      lipgloss.Color("#000000"),
		Error:            lipgloss.Color("#000000"),
		ErrorLight:       lipgloss.Color("#666666"),
		ErrorDark:        lipgloss.Color("#000000"),
		Warning:          lipgloss.Color("#333333"),
		WarningLight:     lipgloss.Color("#777777"),
		WarningDark:      lipgloss.Color("#222222"),
		Info:             lipgloss.Color("#555555"),
		InfoLight:        lipgloss.Color("#888888"),
		InfoDark:         lipgloss.Color("#333333"),
		Highlight:        lipgloss.Color("#DDDDDD"),
		Selection:        lipgloss.Color("#BBBBBB"),
		Shadow:           lipgloss.Color("#000000"),
		CodeForeground:   lipgloss.Color("#000000"),
		CodeBackground:   lipgloss.Color("#F0F0F0"),
	}
}

// ensureContrastCompliance ensures color combinations meet accessibility standards
func (am *AccessibilityManager) ensureContrastCompliance(colors ColorPalette, level AccessibilityLevel) ColorPalette {
	compliant := colors
	minRatio := am.getMinContrastRatio(level)
	
	// Check and fix primary text combinations
	if !am.hasAdequateContrast(colors.Text, colors.Background, minRatio) {
		if am.theme.IsDark {
			compliant.Text = lipgloss.Color("#FFFFFF")
		} else {
			compliant.Text = lipgloss.Color("#000000")
		}
	}
	
	// Check and fix button combinations
	if !am.hasAdequateContrast(colors.TextInverse, colors.Primary, minRatio) {
		compliant.TextInverse = am.findContrastingColor(colors.Primary, minRatio)
	}
	
	// Check and fix state color combinations
	stateColors := []struct {
		bg   *lipgloss.Color
		text lipgloss.Color
	}{
		{&compliant.Success, colors.TextInverse},
		{&compliant.Error, colors.TextInverse},
		{&compliant.Warning, colors.TextInverse},
		{&compliant.Info, colors.TextInverse},
	}
	
	for _, sc := range stateColors {
		if !am.hasAdequateContrast(sc.text, *sc.bg, minRatio) {
			*sc.bg = am.adjustColorForContrast(*sc.bg, sc.text, minRatio)
		}
	}
	
	return compliant
}

// adaptComponentsForAccessibility adapts component styles for accessibility
func (am *AccessibilityManager) adaptComponentsForAccessibility(components ComponentStyles) ComponentStyles {
	adapted := components
	
	// Ensure focus indicators are clearly visible
	adapted.BorderWidth = max(adapted.BorderWidth, 2)
	
	// Simplify borders for screen readers
	if am.screenReaderMode {
		adapted.BorderStyle = lipgloss.NormalBorder()
		adapted.BorderRadius = 0
	}
	
	// Reduce animations for motion sensitivity
	if am.reducedMotionMode {
		adapted.AnimationSpeed = "none"
		adapted.TransitionDuration = "0ms"
	}
	
	// Increase touch targets for motor accessibility
	if am.preferences.ReduceInteraction {
		adapted.ButtonHeight = max(adapted.ButtonHeight, 4)
		adapted.InputHeight = max(adapted.InputHeight, 4)
		adapted.MenuItemHeight = max(adapted.MenuItemHeight, 2)
	}
	
	return adapted
}

// increasedSpacing increases spacing for better readability
func (am *AccessibilityManager) increasedSpacing(spacing Spacing) Spacing {
	increased := spacing
	
	// Increase base spacing
	increased.Base = int(float64(spacing.Base) * 1.5)
	
	// Recalculate derived spacings
	increased.XSmall = increased.Base / 4
	increased.Small = increased.Base / 2
	increased.Medium = increased.Base
	increased.Large = increased.Base + increased.Base/2
	increased.XLarge = increased.Base * 2
	increased.XXLarge = increased.Base * 3
	
	// Increase component spacing
	increased.ComponentSpacing = int(float64(spacing.ComponentSpacing) * 1.3)
	increased.SectionSpacing = int(float64(spacing.SectionSpacing) * 1.3)
	increased.ElementSpacing = int(float64(spacing.ElementSpacing) * 1.2)
	
	return increased
}

// Contrast checking methods

// hasAdequateContrast checks if two colors have adequate contrast
func (am *AccessibilityManager) hasAdequateContrast(foreground, background lipgloss.Color, minRatio float64) bool {
	ratio := am.calculateContrastRatio(foreground, background)
	return ratio >= minRatio
}

// calculateContrastRatio calculates the contrast ratio between two colors
func (am *AccessibilityManager) calculateContrastRatio(color1, background lipgloss.Color) float64 {
	// Convert colors to colorful.Color for luminance calculation
	c1, err := colorful.Hex(string(color1))
	if err != nil {
		return 1.0 // Default to minimal contrast on error
	}
	
	c2, err := colorful.Hex(string(background))
	if err != nil {
		return 1.0
	}
	
	// Calculate relative luminance
	l1 := am.relativeLuminance(c1)
	l2 := am.relativeLuminance(c2)
	
	// Ensure l1 is the lighter color
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	
	// Calculate contrast ratio
	return (l1 + 0.05) / (l2 + 0.05)
}

// relativeLuminance calculates the relative luminance of a color
func (am *AccessibilityManager) relativeLuminance(c colorful.Color) float64 {
	// Convert to linear RGB
	r := am.linearizeColorComponent(c.R)
	g := am.linearizeColorComponent(c.G)
	b := am.linearizeColorComponent(c.B)
	
	// Calculate luminance using ITU-R BT.709 coefficients
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// linearizeColorComponent linearizes a color component for luminance calculation
func (am *AccessibilityManager) linearizeColorComponent(component float64) float64 {
	if component <= 0.04045 {
		return component / 12.92
	}
	return math.Pow((component+0.055)/1.055, 2.4)
}

// findContrastingColor finds a color that has adequate contrast with the given background
func (am *AccessibilityManager) findContrastingColor(background lipgloss.Color, minRatio float64) lipgloss.Color {
	// Try white first
	white := lipgloss.Color("#FFFFFF")
	if am.hasAdequateContrast(white, background, minRatio) {
		return white
	}
	
	// Try black
	black := lipgloss.Color("#000000")
	if am.hasAdequateContrast(black, background, minRatio) {
		return black
	}
	
	// If neither works, find the best contrast
	bg, err := colorful.Hex(string(background))
	if err != nil {
		return black // Fallback
	}
	
	// Calculate luminance of background
	bgLuminance := am.relativeLuminance(bg)
	
	// Choose white or black based on background luminance
	if bgLuminance > 0.5 {
		return black
	}
	return white
}

// adjustColorForContrast adjusts a color to meet contrast requirements
func (am *AccessibilityManager) adjustColorForContrast(color, textColor lipgloss.Color, minRatio float64) lipgloss.Color {
	c, err := colorful.Hex(string(color))
	if err != nil {
		return color
	}
	
	// Try darkening/lightening the color
	h, s, l := c.Hsl()
	
	// Determine if we need to lighten or darken
	textC, err := colorful.Hex(string(textColor))
	if err != nil {
		return color
	}
	
	textLuminance := am.relativeLuminance(textC)
	
	// Adjust lightness to achieve target contrast
	step := 0.05
	if textLuminance > 0.5 {
		// Text is light, darken background
		for l > 0 && !am.hasAdequateContrast(textColor, lipgloss.Color(colorful.Hsl(h, s, l).Hex()), minRatio) {
			l -= step
		}
	} else {
		// Text is dark, lighten background
		for l < 1 && !am.hasAdequateContrast(textColor, lipgloss.Color(colorful.Hsl(h, s, l).Hex()), minRatio) {
			l += step
		}
	}
	
	return lipgloss.Color(colorful.Hsl(h, s, l).Hex())
}

// getMinContrastRatio returns the minimum contrast ratio for the given accessibility level
func (am *AccessibilityManager) getMinContrastRatio(level AccessibilityLevel) float64 {
	switch level {
	case AccessibilityA:
		return 3.0
	case AccessibilityAA:
		return 4.5
	case AccessibilityAAA:
		return 7.0
	default:
		return 4.5
	}
}

// Screen reader support methods

// CreateScreenReaderText creates text optimized for screen readers
func (am *AccessibilityManager) CreateScreenReaderText(visualText, screenReaderText string) string {
	if am.screenReaderMode && screenReaderText != "" {
		return screenReaderText
	}
	return visualText
}

// AddAriaLabel adds ARIA-style labels for better accessibility
func (am *AccessibilityManager) AddAriaLabel(content, label string) string {
	if am.screenReaderMode && label != "" {
		return fmt.Sprintf("%s (%s)", content, label)
	}
	return content
}

// CreateAccessibleButton creates an accessible button with proper labeling
func (am *AccessibilityManager) CreateAccessibleButton(text, ariaLabel, description string) string {
	if am.screenReaderMode {
		var parts []string
		parts = append(parts, "Button: "+text)
		
		if ariaLabel != "" && ariaLabel != text {
			parts = append(parts, ariaLabel)
		}
		
		if description != "" && am.preferences.VerboseDescriptions {
			parts = append(parts, description)
		}
		
		return strings.Join(parts, " - ")
	}
	
	return text
}

// Navigation and structural methods

// CreateHeadingHierarchy creates properly structured headings for screen readers
func (am *AccessibilityManager) CreateHeadingHierarchy(level int, text string) string {
	if am.screenReaderMode {
		return fmt.Sprintf("Heading level %d: %s", level, text)
	}
	
	// Visual heading styling
	markers := []string{"#", "##", "###", "####", "#####", "######"}
	if level > 0 && level <= len(markers) {
		return markers[level-1] + " " + text
	}
	
	return text
}

// CreateLandmark creates ARIA landmark equivalents
func (am *AccessibilityManager) CreateLandmark(landmarkType, content string) string {
	if am.screenReaderMode {
		return fmt.Sprintf("%s landmark: %s", strings.Title(landmarkType), content)
	}
	return content
}

// Utility methods

// IsAccessibilityEnabled returns true if any accessibility features are enabled
func (am *AccessibilityManager) IsAccessibilityEnabled() bool {
	return am.highContrastMode ||
		am.screenReaderMode ||
		am.reducedMotionMode ||
		am.colorBlindSupport ||
		am.preferences.LargeText
}

// GetAccessibilityStatus returns a summary of enabled accessibility features
func (am *AccessibilityManager) GetAccessibilityStatus() string {
	var features []string
	
	if am.highContrastMode {
		features = append(features, "High Contrast")
	}
	if am.screenReaderMode {
		features = append(features, "Screen Reader")
	}
	if am.reducedMotionMode {
		features = append(features, "Reduced Motion")
	}
	if am.colorBlindSupport {
		features = append(features, fmt.Sprintf("Color Blind Support (%s)", am.getColorBlindTypeName()))
	}
	if am.preferences.LargeText {
		features = append(features, "Large Text")
	}
	
	if len(features) == 0 {
		return "No accessibility features enabled"
	}
	
	return "Accessibility: " + strings.Join(features, ", ")
}

// getColorBlindTypeName returns a human-readable name for the color blind type
func (am *AccessibilityManager) getColorBlindTypeName() string {
	switch am.preferences.ColorBlindType {
	case ColorBlindProtanopia:
		return "Protanopia"
	case ColorBlindDeuteranopia:
		return "Deuteranopia"
	case ColorBlindTritanopia:
		return "Tritanopia"
	case ColorBlindProtanomaly:
		return "Protanomaly"
	case ColorBlindDeuteranomaly:
		return "Deuteranomaly"
	case ColorBlindTritanomaly:
		return "Tritanomaly"
	case ColorBlindMonochromacy:
		return "Monochromacy"
	default:
		return "Unknown"
	}
}

// UpdatePreferences updates accessibility preferences
func (am *AccessibilityManager) UpdatePreferences(prefs *AccessibilityPreferences) {
	am.preferences = prefs
	am.applyPreferences()
}

// GetPreferences returns current accessibility preferences
func (am *AccessibilityManager) GetPreferences() *AccessibilityPreferences {
	return am.preferences
}

// Utility function for environment variable checking
func isEnvTrue(envVar string) bool {
	value := strings.ToLower(os.Getenv(envVar))
	return value == "true" || value == "1" || value == "yes" || value == "on"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}