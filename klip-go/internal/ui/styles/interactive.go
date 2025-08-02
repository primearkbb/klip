package styles

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
)

// InteractiveStyler handles dynamic styling with animations and state transitions
type InteractiveStyler struct {
	theme     *Theme
	width     int
	height    int
	animator  *AnimationManager
}

// SpringState tracks position and velocity for a spring animation
type SpringState struct {
	spring   harmonica.Spring
	position float64
	velocity float64
}

// AnimationManager handles smooth transitions and animations
type AnimationManager struct {
	springs     map[string]*SpringState
	transitions map[string]*Transition
	config      AnimationConfig
}

// AnimationConfig defines animation parameters
type AnimationConfig struct {
	DefaultDuration     time.Duration
	EasingFunction      string
	SpringTension       float64
	SpringFriction      float64
	FadeInDuration      time.Duration
	FadeOutDuration     time.Duration
	SlideDistance       int
	BounceAmplitude     float64
	PulseIntensity      float64
}

// Transition represents an animated transition between states
type Transition struct {
	From          interface{}
	To            interface{}
	Current       interface{}
	Progress      float64
	Duration      time.Duration
	StartTime     time.Time
	EasingFunc    func(float64) float64
	OnComplete    func()
	OnUpdate      func(interface{})
	IsActive      bool
}

// AnimationType represents different animation types
type AnimationType int

const (
	AnimationFadeIn AnimationType = iota
	AnimationFadeOut
	AnimationSlideIn
	AnimationSlideOut
	AnimationBounce
	AnimationPulse
	AnimationShake
	AnimationGlow
	AnimationRotate
	AnimationScale
)

// InteractionState represents different interaction states for animations
type InteractionState int

const (
	StateIdle InteractionState = iota
	StateHover
	StateFocus
	StateActive
	StatePressed
	StateReleased
	StateLoading
	StateSuccess
	StateError
	StateWarning
)

// NewInteractiveStyler creates a new interactive styler
func NewInteractiveStyler(theme *Theme, width, height int) *InteractiveStyler {
	return &InteractiveStyler{
		theme:    theme,
		width:    width,
		height:   height,
		animator: NewAnimationManager(),
	}
}

// NewAnimationManager creates a new animation manager
func NewAnimationManager() *AnimationManager {
	return &AnimationManager{
		springs:     make(map[string]*SpringState),
		transitions: make(map[string]*Transition),
		config: AnimationConfig{
			DefaultDuration:  300 * time.Millisecond,
			EasingFunction:   "ease-out",
			SpringTension:    300,
			SpringFriction:   30,
			FadeInDuration:   200 * time.Millisecond,
			FadeOutDuration:  150 * time.Millisecond,
			SlideDistance:    20,
			BounceAmplitude:  0.1,
			PulseIntensity:   0.2,
		},
	}
}

// Interactive Button with smooth state transitions
func (is *InteractiveStyler) InteractiveButton(text string, variant ButtonStyle, size ButtonSize, state InteractionState, progress float64) string {
	baseStyle := is.getBaseButtonStyle(variant, size)
	
	// Apply state-specific styling with animations
	switch state {
	case StateHover:
		baseStyle = is.applyHoverEffect(baseStyle, variant, progress)
	case StateFocus:
		baseStyle = is.applyFocusEffect(baseStyle, variant, progress)
	case StateActive, StatePressed:
		baseStyle = is.applyActiveEffect(baseStyle, variant, progress)
	case StateLoading:
		text = is.AnimatedSpinner(progress) + " " + text
	case StateSuccess:
		baseStyle = is.applySuccessEffect(baseStyle, progress)
		text = "✓ " + text
	case StateError:
		baseStyle = is.applyErrorEffect(baseStyle, progress)
		text = "✗ " + text
	}
	
	return baseStyle.Render(text)
}

