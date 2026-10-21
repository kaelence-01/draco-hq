package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
)

// profileSummary is where a profile takes the issues from and where its
// documents go.
func profileSummary(p *config.Profile) string {
	docs := p.Docs.Provider
	switch {
	case p.Docs.Provider == "confluence" && p.Docs.Space != "":
		docs += " (space " + p.Docs.Space + ")"
	case p.Docs.Provider == "markdown":
		docs += " (" + p.Docs.Path + ")"
	}
	return p.Tracker.Provider + " → " + docs
}

// profiles lists the saved profiles, as draco profile does.
type profiles struct {
	st       *state
	cursor   int
	filter   filter
	removing string // profile waiting for the confirmation to delete it
	note     string
	failed   bool
}

func newProfiles(st *state) *profiles {
	p := &profiles{st: st}
	for i, n := range p.names() {
		if n == st.profile {
			p.cursor = i
		}
	}
	return p
}

func (p *profiles) title() string { return "Perfiles" }

func (p *profiles) keys() []key.Binding {
	return []key.Binding{upKey, downKey, defaultKey, newKey, editKey, removeKey, filterKey}
}

func (p *profiles) captures(k tea.KeyPressMsg) bool {
	return p.removing != "" || p.filter.captures(k)
}

func (p *profiles) names() []string {
	var out []string
	for _, n := range p.st.cfg.ProfileNames() {
		if p.filter.matches(n) {
			out = append(out, n)
		}
	}
	return out
}

func (p *profiles) selected() string {
	names := p.names()
	if len(names) == 0 {
		return ""
	}
	p.cursor = min(p.cursor, len(names)-1)
	return names[p.cursor]
}

func (p *profiles) update(msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || p.st.cfg == nil {
		return p, nil
	}
	switch {
	case p.removing != "":
		p.confirmRemove(key.Matches(k, yesKey))
		return p, nil
	case p.filter.handle(k):
		p.cursor = 0
		return p, nil
	}

	p.note, p.failed = "", false
	name := p.selected()
	switch {
	case key.Matches(k, upKey):
		p.cursor = max(p.cursor-1, 0)
	case key.Matches(k, downKey):
		p.cursor = min(p.cursor+1, max(len(p.names())-1, 0))
	case key.Matches(k, newKey):
		return p, open(newProfileForm(p.st, "", profileNew, p.done, back))
	case name == "":
	case key.Matches(k, defaultKey):
		p.done("Perfil por defecto: "+name, p.st.save(func(c *config.Config) error { return c.SetDefaultProfile(name) }))
	case key.Matches(k, editKey):
		return p, open(newProfileForm(p.st, name, profileEdit, p.done, back))
	case key.Matches(k, removeKey):
		p.removing = name
	}
	return p, nil
}

func (p *profiles) confirmRemove(yes bool) {
	name := p.removing
	p.removing = ""
	if !yes {
		p.note, p.failed = "No se eliminó "+name, false
		return
	}
	wasDefault := p.st.cfg.DefaultProfile == name
	err := p.st.save(func(c *config.Config) error { return c.RemoveProfile(name) })
	note := "Perfil eliminado: " + name
	if wasDefault && p.st.cfg != nil && len(p.st.cfg.Profiles) > 1 {
		note += ". Era el perfil por defecto: elige otro con d"
	}
	p.done(note, err)
}

func (p *profiles) done(note string, err error) {
	if err != nil {
		p.note, p.failed = "No se pudo guardar: "+err.Error(), true
		return
	}
	p.note, p.failed = note, false
}

func (p *profiles) view(width, height int) string {
	if p.st.cfg == nil {
		return pad(lipgloss.NewStyle().Width(width - 2).Render(warnStyle.Render("⚠ " + p.st.notice)))
	}
	list, detail := p.list(), p.detail()
	var body string
	if width >= 80 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, list, "  ", detail)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, list, detail)
	}
	lines := []string{body, ""}
	switch {
	case p.removing != "":
		lines = append(lines, warnStyle.Render("¿Eliminar el perfil «"+p.removing+"»? s confirma, cualquier otra tecla cancela"))
	case p.note != "" && p.failed:
		lines = append(lines, errStyle.Render("✗ "+p.note))
	case p.note != "":
		lines = append(lines, okStyle.Render("✓ ")+p.note)
	}
	return lipgloss.NewStyle().Padding(1, 1, 0).Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (p *profiles) list() string {
	var rows []string
	if f := p.filter.view(); f != "" {
		rows = append(rows, f)
	}
	names := p.names()
	switch {
	case len(names) == 0 && p.filter.text != "":
		rows = append(rows, faintStyle.Render("Ningún perfil coincide"))
	case len(names) == 0:
		implicit, _, _ := p.st.cfg.Profile("")
		rows = append(rows, lipgloss.NewStyle().Width(32).Render(faintStyle.Render(
			"Sin perfiles: se usa el implícito '"+config.LegacyProfile+"' ("+profileSummary(implicit)+"). n agrega uno, y el implícito también se guarda.")))
	}
	sel := p.selected()
	for _, n := range names {
		cursor, label := "  ", n
		if n == sel {
			cursor, label = accentStyle.Render("▸ "), accentStyle.Bold(true).Render(n)
		}
		mark := ""
		if n == p.st.cfg.DefaultProfile {
			mark = "  " + faintStyle.Render("por defecto")
		}
		rows = append(rows, cursor+label+mark)
	}
	return boxStyle.Width(36).Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

func (p *profiles) detail() string {
	name := p.selected()
	if name == "" {
		return ""
	}
	pr := p.st.cfg.Profiles[name]
	field := func(label, v string) string {
		if v == "" {
			v = faintStyle.Render("—")
		}
		return faintStyle.Render(label+strings.Repeat(" ", 11-len(label))) + v
	}
	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		accentStyle.Bold(true).Render(name), "",
		field("tracker", pr.Tracker.Provider),
		field("destino", pr.Docs.Provider),
		field("space", pr.Docs.Space),
		field("carpeta", pr.Docs.Path),
		field("plantillas", pr.Template),
	))
}
