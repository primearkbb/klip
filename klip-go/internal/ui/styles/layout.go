package styles

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LayoutManager handles responsive layouts and positioning
type LayoutManager struct {
	width       int
	height      int
	theme       *Theme
	breakpoints Breakpoints
}

// Breakpoints define responsive breakpoints for different terminal sizes
type Breakpoints struct {
	XSmall int // < 40 columns
	Small  int // 40-79 columns
	Medium int // 80-119 columns
	Large  int // 120-159 columns
	XLarge int // >= 160 columns
}

// ScreenSize represents the current screen size category
type ScreenSize int

const (
	ScreenXSmall ScreenSize = iota
	ScreenSmall
	ScreenMedium
	ScreenLarge
	ScreenXLarge
)

// Layout represents a flexible layout system
type Layout struct {
	Direction FlexDirection
	Wrap      FlexWrap
	Justify   JustifyContent
	Align     AlignItems
	Gap       int
	Padding   [4]int // top, right, bottom, left
	Margin    [4]int
	Width     int
	Height    int
	MinWidth  int
	MinHeight int
	MaxWidth  int
	MaxHeight int
	Grow      int
	Shrink    int
	Basis     int
}

// FlexDirection defines layout direction
type FlexDirection int

const (
	Row FlexDirection = iota
	RowReverse
	Column
	ColumnReverse
)

// FlexWrap defines wrapping behavior
type FlexWrap int

const (
	NoWrap FlexWrap = iota
	Wrap
	WrapReverse
)

// JustifyContent defines main axis alignment
type JustifyContent int

const (
	JustifyStart JustifyContent = iota
	JustifyEnd
	JustifyCenter
	JustifySpaceBetween
	JustifySpaceAround
	JustifySpaceEvenly
)

// AlignItems defines cross axis alignment
type AlignItems int

const (
	AlignStart AlignItems = iota
	AlignEnd
	AlignCenter
	AlignStretch
	AlignBaseline
)

// Grid represents a CSS Grid-like layout system
type Grid struct {
	Columns   []string // e.g., ["1fr", "200px", "1fr"]
	Rows      []string // e.g., ["auto", "1fr", "auto"]
	Gap       int
	RowGap    int
	ColumnGap int
	Padding   [4]int
	Areas     [][]string // Grid areas definition
}

// Container represents different container types
type Container struct {
	Type    ContainerType
	Width   int
	Height  int
	Padding [4]int
	Margin  [4]int
	Border  lipgloss.Border
	Style   lipgloss.Style
}

// ContainerType defines different container styles
type ContainerType int

const (
	ContainerFluid ContainerType = iota
	ContainerFixed
	ContainerCenter
	ContainerSidebar
	ContainerMain
	ContainerModal
	ContainerCard
	ContainerPanel
)

// Panel layouts for common UI patterns
type PanelLayout struct {
	Header  *Container
	Sidebar *Container
	Main    *Container
	Footer  *Container
	Width   int
	Height  int
}

// NewLayoutManager creates a new layout manager
func NewLayoutManager(width, height int, theme *Theme) *LayoutManager {
	return &LayoutManager{
		width:  width,
		height: height,
		theme:  theme,
		breakpoints: Breakpoints{
			XSmall: 40,
			Small:  80,
			Medium: 120,
			Large:  160,
			XLarge: 200,
		},
	}
}

// GetScreenSize determines current screen size category
func (lm *LayoutManager) GetScreenSize() ScreenSize {
	w := lm.width
	bp := lm.breakpoints

	switch {
	case w < bp.XSmall:
		return ScreenXSmall
	case w < bp.Small:
		return ScreenSmall
	case w < bp.Medium:
		return ScreenMedium
	case w < bp.Large:
		return ScreenLarge
	default:
		return ScreenXLarge
	}
}

// Resize updates layout dimensions
func (lm *LayoutManager) Resize(width, height int) {
	lm.width = width
	lm.height = height
}