// Interactive Input with focus animations
func (is *InteractiveStyler) InteractiveInput(value, placeholder string, inputType InputType, state InteractionState, width int, progress float64) string {
	if width <= 0 {
		width = 30
	}
	
	baseStyle := lipgloss.NewStyle().
		Width(width).
		Padding(is.theme.Spacing.InputPadding[0], is.theme.Spacing.InputPadding[1]).
		Background(is.theme.Colors.Surface).
		Foreground(is.theme.Colors.Text).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(is.theme.Colors.Border)
	
	// Apply interactive effects
	switch state {
	case StateFocus:
		baseStyle = is.applyInputFocusEffect(baseStyle, progress)
	case StateError:
		baseStyle = is.applyInputErrorEffect(baseStyle, progress)
	case StateSuccess:
		baseStyle = is.applyInputSuccessEffect(baseStyle)
	}
	
	// Handle content with cursor animation for focus state
	content := value
	if content == "" && placeholder != "" {
		content = placeholder
		baseStyle = baseStyle.Foreground(is.theme.Colors.TextMuted)
	}
	
	// Add animated cursor for focus state
	if state == StateFocus && progress > 0 {
		cursorOpacity := is.calculateBlinkOpacity(progress)
		if cursorOpacity > 0.5 {
			content += "|"
		}
	}
	
	return baseStyle.Render(content)
}

// Animated Chat Bubble with typing indicator and slide-in effect
func (is *InteractiveStyler) AnimatedChatBubble(content, author string, bubbleType ChatBubbleType, timestamp string, animationType AnimationType, progress float64) string {
	// Get base chat bubble
	bubble := is.getBaseChatBubble(content, author, bubbleType, timestamp)
	
	// Apply animation effects
	switch animationType {
	case AnimationFadeIn:
		return is.applyFadeIn(bubble, progress)
	case AnimationSlideIn:
		return is.applySlideIn(bubble, progress, bubbleType == ChatBubbleUser)
	case AnimationBounce:
		return is.applyBounce(bubble, progress)
	default:
		return bubble
	}
}

// Animated List with selection transitions
func (is *InteractiveStyler) AnimatedList(items []string, listType ListType, selectedIndex, previousIndex int, progress float64) string {
	if len(items) == 0 {
		return ""
	}
	
	var styledItems []string
	
	for i, item := range items {
		state := ListItemStateNormal
		itemProgress := float64(0)
		
		if i == selectedIndex {
			state = ListItemStateSelected
			itemProgress = progress
		} else if i == previousIndex && progress < 1.0 {
			// Animate deselection
			itemProgress = 1.0 - progress
		}
		
		styledItem := is.AnimatedListItem(item, listType, state, i, itemProgress)
		styledItems = append(styledItems, styledItem)
	}
	
	return strings.Join(styledItems, "\n")
}

// Animated List Item with smooth selection transitions
func (is *InteractiveStyler) AnimatedListItem(text string, listType ListType, state ListItemState, index int, progress float64) string {
	baseItem := is.getBaseListItem(text, listType, state, index)
	
	if state == ListItemStateSelected && progress > 0 {
		return is.applySelectionAnimation(baseItem, progress)
	}
	
	return baseItem
}

// Progress Bar with smooth value transitions
func (is *InteractiveStyler) AnimatedProgressBar(targetProgress, currentProgress float64, width int, showPercentage bool, animationType AnimationType) string {
	if width <= 0 {
		width = 40
	}
	
	// Use currentProgress for smooth animation
	progress := currentProgress
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}
	
	filled := int(float64(width) * progress)
	empty := width - filled
	
	// Create animated fill based on animation type
	var filledBar string
	switch animationType {
	case AnimationPulse:
		filledBar = is.createPulsingBar(filled, progress)
	case AnimationGlow:
		filledBar = is.createGlowingBar(filled, progress)
	default:
		filledBar = strings.Repeat("█", filled)
	}
	
	emptyBar := strings.Repeat("░", empty)
	
	bar := lipgloss.NewStyle().
		Foreground(is.theme.Colors.Primary).
		Render(filledBar) +
		lipgloss.NewStyle().
			Foreground(is.theme.Colors.BorderSubtle).
			Render(emptyBar)
	
	if showPercentage {
		percentage := fmt.Sprintf(" %.0f%%", progress*100)
		bar += lipgloss.NewStyle().
			Foreground(is.theme.Colors.TextMuted).
			Render(percentage)
	}
	
	return bar
}

