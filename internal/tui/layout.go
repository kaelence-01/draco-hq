package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const trailSep = " › "

// edges puts left at the start of a line and the first of rights that still
// fits at its end, with two spaces between them at least.
func edges(width int, left string, rights ...string) string {
	for _, right := range rights {
		if gap := width - lipgloss.Width(left) - lipgloss.Width(right); gap >= 2 {
			return left + strings.Repeat(" ", gap) + right
		}
	}
	return left
}

// hang puts text beside prefix, wrapping it under itself instead of under the
// prefix, in width columns.
func hang(prefix, text string, width int) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, prefix, lipgloss.NewStyle().Width(max(width-lipgloss.Width(prefix), 1)).Render(text))
}

// trail is the path of open screens, the current one highlighted. When it does
// not fit in width, the first screens give way to an ellipsis.
func trail(titles []string, width int) string {
	last := len(titles) - 1
	for from := 0; ; from++ {
		var parts []string
		if from > 0 {
			parts = append(parts, faintStyle.Render("…"))
		}
		for i := from; i <= last; i++ {
			style := faintStyle
			if i == last {
				style = boldStyle
			}
			parts = append(parts, style.Render(titles[i]))
		}
		out := strings.Join(parts, faintStyle.Render(trailSep))
		if lipgloss.Width(out) <= width || from == last {
			return out
		}
	}
}
