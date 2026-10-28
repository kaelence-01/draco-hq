package tui

import (
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// menuItem is an entry of the home menu. Without open, the function is not in
// the visual mode yet and the item points to its command.
type menuItem struct {
	label   string
	summary string // one line under the label on wide terminals; empty: desc
	desc    string
	steps   []string // the screens the function goes through, for the panel
	example string   // the same function in the command mode
	open    func(*state) screen
}

var (
	upKey    = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "subir"))
	downKey  = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "bajar"))
	moveKey  = key.NewBinding(key.WithKeys("up", "k", "down", "j"), key.WithHelp("↑↓", "elegir"))
	enterKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "abrir"))
	exitKey  = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "salir"))
)

const (
	// panelMinWidth is the narrowest terminal that fits the menu and its panel.
	panelMinWidth = 96
	// contextMinWidth is where the active project and the config fit under the panel.
	contextMinWidth = 120
	menuWidth       = 58
	tagline         = "Documenta despliegues desde la terminal"
)

type home struct {
	st     *state
	items  []menuItem
	cursor int
	note   string
	bar    *cmdBar // nil while closed
}

func newHome(st *state) *home {
	return &home{st: st, items: []menuItem{
		{label: "Generar documento de despliegue", example: "draco g -i OPS-7 -b a1b2c3d", open: func(st *state) screen { return newGenerate(st) },
			summary: "Desde la tarea y tus commits, conservando lo escrito",
			steps:   []string{"Tarea", "Commits", "Vista previa", "Ubicación", "Publicar"},
			desc:    "Lee la tarea y los archivos de tus commits y crea o actualiza su documento de despliegue, conservando lo que el equipo escribió a mano."},
		{label: "Proyectos", example: "draco project list", open: func(st *state) screen { return newProjects(st) },
			summary: "Repos, VCS y space de cada proyecto",
			desc:    "Rutas de los repos, nombres en el VCS y space de cada proyecto, en formularios en vez de editar el YAML."},
		{label: "Consolidado QA", example: "draco qa -s 12 -m Pagos", open: func(st *state) screen { return newQAForm(st) },
			summary: "Evalúa el sprint o la ventana Kanban y lo publica",
			steps:   []string{"Sprint o Kanban", "Evaluar", "Vista previa", "Ubicación", "Publicar"},
			desc:    "Evalúa las tareas del sprint o de la ventana Kanban y publica el consolidado de pruebas."},
		{label: "Exportar página", example: "draco f -i OPS-7", open: func(st *state) screen { return newExportForm(st) },
			summary: "Guarda como texto el documento de una tarea",
			steps:   []string{"Tarea", "Documentos", "Guardar"},
			desc:    "Busca el documento de una tarea y lo guarda como texto."},
		{label: "Backlog técnico", example: "draco backlog scan", open: func(st *state) screen { return newBacklogForm(st) },
			summary: "Deuda técnica de tus repos locales, sin red",
			steps:   []string{"Repos", "Analizar", "Hallazgos"},
			desc:    "Detecta deuda técnica en tus repos locales y la propone como tareas. Sin red."},
		{label: "Configuración", example: "draco init", open: func(st *state) screen { return newSettings(st) },
			summary: "Conexión, proyectos, perfiles, QA y tema",
			desc:    "Conexión con Atlassian y proyectos, lo que necesita el documento. Opcionales: perfiles, QA y tema."},
	}}
}

func (h *home) title() string { return "Inicio" }

func (h *home) keys() []key.Binding {
	if h.bar != nil {
		return h.bar.keys()
	}
	keys := []key.Binding{moveKey, enterKey, h.numberKey()}
	if len(h.st.deps.Commands) > 0 {
		keys = append(keys, barKey)
	}
	return keys
}

func (h *home) moreKeys() []key.Binding {
	if h.bar != nil {
		return nil
	}
	return []key.Binding{upKey, downKey, exitKey}
}