// Animated Status Indicator with pulsing and color transitions
func (is *InteractiveStyler) AnimatedStatusIndicator(status, text string, progress float64) string {
	color := is.getStatusColor(status)
	symbol := is.getStatusSymbol(status, progress)
	
	indicator := lipgloss.NewStyle().
		Foreground(color).
		Render(symbol)
	
	if text != "" {
		indicator += " " + lipgloss.NewStyle().
			Foreground(is.theme.Colors.Text).
			Render(text)
	}
	
	return indicator
}

// Animated Spinner with different patterns
func (is *InteractiveStyler) AnimatedSpinner(progress float64) string {
	// Create different spinner frames
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	
	// Calculate current frame based on progress
	frameIndex := int(progress*float64(len(frames))) % len(frames)
	
	return lipgloss.NewStyle().
		Foreground(is.theme.Colors.Primary).
		Render(frames[frameIndex])
}

// Loading Dots with wave animation
func (is *InteractiveStyler) AnimatedLoadingDots(progress float64) string {
	dots := []string{".", "..", "..."}
	dotIndex := int(progress*float64(len(dots))) % len(dots)
	
	return lipgloss.NewStyle().
		Foreground(is.theme.Colors.Primary).
		Render(dots[dotIndex])
}

// Notification with slide-in and fade effects
func (is *InteractiveStyler) AnimatedNotification(title, message string, notificationType ButtonStyle, animationType AnimationType, progress float64) string {
	// Create base notification
	notification := is.createBaseNotification(title, message, notificationType)
	
	// Apply animation
	switch animationType {
	case AnimationSlideIn:
		return is.applySlideIn(notification, progress, false)
	case AnimationFadeIn:
		return is.applyFadeIn(notification, progress)
	case AnimationBounce:
		return is.applyBounce(notification, progress)
	default:
		return notification
	}
}

// Modal with backdrop fade and content scale
func (is *InteractiveStyler) AnimatedModal(content string, animationType AnimationType, progress float64) string {
	// Create backdrop
	backdrop := is.createModalBackdrop(progress)
	
	// Create modal content with animation
	var animatedContent string
	switch animationType {
	case AnimationFadeIn:
		animatedContent = is.applyFadeIn(content, progress)
	case AnimationScale:
		animatedContent = is.applyScale(content, progress)
	case AnimationSlideIn:
		animatedContent = is.applySlideIn(content, progress, false)
	default:
		animatedContent = content
	}
	
	// Center the modal
	modalStyle := lipgloss.NewStyle().
		Width(is.width).
		Height(is.height).
		Align(lipgloss.Center, lipgloss.Center)
	
	return modalStyle.Render(backdrop + "\n" + animatedContent)
}

// Helper methods for creating base components

func (is *InteractiveStyler) getBaseButtonStyle(variant ButtonStyle, size ButtonSize) lipgloss.Style {
	// Use the ComponentStyler to get base button style
	cs := NewComponentStyler(is.theme, is.width, is.height)
	_ = cs // Placeholder to avoid unused variable error
	return lipgloss.NewStyle() // Simplified - would integrate with ComponentStyler
}

func (is *InteractiveStyler) getBaseChatBubble(content, author string, bubbleType ChatBubbleType, timestamp string) string {
	cs := NewComponentStyler(is.theme, is.width, is.height)
	return cs.ChatBubble(content, author, bubbleType, timestamp)
}

