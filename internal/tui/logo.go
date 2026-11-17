package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var logoLines = []string{
	" ⢄⡀",
	"⢀⣀⣻⣷⣤⣀",
	"⠉⠉⠙⠛⢿⣿⡿⣦⣄       │",
	" ⢀⣴⡶⠿⠿⣿⣦⣽⣷⣤⣤⡀   │   d r a c o",
	"⢰⣿⡏  ⠚⠛⠻⢯⣍⡙⠿⠃   │",
	"⠘⢿⣷⣄     ⠘⠁",
}

var logoWidth = lipgloss.Width(strings.Join(logoLines, "\n"))

// logo is the dragon head with the name, or the plain name when width is too narrow.
func logo(width int) string {
	if width < logoWidth+4 {
		return logoStyle.Render("draco")
	}
	return logoStyle.Render(strings.Join(logoLines, "\n"))
}
