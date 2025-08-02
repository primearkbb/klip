package styles

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Theme represents a complete visual theme for the application
type Theme struct {
	Name        string
	DisplayName string
	Description string
	
	// Color palette - Charm-inspired gradients
	Colors ColorPalette
	
	// Typography settings
	Typography Typography
	
	// Spacing and layout
	Spacing Spacing
	
	// Component configurations
	Components ComponentStyles
	
	// Theme metadata
	IsDark      bool
	IsHighContrast bool
	SupportsGradient bool
}

// ColorPalette defines the complete color system
type ColorPalette struct {
	// Primary palette - Charm purple/magenta gradient
	Primary       lipgloss.Color `json:"primary"`
	PrimaryLight  lipgloss.Color `json:"primary_light"`
	PrimaryDark   lipgloss.Color `json:"primary_dark"`
	
	// Secondary palette - Supporting colors
	Secondary      lipgloss.Color `json:"secondary"`
	SecondaryLight lipgloss.Color `json:"secondary_light"`
	SecondaryDark  lipgloss.Color `json:"secondary_dark"`
	
	// Accent colors
	Accent        lipgloss.Color `json:"accent"`
	AccentLight   lipgloss.Color `json:"accent_light"`
	AccentDark    lipgloss.Color `json:"accent_dark"`
	
	// Background colors
	Background        lipgloss.Color `json:"background"`
	BackgroundAlt     lipgloss.Color `json:"background_alt"`
	BackgroundSubtle  lipgloss.Color `json:"background_subtle"`
	
	// Surface colors
	Surface        lipgloss.Color `json:"surface"`
	SurfaceAlt     lipgloss.Color `json:"surface_alt"`
	SurfaceSubtle  lipgloss.Color `json:"surface_subtle"`
	
	// Text colors
	Text           lipgloss.Color `json:"text"`
	TextSubtle     lipgloss.Color `json:"text_subtle"`
	TextMuted      lipgloss.Color `json:"text_muted"`
	TextInverse    lipgloss.Color `json:"text_inverse"`
	
	// Border colors
	Border         lipgloss.Color `json:"border"`
	BorderSubtle   lipgloss.Color `json:"border_subtle"`
	BorderFocus    lipgloss.Color `json:"border_focus"`
	
	// State colors
	Success        lipgloss.Color `json:"success"`
	SuccessLight   lipgloss.Color `json:"success_light"`
	SuccessDark    lipgloss.Color `json:"success_dark"`
	
	Error          lipgloss.Color `json:"error"`
	ErrorLight     lipgloss.Color `json:"error_light"`
	ErrorDark      lipgloss.Color `json:"error_dark"`
	
	Warning        lipgloss.Color `json:"warning"`
	WarningLight   lipgloss.Color `json:"warning_light"`
	WarningDark    lipgloss.Color `json:"warning_dark"`
	
	Info           lipgloss.Color `json:"info"`
	InfoLight      lipgloss.Color `json:"info_light"`
	InfoDark       lipgloss.Color `json:"info_dark"`
	
	// Special colors
	Highlight      lipgloss.Color `json:"highlight"`
	Selection      lipgloss.Color `json:"selection"`
	Shadow         lipgloss.Color `json:"shadow"`
	
	// Code colors
	CodeForeground lipgloss.Color `json:"code_foreground"`
	CodeBackground lipgloss.Color `json:"code_background"`
}

// Typography defines font and text styling
type Typography struct {
	// Font families (terminal-appropriate)
	FontFamily     string
	FontFamilyMono string
	
	// Font sizes (character-based)
	SizeXSmall  int
	SizeSmall   int
	SizeMedium  int
	SizeLarge   int
	SizeXLarge  int
	SizeXXLarge int
	
	// Line heights
	LineHeightTight  float64
	LineHeightNormal float64
	LineHeightLoose  float64
	
	// Letter spacing
	LetterSpacingTight  string
	LetterSpacingNormal string
	LetterSpacingWide   string
}