// CreateResponsiveGrid creates a responsive grid layout
func (lm *LayoutManager) CreateResponsiveGrid(columns int) *Grid {
	screenSize := lm.GetScreenSize()

	// Adjust columns based on screen size
	switch screenSize {
	case ScreenXSmall:
		columns = 1
	case ScreenSmall:
		columns = min(columns, 2)
	case ScreenMedium:
		columns = min(columns, 3)
	case ScreenLarge:
		columns = min(columns, 4)
	default:
		// Keep original columns for XLarge
	}

	// Create column definitions
	colDefs := make([]string, columns)
	for i := 0; i < columns; i++ {
		colDefs[i] = "1fr"
	}

	return &Grid{
		Columns:   colDefs,
		Rows:      []string{"auto"},
		Gap:       lm.theme.Spacing.ComponentSpacing,
		ColumnGap: lm.theme.Spacing.ComponentSpacing,
		RowGap:    lm.theme.Spacing.ElementSpacing,
		Padding:   [4]int{lm.theme.Spacing.Medium, lm.theme.Spacing.Medium, lm.theme.Spacing.Medium, lm.theme.Spacing.Medium},
	}
}

// CreateFlexLayout creates a flexible layout
func (lm *LayoutManager) CreateFlexLayout(direction FlexDirection) *Layout {
	return &Layout{
		Direction: direction,
		Wrap:      Wrap,
		Justify:   JustifyStart,
		Align:     AlignStretch,
		Gap:       lm.theme.Spacing.ComponentSpacing,
		Padding:   [4]int{lm.theme.Spacing.Medium, lm.theme.Spacing.Medium, lm.theme.Spacing.Medium, lm.theme.Spacing.Medium},
		Width:     lm.width,
		Height:    lm.height,
		Grow:      0,
		Shrink:    1,
		Basis:     0,
	}
}

// CreateContainer creates a styled container
func (lm *LayoutManager) CreateContainer(containerType ContainerType) *Container {
	container := &Container{
		Type:   containerType,
		Width:  lm.width,
		Height: lm.height,
		Border: lm.theme.Components.BorderStyle,
	}

	// Apply type-specific styling
	switch containerType {
	case ContainerFluid:
		container.Width = lm.width
		container.Padding = [4]int{0, 0, 0, 0}
		container.Style = lipgloss.NewStyle().
			Width(container.Width).
			Background(lm.theme.Colors.Background)

	case ContainerFixed:
		maxWidth := lm.getFixedContainerWidth()
		container.Width = min(lm.width, maxWidth)
		container.Padding = [4]int{lm.theme.Spacing.Large, lm.theme.Spacing.Large, lm.theme.Spacing.Large, lm.theme.Spacing.Large}
		container.Style = lipgloss.NewStyle().
			Width(container.Width).
			Padding(container.Padding[0], container.Padding[1], container.Padding[2], container.Padding[3]).
			Background(lm.theme.Colors.Background)

	case ContainerCenter:
		maxWidth := lm.getCenterContainerWidth()
		container.Width = min(lm.width, maxWidth)
		container.Padding = [4]int{lm.theme.Spacing.Large, lm.theme.Spacing.Large, lm.theme.Spacing.Large, lm.theme.Spacing.Large}
		container.Style = lipgloss.NewStyle().
			Width(container.Width).
			Padding(container.Padding[0], container.Padding[1], container.Padding[2], container.Padding[3]).
			Align(lipgloss.Center).
			Background(lm.theme.Colors.Background)

	case ContainerModal:
		container.Width = lm.getModalWidth()
		container.Height = lm.getModalHeight()
		container.Padding = [4]int{lm.theme.Spacing.XLarge, lm.theme.Spacing.XLarge, lm.theme.Spacing.XLarge, lm.theme.Spacing.XLarge}
		container.Style = lipgloss.NewStyle().
			Width(container.Width).
			Height(container.Height).
			Padding(container.Padding[0], container.Padding[1], container.Padding[2], container.Padding[3]).
			Border(lm.theme.Components.BorderStyle).
			BorderForeground(lm.theme.Colors.Border).
			Background(lm.theme.Colors.Surface).
			Align(lipgloss.Center)

	case ContainerCard:
		container.Padding = [4]int{lm.theme.Spacing.Large, lm.theme.Spacing.Large, lm.theme.Spacing.Large, lm.theme.Spacing.Large}
		container.Style = lipgloss.NewStyle().
			Padding(container.Padding[0], container.Padding[1], container.Padding[2], container.Padding[3]).
			Border(lm.theme.Components.BorderStyle).
			BorderForeground(lm.theme.Colors.Border).
			Background(lm.theme.Colors.Surface)

	case ContainerPanel:
		container.Padding = [4]int{lm.theme.Spacing.Medium, lm.theme.Spacing.Medium, lm.theme.Spacing.Medium, lm.theme.Spacing.Medium}
		container.Style = lipgloss.NewStyle().
			Padding(container.Padding[0], container.Padding[1], container.Padding[2], container.Padding[3]).
			Border(lm.theme.Components.BorderStyle).
			BorderForeground(lm.theme.Colors.BorderSubtle).
			Background(lm.theme.Colors.BackgroundAlt)

	case ContainerSidebar:
		container.Width = lm.getSidebarWidth()
		container.Height = lm.height
		container.Padding = [4]int{lm.theme.Spacing.Medium, lm.theme.Spacing.Small, lm.theme.Spacing.Medium, lm.theme.Spacing.Small}
		container.Style = lipgloss.NewStyle().
			Width(container.Width).
			Height(container.Height).
			Padding(container.Padding[0], container.Padding[1], container.Padding[2], container.Padding[3]).
			Border(lipgloss.Border{Right: "│"}).
			BorderForeground(lm.theme.Colors.Border).
			Background(lm.theme.Colors.BackgroundSubtle)

	case ContainerMain:
		sidebarWidth := lm.getSidebarWidth()
		container.Width = lm.width - sidebarWidth
		container.Height = lm.height
		container.Padding = [4]int{lm.theme.Spacing.Medium, lm.theme.Spacing.Medium, lm.theme.Spacing.Medium, lm.theme.Spacing.Medium}
		container.Style = lipgloss.NewStyle().
			Width(container.Width).
			Height(container.Height).
			Padding(container.Padding[0], container.Padding[1], container.Padding[2], container.Padding[3]).
			Background(lm.theme.Colors.Background)
	}

	return container
}

