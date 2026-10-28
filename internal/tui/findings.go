package tui

import (
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/backlog"
)

var (
	repoTabKey    = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "otro repo"))
	saveReportKey = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "guardar reporte"))
)

// findingsDetailLines is the height of the detail below the list when there
// is no room beside it.
const findingsDetailLines = 8

// findings lists what each repo's scan found, with no limit: the list scrolls.
type findings struct {
	run     *backlogRun
	results []scanResult
	tab     int
	cursor  int
	filter  filter
	saved   []string
	note    string
}

func newFindings(r *backlogRun, results []scanResult) *findings {
	return &findings{run: r, results: results}
}

func (f *findings) title() string { return "Hallazgos" }

func (f *findings) keys() []key.Binding {
	keys := []key.Binding{moveKey, saveReportKey}
	if len(f.results) > 1 {
		keys = append([]key.Binding{repoTabKey}, keys...)
	}
	return keys
}

func (f *findings) captures(k tea.KeyPressMsg) bool { return f.filter.captures(k) }

func (f *findings) current() scanResult { return f.results[f.tab] }

// rows are the findings of the current repo that match the filter.
func (f *findings) rows() []backlog.Finding {
	var out []backlog.Finding
	for _, fd := range f.current().rep.Findings {
		if f.filter.matches(strings.Join(append([]string{fd.Title, string(fd.Kind), string(fd.Severity), fd.Module}, fd.Files...), "\n")) {
			out = append(out, fd)
		}
	}
	return out
}

func (f *findings) update(msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return f, nil
	}
	if f.filter.handle(k) {
		f.cursor = 0
		return f, nil
	}
	switch {
	case key.Matches(k, repoTabKey) && len(f.results) > 1:
		f.tab, f.cursor = (f.tab+1)%len(f.results), 0
	case key.Matches(k, upKey):
		f.cursor = max(f.cursor-1, 0)
	case key.Matches(k, downKey):
		f.cursor = min(f.cursor+1, max(len(f.rows())-1, 0))
	case key.Matches(k, saveReportKey):
		f.save()
	}
	return f, nil
}

// save writes every repo's full report where draco backlog scan -o leaves it.
func (f *findings) save() {
	f.saved, f.note = nil, ""
	for _, r := range f.results {
		name := backlog.ReportName(f.run.projName, r.target.Dir, r.target.Repo, f.run.modules)
		path, err := backlog.SaveReport(r.rep, "", name)
		if err != nil {
			f.note = err.Error()
			return
		}
		f.saved = append(f.saved, path)
	}
}

func severityStyle(s backlog.Severity) lipgloss.Style {
	switch s {
	case backlog.SeverityHigh:
		return errStyle
	case backlog.SeverityMedium:
		return warnStyle
	}
	return accentStyle
}

func (f *findings) view(width, height int) string {
	r := f.current()
	wrap := lipgloss.NewStyle().Width(width - 2)
	var head []string
	if len(f.results) > 1 {
		var tabs []string
		for i, res := range f.results {
			label := res.target.Repo + " (" + strconv.Itoa(res.rep.Total) + ")"
			if i == f.tab {
				tabs = append(tabs, accentStyle.Bold(true).Render("▸ "+label))
			} else {
				tabs = append(tabs, faintStyle.Render("  "+label))
			}
		}
		head = append(head, strings.Join(tabs, "   "))
	}
	head = append(head, wrap.Render(f.run.label(r.target)+faintStyle.Render(" · desde "+r.rep.Since)))
	for _, s := range r.rep.Skipped {
		head = append(head, wrap.Render(warnStyle.Render("⚠ Detector "+s.Detector+" omitido: "+s.Reason)))
	}
	head = append(head, wrap.Render(faintStyle.Render(f.summary(r.rep))))
	for _, p := range f.saved {
		head = append(head, wrap.Render(okStyle.Render("✓ Reporte guardado en ")+accentStyle.Render(p)))
	}
	if f.note != "" {
		head = append(head, wrap.Render(errStyle.Render("✗ "+f.note)))
	}
	head = append(head, "")
	rows := f.rows()
	if len(rows) == 0 {
		empty := "Ningún hallazgo coincide"
		if len(r.rep.Findings) == 0 {
			empty = "Sin hallazgos en el período"
		}
		return pad(lipgloss.JoinVertical(lipgloss.Left, append(head, f.filter.line(), okStyle.Render("✓ ")+empty)...))
	}
	cur := min(f.cursor, len(rows)-1)

	if width >= panelMinWidth {
		boxes := max(height-lipgloss.Height(strings.Join(head, "\n")), 6)
		cols := panels(width, boxes, 6, 4,
			func(w, lines int) string { return f.listText(rows, cur, w, lines) },
			func(w, _ int) string { return f.detailText(rows[cur], w) })
		return lipgloss.JoinVertical(lipgloss.Left, lipgloss.NewStyle().PaddingLeft(1).Render(strings.Join(head, "\n")), cols)
	}
	head = append(head, f.filter.line())
	room := max(height-lipgloss.Height(strings.Join(head, "\n"))-1, 3)
	detailHeight := min(findingsDetailLines, room/2)
	list := f.list(rows, cur, room-detailHeight-3, width-2)
	detail := boxStyle.Render(lipgloss.NewStyle().MaxHeight(max(detailHeight, 3)).Render(f.detailText(rows[cur], width-6)))
	return pad(lipgloss.JoinVertical(lipgloss.Left, append(head, list, detail)...))
}

