package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/fetch"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

var (
	saveKey    = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "guardar"))
	replaceKey = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "reemplazar"))
)

// exportForm asks for the task and where to look; the profile opens here.
type exportForm struct {
	st    *state
	issue string
	space string
	form  *huh.Form
	err   string
}

func newExportForm(st *state) *exportForm {
	f := &exportForm{st: st}
	f.form = f.build()
	return f
}

func (f *exportForm) build() *huh.Form {
	return newForm(taskInput(&f.issue),
		huh.NewInput().Title("Space o carpeta").Description("Vacío: el del perfil (en Confluence sin space, todos)").Value(&f.space))
}

func (f *exportForm) title() string       { return "Exportar página" }
func (f *exportForm) keys() []key.Binding { return []key.Binding{nextFieldKey, prevFieldKey} }

func (f *exportForm) captures(k tea.KeyPressMsg) bool { return !key.Matches(k, backKey) }

func (f *exportForm) init() tea.Cmd {
	if f.form.State != huh.StateNormal {
		f.form = f.build()
	}
	return f.form.Init()
}

func (f *exportForm) update(msg tea.Msg) (screen, tea.Cmd) {
	m, cmd := f.form.Update(msg)
	f.form = m.(*huh.Form)
	if f.form.State != huh.StateCompleted {
		return f, cmd
	}
	next, err := f.next()
	if err != nil {
		f.err = err.Error()
		f.form = f.build()
		return f, f.form.Init()
	}
	f.err = ""
	return f, open(next)
}

func (f *exportForm) next() (screen, error) {
	if f.st.cfg == nil {
		return nil, errors.New(f.st.notice)
	}
	target, err := fetch.Resolve(f.st.cfg, fetch.Request{Issue: f.issue, Space: strings.TrimSpace(f.space)})
	if err != nil {
		return nil, err
	}
	bundle, err := f.st.deps.OpenProfile(f.st.cfg, profile.Options{Scope: target.SpaceKey, AnyScope: true})
	if err != nil {
		return nil, explainProfile(err)
	}
	return newExportDocs(&exportRun{bundle: bundle, session: fetch.NewSession(bundle.Docs, target)}), nil
}

func (f *exportForm) view(width, height int) string { return formView(f.form, f.err, width) }

// exportRun is what the screens of Exportar share after the form.
type exportRun struct {
	bundle  *profile.Bundle
	session *fetch.Session
}

// where names the destination searched: its provider and scope.
func (r *exportRun) where() string {
	d := r.bundle.Profile.Docs
	scope := r.session.Target.SpaceKey
	switch {
	case scope != "":
	case d.Provider == "markdown":
		scope = d.Path
	default:
		scope = d.Space
	}
	if scope == "" {
		return d.Provider + " (todos los spaces)"
	}
	return d.Provider + " · " + scope
}

// command is the same export in the command mode.
func (r *exportRun) command(path string) string {
	c := "draco f -i " + r.session.Target.Issue
	if s := r.session.Target.SpaceKey; s != "" {
		c += " -s " + shellQuote(s)
	}
	return c + " -o " + shellQuote(path)
}

func shellQuote(s string) string {
	if strings.ContainsAny(s, " \t'\"") {
		return strconv.Quote(s)
	}
	return s
}

type searchedMsg struct {
	owner *exportDocs
	err   error
}

// exportDocs lists the documents that mention the task. With only one, it
// goes on to the text without asking, as draco f does.
type exportDocs struct {
	run     *exportRun
	done    bool
	err     error
	started bool
	cursor  int
	filter  filter
	spinner spinner.Model
	ctx     context.Context
	cancel  context.CancelFunc
}

func newExportDocs(r *exportRun) *exportDocs {
	d := &exportDocs{run: r, spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	d.ctx, d.cancel = context.WithCancel(context.Background())
	return d
}

func (d *exportDocs) title() string { return "Documentos de " + d.run.session.Target.Issue }
func (d *exportDocs) close()        { d.cancel() }

func (d *exportDocs) keys() []key.Binding {
	if !d.done || d.err != nil {
		return nil
	}
	return []key.Binding{moveKey, pickKey}
}

func (d *exportDocs) captures(k tea.KeyPressMsg) bool { return d.filter.captures(k) }

func (d *exportDocs) init() tea.Cmd {
	if d.started {
		return nil
	}
	d.started = true
	s, ctx := d.run.session, d.ctx
	return tea.Batch(d.spinner.Tick, func() tea.Msg {
		return searchedMsg{owner: d, err: s.Search(ctx)}
	})
}

// pages are the documents that match the filter, most recent first.
func (d *exportDocs) pages() []provider.DocRef {
	var out []provider.DocRef
	for _, p := range d.run.session.Pages {
		if d.filter.matches(p.Title) {
			out = append(out, p)
		}
	}
	return out
}

func (d *exportDocs) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case searchedMsg:
		if msg.owner != d || d.ctx.Err() != nil {
			return d, nil
		}
		d.done, d.err = true, msg.err
		if d.err == nil && len(d.run.session.Pages) == 1 {
			return d, open(newExportText(d.run, d.run.session.Pages[0]))
		}
	case spinner.TickMsg:
		if d.done {
			return d, nil
		}
		var cmd tea.Cmd
		d.spinner, cmd = d.spinner.Update(msg)
		return d, cmd
	case tea.KeyPressMsg:
		if !d.done || d.err != nil {
			return d, nil
		}
		if d.filter.handle(msg) {
			d.cursor = 0
			return d, nil
		}
		pages := d.pages()
		switch {
		case key.Matches(msg, upKey):
			d.cursor = max(d.cursor-1, 0)
		case key.Matches(msg, downKey):
			d.cursor = min(d.cursor+1, max(len(pages)-1, 0))
		case key.Matches(msg, pickKey) && len(pages) > 0:
			return d, open(newExportText(d.run, pages[min(d.cursor, len(pages)-1)]))
		}
	}
	return d, nil
}

