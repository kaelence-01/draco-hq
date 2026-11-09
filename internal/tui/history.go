package tui

import (
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/history"
)

var (
	scopeKey   = key.NewBinding(key.WithKeys("left", "h", "right", "l"), key.WithHelp("←→", "proyecto"))
	prevScope  = key.NewBinding(key.WithKeys("left", "h"))
	nextScope  = key.NewBinding(key.WithKeys("right", "l"))
	browseKey  = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "abrir"))
	redoKey    = key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "actualizar"))
	exportKey  = key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "exportar"))
	recentKey  = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "recientes"))
	menuTabKey = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "menú"))
)

const (
	emptyHistory = "Todavía no hay documentos: se agregan al publicar con Generar o con draco g."
	// recentRows is how many documents the home box shows.
	recentRows = 10
)

// recentList is the history with a tab per project; the home box and
// Historial share it.
type recentList struct {
	st     *state
	scope  string // a project; empty: all of them
	cursor int
}

// scopes are the tabs: all, then each project. With one project there is
// nothing to choose.
func (r *recentList) scopes() []string {
	projects := history.Projects(r.st.recent)
	if len(projects) < 2 {
		return nil
	}
	return append([]string{""}, projects...)
}

func (r *recentList) entries() []history.Entry { return history.Of(r.st.recent, r.scope) }

func (r *recentList) selected() (history.Entry, bool) {
	entries := r.entries()
	if len(entries) == 0 {
		return history.Entry{}, false
	}
	r.cursor = min(r.cursor, len(entries)-1)
	return entries[r.cursor], true
}

// keys are the list's main keys, for the footer.
func (r *recentList) keys() []key.Binding {
	var keys []key.Binding
	if len(r.entries()) > 0 {
		keys = []key.Binding{moveKey, browseKey, redoKey}
	}
	if r.scopes() != nil {
		keys = append(keys, scopeKey)
	}
	return keys
}

// moreKeys are the ones only the full help shows.
func (r *recentList) moreKeys() []key.Binding {
	if len(r.entries()) == 0 {
		return nil
	}
	return []key.Binding{upKey, downKey, exportKey, removeKey}
}

// handle moves the list or acts on the selected document; false when k is
// not one of its keys.
func (r *recentList) handle(k tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(k, upKey):
		r.cursor = max(r.cursor-1, 0)
	case key.Matches(k, downKey):
		r.cursor = min(r.cursor+1, max(len(r.entries())-1, 0))
	case key.Matches(k, prevScope):
		r.shift(-1)
	case key.Matches(k, nextScope):
		r.shift(1)
	case key.Matches(k, browseKey):
		return r.browse(), true
	case key.Matches(k, redoKey):
		return r.redo(), true
	case key.Matches(k, exportKey):
		return r.export(), true
	case key.Matches(k, removeKey):
		if e, ok := r.selected(); ok {
			return open(newRemoveDoc(r.st, e)), true
		}
	default:
		return nil, false
	}
	return nil, true
}

func (r *recentList) shift(by int) {
	scopes := r.scopes()
	if scopes == nil {
		return
	}
	i := 0
	for j, s := range scopes {
		if s == r.scope {
			i = j
		}
	}
	r.scope, r.cursor = scopes[(i+by+len(scopes))%len(scopes)], 0
}

// browse opens the selected document in the browser, outside the TUI.
func (r *recentList) browse() tea.Cmd {
	e, ok := r.selected()
	openURL := r.st.deps.OpenURL
	if !ok || openURL == nil {
		return nil
	}
	return func() tea.Msg {
		if err := openURL(e.URL); err != nil {
			return resultMsg{"No se pudo abrir " + e.Issue + ": " + err.Error() + ". Está en " + e.URL, true}
		}
		return resultMsg{"Abriendo " + e.Issue + ": " + e.URL, false}
	}
}

// redo opens Generar with the task and its project: the progress finds the
// document and offers to update it, which also links it again.
func (r *recentList) redo() tea.Cmd {
	e, ok := r.selected()
	if !ok {
		return nil
	}
	if cfg := r.st.cfg; cfg != nil && e.Project != "" && cfg.Projects[e.Project] == nil {
		g := newGenerate(r.st)
		g.issue, g.err = e.Issue, "El proyecto «"+e.Project+"» ya no está en la configuración: elige otro"
		g.form = g.build()
		return open(g)
	}
	return openGenerate(r.st, &cli.Parsed{Values: map[string]string{"--issue": e.Issue, "--project": e.Project}})
}

func (r *recentList) export() tea.Cmd {
	e, ok := r.selected()
	if !ok {
		return nil
	}
	return openExport(r.st, &cli.Parsed{Values: map[string]string{"--issue": e.Issue}})
}

// tabs is the line of projects, the chosen one in brackets; when it does not
// fit, only the chosen one.
func (r *recentList) tabs(width int) string {
	scopes := r.scopes()
	if scopes == nil {
		return ""
	}
	name := func(s string) string {
		if s == "" {
			return "Todos"
		}
		return s
	}
	parts := make([]string, len(scopes))
	at := 0
	for i, s := range scopes {
		if s == r.scope {
			at, parts[i] = i, accentStyle.Bold(true).Render("["+name(s)+"]")
		} else {
			parts[i] = faintStyle.Render(" " + name(s) + " ")
		}
	}
	if line := strings.Join(parts, " "); lipgloss.Width(line) <= width {
		return line
	}
	return accentStyle.Bold(true).Render("‹ "+name(r.scope)+" ›") + faintStyle.Render("  "+strconv.Itoa(at+1)+" de "+strconv.Itoa(len(scopes)))
}

