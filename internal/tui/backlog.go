package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/backlog"
	"github.com/kaelence-01/draco-hq/internal/config"
)

var findingsKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "ver hallazgos"))

// backlogForm asks what to scan. It needs no profile nor credentials.
type backlogForm struct {
	st      *state
	project string
	repo    string
	path    string
	since   string
	modules string
	deps    bool
	form    *huh.Form
	err     string
}

func newBacklogForm(st *state) *backlogForm {
	f := &backlogForm{st: st, project: st.project}
	f.form = f.build()
	return f
}

func (f *backlogForm) build() *huh.Form {
	var what []huh.Field
	if f.st.cfg != nil && len(f.st.cfg.Projects) > 0 {
		what = append(what, huh.NewSelect[string]().Title("Proyecto").Options(huh.NewOptions(f.st.cfg.ProjectNames()...)...).Value(&f.project))
	}
	what = append(what,
		huh.NewSelect[string]().Title("Repos").Options(
			huh.NewOption("Todos los del proyecto", ""), huh.NewOption("Backend", "backend"), huh.NewOption("Frontend", "frontend"),
		).Value(&f.repo),
		huh.NewInput().Title("Carpeta").Placeholder("en lugar del proyecto (opcional)").Value(&f.path),
	)
	return newPagedForm(huh.NewGroup(what...), huh.NewGroup(
		huh.NewInput().Title("Período").Placeholder("30d, 12w, 3m o una fecha (vacío: el del proyecto o 90d)").Value(&f.since),
		huh.NewInput().Title("Módulos").Placeholder("Pagos, Facturacion (requiere modules en el proyecto)").Value(&f.modules),
		huh.NewConfirm().Title("¿Revisar dependencias con composer audit? (usa internet)").Affirmative("Sí").Negative("No").Value(&f.deps),
	))
}

func (f *backlogForm) title() string       { return "Backlog técnico" }
func (f *backlogForm) keys() []key.Binding { return []key.Binding{nextFieldKey, prevFieldKey} }

func (f *backlogForm) captures(k tea.KeyPressMsg) bool { return !key.Matches(k, backKey) }

func (f *backlogForm) init() tea.Cmd {
	if f.form.State != huh.StateNormal {
		f.form = f.build()
	}
	return f.form.Init()
}