// CreateApplicationLayout creates a standard application layout
func (lm *LayoutManager) CreateApplicationLayout() *PanelLayout {
	screenSize := lm.GetScreenSize()

	layout := &PanelLayout{
		Width:  lm.width,
		Height: lm.height,
	}

	// Header
	headerHeight := lm.theme.Components.HeaderHeight
	layout.Header = &Container{
		Type:    ContainerFluid,
		Width:   lm.width,
		Height:  headerHeight,
		Padding: [4]int{lm.theme.Spacing.Small, lm.theme.Spacing.Medium, lm.theme.Spacing.Small, lm.theme.Spacing.Medium},
		Style: lipgloss.NewStyle().
			Width(lm.width).
			Height(headerHeight).
			Padding(lm.theme.Spacing.Small, lm.theme.Spacing.Medium).
			Border(lipgloss.Border{Bottom: "─"}).
			BorderForeground(lm.theme.Colors.Border).
			Background(lm.theme.Colors.Surface),
	}

	// Footer
	footerHeight := lm.theme.Components.FooterHeight
	layout.Footer = &Container{
		Type:    ContainerFluid,
		Width:   lm.width,
		Height:  footerHeight,
		Padding: [4]int{lm.theme.Spacing.Small, lm.theme.Spacing.Medium, lm.theme.Spacing.Small, lm.theme.Spacing.Medium},
		Style: lipgloss.NewStyle().
			Width(lm.width).
			Height(footerHeight).
			Padding(lm.theme.Spacing.Small, lm.theme.Spacing.Medium).
			Border(lipgloss.Border{Top: "─"}).
			BorderForeground(lm.theme.Colors.Border).
			Background(lm.theme.Colors.Surface),
	}

	// Main content area dimensions
	contentHeight := lm.height - headerHeight - footerHeight

	// Adaptive sidebar based on screen size
	if screenSize >= ScreenMedium {
		// Large enough for sidebar
		sidebarWidth := lm.getSidebarWidth()

		layout.Sidebar = &Container{
			Type:    ContainerSidebar,
			Width:   sidebarWidth,
			Height:  contentHeight,
			Padding: [4]int{lm.theme.Spacing.Medium, lm.theme.Spacing.Small, lm.theme.Spacing.Medium, lm.theme.Spacing.Small},
			Style: lipgloss.NewStyle().
				Width(sidebarWidth).
				Height(contentHeight).
				Padding(lm.theme.Spacing.Medium, lm.theme.Spacing.Small).
				Border(lipgloss.Border{Right: "│"}).
				BorderForeground(lm.theme.Colors.Border).
				Background(lm.theme.Colors.BackgroundSubtle),
		}

		layout.Main = &Container{
			Type:    ContainerMain,
			Width:   lm.width - sidebarWidth,
			Height:  contentHeight,
			Padding: [4]int{lm.theme.Spacing.Medium, lm.theme.Spacing.Medium, lm.theme.Spacing.Medium, lm.theme.Spacing.Medium},
			Style: lipgloss.NewStyle().
				Width(lm.width - sidebarWidth).
				Height(contentHeight).
				Padding(lm.theme.Spacing.Medium).
				Background(lm.theme.Colors.Background),
		}
	} else {
		// Small screen - full width main, no sidebar
		layout.Main = &Container{
			Type:    ContainerMain,
			Width:   lm.width,
			Height:  contentHeight,
			Padding: [4]int{lm.theme.Spacing.Medium, lm.theme.Spacing.Small, lm.theme.Spacing.Medium, lm.theme.Spacing.Small},
			Style: lipgloss.NewStyle().
				Width(lm.width).
				Height(contentHeight).
				Padding(lm.theme.Spacing.Medium, lm.theme.Spacing.Small).
				Background(lm.theme.Colors.Background),
		}
	}

	return layout
}

