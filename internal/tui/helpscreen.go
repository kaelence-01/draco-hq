package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// helpScreen is the help of draco or, with a name, the same text as
// draco help <name>, which the command mode hands over.
type helpScreen struct {
	name string
	text string
	err  error
	vp   viewport.Model
}

func newHelpScreen(st *state, name string) *helpScreen {
	h := &helpScreen{name: name, vp: viewport.New()}
	h.vp.SoftWrap = true
	if name == "" {
		h.text = generalHelp(st)
	} else if st.deps.Help != nil {
		h.text, h.err = st.deps.Help(name)
	}
	h.vp.SetContent(h.text)
	return h
}

func generalHelp(st *state) string {
	var cmds []string
	width := 0
	for _, c := range barCommands(st.deps) {
		width = max(width, len(strings.Join(c.Names, ", ")))
	}
	for _, c := range barCommands(st.deps) {
		names := strings.Join(c.Names, ", ")
		cmds = append(cmds, "  "+names+strings.Repeat(" ", width-len(names)+2)+c.Summary)
	}
	return strings.Join([]string{
		"draco documenta despliegues desde la terminal: el documento de despliegue de una tarea, el consolidado de QA, exportar documentos y la deuda técnica de tus repos. El modo visual y el modo comando recorren los mismos pasos.",
		"",
		"Teclas en todas las pantallas",
		"  ctrl+c  sale de draco",
		"  esc     vuelve a la pantalla anterior",
		"  ?       muestra todas las teclas de la pantalla",
		"  q       sale, desde la principal",
		"",
		"Barra de comandos",
		"  / en la principal la abre. Escribe un comando como en la terminal, con draco o sin él: /g -i OPS-7 -b a1b2c3d. tab completa la sugerencia, ↑↓ la elige, enter ejecuta y esc cierra. /help <comando> muestra la ayuda de un comando.",
		"",
		"Comandos",
		strings.Join(cmds, "\n"),
	}, "\n")
}

func (h *helpScreen) title() string {
	if h.name == "" {
		return "Ayuda"
	}
	return "Ayuda de " + h.name
}

func (h *helpScreen) keys() []key.Binding { return []key.Binding{scrollKey} }

func (h *helpScreen) update(msg tea.Msg) (screen, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		var cmd tea.Cmd
		h.vp, cmd = h.vp.Update(k)
		return h, cmd
	}
	return h, nil
}

func (h *helpScreen) view(width, height int) string {
	if h.err != nil {
		return pad(lipgloss.NewStyle().Width(width - 2).Render(errStyle.Render("✗ " + h.err.Error())))
	}
	h.vp.SetWidth(max(width-2, 10))
	h.vp.SetHeight(max(height-2, 3))
	return pad(h.vp.View())
}
