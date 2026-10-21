package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
)

var (
	tryThemeKey  = key.NewBinding(key.WithKeys("up", "down", "k", "j"), key.WithHelp("↑↓", "probar"))
	saveThemeKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "guardar"))
)

// themePicker shows each theme on the whole screen as the cursor moves;
// enter saves it, and leaving without saving puts the saved one back.
type themePicker struct {
	parent *settings
	cursor int
}

func newThemePicker(parent *settings) *themePicker {
	t := &themePicker{parent: parent}
	for i, th := range themes {
		if th.name == theme {
			t.cursor = i
		}
	}
	return t
}

func (t *themePicker) title() string       { return "Tema" }
func (t *themePicker) keys() []key.Binding { return []key.Binding{tryThemeKey, saveThemeKey} }

func (t *themePicker) close() { applyTheme(t.parent.st.cfg.Theme) }

func (t *themePicker) update(msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return t, nil
	}
	switch {
	case key.Matches(k, upKey):
		t.cursor = max(t.cursor-1, 0)
	case key.Matches(k, downKey):
		t.cursor = min(t.cursor+1, len(themes)-1)
	case key.Matches(k, saveThemeKey):
		name := themes[t.cursor].name
		if name == defaultTheme {
			name = ""
		}
		err := t.parent.st.save(func(c *config.Config) error { c.Theme = name; return nil })
		t.parent.done("Tema guardado: "+themes[t.cursor].name, err)
		return t, back
	}
	applyTheme(themes[t.cursor].name)
	return t, nil
}

func (t *themePicker) view(width, height int) string {
	lines := []string{faintStyle.Render("Los colores siguen la paleta de tu terminal. NO_COLOR quita el color en cualquier tema."), ""}
	for i, th := range themes {
		cursor, name := "  ", th.name
		if i == t.cursor {
			cursor, name = accentStyle.Render("▸ "), accentStyle.Bold(true).Render(name)
		}
		lines = append(lines, cursor+name+"  "+faintStyle.Render(th.desc))
	}
	sample := boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		boldStyle.Render("Así se ve"), "",
		accentStyle.Render("▸ ")+accentStyle.Bold(true).Render("Elegido")+"   "+faintStyle.Render("texto secundario"),
		okStyle.Render("✓ ")+"Guardado",
		warnStyle.Render("⚠ Aviso: algo para revisar"),
		errStyle.Render("✗ Error: qué pasó y qué hacer"),
	))
	lines = append(lines, "", sample)
	return pad(lipgloss.NewStyle().Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}