// Spacing defines consistent spacing values
type Spacing struct {
	// Base spacing unit
	Base int
	
	// Predefined spacing values
	XSmall int // Base * 0.25
	Small  int // Base * 0.5
	Medium int // Base * 1
	Large  int // Base * 1.5
	XLarge int // Base * 2
	XXLarge int // Base * 3
	
	// Component-specific spacing
	ButtonPadding    [2]int // [vertical, horizontal]
	InputPadding     [2]int
	PanelPadding     [2]int
	CardPadding      [2]int
	ModalPadding     [2]int
	
	// Layout spacing
	SectionSpacing   int
	ComponentSpacing int
	ElementSpacing   int
}

// ComponentStyles holds styling configurations for UI components
type ComponentStyles struct {
	// Border styles
	BorderRadius    int
	BorderWidth     int
	BorderStyle     lipgloss.Border
	
	// Shadow and elevation
	ShadowEnabled   bool
	ShadowColor     lipgloss.Color
	ShadowOffset    [2]int
	
	// Animation settings
	AnimationSpeed  string
	TransitionDuration string
	
	// Component-specific settings
	ButtonHeight     int
	InputHeight      int
	MenuItemHeight   int
	ListItemHeight   int
	HeaderHeight     int
	FooterHeight     int
	StatusBarHeight  int
}

