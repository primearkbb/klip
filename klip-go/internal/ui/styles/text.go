package styles

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// TextFormatter handles advanced text formatting, syntax highlighting, and markdown rendering
type TextFormatter struct {
	theme       *Theme
	width       int
	height      int
	glamour     *glamour.TermRenderer
	chromaStyle *chroma.Style
	capabilities *TerminalCapabilities
}

// TextStyle represents different text formatting styles
type TextStyle int

const (
	TextStylePlain TextStyle = iota
	TextStyleBold
	TextStyleItalic
	TextStyleUnderline
	TextStyleStrikethrough
	TextStyleCode
	TextStyleCodeBlock
	TextStyleQuote
	TextStyleLink
	TextStyleHeading1
	TextStyleHeading2
	TextStyleHeading3
	TextStyleEmphasis
	TextStyleStrong
	TextStyleHighlight
)

// CodeLanguage represents supported programming languages for syntax highlighting
type CodeLanguage string

const (
	LangGo         CodeLanguage = "go"
	LangJavaScript CodeLanguage = "javascript"
	LangTypeScript CodeLanguage = "typescript"
	LangPython     CodeLanguage = "python"
	LangJava       CodeLanguage = "java"
	LangC          CodeLanguage = "c"
	LangCpp        CodeLanguage = "cpp"
	LangRust       CodeLanguage = "rust"
	LangJSON       CodeLanguage = "json"
	LangYAML       CodeLanguage = "yaml"
	LangMarkdown   CodeLanguage = "markdown"
	LangHTML       CodeLanguage = "html"
	LangCSS        CodeLanguage = "css"
	LangSQL        CodeLanguage = "sql"
	LangShell      CodeLanguage = "bash"
	LangPlainText  CodeLanguage = "text"
)

// TextListType represents different text list formatting styles
type TextListType int

const (
	TextListTypeBulleted TextListType = iota
	TextListTypeNumbered
	TextListTypeChecklist
)

// NewTextFormatter creates a new text formatter with the given theme and dimensions
func NewTextFormatter(theme *Theme, width, height int, capabilities *TerminalCapabilities) *TextFormatter {
	tf := &TextFormatter{
		theme:        theme,
		width:        width,
		height:       height,
		capabilities: capabilities,
	}
	
	// Initialize Glamour renderer
	tf.initGlamourRenderer()
	
	// Initialize Chroma style for syntax highlighting
	tf.initChromaStyle()
	
	return tf
}

// initGlamourRenderer initializes the Glamour markdown renderer
func (tf *TextFormatter) initGlamourRenderer() {
	// Use appropriate style based on theme darkness
	if !tf.theme.IsDark {
		// Light theme styling would be applied here
	}
	
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte("{}")), // Use default styles
		glamour.WithWordWrap(tf.width-4),
	)
	
	if err != nil {
		// Fallback to simple renderer
		renderer, _ = glamour.NewTermRenderer(
			glamour.WithWordWrap(tf.width-4),
		)
	}
	
	tf.glamour = renderer
}

// initChromaStyle initializes the Chroma syntax highlighting style
func (tf *TextFormatter) initChromaStyle() {
	if tf.theme.IsDark {
		tf.chromaStyle = styles.Get("monokai")
		if tf.chromaStyle == nil {
			tf.chromaStyle = styles.Get("monokai")
		}
	} else {
		tf.chromaStyle = styles.Get("github")
		if tf.chromaStyle == nil {
			tf.chromaStyle = styles.Get("colorful")
		}
	}
	
	// Fallback to default if none found
	if tf.chromaStyle == nil {
		tf.chromaStyle = styles.Fallback
	}
}

// Markdown Rendering

// RenderMarkdown renders markdown text with full formatting
func (tf *TextFormatter) RenderMarkdown(markdown string) (string, error) {
	if tf.glamour == nil {
		return markdown, fmt.Errorf("glamour renderer not initialized")
	}
	
	// Note: Width is set during renderer initialization
	// The glamour renderer width cannot be changed after creation
	
	rendered, err := tf.glamour.Render(markdown)
	if err != nil {
		return markdown, err // Return original on error
	}
	
	return strings.TrimSpace(rendered), nil
}

// RenderMarkdownInline renders markdown text without block formatting
func (tf *TextFormatter) RenderMarkdownInline(markdown string) string {
	// Simple inline markdown processing
	text := markdown
	
	// Bold
	boldRegex := regexp.MustCompile(`\*\*(.*?)\*\*`)
	text = boldRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := boldRegex.FindStringSubmatch(match)[1]
		return tf.ApplyTextStyle(content, TextStyleBold)
	})
	
	// Italic
	italicRegex := regexp.MustCompile(`\*(.*?)\*`)
	text = italicRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := italicRegex.FindStringSubmatch(match)[1]
		return tf.ApplyTextStyle(content, TextStyleItalic)
	})
	
	// Code
	codeRegex := regexp.MustCompile("`([^`]+)`")
	text = codeRegex.ReplaceAllStringFunc(text, func(match string) string {
		content := codeRegex.FindStringSubmatch(match)[1]
		return tf.ApplyTextStyle(content, TextStyleCode)
	})
	
	return text
}