// Layout rendering methods

// RenderGrid renders items in a grid layout
func (lm *LayoutManager) RenderGrid(grid *Grid, items []string) string {
	if len(items) == 0 {
		return ""
	}

	cols := len(grid.Columns)
	if cols == 0 {
		return strings.Join(items, "\n")
	}

	// Calculate column widths
	availableWidth := lm.width - (grid.Padding[1] + grid.Padding[3]) - ((cols - 1) * grid.ColumnGap)
	colWidths := lm.calculateColumnWidths(grid.Columns, availableWidth)

	var rows []string
	for i := 0; i < len(items); i += cols {
		var rowItems []string
		for j := 0; j < cols && i+j < len(items); j++ {
			item := items[i+j]
			if colWidths[j] > 0 {
				style := lipgloss.NewStyle().Width(colWidths[j])
				rowItems = append(rowItems, style.Render(item))
			}
		}

		if len(rowItems) > 0 {
			gap := strings.Repeat(" ", grid.ColumnGap)
			rows = append(rows, strings.Join(rowItems, gap))
		}
	}

	rowGap := strings.Repeat("\n", grid.RowGap+1)
	content := strings.Join(rows, rowGap)

	// Apply padding
	if grid.Padding[0] > 0 || grid.Padding[1] > 0 || grid.Padding[2] > 0 || grid.Padding[3] > 0 {
		style := lipgloss.NewStyle().
			Padding(grid.Padding[0], grid.Padding[1], grid.Padding[2], grid.Padding[3])
		content = style.Render(content)
	}

	return content
}

// RenderFlex renders items in a flex layout
func (lm *LayoutManager) RenderFlex(layout *Layout, items []string) string {
	if len(items) == 0 {
		return ""
	}

	var content string

	switch layout.Direction {
	case Row, RowReverse:
		content = lm.renderFlexRow(layout, items)
	case Column, ColumnReverse:
		content = lm.renderFlexColumn(layout, items)
	}

	// Apply padding
	if layout.Padding[0] > 0 || layout.Padding[1] > 0 || layout.Padding[2] > 0 || layout.Padding[3] > 0 {
		style := lipgloss.NewStyle().
			Padding(layout.Padding[0], layout.Padding[1], layout.Padding[2], layout.Padding[3])
		content = style.Render(content)
	}

	return content
}

// RenderContainer renders content within a container
func (lm *LayoutManager) RenderContainer(container *Container, content string) string {
	return container.Style.Render(content)
}