// Predefined themes following Charm's design system
var (
	// CharmLight - The signature Charm theme with purple gradients
	CharmLight = Theme{
		Name:        "charm-light",
		DisplayName: "Charm Light",
		Description: "The signature Charm theme with beautiful purple gradients",
		IsDark:      false,
		IsHighContrast: false,
		SupportsGradient: true,
		Colors: ColorPalette{
			// Primary - Charm's signature purple gradient
			Primary:       lipgloss.Color("#7C3AED"), // Purple-600
			PrimaryLight:  lipgloss.Color("#A855F7"), // Purple-500
			PrimaryDark:   lipgloss.Color("#5B21B6"), // Purple-700
			
			// Secondary - Magenta/Pink gradient
			Secondary:     lipgloss.Color("#EC4899"), // Pink-500
			SecondaryLight: lipgloss.Color("#F472B6"), // Pink-400
			SecondaryDark: lipgloss.Color("#BE185D"), // Pink-700
			
			// Accent - Violet gradient
			Accent:        lipgloss.Color("#8B5CF6"), // Violet-500
			AccentLight:   lipgloss.Color("#A78BFA"), // Violet-400
			AccentDark:    lipgloss.Color("#7C3AED"), // Violet-600
			
			// Backgrounds
			Background:    lipgloss.Color("#FFFFFF"), // White
			BackgroundAlt: lipgloss.Color("#F8FAFC"), // Slate-50
			BackgroundSubtle: lipgloss.Color("#F1F5F9"), // Slate-100
			
			// Surfaces
			Surface:       lipgloss.Color("#FFFFFF"), // White
			SurfaceAlt:    lipgloss.Color("#F8FAFC"), // Slate-50
			SurfaceSubtle: lipgloss.Color("#E2E8F0"), // Slate-200
			
			// Text
			Text:          lipgloss.Color("#0F172A"), // Slate-900
			TextSubtle:    lipgloss.Color("#475569"), // Slate-600
			TextMuted:     lipgloss.Color("#94A3B8"), // Slate-400
			TextInverse:   lipgloss.Color("#FFFFFF"), // White
			
			// Borders
			Border:        lipgloss.Color("#E2E8F0"), // Slate-200
			BorderSubtle:  lipgloss.Color("#F1F5F9"), // Slate-100
			BorderFocus:   lipgloss.Color("#7C3AED"), // Purple-600
			
			// States
			Success:       lipgloss.Color("#10B981"), // Emerald-500
			SuccessLight:  lipgloss.Color("#34D399"), // Emerald-400
			SuccessDark:   lipgloss.Color("#059669"), // Emerald-600
			
			Error:         lipgloss.Color("#EF4444"), // Red-500
			ErrorLight:    lipgloss.Color("#F87171"), // Red-400
			ErrorDark:     lipgloss.Color("#DC2626"), // Red-600
			
			Warning:       lipgloss.Color("#F59E0B"), // Amber-500
			WarningLight:  lipgloss.Color("#FBBF24"), // Amber-400
			WarningDark:   lipgloss.Color("#D97706"), // Amber-600
			
			Info:          lipgloss.Color("#3B82F6"), // Blue-500
			InfoLight:     lipgloss.Color("#60A5FA"), // Blue-400
			InfoDark:      lipgloss.Color("#2563EB"), // Blue-600
			
			// Special
			Highlight:     lipgloss.Color("#FEF3C7"), // Amber-100
			Selection:     lipgloss.Color("#E0E7FF"), // Indigo-100
			Shadow:        lipgloss.Color("#00000040"), // Black with 25% opacity
			
			// Code
			CodeForeground: lipgloss.Color("#1E293B"), // Slate-800
			CodeBackground: lipgloss.Color("#F1F5F9"), // Slate-100
		},
		Typography: defaultTypography(),
		Spacing:    defaultSpacing(),
		Components: defaultComponents(),
	}
	
	// CharmDark - Dark variant with the same purple gradients
	CharmDark = Theme{
		Name:        "charm-dark",
		DisplayName: "Charm Dark",
		Description: "Dark variant of the Charm theme with glowing purple accents",
		IsDark:      true,
		IsHighContrast: false,
		SupportsGradient: true,
		Colors: ColorPalette{
			// Primary - Brighter purples for dark theme
			Primary:       lipgloss.Color("#A855F7"), // Purple-500
			PrimaryLight:  lipgloss.Color("#C084FC"), // Purple-400
			PrimaryDark:   lipgloss.Color("#8B5CF6"), // Purple-500
			
			// Secondary - Bright magenta/pink
			Secondary:     lipgloss.Color("#F472B6"), // Pink-400
			SecondaryLight: lipgloss.Color("#F9A8D4"), // Pink-300
			SecondaryDark: lipgloss.Color("#EC4899"), // Pink-500
			
			// Accent - Bright violet
			Accent:        lipgloss.Color("#A78BFA"), // Violet-400
			AccentLight:   lipgloss.Color("#C4B5FD"), // Violet-300
			AccentDark:    lipgloss.Color("#8B5CF6"), // Violet-500
			
			// Backgrounds - Deep dark with subtle variations
			Background:    lipgloss.Color("#0F172A"), // Slate-900
			BackgroundAlt: lipgloss.Color("#1E293B"), // Slate-800
			BackgroundSubtle: lipgloss.Color("#334155"), // Slate-700
			
			// Surfaces
			Surface:       lipgloss.Color("#1E293B"), // Slate-800
			SurfaceAlt:    lipgloss.Color("#334155"), // Slate-700
			SurfaceSubtle: lipgloss.Color("#475569"), // Slate-600
			
			// Text - Light colors for contrast
			Text:          lipgloss.Color("#F8FAFC"), // Slate-50
			TextSubtle:    lipgloss.Color("#CBD5E1"), // Slate-300
			TextMuted:     lipgloss.Color("#94A3B8"), // Slate-400
			TextInverse:   lipgloss.Color("#0F172A"), // Slate-900
			
			// Borders
			Border:        lipgloss.Color("#475569"), // Slate-600
			BorderSubtle:  lipgloss.Color("#334155"), // Slate-700
			BorderFocus:   lipgloss.Color("#A855F7"), // Purple-500
			
			// States - Adjusted for dark theme
			Success:       lipgloss.Color("#34D399"), // Emerald-400
			SuccessLight:  lipgloss.Color("#6EE7B7"), // Emerald-300
			SuccessDark:   lipgloss.Color("#10B981"), // Emerald-500
			
			Error:         lipgloss.Color("#F87171"), // Red-400
			ErrorLight:    lipgloss.Color("#FCA5A5"), // Red-300
			ErrorDark:     lipgloss.Color("#EF4444"), // Red-500
			
			Warning:       lipgloss.Color("#FBBF24"), // Amber-400
			WarningLight:  lipgloss.Color("#FCD34D"), // Amber-300
			WarningDark:   lipgloss.Color("#F59E0B"), // Amber-500
			
			Info:          lipgloss.Color("#60A5FA"), // Blue-400
			InfoLight:     lipgloss.Color("#93C5FD"), // Blue-300
			InfoDark:      lipgloss.Color("#3B82F6"), // Blue-500
			
			// Special
			Highlight:     lipgloss.Color("#451A03"), // Amber-950
			Selection:     lipgloss.Color("#312E81"), // Indigo-900
			Shadow:        lipgloss.Color("#00000080"), // Black with 50% opacity
			
			// Code
			CodeForeground: lipgloss.Color("#CBD5E1"), // Slate-300
			CodeBackground: lipgloss.Color("#334155"), // Slate-700
		},
		Typography: defaultTypography(),
		Spacing:    defaultSpacing(),
		Components: defaultComponents(),
	}
	
	// HighContrast - Accessibility-focused high contrast theme
	HighContrast = Theme{
		Name:        "high-contrast",
		DisplayName: "High Contrast",
		Description: "High contrast theme for accessibility",
		IsDark:      false,
		IsHighContrast: true,
		SupportsGradient: false,
		Colors: ColorPalette{
			Primary:       lipgloss.Color("#000000"),
			PrimaryLight:  lipgloss.Color("#333333"),
			PrimaryDark:   lipgloss.Color("#000000"),
			
			Secondary:     lipgloss.Color("#FFFFFF"),
			SecondaryLight: lipgloss.Color("#F0F0F0"),
			SecondaryDark: lipgloss.Color("#CCCCCC"),
			
			Accent:        lipgloss.Color("#000000"),
			AccentLight:   lipgloss.Color("#333333"),
			AccentDark:    lipgloss.Color("#000000"),
			
			Background:    lipgloss.Color("#FFFFFF"),
			BackgroundAlt: lipgloss.Color("#F0F0F0"),
			BackgroundSubtle: lipgloss.Color("#E0E0E0"),
			
			Surface:       lipgloss.Color("#FFFFFF"),
			SurfaceAlt:    lipgloss.Color("#F0F0F0"),
			SurfaceSubtle: lipgloss.Color("#E0E0E0"),
			
			Text:          lipgloss.Color("#000000"),
			TextSubtle:    lipgloss.Color("#333333"),
			TextMuted:     lipgloss.Color("#666666"),
			TextInverse:   lipgloss.Color("#FFFFFF"),
			
			Border:        lipgloss.Color("#000000"),
			BorderSubtle:  lipgloss.Color("#666666"),
			BorderFocus:   lipgloss.Color("#000000"),
			
			Success:       lipgloss.Color("#008000"),
			SuccessLight:  lipgloss.Color("#00A000"),
			SuccessDark:   lipgloss.Color("#006000"),
			
			Error:         lipgloss.Color("#FF0000"),
			ErrorLight:    lipgloss.Color("#FF3333"),
			ErrorDark:     lipgloss.Color("#CC0000"),
			
			Warning:       lipgloss.Color("#FF8000"),
			WarningLight:  lipgloss.Color("#FFA000"),
			WarningDark:   lipgloss.Color("#CC6600"),
			
			Info:          lipgloss.Color("#0000FF"),
			InfoLight:     lipgloss.Color("#3333FF"),
			InfoDark:      lipgloss.Color("#0000CC"),
			
			Highlight:     lipgloss.Color("#FFFF00"),
			Selection:     lipgloss.Color("#0080FF"),
			Shadow:        lipgloss.Color("#000000"),
			
			// Code
			CodeForeground: lipgloss.Color("#000000"),
			CodeBackground: lipgloss.Color("#F0F0F0"),
		},
		Typography: defaultTypography(),
		Spacing:    defaultSpacing(),
		Components: defaultComponents(),
	}
)