// Syntax Highlighting

// HighlightCode applies syntax highlighting to code
func (tf *TextFormatter) HighlightCode(code string, language CodeLanguage) (string, error) {
	lexer := lexers.Get(string(language))
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	
	// Create terminal formatter
	formatter := formatters.Get("terminal")
	if formatter == nil {
		return code, fmt.Errorf("terminal formatter not available")
	}
	
	// Note: Terminal formatter options are handled differently in newer chroma versions
	// The formatter configuration is handled internally by the terminal formatter
	
	// Tokenize code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, err
	}
	
	// Format with syntax highlighting
	var highlighted strings.Builder
	err = formatter.Format(&highlighted, tf.chromaStyle, iterator)
	if err != nil {
		return code, err
	}
	
	return highlighted.String(), nil
}

// HighlightCodeBlock creates a styled code block with syntax highlighting
func (tf *TextFormatter) HighlightCodeBlock(code string, language CodeLanguage, showLineNumbers bool) string {
	// Apply syntax highlighting
	highlighted, err := tf.HighlightCode(code, language)
	if err != nil {
		highlighted = code // Fallback to original
	}
	
	lines := strings.Split(highlighted, "\n")
	
	// Add line numbers if requested
	if showLineNumbers {
		lineNumStyle := lipgloss.NewStyle().Foreground(tf.theme.Colors.TextMuted)
		for i, line := range lines {
			lineNum := fmt.Sprintf("%3d ", i+1)
			lineNumStyled := lineNumStyle.
				Render(lineNum)
			lines[i] = lineNumStyled + line
		}
	}
	
	content := strings.Join(lines, "\n")
	
	// Create code block style
	codeBlockStyle := lipgloss.NewStyle().
		Foreground(tf.theme.Colors.CodeForeground).
		Background(tf.theme.Colors.CodeBackground).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tf.theme.Colors.Border)
	
	// Add language label if specified
	if language != LangPlainText && language != "" {
		langStyle := lipgloss.NewStyle().
			Foreground(tf.theme.Colors.TextMuted).
			Italic(true)
		langLabel := langStyle.Render(fmt.Sprintf("# %s", string(language)))
		
		return langLabel + "\n" + codeBlockStyle.Render(content)
	}
	
	return codeBlockStyle.Render(content)
}

// Text Styling

// ApplyTextStyle applies the specified text style
func (tf *TextFormatter) ApplyTextStyle(text string, style TextStyle) string {
	switch style {
	case TextStyleBold:
		return lipgloss.NewStyle().Bold(true).Render(text)
		
	case TextStyleItalic:
		if tf.capabilities != nil && tf.capabilities.SupportsItalic {
			return lipgloss.NewStyle().Italic(true).Render(text)
		}
		return text // Fallback if italic not supported
		
	case TextStyleUnderline:
		if tf.capabilities != nil && tf.capabilities.SupportsUnderline {
			return lipgloss.NewStyle().Underline(true).Render(text)
		}
		return text
		
	case TextStyleStrikethrough:
		if tf.capabilities != nil && tf.capabilities.SupportsStrike {
			return lipgloss.NewStyle().Strikethrough(true).Render(text)
		}
		return text
		
	case TextStyleCode:
		return lipgloss.NewStyle().
			Foreground(tf.theme.Colors.CodeForeground).
			Background(tf.theme.Colors.CodeBackground).
			Padding(0, 1).
			Render(text)
		
	case TextStyleQuote:
		return lipgloss.NewStyle().
			Foreground(tf.theme.Colors.TextMuted).
			Italic(true).
			Render("❝ " + text + " ❞")
		
	case TextStyleLink:
		style := lipgloss.NewStyle().
			Foreground(tf.theme.Colors.Primary).
			Bold(true)
		if tf.capabilities != nil && tf.capabilities.SupportsUnderline {
			style = style.Underline(true)
		}
		return style.Render(text)
		
	case TextStyleHeading1:
		return lipgloss.NewStyle().
			Foreground(tf.theme.Colors.Primary).
			Bold(true).
			Render("# " + text)
		
	case TextStyleHeading2:
		return lipgloss.NewStyle().
			Foreground(tf.theme.Colors.Primary).
			Bold(true).
			Render("## " + text)
		
	case TextStyleHeading3:
		return lipgloss.NewStyle().
			Foreground(tf.theme.Colors.Secondary).
			Bold(true).
			Render("### " + text)
		
	case TextStyleEmphasis:
		return lipgloss.NewStyle().
			Foreground(tf.theme.Colors.Accent).
			Italic(true).
			Render(text)
		
	case TextStyleStrong:
		return lipgloss.NewStyle().
			Foreground(tf.theme.Colors.Primary).
			Bold(true).
			Render(text)
		
	case TextStyleHighlight:
		return lipgloss.NewStyle().
			Foreground(tf.theme.Colors.Background).
			Background(tf.theme.Colors.Accent).
			Render(text)
		
	default:
		return text
	}
}