func (d *exportDocs) view(width, height int) string {
	issue := d.run.session.Target.Issue
	switch {
	case d.err != nil:
		return pad(lipgloss.NewStyle().Width(width - 2).Render(errStyle.Render("✗ " + d.explain().Error())))
	case !d.done:
		return pad(d.spinner.View() + " Buscando documentos que mencionen " + issue + " en " + d.run.where() + "…")
	}
	summary := faintStyle.Render(plural(len(d.run.session.Pages), "documento menciona", "documentos mencionan") + " " + issue + " en " + d.run.where() + ", el más reciente primero.")
	pages := d.pages()
	if width >= panelMinWidth {
		head := lipgloss.NewStyle().Width(width - 2).PaddingLeft(1).Render(summary)
		cols := panels(width, max(height-lipgloss.Height(head)-1, 6), 6, 4,
			func(w, lines int) string { return d.listText(pages, w, lines) },
			func(w, _ int) string { return d.detailText(pages, w) })
		return lipgloss.JoinVertical(lipgloss.Left, head, "", cols)
	}
	lines := []string{summary, "", d.filter.line()}
	if len(pages) == 0 {
		lines = append(lines, faintStyle.Render("Ningún documento coincide"))
	}
	lines = append(lines, docRows(pages, d.cursor, max((height-len(lines)-2)/2, 1), width-2)...)
	return pad(lipgloss.NewStyle().Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

// listText is the filter line and the documents that fit in lines lines.
func (d *exportDocs) listText(pages []provider.DocRef, width, lines int) string {
	rows := []string{d.filter.line()}
	if len(pages) == 0 {
		rows = append(rows, faintStyle.Render("Ningún documento coincide"))
	}
	return strings.Join(append(rows, docRows(pages, d.cursor, max((lines-len(rows))/2, 1), width)...), "\n")
}

// detailText is the document under the cursor and what enter does with it.
func (d *exportDocs) detailText(pages []provider.DocRef, width int) string {
	if len(pages) == 0 {
		return faintStyle.Render("Ningún documento coincide con el filtro.")
	}
	p := pages[min(d.cursor, len(pages)-1)]
	lines := []string{boldStyle.Render(p.Title)}
	if p.Version > 0 {
		lines = append(lines, faintStyle.Render("Versión "+strconv.Itoa(p.Version)))
	}
	lines = append(lines, accentStyle.Render(p.URL), "", faintStyle.Render("enter lee el documento y propone dónde guardarlo como texto."))
	return strings.Join(lines, "\n")
}

func (d *exportDocs) moreKeys() []key.Binding {
	if !d.done || d.err != nil {
		return nil
	}
	return []key.Binding{filterKey, upKey, downKey}
}

func (d *exportDocs) explain() error {
	if errors.Is(d.err, fetch.ErrNoPages) {
		return fmt.Errorf("%w (se buscó en %s): prueba con otro space o carpeta", d.err, d.run.where())
	}
	return d.err
}

type exportedMsg struct {
	owner *exportText
	exp   *fetch.Export
	err   error
}

// exportText shows the document as text and saves it where the user says.
type exportText struct {
	run     *exportRun
	page    provider.DocRef
	exp     *fetch.Export
	err     error
	started bool
	path    textinput.Model
	confirm bool // the file exists: s replaces it
	note    string
	saved   string
	vp      viewport.Model
	spinner spinner.Model
	ctx     context.Context
	cancel  context.CancelFunc
}

func newExportText(r *exportRun, page provider.DocRef) *exportText {
	t := &exportText{run: r, page: page, path: textinput.New(), vp: viewport.New(), spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	t.path.Prompt = ""
	t.vp.SoftWrap = true
	t.ctx, t.cancel = context.WithCancel(context.Background())
	return t
}

func (t *exportText) title() string { return "Exportar" }
func (t *exportText) close()        { t.cancel() }

func (t *exportText) keys() []key.Binding {
	switch {
	case t.exp == nil:
		return nil
	case t.saved != "":
		return []key.Binding{homeKey}
	case t.confirm:
		return []key.Binding{replaceKey}
	}
	return []key.Binding{scrollKey, saveKey}
}

// captures keeps the keys for the path; esc too while confirming or done.
func (t *exportText) captures(k tea.KeyPressMsg) bool {
	switch {
	case t.exp == nil:
		return false
	case t.confirm || t.saved != "":
		return key.Matches(k, backKey)
	}
	return !key.Matches(k, backKey)
}

func (t *exportText) init() tea.Cmd {
	if t.started {
		return nil
	}
	t.started = true
	s, ctx, page := t.run.session, t.ctx, t.page
	return tea.Batch(t.spinner.Tick, func() tea.Msg {
		exp, err := s.Export(ctx, &page)
		return exportedMsg{owner: t, exp: exp, err: err}
	})
}

func (t *exportText) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case exportedMsg:
		if msg.owner != t || t.ctx.Err() != nil {
			return t, nil
		}
		t.exp, t.err = msg.exp, msg.err
		if t.exp == nil {
			return t, nil
		}
		t.vp.SetContent(t.exp.Text)
		t.path.SetValue(defaultExportPath(t.exp.FileName))
		return t, t.path.Focus()
	case spinner.TickMsg:
		if t.exp != nil || t.err != nil {
			return t, nil
		}
		var cmd tea.Cmd
		t.spinner, cmd = t.spinner.Update(msg)
		return t, cmd
	case tea.KeyPressMsg:
		return t.key(msg)
	}
	return t, nil
}

func (t *exportText) key(k tea.KeyPressMsg) (screen, tea.Cmd) {
	switch {
	case t.exp == nil:
	case t.saved != "":
		if key.Matches(k, homeKey) {
			return t, toHome
		}
	case t.confirm:
		switch {
		case key.Matches(k, replaceKey):
			t.save()
		case key.Matches(k, backKey):
			t.confirm = false
		}
	case key.Matches(k, saveKey):
		path := strings.TrimSpace(t.path.Value())
		switch _, err := os.Stat(path); {
		case path == "":
			t.note = "Falta la ruta del archivo"
		case err == nil:
			t.confirm, t.note = true, ""
		default:
			t.save()
		}
	case key.Matches(k, scrollKey):
		var cmd tea.Cmd
		t.vp, cmd = t.vp.Update(k)
		return t, cmd
	default:
		t.note = ""
		var cmd tea.Cmd
		t.path, cmd = t.path.Update(k)
		return t, cmd
	}
	return t, nil
}

func (t *exportText) save() {
	path := strings.TrimSpace(t.path.Value())
	t.confirm = false
	if err := t.exp.Save(path); err != nil {
		t.note = err.Error()
		return
	}
	t.saved, t.note = path, ""
	t.path.Blur()
}

// defaultExportPath is the suggested name in the current folder.
func defaultExportPath(name string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return name
	}
	return filepath.Join(cwd, name)
}