// numberKey opens the function with that number in the menu.
func (h *home) numberKey() key.Binding {
	n := min(len(h.items), 9)
	keys := make([]string, n)
	for i := range keys {
		keys[i] = strconv.Itoa(i + 1)
	}
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp("1-"+strconv.Itoa(n), "abrir directo"))
}

// captures gives every key to the command bar while it is open.
func (h *home) captures(tea.KeyPressMsg) bool { return h.bar != nil }

func (h *home) update(msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return h, nil
	}
	if h.bar != nil {
		cmd, closed := h.bar.handle(k)
		if closed {
			h.bar = nil
		}
		return h, cmd
	}
	switch {
	case key.Matches(k, barKey) && len(h.st.deps.Commands) > 0:
		h.bar, h.note = &cmdBar{st: h.st}, ""
	case key.Matches(k, exitKey):
		return h, tea.Quit
	case key.Matches(k, upKey):
		h.cursor = (h.cursor + len(h.items) - 1) % len(h.items)
		h.note = ""
	case key.Matches(k, downKey):
		h.cursor = (h.cursor + 1) % len(h.items)
		h.note = ""
	case key.Matches(k, h.numberKey()):
		n, _ := strconv.Atoi(k.String())
		h.cursor, h.note = n-1, ""
		return h, h.openCursor()
	case key.Matches(k, enterKey):
		return h, h.openCursor()
	}
	return h, nil
}

// openCursor opens the selected function, or says where to find it.
func (h *home) openCursor() tea.Cmd {
	it := h.items[h.cursor]
	if it.open == nil {
		h.note = "«" + it.label + "» todavía no está en el modo visual: usa " + command(it.example)
		return nil
	}
	return open(it.open(h.st))
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
	if h.bar != nil {
		// The menu is not used while typing; without it, the bar fits in 24 lines.
		bar := h.bar.view(width)
		top := lipgloss.JoinVertical(lipgloss.Center, logo(width), faintStyle.Render(tagline))
		return lipgloss.JoinVertical(lipgloss.Left, lipgloss.Place(width, max(height-lipgloss.Height(bar), 0), lipgloss.Center, lipgloss.Center, top), bar)
	}
	var above, below []string
	if n := h.notices(width); n != "" {
		above = []string{n, ""}
	}
	if h.note != "" {
		below = []string{"", warnStyle.Width(width - 2).Render(h.note)}
	}
	room := height - blocksHeight(above) - blocksHeight(below)
	var main string
	if width >= panelMinWidth {
		main = h.wide(width, room)
	} else {
		main = h.narrow(width, room)
	}
	out := lipgloss.JoinVertical(lipgloss.Left, slices.Concat(above, []string{main}, below)...)
	return lipgloss.NewStyle().PaddingLeft(1).Render(out)
}

func blocksHeight(blocks []string) int {
	if len(blocks) == 0 {
		return 0
	}
	return lipgloss.Height(lipgloss.JoinVertical(lipgloss.Left, blocks...))
}

// notices are the problems with the config and a newer version, over the menu.
func (h *home) notices(width int) string {
	var lines []string
	if h.st.notice != "" {
		lines = append(lines, warnStyle.Render("⚠ "+h.st.notice))
	}
	if v := h.st.newVersion; v != "" {
		lines = append(lines, warnStyle.Render("Nueva versión "+v+" (tienes "+h.st.deps.Version+"): sal con q y corre draco update"))
	}
	if len(lines) == 0 {
		return ""
	}
	return lipgloss.NewStyle().Width(width - 2).Render(strings.Join(lines, "\n"))
}

type menuMode int

const (
	menuSpaced    menuMode = iota // a description under each option, with a blank line between them
	menuDescribed                 // a description under each option
	menuPlain                     // only the options
)

