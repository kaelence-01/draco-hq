package tui

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/profile"
)

type step int

const (
	stepFetch step = iota
	stepFind
	stepChoose // the user decides about an existing document
	stepRead
	stepDone
)

type stepDoneMsg struct {
	owner *progress
	step  step
	err   error
}

var (
	updateKey  = key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "actualizar"))
	createKey  = key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "crear otro"))
	discardKey = key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "cancelar"))
	previewKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "vista previa"))
)

// progress runs the steps of deploy.Session up to the files of the commits.
// While a step runs, the Session is not read: the step writes it from its own
// goroutine.
type progress struct {
	st      *state
	bundle  *profile.Bundle
	project string
	back    []string
	front   []string
	issue   string
	target  *deploy.Target
	session *deploy.Session
	step    step
	err     error
	replace bool // the user chose to update the existing document
	started bool
	spinner spinner.Model
	ctx     context.Context
	cancel  context.CancelFunc
}

func newProgress(st *state, bundle *profile.Bundle, project, issue string, back, front []string) *progress {
	p := &progress{st: st, bundle: bundle, project: project, issue: issue, back: back, front: front,
		spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p
}

func (p *progress) title() string { return "Preparar documento" }

func (p *progress) keys() []key.Binding {
	switch {
	case p.err != nil:
	case p.step == stepChoose:
		return []key.Binding{updateKey, createKey, discardKey}
	case p.step == stepDone:
		return []key.Binding{previewKey}
	}
	return nil
}

func (p *progress) close() { p.cancel() }

func (p *progress) init() tea.Cmd {
	if p.started {
		return nil
	}
	p.started = true
	var err error
	p.target, err = deploy.Resolve(p.st.cfg, deploy.Request{
		Issue: p.issue, BackendHashes: p.back, FrontendHashes: p.front,
		Project: p.project, Template: p.bundle.Profile.TemplateDir("deploy"),
	})
	if err != nil {
		p.err = err
		return nil
	}
	p.session = deploy.NewSession(p.bundle.Tracker, p.bundle.Docs, p.bundle.Changes, p.target)
	return tea.Batch(p.spinner.Tick, p.run(stepFetch))
}

func (p *progress) run(st step) tea.Cmd {
	p.step = st
	s, ctx := p.session, p.ctx
	return func() tea.Msg {
		var err error
		switch st {
		case stepFetch:
			err = s.FetchIssue(ctx)
		case stepFind:
			err = s.FindExisting(ctx)
		case stepRead:
			err = s.ReadCommits(ctx)
		}
		return stepDoneMsg{owner: p, step: st, err: err}
	}
}

func (p *progress) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case stepDoneMsg:
		if msg.owner != p || p.ctx.Err() != nil {
			return p, nil
		}
		if msg.err != nil {
			p.err = msg.err
			return p, nil
		}
		switch msg.step {
		case stepFetch:
			return p, p.run(stepFind)
		case stepFind:
			if p.session.Existing != nil {
				p.step = stepChoose
				return p, nil
			}
			return p, p.run(stepRead)
		case stepRead:
			p.step = stepDone
		}
	case spinner.TickMsg:
		if p.step == stepDone || p.step == stepChoose || p.err != nil {
			return p, nil
		}
		var cmd tea.Cmd
		p.spinner, cmd = p.spinner.Update(msg)
		return p, cmd
	case tea.KeyPressMsg:
		if p.step == stepDone && key.Matches(msg, previewKey) {
			return p, open(newPreview(&genRun{st: p.st, bundle: p.bundle, session: p.session, replace: p.replace, command: p.command()}))
		}
		if p.step != stepChoose || p.err != nil {
			return p, nil
		}
		switch {
		case key.Matches(msg, updateKey):
			p.replace = true
		case key.Matches(msg, createKey):
		case key.Matches(msg, discardKey):
			return p, back
		default:
			return p, nil
		}
		return p, tea.Batch(p.spinner.Tick, p.run(stepRead))
	}
	return p, nil
}

