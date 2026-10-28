package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

var filterKey = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filtrar"))

// filter is the text a list is filtered by: / starts typing it, enter keeps
// it and esc clears it.
type filter struct {
	text   string
	typing bool
}

// captures keeps the keys while typing, and esc while a filter is applied.
func (f *filter) captures(k tea.KeyPressMsg) bool {
	return f.typing || f.text != "" && key.Matches(k, backKey)
}

// handle reports whether k was for the filter.
func (f *filter) handle(k tea.KeyPressMsg) bool {
	switch {
	case f.typing:
		f.typeKey(k)
	case key.Matches(k, filterKey):
		f.typing = true
	case f.text != "" && key.Matches(k, backKey):
		f.text = ""
	default:
		return false
	}
	return true
}

func (f *filter) typeKey(k tea.KeyPressMsg) {
	switch {
	case key.Matches(k, backKey):
		f.text, f.typing = "", false
	case k.Code == tea.KeyEnter:
		f.typing = false
	case k.Code == tea.KeyBackspace:
		if r := []rune(f.text); len(r) > 0 {
			f.text = string(r[:len(r)-1])
		}
	case k.Text != "":
		f.text += k.Text
	}
}

func (f *filter) matches(s string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(f.text))
}

// window is the first of rows rows of a list of n that keeps cursor in view.
func window(cursor, n, rows int) int {
	return min(max(cursor-rows/2, 0), max(n-rows, 0))
}

// line is the filter line of a list that always shows it: what is typed or
// applied, or what / does.
func (f *filter) line() string {
	if v := f.view(); v != "" {
		return v
	}
	return accentStyle.Render("/ ") + faintStyle.Render("filtrar")
}

// view is the filter line of the list; empty without a filter.
func (f *filter) view() string {
	switch {
	case f.typing:
		return "/ " + f.text + accentStyle.Render("█")
	case f.text != "":
		return faintStyle.Render("filtro: " + f.text + "  (esc lo quita)")
	}
	return ""
}