func (is *InteractiveStyler) getBaseListItem(text string, listType ListType, state ListItemState, index int) string {
	cs := NewComponentStyler(is.theme, is.width, is.height)
	return cs.ListItem(text, listType, state, index)
}

// Animation effect methods

func (is *InteractiveStyler) applyHoverEffect(style lipgloss.Style, variant ButtonStyle, progress float64) lipgloss.Style {
	// Interpolate to hover color
	if variant == ButtonPrimary {
		// Gradually transition to darker color
		return style.Background(is.interpolateColor(is.theme.Colors.Primary, is.theme.Colors.PrimaryDark, progress))
	}
	return style
}

func (is *InteractiveStyler) applyFocusEffect(style lipgloss.Style, variant ButtonStyle, progress float64) lipgloss.Style {
	// Add animated border
	borderColor := is.interpolateColor(is.theme.Colors.Border, is.theme.Colors.Primary, progress)
	return style.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor)
}

func (is *InteractiveStyler) applyActiveEffect(style lipgloss.Style, variant ButtonStyle, progress float64) lipgloss.Style {
	// Scale down slightly for press effect
	scale := 1.0 - (progress * 0.05) // 5% scale down
	_ = scale // Placeholder to avoid unused variable error
	// In a real implementation, you'd apply scaling
	return style
}

func (is *InteractiveStyler) applySuccessEffect(style lipgloss.Style, progress float64) lipgloss.Style {
	bgColor := style.GetBackground()
	if color, ok := bgColor.(lipgloss.Color); ok {
		successColor := is.interpolateColor(color, is.theme.Colors.Success, progress)
		return style.Background(successColor)
	}
	return style.Background(is.theme.Colors.Success)
}

func (is *InteractiveStyler) applyErrorEffect(style lipgloss.Style, progress float64) lipgloss.Style {
	bgColor := style.GetBackground()
	if color, ok := bgColor.(lipgloss.Color); ok {
		errorColor := is.interpolateColor(color, is.theme.Colors.Error, progress)
		return style.Background(errorColor)
	}
	return style.Background(is.theme.Colors.Error)
}

func (is *InteractiveStyler) applyInputFocusEffect(style lipgloss.Style, progress float64) lipgloss.Style {
	borderColor := is.interpolateColor(is.theme.Colors.Border, is.theme.Colors.Primary, progress)
	return style.
		BorderForeground(borderColor).
		Border(lipgloss.NormalBorder(), true)
}

func (is *InteractiveStyler) applyInputErrorEffect(style lipgloss.Style, progress float64) lipgloss.Style {
	// Shake effect for errors
	if progress > 0 {
		// In a real implementation, you'd apply actual shaking
		return style.
			BorderForeground(is.theme.Colors.Error).
			Background(is.theme.Colors.BackgroundSubtle)
	}
	return style
}

func (is *InteractiveStyler) applyInputSuccessEffect(style lipgloss.Style) lipgloss.Style {
	return style.BorderForeground(is.theme.Colors.Success)
}

func (is *InteractiveStyler) applyFadeIn(content string, progress float64) string {
	if progress <= 0 {
		return ""
	} else if progress >= 1 {
		return content
	}
	
	// Simulate fade by adjusting opacity (simplified for terminal)
	opacity := int(progress * 100)
	if opacity < 30 {
		return "" // Too transparent to show
	}
	
	return content
}

func (is *InteractiveStyler) applySlideIn(content string, progress float64, fromRight bool) string {
	if progress <= 0 {
		return ""
	} else if progress >= 1 {
		return content
	}
	
	// Calculate slide distance
	slideDistance := int(float64(is.width) * (1.0 - progress) * 0.3)
	
	var padding string
	if fromRight {
		padding = strings.Repeat(" ", slideDistance)
	} else {
		// For left slide, we'd need to clip content
	}
	
	return padding + content
}

