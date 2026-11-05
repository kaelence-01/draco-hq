package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
)

var (
	useKey     = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "usar"))
	resumeKey  = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "retomar"))
	defaultKey = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "por defecto"))
	newKey     = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "nuevo"))
	editKey    = key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "editar"))
	removeKey  = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "eliminar"))
	yesKey     = key.NewBinding(key.WithKeys("s", "y"))
)

// projects lists the configured projects, then the drafts: the active one is
// chosen for this session, the default one is saved, and a draft can only be
// resumed or deleted.
type projects struct {
	st       *state
	cursor   int
	filter   filter
	removing *entry // waiting for the confirmation to delete it
	note     string
	failed   bool // note is an error
}

// entry is a project of the list, or a draft.
type entry struct {
	name  string
	draft bool
}

func newProjects(st *state) *projects {
	p := &projects{st: st}
	for i, e := range p.entries() {
		if !e.draft && e.name == st.project {
			p.cursor = i
		}
	}
	return p
}

func (p *projects) title() string { return "Proyectos" }

func (p *projects) keys() []key.Binding {
	if p.selected().draft {
		return []key.Binding{moveKey, resumeKey, newKey, filterKey}
	}
	return []key.Binding{moveKey, useKey, newKey, editKey, filterKey}
}

func (p *projects) moreKeys() []key.Binding {
	return []key.Binding{upKey, downKey, defaultKey, removeKey}
}

// captures keeps the keys while the user confirms, and those of the filter.
func (p *projects) captures(k tea.KeyPressMsg) bool {
	return p.removing != nil || p.filter.captures(k)
}

// entries are the projects that match the filter, then the drafts, sorted.
func (p *projects) entries() []entry {
	if p.st.cfg == nil {
		return nil
	}
	var out []entry
	for _, n := range p.st.cfg.ProjectNames() {
		if p.filter.matches(n) {
			out = append(out, entry{name: n})
		}
	}
	for _, n := range p.st.cfg.DraftNames() {
		if p.filter.matches(n) {
			out = append(out, entry{name: n, draft: true})
		}
	}
	return out
}

func (p *projects) selected() entry {
	entries := p.entries()
	if len(entries) == 0 {
		return entry{}
	}
	p.cursor = min(p.cursor, len(entries)-1)
	return entries[p.cursor]
}

func (p *projects) update(msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || p.st.cfg == nil {
		return p, nil
	}
	switch {
	case p.removing != nil:
		p.confirmRemove(key.Matches(k, yesKey))
		return p, nil
	case p.filter.handle(k):
		p.cursor = 0
		return p, nil
	}

	p.note, p.failed = "", false
	sel := p.selected()
	name := sel.name
	switch {
	case key.Matches(k, upKey):
		p.cursor = max(p.cursor-1, 0)
	case key.Matches(k, downKey):
		p.cursor = min(p.cursor+1, max(len(p.entries())-1, 0))
	case key.Matches(k, newKey):
		return p, open(newProjectForm(p, ""))
	case name == "":
	case key.Matches(k, removeKey):
		p.removing = &sel
	case sel.draft && (key.Matches(k, resumeKey) || key.Matches(k, editKey)):
		return p, open(newDraftForm(p, name))
	case sel.draft && key.Matches(k, defaultKey):
		p.note, p.failed = "Un borrador no se puede usar: enter lo retoma y, al completarlo, pasa a ser un proyecto", true
	case sel.draft:
	case key.Matches(k, useKey):
		p.st.useProject(name)
		if p.st.profile == "" {
			p.note, p.failed = p.st.notice, true
			break
		}
		p.note = "Proyecto activo en esta sesión: " + name + ", con el perfil " + p.st.profile + " (la configuración no cambia)"
	case key.Matches(k, defaultKey):
		p.done("Proyecto por defecto: "+name, p.st.save(func(c *config.Config) error { return c.SetDefaultProject(name) }))
	case key.Matches(k, editKey):
		return p, open(newProjectForm(p, name))
	}
	return p, nil
}

func (p *projects) confirmRemove(yes bool) {
	e := *p.removing
	name := e.name
	p.removing = nil
	if !yes {
		p.note, p.failed = "No se eliminó "+name, false
		return
	}
	if e.draft {
		p.done("Borrador eliminado: "+name, p.st.save(func(c *config.Config) error { return c.RemoveProjectDraft(name) }))
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
	status := p.status()
	if width >= panelMinWidth {
		boxes := height
		if status != "" {
			boxes--
		}
		out := panels(width, boxes, 4, 6,
			func(_, lines int) string { return p.listText(lines) },
			func(w, _ int) string { return p.detailText(w) })
		if status != "" {
			out = lipgloss.JoinVertical(lipgloss.Left, out, lipgloss.NewStyle().PaddingLeft(1).Render(status))
		}
		return out
	}
	detailWidth := width - 6
	if width >= 80 {
		detailWidth = width - 44
	}
	list, detail := p.list(), p.detail(detailWidth)
	var body string
	if width >= 80 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, list, "  ", detail)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, list, detail)
	}
	lines := []string{body, ""}
	if status != "" {
		lines = append(lines, status)
	}
	return lipgloss.NewStyle().Padding(1, 1, 0).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// status is the question being asked or the result of the last change.
