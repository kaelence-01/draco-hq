package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// settings is the configuration, by sections: first what generating a
// document needs, then the optional ones. When the file has no credentials
// and no profiles, it starts with the connection, in the order of draco init.
type settings struct {
	st      *state
	cursor  int
	wizard  bool
	started bool
	note    string
	failed  bool
}

func newSettings(st *state) *settings {
	s := &settings{st: st}
	if f, err := st.deps.LoadFile(); err == nil {
		s.wizard = !f.HasAtlassianCredentials() && len(f.Profiles) == 0
	}
	return s
}

func (s *settings) title() string { return "Configuración" }

func (s *settings) keys() []key.Binding { return []key.Binding{upKey, downKey, enterKey} }

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
	ok, optional bool
}

func (s *settings) sections() []section {
	c := s.st.cfg
	conn := section{name: "Conexión con Atlassian", status: "faltan el email, el token o la URL del sitio"}
	if c.HasAtlassianCredentials() {
		conn.ok, conn.status = true, c.AtlassianEmail+" en "+c.BaseURL
		if len(c.Profiles) == 0 && c.ConfluenceSpaceKey != "" {
			conn.status += " · space " + c.ConfluenceSpaceKey
		}
	}
	proj := section{name: "Proyectos", status: "ninguno: hace falta uno para generar el documento"}
	switch n := len(c.Projects); {
	case n == 1:
		proj.ok, proj.status = true, "1 proyecto"
	case n > 1:
		proj.ok, proj.status = true, fmt.Sprintf("%d proyectos", n)
	}
	if c.DefaultProject != "" {
		proj.status += "; por defecto: " + c.DefaultProject
	}
	prof := section{name: "Perfiles", optional: true}
	switch p, name, err := c.Profile(""); {
	case err != nil:
		prof.status = err.Error()
	case len(c.Profiles) == 0:
		prof.ok, prof.status = true, "ninguno: se usa "+profileSummary(p)
	default:
		prof.ok, prof.status = true, name+": "+profileSummary(p)
	}
	qa := section{name: "QA", status: "sin configurar: solo hace falta para el consolidado QA", optional: true}
	switch {
	case c.QAJira == nil:
	case c.QAJira.Validate() != nil:
		qa.status = "incompleta: ábrela para ver qué falta"
	case c.QAEmail == "":
		qa.status = "falta el email de la cuenta de QA: nadie podría publicar el consolidado"
	default:
		qa.ok, qa.status = true, "proyecto "+c.QAJira.Project+"; publica "+c.QAEmail
	}
	look := section{name: "Tema", ok: true, status: knownTheme(c.Theme), optional: true}
	switch {
	case c.Theme == "":
		look.status += " (por defecto)"
	case knownTheme(c.Theme) != c.Theme:
		look.ok, look.status = false, "«"+c.Theme+"» no existe: se usa "+defaultTheme
	}
	return []section{conn, proj, prof, qa, look}
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
			return s, open(newProjects(s.st))
		case 2:
			return s, open(newProfiles(s.st))
		case 3:
			return s, open(newQAConfig(s))
		}
		return s, open(newThemePicker(s))
	}
	return s, nil
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
	optional := false
	for i, sec := range s.sections() {
		if sec.optional && !optional {
			optional = true
			lines = append(lines, "", faintStyle.Render("Opcional"))
		}
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
	return pad(lipgloss.NewStyle().Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}