// Advanced Text Features

// WrapText wraps text to specified width
func (tf *TextFormatter) WrapText(text string, width int) string {
	if width <= 0 {
		width = tf.width
	}
	
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	
	var lines []string
	var currentLine strings.Builder
	
	for _, word := range words {
		if currentLine.Len()+len(word)+1 > width && currentLine.Len() > 0 {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
		}
		
		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)
	}
	
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}
	
	return strings.Join(lines, "\n")
}

// TruncateText truncates text with ellipsis
func (tf *TextFormatter) TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	
	ellipsis := "..."
	if tf.capabilities != nil && tf.capabilities.SupportsUnicode {
		ellipsis = "…"
	}
	
	if maxLength <= len(ellipsis) {
		return ellipsis[:maxLength]
	}
	
	return text[:maxLength-len(ellipsis)] + ellipsis
}

// PadText pads text to specified width with alignment
func (tf *TextFormatter) PadText(text string, width int, align lipgloss.Position) string {
	return lipgloss.NewStyle().
		Width(width).
		Align(align).
		Render(text)
}

// CreateDivider creates a styled divider line
func (tf *TextFormatter) CreateDivider(width int, character string) string {
	if width <= 0 {
		width = tf.width
	}
	if character == "" {
		character = "─"
		if tf.capabilities != nil && !tf.capabilities.SupportsUnicode {
			character = "-"
		}
	}
	
	return lipgloss.NewStyle().
		Foreground(tf.theme.Colors.Border).
		Render(strings.Repeat(character, width))
}

// FormatList creates a formatted list with various styles
func (tf *TextFormatter) FormatList(items []string, listType TextListType) string {
	if len(items) == 0 {
		return ""
	}
	
	var formattedItems []string
	for i, item := range items {
		var prefix string
		switch listType {
		case TextListTypeNumbered:
			prefix = fmt.Sprintf("%d. ", i+1)
		case TextListTypeBulleted:
			prefix = "• "
			if tf.capabilities != nil && !tf.capabilities.SupportsUnicode {
				prefix = "* "
			}
		case TextListTypeChecklist:
			prefix = "☐ "
			if tf.capabilities != nil && !tf.capabilities.SupportsUnicode {
				prefix = "[ ] "
			}
		default:
			prefix = "- "
		}
		
		formattedItems = append(formattedItems, prefix+item)
	}
	
	return strings.Join(formattedItems, "\n")
}

// DetectLanguage attempts to detect programming language from code
func (tf *TextFormatter) DetectLanguage(code string) CodeLanguage {
	code = strings.ToLower(strings.TrimSpace(code))
	
	// Check for common patterns
	switch {
	case strings.Contains(code, "package main") || strings.Contains(code, "func main()"):
		return LangGo
	case strings.Contains(code, "function ") || strings.Contains(code, "const ") || strings.Contains(code, "let "):
		return LangJavaScript
	case strings.Contains(code, "interface ") || strings.Contains(code, "type "):
		return LangTypeScript
	case strings.Contains(code, "def ") || strings.Contains(code, "import ") || strings.Contains(code, "from "):
		return LangPython
	case strings.Contains(code, "public class") || strings.Contains(code, "public static void"):
		return LangJava
	case strings.Contains(code, "#include") || strings.Contains(code, "int main"):
		return LangC
	case strings.Contains(code, "fn main") || strings.Contains(code, "let mut"):
		return LangRust
	case strings.Contains(code, "{") && strings.Contains(code, "}") && (strings.Contains(code, "\"") || strings.Contains(code, "'")):
		return LangJSON
	case strings.Contains(code, "<!doctype") || strings.Contains(code, "<html"):
		return LangHTML
	case strings.Contains(code, "select ") || strings.Contains(code, "from ") || strings.Contains(code, "where "):
		return LangSQL
	case strings.Contains(code, "#!/bin/bash") || strings.Contains(code, "echo "):
		return LangShell
	default:
		return LangPlainText
	}
}

// Resize updates the formatter dimensions
func (tf *TextFormatter) Resize(width, height int) {
	tf.width = width
	tf.height = height
	
	// Reinitialize glamour with new width
	tf.initGlamourRenderer()
}