// Default configuration functions
func defaultTypography() Typography {
	return Typography{
		FontFamily:     "monospace",
		FontFamilyMono: "monospace",
		
		SizeXSmall:  10,
		SizeSmall:   12,
		SizeMedium:  14,
		SizeLarge:   16,
		SizeXLarge:  18,
		SizeXXLarge: 24,
		
		LineHeightTight:  1.2,
		LineHeightNormal: 1.4,
		LineHeightLoose:  1.6,
		
		LetterSpacingTight:  "-0.05em",
		LetterSpacingNormal: "0",
		LetterSpacingWide:   "0.05em",
	}
}

func defaultSpacing() Spacing {
	base := 4
	return Spacing{
		Base: base,
		
		XSmall:  base / 4,      // 1
		Small:   base / 2,      // 2
		Medium:  base,          // 4
		Large:   base + base/2, // 6
		XLarge:  base * 2,      // 8
		XXLarge: base * 3,      // 12
		
		ButtonPadding:    [2]int{1, 2},
		InputPadding:     [2]int{1, 2},
		PanelPadding:     [2]int{2, 3},
		CardPadding:      [2]int{2, 3},
		ModalPadding:     [2]int{3, 4},
		
		SectionSpacing:   base * 2,     // 8
		ComponentSpacing: base + base/2, // 6
		ElementSpacing:   base,          // 4
	}
}