func (f *backlogForm) update(msg tea.Msg) (screen, tea.Cmd) {
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

func (f *backlogForm) next() (screen, error) {
	r := &backlogRun{st: f.st, repo: f.repo, path: strings.TrimSpace(f.path), since: strings.TrimSpace(f.since),
		modules: backlog.SplitModules(f.modules), deps: f.deps}
	if f.st.cfg != nil {
		if p, ok := f.st.cfg.Projects[f.project]; ok {
			r.proj, r.projName = p, f.project
		}
	}
	var err error
	if r.targets, err = backlog.Targets(r.proj, r.repo, r.path); errors.Is(err, backlog.ErrNoRepoPath) {
		return nil, fmt.Errorf("el proyecto %s no tiene %s_path: elige otro repo o una carpeta, o agrégalo en Proyectos", r.projName, r.repo)
	}
	if err != nil {
		return nil, err
	}
	return newScanning(r), nil
}

func (f *backlogForm) view(width, height int) string { return formView(f.form, f.err, width) }

// backlogRun is a scan asked for in the form, resolved to its repos.
type backlogRun struct {
	st       *state
	proj     *config.ProjectConfig // nil without a project
	projName string
	repo     string
	path     string
	since    string
	modules  []string
	deps     bool
	targets  []backlog.Target
}

// label names a repo of the scan: the repo and its folder.
func (r *backlogRun) label(t backlog.Target) string { return t.Repo + " · " + t.Dir }

// command is the same scan in the command mode, with every finding.
func (r *backlogRun) command() string {
	parts := []string{"draco backlog scan"}
	if r.projName != "" {
		parts = append(parts, "-p "+r.projName)
	}
	if r.repo != "" {
		parts = append(parts, "-r "+r.repo)
	}
	if r.path != "" {
		parts = append(parts, "--path "+shellQuote(r.path))
	}
	if r.since != "" {
		parts = append(parts, "--since "+shellQuote(r.since))
	}
	if len(r.modules) > 0 {
		parts = append(parts, "-m "+shellQuote(strings.Join(r.modules, ",")))
	}
	if r.deps {
		parts = append(parts, "--deps")
	}
	return strings.Join(append(parts, "-l 0"), " ")
}

// scanResult is one repo scanned: its report, or why it failed.
type scanResult struct {
	target backlog.Target
	rep    *backlog.Report
	err    error
}

type scannedMsg struct {
	owner *scanning
	res   scanResult
}

// scanning runs the scan of each repo, one after the other. A repo that fails
// does not stop the rest, as in draco backlog scan.
type scanning struct {
	run     *backlogRun
	results []scanResult // in the order of run.targets, as they finish
	started bool
	spinner spinner.Model
	ctx     context.Context
	cancel  context.CancelFunc
}

func newScanning(r *backlogRun) *scanning {
	s := &scanning{run: r, spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s
}

func (s *scanning) title() string { return "Analizar" }
func (s *scanning) close()        { s.cancel() }

func (s *scanning) done() bool { return len(s.results) == len(s.run.targets) }

func (s *scanning) keys() []key.Binding {
	if s.done() && len(s.reports()) > 0 {
		return []key.Binding{findingsKey}
	}
	return nil
}

func (s *scanning) init() tea.Cmd {
	if s.started {
		return nil
	}
	s.started = true
	return tea.Batch(s.spinner.Tick, s.scan(0))
}

func (s *scanning) scan(i int) tea.Cmd {
	r, ctx, t := s.run, s.ctx, s.run.targets[i]
	scan := r.st.deps.Scan
	return func() tea.Msg {
		opts := backlog.ScanOptions{Options: backlog.OptionsFor(r.proj.BacklogFor(t.Repo), r.since), Dependencies: r.deps}
		rep, err := scan(ctx, t.Dir, opts)
		if err == nil {
			if err = backlog.ApplyModules(rep, r.modules); err != nil {
				rep, err = nil, fmt.Errorf("%w: agrega backlog.%s.modules al proyecto (draco help backlog)", err, t.Repo)
			}
		}
		return scannedMsg{owner: s, res: scanResult{target: t, rep: rep, err: err}}
	}
}

// reports are the repos that were scanned.
func (s *scanning) reports() []scanResult {
	var out []scanResult
	for _, r := range s.results {
		if r.err == nil {
			out = append(out, r)
		}
	}
	return out
}

func (s *scanning) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case scannedMsg:
		if msg.owner != s || s.ctx.Err() != nil {
			return s, nil
		}
		s.results = append(s.results, msg.res)
		if !s.done() {
			return s, s.scan(len(s.results))
		}
	case spinner.TickMsg:
		if s.done() {
			return s, nil
		}
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	case tea.KeyPressMsg:
		if s.done() && len(s.reports()) > 0 && key.Matches(msg, findingsKey) {
			return s, open(newFindings(s.run, s.reports()))
		}
	}
	return s, nil
}

func (s *scanning) view(width, height int) string {
	lines := []string{faintStyle.Render("Solo lee los repos: no modifica nada ni necesita credenciales."), ""}
	for i, t := range s.run.targets {
		switch {
		case i < len(s.results) && s.results[i].err != nil:
			lines = append(lines, errStyle.Render("✗ "+s.run.label(t)+": "+s.results[i].err.Error()))
		case i < len(s.results):
			lines = append(lines, okStyle.Render("✓ ")+s.run.label(t)+faintStyle.Render(" · "+plural(s.results[i].rep.Total, "hallazgo", "hallazgos")))
		case i == len(s.results):
			lines = append(lines, s.spinner.View()+" Analizando "+s.run.label(t)+"…")
		default:
			lines = append(lines, faintStyle.Render("  "+s.run.label(t)))
		}
	}
	if s.done() {
		lines = append(lines, "", faintStyle.Render("Lo mismo en modo comando:"), "$ "+s.run.command())
	}
	return pad(lipgloss.NewStyle().Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}