func (is *InteractiveStyler) applyBounce(content string, progress float64) string {
	// Simple bounce effect using sine wave
	bounceOffset := int(math.Sin(progress*math.Pi*4) * is.animator.config.BounceAmplitude * 10)
	
	if bounceOffset > 0 {
		padding := strings.Repeat(" ", bounceOffset)
		return padding + content
	}
	
	return content
}

func (is *InteractiveStyler) applyScale(content string, progress float64) string {
	// Simplified scaling for terminal - would need more sophisticated implementation
	if progress < 0.5 {
		// Scale up from small
		return content
	}
	return content
}

func (is *InteractiveStyler) applySelectionAnimation(content string, progress float64) string {
	// Highlight selection with animated background
	if progress > 0 {
		style := lipgloss.NewStyle().
			Background(is.theme.Colors.Selection).
			Foreground(is.theme.Colors.Primary)
		return style.Render(content)
	}
	return content
}

func (is *InteractiveStyler) createPulsingBar(length int, progress float64) string {
	// Create pulsing effect with sine wave
	// In a real implementation, you'd adjust color intensity based on:
	// intensity := math.Sin(progress*math.Pi*2) * 0.3 + 0.7
	return strings.Repeat("█", length)
}

func (is *InteractiveStyler) createGlowingBar(length int, progress float64) string {
	// Create glowing effect
	return strings.Repeat("▓", length)
}

func (is *InteractiveStyler) createBaseNotification(title, message string, notificationType ButtonStyle) string {
	cs := NewComponentStyler(is.theme, is.width, is.height)
	
	var content strings.Builder
	if title != "" {
		titleStyle := lipgloss.NewStyle().
			Foreground(is.theme.GetStateColor(is.getNotificationState(notificationType))).
			Bold(true)
		content.WriteString(titleStyle.Render(title))
		content.WriteString("\n")
	}
	
	if message != "" {
		content.WriteString(message)
	}
	
	return cs.Card("", content.String(), CardTypeElevated, 0)
}

func (is *InteractiveStyler) createModalBackdrop(progress float64) string {
	// Create semi-transparent backdrop
	if progress <= 0 {
		return ""
	}
	
	// In a real implementation, you'd create an actual backdrop overlay
	return ""
}

// Utility methods

func (is *InteractiveStyler) calculateBlinkOpacity(progress float64) float64 {
	// Create blinking cursor effect
	return (math.Sin(progress*math.Pi*2) + 1.0) / 2.0
}

func (is *InteractiveStyler) interpolateColor(from, to lipgloss.Color, progress float64) lipgloss.Color {
	// Simplified color interpolation
	// In a real implementation, you'd convert to RGB, interpolate, and convert back
	if progress <= 0 {
		return from
	} else if progress >= 1 {
		return to
	}
	
	return to // Simplified
}

func (is *InteractiveStyler) getStatusColor(status string) lipgloss.Color {
	return is.theme.GetStateColor(status)
}

func (is *InteractiveStyler) getStatusSymbol(status string, progress float64) string {
	baseSymbol := "●"
	
	// Animate based on status
	switch strings.ToLower(status) {
	case "loading":
		// Rotate through different symbols
		symbols := []string{"◐", "◓", "◑", "◒"}
		index := int(progress*float64(len(symbols))) % len(symbols)
		return symbols[index]
	case "connecting":
		// Pulsing dots
		intensity := int(progress*3) % 4
		return strings.Repeat(".", intensity+1)
	default:
		return baseSymbol
	}
}

func (is *InteractiveStyler) getNotificationState(notificationType ButtonStyle) string {
	switch notificationType {
	case ButtonSuccess:
		return "success"
	case ButtonError:
		return "error"
	case ButtonWarning:
		return "warning"
	case ButtonInfo:
		return "info"
	default:
		return "primary"
	}
}

// Animation management methods

