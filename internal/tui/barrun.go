package tui

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/deploy"
)

// barFlags are the flags each screen takes from the command bar; the others
// only make sense in the command mode.
var barFlags = map[string][]string{
	"generate": {"--issue", "--commit-backend", "--commit-frontend", "--project"},
	"qa":       {"--sprint", "--module"},
	"fetch":    {"--issue", "--space"},
	"backlog":  {"--project", "--repo", "--path", "--since", "--modulo", "--deps"},
}

// openFromBar opens the screen of a command with the values of its line, at
// the first step the user reviews: a form whose values are enough is sent,
// and an error stays on the form, as if the user had sent it.
func openFromBar(st *state, l barLine) (tea.Cmd, error) {
	if err := barUnsupported(l); err != nil {
		return nil, err
	}
	name, f := l.cmd.Names[0], l.flags
	// A form's list would replace an unknown project with the first one.
	if p := f.Get("--project"); p != "" && st.cfg != nil {
		if _, _, err := st.cfg.GetProject(p); err != nil {
			return nil, err
		}
	}
	switch name {
	case "generate":
		return openGenerate(st, f), nil
	case "qa":
		return openQA(st, f), nil
	case "fetch":
		return openExport(st, f), nil
	case "backlog":
		return openBacklog(st, f, l.sub != nil), nil
	case "project":
		ps := newProjects(st)
		if l.sub != nil && l.sub.Names[0] == "add" {
			return tea.Sequence(open(ps), open(newProjectForm(ps, ""))), nil
		}
		return open(ps), nil
	case "init", "profile":
		return openSettings(st, name, l.sub), nil
	}
	return nil, fmt.Errorf("«%s» se corre fuera del modo visual: sal con q y usa draco %s", name, name)
}

// barUnsupported explains the first flag of the line, in alphabetical order,
// that its screen does not take.
func barUnsupported(l barLine) error {
	name := l.cmd.Names[0]
	for _, long := range slices.Sorted(maps.Keys(l.flags.Values)) {
		switch {
		case slices.Contains(barFlags[name], long):
		case long == "--dry-run":
			return fmt.Errorf("--dry-run sobra en el modo visual: la vista previa no publica nada")
		default:
			return fmt.Errorf("%s no está en el modo visual: usa draco %s en la terminal", long, name)
		}
	}
	return nil
}

func openGenerate(st *state, f *cli.Parsed) tea.Cmd {
	g := newGenerate(st)
	g.issue = strings.TrimSpace(f.Get("--issue"))
	if p := f.Get("--project"); p != "" {
		g.project = p
	}
	if g.issue == "" {
		g.form = g.build()
		return open(g)
	}
	next, err := g.next()
	g.form = g.build()
	if err != nil {
		g.err = err.Error()
		return open(g)
	}
	next.(*commits).preset(deploy.SplitHashes(f.Get("--commit-backend")), deploy.SplitHashes(f.Get("--commit-frontend")))
	return tea.Sequence(open(g), open(next))
}

func openQA(st *state, f *cli.Parsed) tea.Cmd {
	q := newQAForm(st)
	q.sprint, q.module = f.Get("--sprint"), f.Get("--module")
	q.form = q.build()
	if err := validSprint(q.sprint); err != nil {
		q.err = err.Error()
		return open(q)
	}
	next, err := q.next()
	if err != nil {
		q.err = err.Error()
		return open(q)
	}
	return tea.Sequence(open(q), open(next))
}

func openExport(st *state, f *cli.Parsed) tea.Cmd {
	e := newExportForm(st)
	e.issue, e.space = strings.TrimSpace(f.Get("--issue")), f.Get("--space")
	e.form = e.build()
	if e.issue == "" {
		return open(e)
	}
	next, err := e.next()
	if err != nil {
		e.err = err.Error()
		return open(e)
	}
	return tea.Sequence(open(e), open(next))
}

// openBacklog sends the form only with scan, as draco backlog without it
// shows its help.
func openBacklog(st *state, f *cli.Parsed, scan bool) tea.Cmd {
	b := newBacklogForm(st)
	if p := f.Get("--project"); p != "" {
		b.project = p
	}
	b.repo, b.path, b.since, b.modules, b.deps = f.Get("--repo"), f.Get("--path"), f.Get("--since"), f.Get("--modulo"), f.Has("--deps")
	if !scan {
		b.form = b.build()
		return open(b)
	}
	next, err := b.next()
	b.form = b.build()
	if err != nil {
		b.err = err.Error()
		return open(b)
	}
	return tea.Sequence(open(b), open(next))
}

// openSettings opens Configuración at the connection (init) or the profiles.
func openSettings(st *state, name string, sub *cli.Command) tea.Cmd {
	s := newSettings(st)
	s.started = true
	if st.cfg == nil {
		return open(s)
	}
	if name == "init" {
		return tea.Sequence(open(s), open(newConnection(s)))
	}
	ps := newProfiles(st)
	if sub != nil && sub.Names[0] == "add" {
		return tea.Sequence(open(s), open(ps), open(newProfileForm(st, "", profileNew, ps.done, back)))
	}
	return tea.Sequence(open(s), open(ps))
}
