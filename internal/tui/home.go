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
	desc    string
	example string // the same function in the command mode
	open    func(*state) screen
}

var (
	upKey    = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "subir"))
	downKey  = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "bajar"))
	enterKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "abrir"))
	exitKey  = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "salir"))
)

// panelMinWidth is the narrowest terminal that fits the menu and its panel.
const panelMinWidth = 96

type home struct {
	st     *state
	items  []menuItem
	cursor int
	note   string
}

func newHome(st *state) *home {
	return &home{st: st, items: []menuItem{
		{label: "Generar documento de despliegue", example: "draco g -i OPS-7 -b a1b2c3d", open: func(st *state) screen { return newGenerate(st) },
			desc: "Lee la tarea y los archivos de tus commits y crea o actualiza su documento de despliegue, conservando lo que el equipo escribió a mano."},
		{label: "Proyectos", example: "draco project list", open: func(st *state) screen { return newProjects(st) },
			desc: "Rutas de los repos, nombres en el VCS y space de cada proyecto, en formularios en vez de editar el YAML."},
		{label: "Consolidado QA", example: "draco qa -s 12 -m Pagos", open: func(st *state) screen { return newQAForm(st) },
			desc: "Evalúa las tareas del sprint o de la ventana Kanban y publica el consolidado de pruebas."},
		{label: "Exportar página", example: "draco f -i OPS-7", open: func(st *state) screen { return newExportForm(st) },
			desc: "Busca el documento de una tarea y lo guarda como texto."},
		{label: "Backlog técnico", example: "draco backlog scan", open: func(st *state) screen { return newBacklogForm(st) },
			desc: "Detecta deuda técnica en tus repos locales y la propone como tareas. Sin red."},
		{label: "Configuración", example: "draco init", open: func(st *state) screen { return newSettings(st) },
			desc: "Credenciales, perfiles (tracker y destino de los documentos) y el primer proyecto."},
	}}
}

func (h *home) title() string       { return "Inicio" }
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
			h.note = "«" + it.label + "» todavía no está en el modo visual: usa " + command(it.example)
			return h, nil
		}
		return h, open(it.open(h.st))
	}
	return h, nil
}

// command is the command of an example, without its flags.
func command(example string) string {
	f := strings.Fields(example)
	for i, w := range f {
		if strings.HasPrefix(w, "-") {
			return strings.Join(f[:i], " ")
		}
	}
	return example
}

func (h *home) view(width, height int) string {
	labelWidth := 0
	for _, it := range h.items {
		labelWidth = max(labelWidth, lipgloss.Width(it.label))
	}
	wide := width >= panelMinWidth
	var menu []string
	for i, it := range h.items {
		label := it.label + strings.Repeat(" ", labelWidth-lipgloss.Width(it.label))
		cursor := "  "
		if i == h.cursor {
			cursor = accentStyle.Render("▸ ")
		}
		switch {
		case it.open == nil && !wide:
			label = faintStyle.Render(label + "   " + command(it.example))
		case it.open == nil:
			label = faintStyle.Render(label)
		case i == h.cursor:
			label = accentStyle.Bold(true).Render(label)
		}
		menu = append(menu, cursor+label)
	}
	body := lipgloss.JoinVertical(lipgloss.Left, menu...)
	if wide {
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, "    ", h.panel(width-labelWidth-10))
	}

	blocks := []string{logo(width), faintStyle.Render("Documenta despliegues desde la terminal"), ""}
	if h.st.notice != "" {
		blocks = append(blocks, warnStyle.Render("⚠ "+h.st.notice), "")
	}
	blocks = append(blocks, body)
	if h.note != "" {
		blocks = append(blocks, "", warnStyle.Render(h.note))
	}
	content := lipgloss.JoinVertical(lipgloss.Center, blocks...)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// panel describes the selected function and shows its command.
func (h *home) panel(maxWidth int) string {
	it := h.items[h.cursor]
	w := min(maxWidth, 52)
	text := lipgloss.NewStyle().Width(w)
	lines := []string{accentStyle.Bold(true).Render(it.label), text.Render(it.desc), ""}
	if it.open == nil {
		lines = append(lines, warnStyle.Render("Todavía no está en el modo visual."))
	}
	lines = append(lines, faintStyle.Render("En modo comando, para scripts y CI:"), "$ "+it.example)
	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