func defaultComponents() ComponentStyles {
	return ComponentStyles{
		BorderRadius:    1,
		BorderWidth:     1,
		BorderStyle:     lipgloss.RoundedBorder(),
		
		ShadowEnabled:   true,
		ShadowColor:     lipgloss.Color("#00000020"),
		ShadowOffset:    [2]int{0, 1},
		
		AnimationSpeed:     "fast",
		TransitionDuration: "200ms",
		
		ButtonHeight:     3,
		InputHeight:      3,
		MenuItemHeight:   1,
		ListItemHeight:   1,
		HeaderHeight:     3,
		FooterHeight:     2,
		StatusBarHeight:  1,
	}
}

// ThemeManager handles theme switching and persistence
type ThemeManager struct {
	currentTheme    *Theme
	availableThemes map[string]*Theme
	terminalProfile termenv.Profile
	colorSupport   ColorSupport
}

// ColorSupport represents terminal color capabilities
type ColorSupport struct {
	HasTrueColor bool
	Has256Color  bool
	HasColor     bool
	IsMonochrome bool
}

// NewThemeManager creates a new theme manager
func NewThemeManager() *ThemeManager {
	tm := &ThemeManager{
		availableThemes: make(map[string]*Theme),
		terminalProfile: termenv.ColorProfile(),
	}
	
	// Detect terminal capabilities
	tm.colorSupport = tm.detectColorSupport()
	
	// Register built-in themes
	tm.RegisterTheme(&CharmLight)
	tm.RegisterTheme(&CharmDark)
	tm.RegisterTheme(&HighContrast)
	
	// Set default theme based on environment
	tm.SetDefaultTheme()
	
	return tm
}

// RegisterTheme registers a new theme
func (tm *ThemeManager) RegisterTheme(theme *Theme) {
	tm.availableThemes[theme.Name] = theme
}

// SetTheme activates a theme by name
func (tm *ThemeManager) SetTheme(name string) error {
	theme, exists := tm.availableThemes[name]
	if !exists {
		return fmt.Errorf("theme '%s' not found", name)
	}
	
	// Adapt theme to terminal capabilities
	adaptedTheme := tm.adaptThemeToTerminal(theme)
	tm.currentTheme = adaptedTheme
	
	return nil
}

// GetCurrentTheme returns the currently active theme
func (tm *ThemeManager) GetCurrentTheme() *Theme {
	return tm.currentTheme
}

// GetAvailableThemes returns all registered themes
func (tm *ThemeManager) GetAvailableThemes() map[string]*Theme {
	return tm.availableThemes
}

// SetDefaultTheme sets a reasonable default theme
func (tm *ThemeManager) SetDefaultTheme() {
	// Check for user preference or environment
	preferredTheme := os.Getenv("KLIP_THEME")
	if preferredTheme != "" {
		if err := tm.SetTheme(preferredTheme); err == nil {
			return
		}
	}
	
	// Auto-detect based on terminal capabilities and environment
	if tm.shouldUseHighContrast() {
		tm.SetTheme("high-contrast")
	} else if tm.shouldUseDarkTheme() {
		tm.SetTheme("charm-dark")
	} else {
		tm.SetTheme("charm-light")
	}
}

// detectColorSupport analyzes terminal color capabilities
func (tm *ThemeManager) detectColorSupport() ColorSupport {
	profile := tm.terminalProfile
	
	support := ColorSupport{
		HasTrueColor: profile == termenv.TrueColor,
		Has256Color:  profile >= termenv.ANSI256,
		HasColor:     profile >= termenv.ANSI,
		IsMonochrome: profile == termenv.Ascii,
	}
	
	return support
}

