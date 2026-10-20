package tui

import "charm.land/lipgloss/v2"

// Basic ANSI colors follow the terminal's own palette, light or dark.
var (
	accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Cyan)
	boldStyle   = lipgloss.NewStyle().Bold(true)
	faintStyle  = lipgloss.NewStyle().Faint(true)
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Yellow)
	logoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Cyan).Bold(true)
	ruleStyle   = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
)
