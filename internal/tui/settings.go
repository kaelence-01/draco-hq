package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var projectKey = key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "crear un proyecto"))

// settings is the configuration, by sections. When the file has no
// credentials and no profiles, it starts with the connection, in the order of
// draco init.
type settings struct {
	st           *state
	cursor       int
	wizard       bool
	started      bool
	offerProject bool // the first profile was saved and there are no projects
	note         string
	failed       bool
}

func newSettings(st *state) *settings {
	s := &settings{st: st}
	if f, err := st.deps.LoadFile(); err == nil {
		s.wizard = !f.HasAtlassianCredentials() && len(f.Profiles) == 0
	}
	return s
}

func (s *settings) title() string { return "Configuración" }

func (s *settings) keys() []key.Binding {
	if s.offerProject {
		return []key.Binding{upKey, downKey, enterKey, projectKey}
	}
	return []key.Binding{upKey, downKey, enterKey}
}

func (s *settings) init() tea.Cmd {
	if !s.wizard || s.started || s.st.cfg == nil {
		return nil
	}
	s.started = true
	return open(newConnection(s))
}

// section is one entry of the list, with what it has or what it lacks.
type section struct {
	name, status string
	ok           bool
}

func (s *settings) sections() []section {
	c := s.st.cfg
	conn := section{name: "Conexión con Atlassian", status: "faltan el email, el token o la URL del sitio"}
	if c.HasAtlassianCredentials() {
		conn.ok, conn.status = true, c.AtlassianEmail+" en "+c.BaseURL
	}
	prof := section{name: "Perfiles"}
	if p, name, err := c.Profile(""); err != nil {
		prof.status = err.Error()
	} else {
		prof.ok, prof.status = true, name+": "+profileSummary(p)
		if len(c.Profiles) == 0 {
			prof.status += " (implícito)"
		}
	}
	qa := section{name: "QA", status: "sin configurar: solo hace falta para el consolidado QA"}
	switch {
	case c.QAJira == nil:
	case c.QAJira.Validate() != nil:
		qa.status = "incompleta: ábrela para ver qué falta"
	case c.QAEmail == "":
		qa.status = "falta el email de la cuenta de QA: nadie podría publicar el consolidado"
	default:
		qa.ok, qa.status = true, "proyecto "+c.QAJira.Project+"; publica "+c.QAEmail
	}
	return []section{conn, prof, qa}
}

func (s *settings) update(msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok || s.st.cfg == nil {
		return s, nil
	}
	s.note, s.failed = "", false
	switch {
	case key.Matches(k, upKey):
		s.cursor = max(s.cursor-1, 0)
	case key.Matches(k, downKey):
		s.cursor = min(s.cursor+1, len(s.sections())-1)
	case key.Matches(k, enterKey):
		switch s.cursor {
		case 0:
			return s, open(newConnection(s))
		case 1:
			return s, open(newProfiles(s.st))
		}
		return s, open(newQAConfig(s))
	case key.Matches(k, projectKey) && s.offerProject:
		s.offerProject = false
		ps := newProjects(s.st)
		return s, tea.Sequence(open(ps), open(newProjectForm(ps, "")))
	}
	return s, nil
}

// firstProfile reports the first profile saved by the wizard.
func (s *settings) firstProfile(note string, err error) {
	s.done(note, err)
	s.offerProject = err == nil && len(s.st.cfg.Projects) == 0
	s.cursor = 1
}

func (s *settings) done(note string, err error) {
	if err != nil {
		s.note, s.failed = "No se pudo guardar: "+err.Error(), true
		return
	}
	s.note, s.failed = note, false
}

func (s *settings) view(width, height int) string {
	if s.st.cfg == nil {
		return pad(lipgloss.NewStyle().Width(width - 2).Render(warnStyle.Render("⚠ " + s.st.notice)))
	}
	lines := []string{faintStyle.Render("Se guarda en config.yaml, legible solo por tu usuario."), ""}
	for i, sec := range s.sections() {
		cursor, name := "  ", sec.name
		if i == s.cursor {
			cursor, name = accentStyle.Render("▸ "), accentStyle.Bold(true).Render(name)
		}
		mark := okStyle.Render("✓ ")
		if !sec.ok {
			mark = warnStyle.Render("⚠ ")
		}
		lines = append(lines, cursor+name, "    "+mark+sec.status)
	}
	switch {
	case s.note != "" && s.failed:
		lines = append(lines, "", errStyle.Render("✗ "+s.note))
	case s.note != "":
		lines = append(lines, "", okStyle.Render("✓ ")+s.note)
	}
	if s.offerProject {
		lines = append(lines, faintStyle.Render("Configuración lista. Si quieres, p crea el primer proyecto."))
	}
	return pad(lipgloss.NewStyle().Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}
