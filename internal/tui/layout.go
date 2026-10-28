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

// columns splits width into two columns in the ratio a:b, with gap columns
// between them.
func columns(width, gap, a, b int) (left, right int) {
	usable := max(width-gap, 0)
	left = usable * a / (a + b)
	return left, usable - left
}

// stretch is a box of width columns that fills height lines, whatever its
// content; the caller keeps the content within height-2 lines.
func stretch(content string, width, height int) string {
	return boxStyle.Width(width).Height(max(height, 3)).Render(content)
}

// panel is a stretched box with content wrapped to it; what does not fit in
// the height is cut with an ellipsis.
func panel(content string, width, height int) string {
	lines := strings.Split(lipgloss.NewStyle().Width(max(width-4, 1)).Render(content), "\n")
	if room := height - 2; len(lines) > room {
		lines = append(lines[:max(room-1, 0)], faintStyle.Render("…"))
	}
	return stretch(strings.Join(lines, "\n"), width, height)
}

// panels lays a list and its detail side by side, in the ratio a:b, as two
// boxes as tall as height. Each side is given the room inside its box (width
// and lines) and returns its content.
func panels(width, height, a, b int, left, right func(width, lines int) string) string {
	leftWidth, rightWidth := columns(width-2, 2, a, b)
	boxes := lipgloss.JoinHorizontal(lipgloss.Top,
		panel(left(leftWidth-4, height-2), leftWidth, height), "  ",
		panel(right(rightWidth-4, height-2), rightWidth, height))
	return lipgloss.NewStyle().PaddingLeft(1).Render(boxes)
}

// stepper draws the steps of a flow: the ones done checked, the current one
// highlighted, the rest dim.
func stepper(names []string, current int) string {
	parts := make([]string, len(names))
	for i, n := range names {
		switch {
		case i < current:
			parts[i] = okStyle.Render("✓ " + n)
		case i == current:
			parts[i] = accentStyle.Bold(true).Render(n)
		default:
			parts[i] = faintStyle.Render(n)
		}
	}
	return strings.Join(parts, faintStyle.Render("  ›  "))
}

// titleLine is the first line of a step of a flow: its title and, from
// panelMinWidth columns on, the steps at the end of the line.
func titleLine(title string, steps []string, current, width int) string {
	if width < panelMinWidth {
		return title
	}
	return edges(width, title, stepper(steps, current))
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
