package tui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
)

type lineState int

const (
	lineOK lineState = iota
	lineWarn
	lineNone // optional and not set up
)

// configLine is one line of «Tu configuración».
type configLine struct {
	state         lineState
	label, detail string
}

// configChecks summarizes the config with the rules of Configuración.
func configChecks(st *state) []configLine {
	c := st.cfg
	conn := configLine{lineWarn, "Conexión", "faltan el email, el token o la URL"}
	if c.HasAtlassianCredentials() {
		conn = configLine{lineOK, "Conexión", c.AtlassianEmail}
	}
	projects := configLine{lineWarn, "Proyectos", "ninguno"}
	if n := len(c.Projects); n > 0 {
		projects = configLine{lineOK, "Proyectos", strconv.Itoa(n)}
	}
	profile := configLine{lineWarn, "Perfil", "ninguno"}
	if st.profile != "" {
		profile = configLine{lineOK, "Perfil", st.profile}
	}
	var qa configLine
	switch c.QAState() {
	case config.QAUnset:
		qa = configLine{lineNone, "QA", "sin configurar"}
	case config.QAIncomplete:
		qa = configLine{lineWarn, "QA", "qa_jira incompleta"}
	case config.QANoAccount:
		qa = configLine{lineWarn, "QA", "falta qa_email"}
	default:
		qa = configLine{lineOK, "QA", "proyecto " + c.QAJira.Project}
		if !c.HasQAReport() {
			qa = configLine{lineWarn, "QA", "falta qa_report"}
		}
	}
	return []configLine{conn, projects, profile, qa, {lineOK, "Tema", theme}}
}

func (c configLine) line(width int) string {
	mark, detail := okStyle.Render("✓"), faintStyle.Render(c.detail)
	switch c.state {
	case lineWarn:
		mark, detail = warnStyle.Render("⚠"), warnStyle.Render(c.detail)
	case lineNone:
		mark = faintStyle.Render("·")
	}
	return hang(mark+" "+padRight(c.label, 10), detail, width)
}

// context is the active project and the config, side by side.
func (h *home) context(width int) string {
	leftWidth := (width - 1) / 2
	rightWidth := width - 1 - leftWidth
	// A box is its text, a border and a padding on each side.
	project := append([]string{faintStyle.Render("PROYECTO ACTIVO")}, h.projectLines(leftWidth-4)...)
	status := []string{faintStyle.Render("TU CONFIGURACIÓN")}
	for _, c := range configChecks(h.st) {
		status = append(status, c.line(rightWidth-4))
	}

	render := func(w, height int, lines []string) string {
		return boxStyle.Width(w).Height(height).Render(strings.Join(lines, "\n"))
	}
	left, right := render(leftWidth, 0, project), render(rightWidth, 0, status)
	if hl, hr := lipgloss.Height(left), lipgloss.Height(right); hl != hr {
		left, right = render(leftWidth, max(hl, hr), project), render(rightWidth, max(hl, hr), status)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (h *home) projectLines(width int) []string {
	name := h.st.project
	pc := h.st.cfg.Projects[name]
	if pc == nil {
		return []string{faintStyle.Render("Ninguno. Crea uno en Proyectos.")}
	}
	title := accentStyle.Bold(true).Render(name)
	if name == h.st.cfg.DefaultProject {
		title += faintStyle.Render(" · por defecto")
	}
	vcs := pc.VCSHost
	if pc.VCSOrg != "" {
		if vcs != "" {
			vcs += " / "
		}
		vcs += pc.VCSOrg
	}
	return []string{title, kv("backend", pc.BackendPath, width), kv("frontend", pc.FrontendPath, width), kv("vcs", vcs, width), kv("space", pc.ConfluenceSpaceKey, width)}
}

// kv is a label and its value, or a dash when it has none.
func kv(label, value string, width int) string {
	if value == "" {
		value = faintStyle.Render("—")
	}
	return hang(faintStyle.Render(padRight(label, 9)), value, width)
}