func (p *projects) status() string {
	switch {
	case p.removing != nil && p.removing.draft:
		return warnStyle.Render("¿Eliminar el borrador «" + p.removing.name + "»? s confirma, cualquier otra tecla cancela")
	case p.removing != nil:
		return warnStyle.Render("¿Eliminar el proyecto «" + p.removing.name + "»? s confirma, cualquier otra tecla cancela")
	case p.note != "" && p.failed:
		return errStyle.Render("✗ " + p.note)
	case p.note != "":
		return okStyle.Render("✓ ") + p.note
	}
	return ""
}

func (p *projects) list() string {
	return boxStyle.Width(36).Render(p.listText(1 << 10))
}

// listText is the filter line and the projects that fit in lines lines.
func (p *projects) listText(lines int) string {
	rows := []string{p.filter.line()}
	entries := p.entries()
	if len(entries) == 0 {
		if p.filter.text != "" {
			rows = append(rows, faintStyle.Render("Ningún proyecto coincide"))
		} else {
			rows = append(rows, faintStyle.Render("No hay proyectos. n crea uno."))
		}
	}
	sel := p.selected()
	start := window(p.cursor, len(entries), max(lines-len(rows), 1))
	for i := start; i < min(start+max(lines-len(rows), 1), len(entries)); i++ {
		e := entries[i]
		cursor, label := "  ", e.name
		switch {
		case e == sel:
			cursor, label = accentStyle.Render("▸ "), accentStyle.Bold(true).Render(e.name)
		case e.draft:
			label = faintStyle.Render(e.name)
		}
		var marks []string
		switch {
		case e.draft:
			marks = append(marks, warnStyle.Render("borrador"))
		case e.name == p.st.project:
			marks = append(marks, okStyle.Render("activo"))
		}
		if !e.draft && e.name == p.st.cfg.DefaultProject {
			marks = append(marks, faintStyle.Render("por defecto"))
		}
		rows = append(rows, cursor+label+"  "+strings.Join(marks, " "))
	}
	return strings.Join(rows, "\n")
}

func (p *projects) detail(width int) string {
	if p.selected().name == "" {
		return ""
	}
	return boxStyle.Render(p.detailText(width))
}

// detailText is everything the config says about the project under the cursor.
func (p *projects) detailText(width int) string {
	sel := p.selected()
	name := sel.name
	if name == "" {
		return ""
	}
	pc := p.st.cfg.Projects[name]
	lines := []string{accentStyle.Bold(true).Render(name), ""}
	if sel.draft {
		pc = p.st.cfg.ProjectDrafts[name]
		why := "falta guardarlo"
		check := *pc
		if err := check.Complete(); err != nil {
			why = err.Error()
		}
		lines = append(lines, hang(warnStyle.Render("Borrador: "), "no se puede usar hasta completarlo ("+why+"). enter lo retoma; x lo elimina.", width), "")
	}
	field := func(label, v string) string {
		if v == "" {
			v = faintStyle.Render("—")
		}
		return hang(faintStyle.Render(padRight(label, 15)), v, width)
	}
	lines = append(lines,
		field("backend_path", pc.BackendPath), field("backend_repo", pc.BackendRepo),
		field("frontend_path", pc.FrontendPath), field("frontend_repo", pc.FrontendRepo),
		field("vcs_host", pc.VCSHost), field("vcs_org", pc.VCSOrg),
		field("space", pc.ConfluenceSpaceKey), field("perfil", pc.Profile),
	)
	for _, repo := range []string{"backend", "frontend"} {
		if b := pc.BacklogFor(repo); b != nil && len(b.ClassGlobs)+len(b.Modules) > 0 {
			lines = append(lines, "", faintStyle.Render("backlog "+repo))
			for _, g := range b.ClassGlobs {
				lines = append(lines, "- clases: "+g)
			}
			for _, m := range b.Modules {
				lines = append(lines, "- módulos: "+m)
			}
		}
	}
	if checklist := p.st.cfg.ResolveDeployChecklist(pc); len(checklist) > 0 {
		lines = append(lines, "", faintStyle.Render("deploy_checklist"))
		for _, c := range checklist {
			lines = append(lines, "- "+c)
		}
	}
	return strings.Join(lines, "\n")
}