// rows draws up to n documents, one line each, keeping the cursor in view;
// with focus the cursor shows.
func (r *recentList) rows(width, n int, focus bool) []string {
	if r.st.recentErr != "" {
		return []string{lipgloss.NewStyle().Width(width).Render(warnStyle.Render("⚠ " + r.st.recentErr))}
	}
	entries := r.entries()
	if len(entries) == 0 {
		text := emptyHistory
		if r.scope != "" {
			text = "No hay documentos de " + r.scope + "."
		}
		return []string{lipgloss.NewStyle().Width(width).Render(faintStyle.Render(text))}
	}
	start := window(r.cursor, len(entries), n)
	shown := entries[start:min(start+n, len(entries))]
	issueWidth := 0
	for _, e := range shown {
		issueWidth = max(issueWidth, lipgloss.Width(e.Issue))
	}
	now := r.st.now()
	var out []string
	for i, e := range shown {
		cursor, issue := "  ", padRight(e.Issue, issueWidth)
		if focus && start+i == r.cursor {
			cursor, issue = accentStyle.Render("▸ "), accentStyle.Bold(true).Render(issue)
		}
		action := "creado"
		if e.Updated {
			action = "actualizado"
		}
		tail := faintStyle.Render(e.Title)
		switch {
		case e.LinkFailed:
			tail = warnStyle.Render("⚠ sin enlace")
		case r.scope == "" && e.Project != "":
			tail = faintStyle.Render(e.Project)
		}
		row := cursor + issue + "  " + padRight(when(e.At, now), 12) + faintStyle.Render(padRight(action, 12)) + tail
		out = append(out, lipgloss.NewStyle().MaxWidth(width).Render(row))
	}
	return out
}

// detail is the selected document: what, where and when it was published,
// what is wrong with it and what can be done.
func (r *recentList) detail(width int) string {
	e, ok := r.selected()
	if !ok {
		return ""
	}
	wrap := lipgloss.NewStyle().Width(width)
	action := "Creado"
	if e.Updated {
		action = "Actualizado"
	}
	lines := []string{
		accentStyle.Bold(true).Render(e.Issue) + faintStyle.Render("  "+action+" el "+e.At.Local().Format("02/01/2006 15:04")),
		wrap.Render(e.Title),
		wrap.Render(accentStyle.Render(e.URL)),
		"",
		kv("proyecto", e.Project, width), kv("perfil", e.Profile, width), kv("destino", e.Dest, width),
	}
	if e.LinkFailed {
		lines = append(lines, "", wrap.Render(warnStyle.Render("⚠ La tarea no enlaza este documento: el consolidado QA la marcará sin documentación. u lo actualiza y lo vuelve a enlazar.")))
	}
	if len(e.NotPreserved) > 0 {
		lines = append(lines, "", wrap.Render(warnStyle.Render("⚠ Se publicó sin conservar "+quoted(e.NotPreserved)+": salió como la trae la plantilla.")))
	}
	lines = append(lines, "", wrap.Render(faintStyle.Render("enter lo abre · u lo actualiza · e lo exporta · x lo borra")))
	return strings.Join(lines, "\n")
}

// command is the history in the command mode, with the chosen project.
func (r *recentList) command() string {
	if r.scope == "" {
		return "draco history"
	}
	return "draco history -p " + r.scope
}

// when is a date as people say it: today and yesterday with the time, the
// day of this year, or the full date.
func when(at, now time.Time) string {
	at, now = at.Local(), now.Local()
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	switch {
	case !at.Before(today):
		return "hoy " + at.Format("15:04")
	case !at.Before(today.AddDate(0, 0, -1)):
		return "ayer " + at.Format("15:04")
	case at.Year() == y:
		return at.Format("02/01")
	}
	return at.Format("02/01/2006")
}

// historyScreen is Historial: every recorded document, a project at a time,
// with its detail.
type historyScreen struct {
	list *recentList
}

func newHistory(st *state) *historyScreen {
	return &historyScreen{list: &recentList{st: st}}
}

func (h *historyScreen) title() string           { return "Historial" }
func (h *historyScreen) keys() []key.Binding     { return h.list.keys() }
func (h *historyScreen) moreKeys() []key.Binding { return h.list.moreKeys() }

// init reads the history again when the screen shows, as after publishing.
func (h *historyScreen) init() tea.Cmd {
	h.list.st.loadRecent()
	return nil
}

func (h *historyScreen) update(msg tea.Msg) (screen, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		cmd, _ := h.list.handle(k)
		return h, cmd
	}
	return h, nil
}

func (h *historyScreen) view(width, height int) string {
	l := h.list
	if width >= panelMinWidth {
		return panels(width, height, 5, 5,
			func(w, lines int) string {
				head := []string{faintStyle.Render("DOCUMENTOS PUBLICADOS")}
				if t := l.tabs(w); t != "" {
					head = append(head, t)
				}
				head = append(head, "")
				return strings.Join(append(head, l.rows(w, max(lines-len(head), 1), true)...), "\n")
			},
			func(w, _ int) string {
				detail := l.detail(w)
				if detail != "" {
					detail += "\n\n"
				}
				return detail + faintStyle.Render("En modo comando:") + "\n$ " + l.command()
			})
	}
	w := width - 2
	var lines []string
	if t := l.tabs(w); t != "" {
		lines = append(lines, t, "")
	}
	detail := l.detail(w)
	rows := max(height-len(lines)-lipgloss.Height(detail)-2, 3)
	lines = append(lines, l.rows(w, rows, true)...)
	if detail != "" {
		lines = append(lines, "", detail)
	}
	return lipgloss.NewStyle().PaddingLeft(1).Render(strings.Join(lines, "\n"))
}