// StartTransition starts a new animated transition
func (am *AnimationManager) StartTransition(id string, from, to interface{}, duration time.Duration, onComplete func()) {
	transition := &Transition{
		From:       from,
		To:         to,
		Current:    from,
		Progress:   0,
		Duration:   duration,
		StartTime:  time.Now(),
		EasingFunc: am.getEasingFunction(am.config.EasingFunction),
		OnComplete: onComplete,
		IsActive:   true,
	}
	
	am.transitions[id] = transition
}

// UpdateTransitions updates all active transitions
func (am *AnimationManager) UpdateTransitions() {
	now := time.Now()
	
	for id, transition := range am.transitions {
		if !transition.IsActive {
			continue
		}
		
		elapsed := now.Sub(transition.StartTime)
		progress := float64(elapsed) / float64(transition.Duration)
		
		if progress >= 1.0 {
			// Transition complete
			transition.Progress = 1.0
			transition.Current = transition.To
			transition.IsActive = false
			
			if transition.OnComplete != nil {
				transition.OnComplete()
			}
		} else {
			// Update progress with easing
			easedProgress := transition.EasingFunc(progress)
			transition.Progress = easedProgress
			
			// Interpolate current value
			// This would need type-specific interpolation logic
			transition.Current = am.interpolateValue(transition.From, transition.To, easedProgress)
		}
		
		if transition.OnUpdate != nil {
			transition.OnUpdate(transition.Current)
		}
		
		// Clean up completed transitions
		if !transition.IsActive {
			delete(am.transitions, id)
		}
	}
}

// GetTransitionProgress returns the current progress of a transition
func (am *AnimationManager) GetTransitionProgress(id string) float64 {
	if transition, exists := am.transitions[id]; exists {
		return transition.Progress
	}
	return 0
}

// Spring animation support
func (am *AnimationManager) CreateSpring(id string, tension, friction float64) {
	spring := harmonica.NewSpring(harmonica.FPS(60), tension, friction)
	am.springs[id] = &SpringState{
		spring:   spring,
		position: 0.0,
		velocity: 0.0,
	}
}

func (am *AnimationManager) UpdateSpring(id string, target float64) float64 {
	if springState, exists := am.springs[id]; exists {
		springState.position, springState.velocity = springState.spring.Update(
			springState.position,
			springState.velocity,
			target,
		)
		return springState.position
	}
	return target
}

// Easing functions
func (am *AnimationManager) getEasingFunction(name string) func(float64) float64 {
	switch name {
	case "ease-in":
		return am.easeIn
	case "ease-out":
		return am.easeOut
	case "ease-in-out":
		return am.easeInOut
	case "bounce":
		return am.bounce
	case "elastic":
		return am.elastic
	default:
		return am.linear
	}
}

func (am *AnimationManager) linear(t float64) float64 {
	return t
}

func (am *AnimationManager) easeIn(t float64) float64 {
	return t * t
}

func (am *AnimationManager) easeOut(t float64) float64 {
	return 1 - (1-t)*(1-t)
}

func (am *AnimationManager) easeInOut(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return 1 - 2*(1-t)*(1-t)
}

func (am *AnimationManager) bounce(t float64) float64 {
	if t < 1/2.75 {
		return 7.5625 * t * t
	} else if t < 2/2.75 {
		t -= 1.5 / 2.75
		return 7.5625*t*t + 0.75
	} else if t < 2.5/2.75 {
		t -= 2.25 / 2.75
		return 7.5625*t*t + 0.9375
	} else {
		t -= 2.625 / 2.75
		return 7.5625*t*t + 0.984375
	}
}

func (am *AnimationManager) elastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	
	p := 0.3
	s := p / 4
	return math.Pow(2, -10*t) * math.Sin((t-s)*(2*math.Pi)/p) + 1
}

// Interpolation helpers
func (am *AnimationManager) interpolateValue(from, to interface{}, progress float64) interface{} {
	// This would need type-specific logic for different value types
	// For now, return a simple interpolation placeholder
	return to
}