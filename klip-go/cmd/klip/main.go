package klip

import (
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
	// Setup logging
	log.SetLevel(log.DebugLevel)
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