func (p *progress) view(width, height int) string {
	var lines []string
	if p.session == nil {
		lines = append(lines, errStyle.Render("✗ "+p.err.Error()))
		return p.frame(width, lines)
	}
	for _, w := range p.target.Warnings {
		lines = append(lines, warnStyle.Render("⚠ "+w))
	}
	if len(p.target.Warnings) > 0 {
		lines = append(lines, "")
	}

	lines = append(lines, p.line(stepFetch, "Buscando la tarea "+p.issue, func() string {
		return boldStyle.Render(p.session.Issue.Key) + " — " + p.session.Issue.Summary
	}))
	if p.step >= stepFind {
		lines = append(lines, p.line(stepFind, "Buscando un documento existente", func() string {
			if p.session.Existing == nil {
				return "Sin documento previo: se creará uno nuevo"
			}
			return "Ya existe: " + p.session.Existing.Title
		}))
	}
	if p.step == stepChoose && p.err == nil {
		ex := p.session.Existing
		lines = append(lines, "", "   "+faintStyle.Render(ex.URL), "",
			"   "+warnStyle.Render("¿Qué hacer con el documento existente?")+"  1 actualizar · 2 crear otro · 3 cancelar")
	}
	if p.step >= stepRead {
		if p.replace {
			lines = append(lines, "  "+faintStyle.Render("se actualizará el documento existente"))
		}
		lines = append(lines, p.line(stepRead, "Leyendo los archivos de los commits", func() string { return "Archivos leídos" }))
	}
	if p.step == stepDone {
		lines = append(lines, "")
		lines = append(lines, readFiles(p.session)...)
		lines = append(lines, "", faintStyle.Render("enter abre la vista previa; nada se publica hasta que lo confirmes."))
	}
	return p.frame(width, lines)
}

func (p *progress) frame(width int, lines []string) string {
	return lipgloss.NewStyle().Padding(1, 1, 0).Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// line is a step: running, failed or done with what it found.
func (p *progress) line(st step, running string, done func() string) string {
	switch {
	case p.step == st && p.err != nil:
		return errStyle.Render("✗ ") + running + "\n  " + errStyle.Render(p.err.Error())
	case p.step == st:
		return p.spinner.View() + " " + running + "…"
	default:
		return okStyle.Render("✓ ") + done()
	}
}

// readFiles lists what each side read, by folder.
func readFiles(s *deploy.Session) []string {
	var out []string
	for _, sd := range []struct {
		label string
		repo  deploy.Repo
		c     *deploy.Commits
	}{{"Backend", s.Target.Backend, s.Backend}, {"Frontend", s.Target.Frontend, s.Frontend}} {
		switch {
		case sd.c == nil:
		case sd.c.Err != nil:
			out = append(out, errStyle.Render("✗ "+sd.label+" ["+strings.Join(sd.c.Hashes, ", ")+"]: ")+sd.c.Err.Error())
		default:
			out = append(out, accentStyle.Render(sd.label+" · "+sd.repo.Name)+faintStyle.Render("  "+strings.Join(sd.c.Hashes, ", ")+" · "+plural(sd.c.Count, "archivo", "archivos")))
			dirs := make([]string, 0, len(sd.c.Files))
			for d := range sd.c.Files {
				dirs = append(dirs, d)
			}
			sort.Strings(dirs)
			for _, d := range dirs {
				out = append(out, "  "+faintStyle.Render(d+"/")+" "+strings.Join(sd.c.Files[d], ", "))
			}
		}
	}
	if s.PartialFailure() {
		out = append(out, warnStyle.Render("⚠ Uno de los lados falló: el documento se arma con lo que se pudo leer."))
	}
	return out
}

// command is the same run in the command mode.
func (p *progress) command() string {
	parts := []string{"draco g -i " + p.issue}
	if len(p.back) > 0 {
		parts = append(parts, "-b "+strings.Join(p.back, ","))
	}
	if len(p.front) > 0 {
		parts = append(parts, "-f "+strings.Join(p.front, ","))
	}
	if p.project != "" {
		parts = append(parts, "-p "+p.project)
	}
	return strings.Join(parts, " ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}