// adaptThemeToTerminal adapts a theme to terminal capabilities
func (tm *ThemeManager) adaptThemeToTerminal(theme *Theme) *Theme {
	// Create a copy of the theme to avoid modifying the original
	adaptedTheme := *theme
	adaptedColors := theme.Colors
	
	// Adapt colors based on terminal capabilities
	if tm.colorSupport.IsMonochrome {
		// Convert to monochrome
		adaptedColors = tm.convertToMonochrome(adaptedColors)
	} else if !tm.colorSupport.HasTrueColor && !tm.colorSupport.Has256Color {
		// Fallback to basic 16 colors
		adaptedColors = tm.convertToBasicColors(adaptedColors)
	} else if !tm.colorSupport.HasTrueColor && tm.colorSupport.Has256Color {
		// Convert true colors to 256-color approximations
		adaptedColors = tm.convertTo256Colors(adaptedColors)
	}
	
	adaptedTheme.Colors = adaptedColors
	return &adaptedTheme
}

// shouldUseHighContrast determines if high contrast should be used
func (tm *ThemeManager) shouldUseHighContrast() bool {
	// Check environment variables or system settings
	return strings.ToLower(os.Getenv("FORCE_HIGH_CONTRAST")) == "true" ||
		   strings.ToLower(os.Getenv("ACCESSIBILITY_HIGH_CONTRAST")) == "true"
}

// shouldUseDarkTheme determines if dark theme should be used
func (tm *ThemeManager) shouldUseDarkTheme() bool {
	// Check various indicators for dark theme preference
	colorTerm := strings.ToLower(os.Getenv("COLORFGBG"))
	if strings.Contains(colorTerm, "15;0") || strings.Contains(colorTerm, "7;0") {
		return true // Light text on dark background
	}
	
	// Check for explicit dark theme preference
	return strings.ToLower(os.Getenv("KLIP_DARK_MODE")) == "true" ||
		   strings.ToLower(os.Getenv("DARK_MODE")) == "true"
}

// Color conversion methods for terminal adaptation
func (tm *ThemeManager) convertToMonochrome(colors ColorPalette) ColorPalette {
	// Convert all colors to black/white/gray equivalents
	white := lipgloss.Color("#FFFFFF")
	black := lipgloss.Color("#000000")
	gray := lipgloss.Color("#808080")
	
	return ColorPalette{
		Primary:          black,
		PrimaryLight:     gray,
		PrimaryDark:      black,
		Secondary:        gray,
		SecondaryLight:   white,
		SecondaryDark:    black,
		Accent:           black,
		AccentLight:      gray,
		AccentDark:       black,
		Background:       white,
		BackgroundAlt:    white,
		BackgroundSubtle: white,
		Surface:          white,
		SurfaceAlt:       white,
		SurfaceSubtle:    white,
		Text:             black,
		TextSubtle:       gray,
		TextMuted:        gray,
		TextInverse:      white,
		Border:           black,
		BorderSubtle:     gray,
		BorderFocus:      black,
		Success:          black,
		SuccessLight:     gray,
		SuccessDark:      black,
		Error:            black,
		ErrorLight:       gray,
		ErrorDark:        black,
		Warning:          black,
		WarningLight:     gray,
		WarningDark:      black,
		Info:             black,
		InfoLight:        gray,
		InfoDark:         black,
		Highlight:        gray,
		Selection:        gray,
		Shadow:           black,
		CodeForeground:   black,
		CodeBackground:   white,
	}
}

