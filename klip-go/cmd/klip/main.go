package klip

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/john/klip/internal/app"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			MarginBottom(1)

	bannerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B5CF6")).
			MarginBottom(1)
)

const banner = `
██╗  ██╗██╗     ██╗██████╗ 
██║ ██╔╝██║     ██║██╔══██╗
█████╔╝ ██║     ██║██████╔╝
██╔═██╗ ██║     ██║██╔═══╝ 
██║  ██╗███████╗██║██║     
╚═╝  ╚═╝╚══════╝╚═╝╚═╝     
`

// Execute runs the main application
func Execute() {
	// Parse command line flags
	var showHelp bool
	var showVersion bool
	flag.BoolVar(&showHelp, "help", false, "Show help information")
	flag.BoolVar(&showHelp, "h", false, "Show help information")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information")
	flag.Parse()

	// Handle help flag
	if showHelp {
		showHelpText()
		return
	}

	// Handle version flag
	if showVersion {
		showVersionText()
		return
	}

	// Setup logging
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stderr)

	// Display banner
	fmt.Print(titleStyle.Render("Klip - Terminal AI Chat"))
	fmt.Print(bannerStyle.Render(banner))
	fmt.Println()

	// Initialize the application model
	model := app.New()

	// Create the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run the program
	if _, err := p.Run(); err != nil {
		log.Error("Error running application", "error", err)
		os.Exit(1)
	}
}

// showHelpText displays help information
func showHelpText() {
	fmt.Print(titleStyle.Render("Klip - Terminal AI Chat"))
	fmt.Print(bannerStyle.Render(banner))
	fmt.Println()
	fmt.Println("Usage: klip [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help     Show this help message")
	fmt.Println("  -v, --version  Show version information")
	fmt.Println()
	fmt.Println("Interactive Commands:")
	fmt.Println("  /help          Show help within the application")
	fmt.Println("  /models        Switch between AI models")
	fmt.Println("  /settings      Configure API keys and preferences")
	fmt.Println("  /history       View chat history")
	fmt.Println("  /clear         Clear current conversation")
	fmt.Println("  /quit          Exit the application")
	fmt.Println()
	fmt.Println("Getting Started:")
	fmt.Println("  1. Run 'klip' to start the application")
	fmt.Println("  2. Use '/settings' to configure your API keys")
	fmt.Println("  3. Start chatting with AI models!")
	fmt.Println()
}

// showVersionText displays version information
func showVersionText() {
	fmt.Print(titleStyle.Render("Klip - Terminal AI Chat"))
	fmt.Println()
	fmt.Println("Version: 1.0.0")
	fmt.Println("Build: Go version")
	fmt.Println("Repository: https://github.com/john/klip")
	fmt.Println()
}
