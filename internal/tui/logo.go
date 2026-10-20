package tui

import "strings"

var logoLines = []string{
	"██████  ██████   █████   ██████  ██████ ",
	"██   ██ ██   ██ ██   ██ ██      ██    ██",
	"██   ██ ██████  ███████ ██      ██    ██",
	"██   ██ ██   ██ ██   ██ ██      ██    ██",
	"██████  ██   ██ ██   ██  ██████  ██████ ",
}

// logo is DRACO in large letters, or the plain name when width is too narrow.
func logo(width int) string {
	if width < len([]rune(logoLines[0]))+4 {
		return logoStyle.Render("draco")
	}
	return logoStyle.Render(strings.Join(logoLines, "\n"))
}
