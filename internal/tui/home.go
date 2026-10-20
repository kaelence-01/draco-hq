package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// menuItem is an entry of the home menu. Without open, the function is not in
// the visual mode yet and the item points to its command.
type menuItem struct {
	label   string
	command string
	open    func(*state) screen
}

var (
	upKey    = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "subir"))
	downKey  = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "bajar"))
	enterKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "abrir"))
	exitKey  = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "salir"))
)

type home struct {
	st     *state
	items  []menuItem
	cursor int
	note   string
}

func newHome(st *state) *home {
	return &home{st: st, items: []menuItem{
		{label: "Generar documento de despliegue", command: "draco g"},
		{label: "Proyectos", command: "draco project"},
		{label: "Consolidado QA", command: "draco qa"},
		{label: "Exportar página", command: "draco f"},
		{label: "Backlog técnico", command: "draco backlog scan"},
		{label: "Configuración", command: "draco init"},
	}}
}

func (h *home) keys() []key.Binding { return []key.Binding{upKey, downKey, enterKey, exitKey} }

func (h *home) update(msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return h, nil
	}
	switch {
	case key.Matches(k, exitKey):
		return h, tea.Quit
	case key.Matches(k, upKey):
		h.cursor = (h.cursor + len(h.items) - 1) % len(h.items)
		h.note = ""
	case key.Matches(k, downKey):
		h.cursor = (h.cursor + 1) % len(h.items)
		h.note = ""
	case key.Matches(k, enterKey):
		it := h.items[h.cursor]
		if it.open == nil {
			h.note = "«" + it.label + "» todavía no está en el modo visual: usa " + it.command
			return h, nil
		}
		return h, open(it.open(h.st))
	}
	return h, nil
}

func (h *home) view(width, height int) string {
	labelWidth := 0
	for _, it := range h.items {
		labelWidth = max(labelWidth, lipgloss.Width(it.label))
	}
	var menu []string
	for i, it := range h.items {
		label := it.label + strings.Repeat(" ", labelWidth-lipgloss.Width(it.label))
		cursor := "  "
		if i == h.cursor {
			cursor = accentStyle.Render("▸ ")
		}
		switch {
		case it.open == nil:
			label = faintStyle.Render(label + "   " + it.command)
		case i == h.cursor:
			label = accentStyle.Bold(true).Render(label)
		}
		menu = append(menu, cursor+label)
	}

	blocks := []string{logo(width), faintStyle.Render("Documenta despliegues desde la terminal"), ""}
	if h.st.notice != "" {
		blocks = append(blocks, warnStyle.Render("⚠ "+h.st.notice), "")
	}
	blocks = append(blocks, lipgloss.JoinVertical(lipgloss.Left, menu...))
	if h.note != "" {
		blocks = append(blocks, "", warnStyle.Render(h.note))
	}
	content := lipgloss.JoinVertical(lipgloss.Center, blocks...)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