func (t *exportText) view(width, height int) string {
	wrap := lipgloss.NewStyle().Width(width - 2)
	switch {
	case t.err != nil:
		return pad(wrap.Render(errStyle.Render("✗ " + t.err.Error())))
	case t.exp == nil:
		return pad(t.spinner.View() + " Descargando «" + t.page.Title + "»…")
	}
	head := []string{boldStyle.Render(t.exp.Doc.Title), faintStyle.Render(strconv.Itoa(len(t.exp.Text)) + " caracteres · " + t.run.where())}

	var foot []string
	switch {
	case t.saved != "":
		foot = append(foot, okStyle.Bold(true).Render("✓ Exportado"), "  "+accentStyle.Render(t.saved), "",
			faintStyle.Render("Lo mismo en modo comando:"), "$ "+t.run.command(t.saved))
	case t.confirm:
		foot = append(foot, warnStyle.Render("⚠ Ya existe "+strings.TrimSpace(t.path.Value())+". s lo reemplaza · esc elige otra ruta"))
	default:
		t.path.SetWidth(max(width-16, 10))
		foot = append(foot, faintStyle.Render("Guardar en ")+t.path.View())
	}
	if t.note != "" {
		foot = append(foot, errStyle.Render("✗ "+t.note))
	}
	for i, l := range foot {
		foot[i] = wrap.Render(l)
	}

	t.vp.SetWidth(max(width-6, 10))
	t.vp.SetHeight(max(height-len(head)-lipgloss.Height(strings.Join(foot, "\n"))-5, 3))
	body := append(head, boxStyle.Render(t.vp.View()))
	return pad(lipgloss.JoinVertical(lipgloss.Left, append(body, foot...)...))
}
