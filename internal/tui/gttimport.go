package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
)

var (
	importKey  = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "importar"))
	scratchKey = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "configurar desde cero"))
)

// gttImport offers gtt's config to a draco without one, as draco init does:
// importing saves it as is and goes on to verify the account.
type gttImport struct {
	parent *settings
	err    string
}

func (g *gttImport) title() string       { return "Importar de gtt" }
func (g *gttImport) keys() []key.Binding { return []key.Binding{importKey, scratchKey} }

func (g *gttImport) update(msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return g, nil
	}
	s := g.parent
	switch {
	case key.Matches(k, importKey):
		if err := s.st.save(func(c *config.Config) error { *c = *s.gtt; return nil }); err != nil {
			g.err = "No se pudo guardar: " + err.Error()
			return g, nil
		}
		s.done("Configuración importada de gtt", nil)
		return g, tea.Sequence(back, open(newAccountCheck(s)))
	case key.Matches(k, scratchKey):
		return g, tea.Sequence(back, open(newConnection(s)))
	}
	return g, nil
}

func (g *gttImport) view(width, height int) string {
	lines := []string{"Se encontró la configuración de gtt en " + g.parent.gttPath + ".", ""}
	for _, note := range config.ImportNotes(g.parent.gtt) {
		lines = append(lines, "  "+note)
	}
	lines = append(lines, "", "Pasa tal cual a draco, y después se verifica la cuenta. La de gtt no se modifica.")
	if g.err != "" {
		lines = append(lines, "", errStyle.Render("✗ "+g.err))
	}
	lines = append(lines, "", faintStyle.Render("enter la importa · n configura desde cero"))
	return pad(lipgloss.NewStyle().Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}
