package tui

import "github.com/charmbracelet/lipgloss"

// Define styles used across the TUI
var (
	// StatusBarStyle is used for the bottom status bar showing provider/model info
	StatusBarStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)

	// HelpBarStyle is used for help text display
	HelpBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 1)

	// UserMsgStyle is used for "You:" label
	UserMsgStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("117"))

	// AssistantStyle is used for "Assistant:" label
	AssistantStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	// ErrorStyle is used for error messages
	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	// TitleStyle is used for titles and headers
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230"))

	// AccentStyle is used for active/highlighted elements
	AccentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)
)