// wide is the menu on the left and, on the right, the panel of the selected
// function and, when there is room, the active project and the config. The
// menu gives up its descriptions before it runs out of lines.
func (h *home) wide(width, room int) string {
	head := []string{logo(width), faintStyle.Render(tagline), ""}
	var left string
	for _, mode := range []menuMode{menuSpaced, menuDescribed, menuPlain} {
		left = lipgloss.JoinVertical(lipgloss.Left, append(slices.Clone(head), h.menu(mode))...)
		if lipgloss.Height(left) <= room {
			break
		}
	}
	rightWidth := width - menuWidth - 5
	right := h.panel(rightWidth)
	if h.st.cfg != nil && width >= contextMinWidth {
		if ctx := h.context(rightWidth); lipgloss.Height(right)+1+lipgloss.Height(ctx) <= room {
			right = lipgloss.JoinVertical(lipgloss.Left, right, "", ctx)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "   ", right)
}

func (h *home) menu(mode menuMode) string {
	var rows []string
	for i, it := range h.items {
		cursor, label := "  ", it.label
		switch {
		case it.open == nil:
			label = faintStyle.Render(label)
		case i == h.cursor:
			label = accentStyle.Bold(true).Render(label)
		}
		if i == h.cursor {
			cursor = accentStyle.Render("▸ ")
		}
		number := ""
		if i < 9 {
			number = faintStyle.Render(strconv.Itoa(i + 1))
		}
		rows = append(rows, padRight(cursor+label, menuWidth-1)+number)
		if mode == menuPlain {
			continue
		}
		summary := it.summary
		if summary == "" {
			summary = it.desc
		}
		rows = append(rows, lipgloss.NewStyle().MaxWidth(menuWidth).Render(faintStyle.Render("   "+summary)))
		if mode == menuSpaced && i < len(h.items)-1 {
			rows = append(rows, "")
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// panel describes the selected function, its steps and its command.
func (h *home) panel(width int) string {
	it := h.items[h.cursor]
	lines := []string{accentStyle.Bold(true).Render(it.label), it.desc}
	if len(it.steps) > 0 {
		parts := make([]string, len(it.steps))
		for i, s := range it.steps {
			parts[i] = accentStyle.Render(strconv.Itoa(i+1)) + " " + s
		}
		lines = append(lines, "", strings.Join(parts, faintStyle.Render("  ›  ")))
	}
	lines = append(lines, "")
	if it.open == nil {
		lines = append(lines, warnStyle.Render("Todavía no está en el modo visual."), "")
	}
	lines = append(lines, faintStyle.Render("En modo comando, para scripts y CI:"), "$ "+it.example)
	return boxStyle.Width(width).Render(strings.Join(lines, "\n"))
}

// narrow is the menu without descriptions and, under it, the text of the
// selected function. Without room, the tagline and then the big letters go
// before the text or the command do.
func (h *home) narrow(width, room int) string {
	heads := [][]string{
		{logo(width), faintStyle.Render(tagline), ""},
		{logo(width), ""},
		{logoStyle.Render("draco"), ""},
	}
	var out string
	for _, head := range heads {
		if out = h.narrowWith(width, head); lipgloss.Height(out) <= room {
			break
		}
	}
	return out
}

func (h *home) narrowWith(width int, head []string) string {
	it := h.items[h.cursor]
	lines := slices.Clone(head)
	for i, m := range h.items {
		cursor, label := "  ", m.label
		switch {
		case m.open == nil:
			label = faintStyle.Render(label)
		case i == h.cursor:
			label = accentStyle.Bold(true).Render(label)
		}
		if i == h.cursor {
			cursor = accentStyle.Render("▸ ")
		}
		number := "  "
		if i < 9 {
			number = faintStyle.Render(strconv.Itoa(i+1)) + " "
		}
		lines = append(lines, number+cursor+label)
	}
	lines = append(lines, "", lipgloss.NewStyle().Width(width-2).Render(it.desc))
	if it.open == nil {
		lines = append(lines, warnStyle.Render("Todavía no está en el modo visual."))
	}
	return strings.Join(append(lines, "$ "+it.example), "\n")
}
