package pages

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Page represents a UI page in the application
type Page interface {
	tea.Model
	Name() string
	Description() string
}

// PageType defines the different pages available in the application
type PageType int

const (
	PageChat PageType = iota
	PageSettings
	PageModels
	PageHelp
)

// PageNames maps page types to their string names
var PageNames = map[PageType]string{
	PageChat:     "chat",
	PageSettings: "settings",
	PageModels:   "models",
	PageHelp:     "help",
}

// GetPageName returns the string name for a page type
func GetPageName(pt PageType) string {
	if name, ok := PageNames[pt]; ok {
		return name
	}
	return "unknown"
}
