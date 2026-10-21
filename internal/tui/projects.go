package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
)

var (
	useKey     = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "usar en esta sesión"))
	defaultKey = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "por defecto"))
	newKey     = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "nuevo"))
	editKey    = key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "editar"))
	removeKey  = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "eliminar"))
	filterKey  = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filtrar"))
	yesKey     = key.NewBinding(key.WithKeys("s", "y"))
)

// projects lists the configured projects: the active one is chosen for this
// session, the default one is saved.
type projects struct {
	st        *state
	cursor    int
	filter    string
	filtering bool
	removing  string // project waiting for the confirmation to delete it
	note      string
	failed    bool // note is an error
}

func newProjects(st *state) *projects {
	p := &projects{st: st}
	for i, n := range p.names() {
		if n == st.project {
			p.cursor = i
		}
	}
	return p
}

func (p *projects) title() string { return "Proyectos" }

func (p *projects) keys() []key.Binding {
	return []key.Binding{upKey, downKey, useKey, defaultKey, newKey, editKey, removeKey, filterKey}
}

// captures keeps the keys while the user types a filter or confirms, and esc
// while a filter is applied, which clears it.
func (p *projects) captures(k tea.KeyPressMsg) bool {
	return p.filtering || p.removing != "" || p.filter != "" && key.Matches(k, backKey)
}

// names are the projects that match the filter, sorted.
func (p *projects) names() []string {
	if p.st.cfg == nil {
		return nil
	}
	var out []string
	for _, n := range p.st.cfg.ProjectNames() {
		if strings.Contains(strings.ToLower(n), strings.ToLower(p.filter)) {
			out = append(out, n)
		}
	}
	return out
}

func (p *projects) selected() string {
	names := p.names()
	if len(names) == 0 {
		return ""
	}
	p.cursor = min(p.cursor, len(names)-1)
	return names[p.cursor]
}

func (p *projects) update(msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || p.st.cfg == nil {
		return p, nil
	}
	switch {
	case p.removing != "":
		p.confirmRemove(key.Matches(k, yesKey))
		return p, nil
	case p.filtering:
		p.typeFilter(k)
		return p, nil
	}

	p.note, p.failed = "", false
	name := p.selected()
	switch {
	case key.Matches(k, backKey):
		p.filter, p.cursor = "", 0
	case key.Matches(k, upKey):
		p.cursor = max(p.cursor-1, 0)
	case key.Matches(k, downKey):
		p.cursor = min(p.cursor+1, max(len(p.names())-1, 0))
	case key.Matches(k, filterKey):
		p.filtering = true
	case key.Matches(k, newKey):
		return p, open(newProjectForm(p, ""))
	case name == "":
	case key.Matches(k, useKey):
		p.st.project = name
		p.note = "Proyecto activo en esta sesión: " + name + " (la configuración no cambia)"
	case key.Matches(k, defaultKey):
		p.done("Proyecto por defecto: "+name, p.st.save(func(c *config.Config) error { return c.SetDefaultProject(name) }))
	case key.Matches(k, editKey):
		return p, open(newProjectForm(p, name))
	case key.Matches(k, removeKey):
		p.removing = name
	}
	return p, nil
}

func (p *projects) typeFilter(k tea.KeyPressMsg) {
	switch {
	case key.Matches(k, backKey):
		p.filter, p.filtering = "", false
	case k.Code == tea.KeyEnter:
		p.filtering = false
	case k.Code == tea.KeyBackspace:
		if r := []rune(p.filter); len(r) > 0 {
			p.filter = string(r[:len(r)-1])
		}
	case k.Text != "":
		p.filter += k.Text
	}
	p.cursor = 0
}

func (p *projects) confirmRemove(yes bool) {
	name := p.removing
	p.removing = ""
	if !yes {
		p.note, p.failed = "No se eliminó "+name, false
		return
	}
	wasDefault := p.st.cfg.DefaultProject == name
	err := p.st.save(func(c *config.Config) error { return c.RemoveProject(name) })
	note := "Proyecto eliminado: " + name
	if wasDefault {
		note += ". Era el proyecto por defecto: elige otro con d"
	}
	p.done(note, err)
}

// done shows the result of a change.
func (p *projects) done(note string, err error) {
	if err != nil {
		p.note, p.failed = "No se pudo guardar: "+err.Error(), true
		return
	}
	p.note, p.failed = note, false
}

func (p *projects) view(width, height int) string {
	if p.st.cfg == nil {
		return " " + warnStyle.Render("⚠ "+p.st.notice)
	}
	list, detail := p.list(), p.detail()
	var body string
	if width >= 80 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, list, "  ", detail)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, list, detail)
	}
	var lines []string
	switch {
	case p.removing != "":
		lines = append(lines, warnStyle.Render("¿Eliminar el proyecto «"+p.removing+"»? s confirma, cualquier otra tecla cancela"))
	case p.note != "" && p.failed:
		lines = append(lines, errStyle.Render("✗ "+p.note))
	case p.note != "":
		lines = append(lines, okStyle.Render("✓ ")+p.note)
	}
	return lipgloss.NewStyle().Padding(1, 1, 0).Render(lipgloss.JoinVertical(lipgloss.Left, append([]string{body, ""}, lines...)...))
}

func (p *projects) list() string {
	var rows []string
	switch {
	case p.filtering:
		rows = append(rows, "/ "+p.filter+accentStyle.Render("█"))
	case p.filter != "":
		rows = append(rows, faintStyle.Render("filtro: "+p.filter+"  (esc lo quita)"))
	}
	names := p.names()
	if len(names) == 0 {
		if p.filter != "" {
			rows = append(rows, faintStyle.Render("Ningún proyecto coincide"))
		} else {
			rows = append(rows, faintStyle.Render("No hay proyectos. n crea uno."))
		}
	}
	sel := p.selected()
	for _, n := range names {
		cursor, label := "  ", n
		if n == sel {
			cursor, label = accentStyle.Render("▸ "), accentStyle.Bold(true).Render(n)
		}
		var marks []string
		if n == p.st.project {
			marks = append(marks, okStyle.Render("activo"))
		}
		if n == p.st.cfg.DefaultProject {
			marks = append(marks, faintStyle.Render("por defecto"))
		}
		rows = append(rows, cursor+label+"  "+strings.Join(marks, " "))
	}
	return boxStyle.Width(36).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (p *projects) detail() string {
	name := p.selected()
	if name == "" {
		return ""
	}
	pc := p.st.cfg.Projects[name]
	field := func(label, v string) string {
		if v == "" {
			v = faintStyle.Render("—")
		}
		return faintStyle.Render(label+strings.Repeat(" ", 15-len(label))) + v
	}
	lines := []string{
		accentStyle.Bold(true).Render(name), "",
		field("backend_path", pc.BackendPath), field("backend_repo", pc.BackendRepo),
		field("frontend_path", pc.FrontendPath), field("frontend_repo", pc.FrontendRepo),
		field("vcs_host", pc.VCSHost), field("vcs_org", pc.VCSOrg),
		field("space", pc.ConfluenceSpaceKey),
	}
	if checklist := p.st.cfg.ResolveDeployChecklist(pc); len(checklist) > 0 {
		lines = append(lines, "", faintStyle.Render("deploy_checklist"))
		for _, c := range checklist {
			lines = append(lines, "- "+c)
		}
	}
	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