func (tm *ThemeManager) convertToBasicColors(colors ColorPalette) ColorPalette {
	// Map to basic ANSI colors
	return ColorPalette{
		Primary:          lipgloss.Color("5"),  // Magenta
		PrimaryLight:     lipgloss.Color("13"), // Bright Magenta
		PrimaryDark:      lipgloss.Color("5"),  // Magenta
		Secondary:        lipgloss.Color("4"),  // Blue
		SecondaryLight:   lipgloss.Color("12"), // Bright Blue
		SecondaryDark:    lipgloss.Color("4"),  // Blue
		Accent:           lipgloss.Color("6"),  // Cyan
		AccentLight:      lipgloss.Color("14"), // Bright Cyan
		AccentDark:       lipgloss.Color("6"),  // Cyan
		Background:       lipgloss.Color("0"),  // Black or White depending on theme
		BackgroundAlt:    lipgloss.Color("0"),
		BackgroundSubtle: lipgloss.Color("8"),  // Bright Black (Gray)
		Surface:          lipgloss.Color("0"),
		SurfaceAlt:       lipgloss.Color("8"),
		SurfaceSubtle:    lipgloss.Color("8"),
		Text:             lipgloss.Color("7"),  // White or Black depending on theme
		TextSubtle:       lipgloss.Color("8"),  // Bright Black
		TextMuted:        lipgloss.Color("8"),
		TextInverse:      lipgloss.Color("0"),
		Border:           lipgloss.Color("8"),
		BorderSubtle:     lipgloss.Color("8"),
		BorderFocus:      lipgloss.Color("5"),  // Magenta
		Success:          lipgloss.Color("2"),  // Green
		SuccessLight:     lipgloss.Color("10"), // Bright Green
		SuccessDark:      lipgloss.Color("2"),
		Error:            lipgloss.Color("1"),  // Red
		ErrorLight:       lipgloss.Color("9"),  // Bright Red
		ErrorDark:        lipgloss.Color("1"),
		Warning:          lipgloss.Color("3"),  // Yellow
		WarningLight:     lipgloss.Color("11"), // Bright Yellow
		WarningDark:      lipgloss.Color("3"),
		Info:             lipgloss.Color("4"),  // Blue
		InfoLight:        lipgloss.Color("12"), // Bright Blue
		InfoDark:         lipgloss.Color("4"),
		Highlight:        lipgloss.Color("11"), // Bright Yellow
		Selection:        lipgloss.Color("4"),  // Blue
		Shadow:           lipgloss.Color("0"),  // Black
		CodeForeground:   lipgloss.Color("7"),  // White
		CodeBackground:   lipgloss.Color("8"),  // Bright Black (Gray)
	}
}

func (tm *ThemeManager) convertTo256Colors(colors ColorPalette) ColorPalette {
	// Convert hex colors to closest 256-color approximations
	// This is a simplified conversion - in a real implementation,
	// you'd use a proper color distance algorithm
	return colors // For now, return as-is since lipgloss handles this
}

// Utility functions for working with themes

// CreateGradient creates a gradient effect using the theme colors
func (theme *Theme) CreateGradient(text string, startColor, endColor lipgloss.Color) string {
	if !theme.SupportsGradient || len(text) == 0 {
		return lipgloss.NewStyle().Foreground(startColor).Render(text)
	}
	
	// Simple gradient implementation
	// In a real implementation, you'd interpolate between colors
	return lipgloss.NewStyle().
		Foreground(startColor).
		Background(lipgloss.Color("")).
		Render(text)
}

// GetStateColor returns the appropriate color for a given state
func (theme *Theme) GetStateColor(state string) lipgloss.Color {
	switch strings.ToLower(state) {
	case "success", "ok", "done", "complete":
		return theme.Colors.Success
	case "error", "fail", "failed":
		return theme.Colors.Error
	case "warning", "warn":
		return theme.Colors.Warning
	case "info", "information":
		return theme.Colors.Info
	case "primary":
		return theme.Colors.Primary
	case "secondary":
		return theme.Colors.Secondary
	case "accent":
		return theme.Colors.Accent
	default:
		return theme.Colors.Text
	}
}

// GetContrastColor returns a color with good contrast against the given background
func (theme *Theme) GetContrastColor(background lipgloss.Color) lipgloss.Color {
	// Simplified contrast calculation
	// In a real implementation, you'd calculate luminance and choose accordingly
	if theme.IsDark {
		return theme.Colors.Text
	}
	return theme.Colors.TextInverse
}

// ApplyColorProfile applies the terminal's color profile to a color
func ApplyColorProfile(color lipgloss.Color) lipgloss.Color {
	// In newer versions, lipgloss handles color profile adaptation internally
	// Just return the color as-is since lipgloss will handle the conversion
	return color
}

// Global theme manager instance
var DefaultThemeManager = NewThemeManager()

// Convenience functions for global theme access
func GetCurrentTheme() *Theme {
	return DefaultThemeManager.GetCurrentTheme()
}

func SetTheme(name string) error {
	return DefaultThemeManager.SetTheme(name)
}

func RegisterTheme(theme *Theme) {
	DefaultThemeManager.RegisterTheme(theme)
}