// RenderPanelLayout renders a complete panel layout
func (lm *LayoutManager) RenderPanelLayout(layout *PanelLayout, header, sidebar, main, footer string) string {
	var parts []string

	// Header
	if layout.Header != nil && header != "" {
		parts = append(parts, lm.RenderContainer(layout.Header, header))
	}

	// Content area (sidebar + main)
	var contentParts []string
	if layout.Sidebar != nil && sidebar != "" {
		contentParts = append(contentParts, lm.RenderContainer(layout.Sidebar, sidebar))
	}
	if layout.Main != nil {
		contentParts = append(contentParts, lm.RenderContainer(layout.Main, main))
	}

	if len(contentParts) > 0 {
		content := lipgloss.JoinHorizontal(lipgloss.Top, contentParts...)
		parts = append(parts, content)
	}

	// Footer
	if layout.Footer != nil && footer != "" {
		parts = append(parts, lm.RenderContainer(layout.Footer, footer))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// Helper methods for responsive calculations

func (lm *LayoutManager) getFixedContainerWidth() int {
	screenSize := lm.GetScreenSize()
	switch screenSize {
	case ScreenXSmall:
		return lm.width - 4 // Very small margin
	case ScreenSmall:
		return lm.width - 8
	case ScreenMedium:
		return min(lm.width-16, 100)
	case ScreenLarge:
		return min(lm.width-24, 120)
	default:
		return min(lm.width-32, 140)
	}
}

func (lm *LayoutManager) getCenterContainerWidth() int {
	screenSize := lm.GetScreenSize()
	switch screenSize {
	case ScreenXSmall:
		return lm.width - 2
	case ScreenSmall:
		return lm.width - 6
	case ScreenMedium:
		return min(lm.width-12, 80)
	case ScreenLarge:
		return min(lm.width-20, 100)
	default:
		return min(lm.width-28, 120)
	}
}

func (lm *LayoutManager) getSidebarWidth() int {
	screenSize := lm.GetScreenSize()
	switch screenSize {
	case ScreenXSmall, ScreenSmall:
		return 0 // No sidebar on small screens
	case ScreenMedium:
		return min(lm.width/4, 20)
	case ScreenLarge:
		return min(lm.width/4, 25)
	default:
		return min(lm.width/4, 30)
	}
}

func (lm *LayoutManager) getModalWidth() int {
	return min(lm.width-10, max(lm.width*3/4, 40))
}

func (lm *LayoutManager) getModalHeight() int {
	return min(lm.height-6, max(lm.height*3/4, 15))
}

func (lm *LayoutManager) calculateColumnWidths(columns []string, availableWidth int) []int {
	widths := make([]int, len(columns))
	totalFr := 0.0
	fixedWidth := 0

	// First pass: calculate fixed widths and count fr units
	for i, col := range columns {
		if strings.HasSuffix(col, "fr") {
			fr := 1.0
			if len(col) > 2 {
				if parsed, err := fmt.Sscanf(col, "%ffr", &fr); parsed == 1 && err == nil {
					totalFr += fr
				} else {
					totalFr += 1.0
				}
			} else {
				totalFr += 1.0
			}
		} else if strings.HasSuffix(col, "px") {
			var px int
			if parsed, err := fmt.Sscanf(col, "%dpx", &px); parsed == 1 && err == nil {
				widths[i] = px
				fixedWidth += px
			}
		} else if col == "auto" {
			// Auto width - will be calculated based on content
			widths[i] = -1 // Mark as auto
		} else {
			// Try to parse as integer (assume characters)
			var chars int
			if parsed, err := fmt.Sscanf(col, "%d", &chars); parsed == 1 && err == nil {
				widths[i] = chars
				fixedWidth += chars
			}
		}
	}

	// Second pass: distribute remaining width among fr units
	if totalFr > 0 {
		remainingWidth := availableWidth - fixedWidth
		if remainingWidth > 0 {
			frWidth := float64(remainingWidth) / totalFr

			for i, col := range columns {
				if strings.HasSuffix(col, "fr") {
					fr := 1.0
					if len(col) > 2 {
						fmt.Sscanf(col, "%ffr", &fr)
					}
					widths[i] = int(math.Round(fr * frWidth))
				}
			}
		}
	}

	// Handle auto widths (simplified - equal distribution of remaining space)
	autoCount := 0
	for _, width := range widths {
		if width == -1 {
			autoCount++
		}
	}

	if autoCount > 0 {
		usedWidth := 0
		for _, width := range widths {
			if width > 0 {
				usedWidth += width
			}
		}

		remainingWidth := availableWidth - usedWidth
		if remainingWidth > 0 {
			autoWidth := remainingWidth / autoCount
			for i, width := range widths {
				if width == -1 {
					widths[i] = autoWidth
				}
			}
		}
	}

	return widths
}

func (lm *LayoutManager) renderFlexRow(layout *Layout, items []string) string {
	if layout.Direction == RowReverse {
		// Reverse items
		reversed := make([]string, len(items))
		for i, item := range items {
			reversed[len(items)-1-i] = item
		}
		items = reversed
	}

	gap := strings.Repeat(" ", layout.Gap)
	return strings.Join(items, gap)
}

func (lm *LayoutManager) renderFlexColumn(layout *Layout, items []string) string {
	if layout.Direction == ColumnReverse {
		// Reverse items
		reversed := make([]string, len(items))
		for i, item := range items {
			reversed[len(items)-1-i] = item
		}
		items = reversed
	}

	gap := strings.Repeat("\n", layout.Gap+1)
	return strings.Join(items, gap)
}

// Utility functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
