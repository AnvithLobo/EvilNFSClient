package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles for the UI
var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			Padding(0, 1)

	PromptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))

	// ls output colors
	DirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4B9BFF")).
			Bold(true)

	ExecStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	LinkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFF00"))

	FileStyle = lipgloss.NewStyle()

	// Help menu styles
	HelpTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			MarginBottom(1)

	HelpSectionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#04B575")).
				Bold(true).
				MarginTop(1).
				MarginBottom(1)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0"))

	HelpArgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFF00"))

	HelpOptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4B9BFF"))

	HelpExampleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00"))

	HelpExampleBgStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#1A1A1A")).
				Foreground(lipgloss.Color("#00FF00"))
)