// listText is the filter line and the findings that fit in lines lines.
func (f *findings) listText(rows []backlog.Finding, cur, width, lines int) string {
	return f.filter.line() + "\n" + f.list(rows, cur, max(lines-1, 1), width)
}

// summary is the count, points and modules of a report.
func (f *findings) summary(rep *backlog.Report) string {
	points := 0
	for _, fd := range rep.Findings {
		points += fd.Points
	}
	s := plural(rep.Total, "hallazgo", "hallazgos") + " · " + strconv.Itoa(points) + " pts sugeridos"
	if len(rep.ModuleFilter) > 0 {
		s += " · módulos " + strings.Join(rep.ModuleFilter, ", ")
	}
	if len(rep.Modules) > 0 {
		s += " · por módulo: " + backlog.ModuleSummary(rep.Modules)
	}
	return s
}

// list draws the window of findings around the cursor, one line each.
func (f *findings) list(rows []backlog.Finding, cur, height, width int) string {
	row := lipgloss.NewStyle().MaxWidth(width)
	var out []string
	start := window(cur, len(rows), height)
	for i := start; i < min(start+height, len(rows)); i++ {
		fd := rows[i]
		mark, title := "  ", fd.Title
		if fd.Module != "" {
			title = fd.Module + " · " + title
		}
		if i == cur {
			mark, title = accentStyle.Render("▸ "), accentStyle.Bold(true).Render(title)
		}
		sev := severityStyle(fd.Severity).Render("[" + string(fd.Severity) + strings.Repeat(" ", 5-len(fd.Severity)) + "]")
		pts := boldStyle.Render(strconv.Itoa(fd.Points) + " pts")
		out = append(out, row.Render(mark+sev+" "+pts+"  "+title))
	}
	return lipgloss.JoinVertical(lipgloss.Left, out...)
}

// detailText is everything about a finding: kind, files and evidence.
func (f *findings) detailText(fd backlog.Finding, width int) string {
	text := lipgloss.NewStyle().Width(max(width, 10))
	lines := []string{
		text.Render(boldStyle.Render(fd.Title)),
		faintStyle.Render(string(fd.Kind)+" · "+fd.ID) + "  " + severityStyle(fd.Severity).Render(string(fd.Severity)) + faintStyle.Render(" · "+strconv.Itoa(fd.Points)+" pts"),
	}
	if fd.Module != "" {
		lines = append(lines, faintStyle.Render("Módulo ")+fd.Module)
	}
	for _, file := range fd.Files {
		lines = append(lines, text.Render(filepath.FromSlash(file)))
	}
	for _, e := range fd.Evidence {
		lines = append(lines, text.Render(faintStyle.Render("· "+e)))
	}
	return strings.Join(lines, "\n")
}

func (f *findings) moreKeys() []key.Binding { return []key.Binding{filterKey, upKey, downKey} }
