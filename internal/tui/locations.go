package tui

import (
	"context"
	"errors"
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

var pickKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "elegir"))

type locationsMsg struct {
	owner *locations
	loc   *deploy.Locations
	err   error
}

// locations picks where a new document goes: next to one of the recent ones.
type locations struct {
	run     *genRun
	draft   *deploy.Draft
	loc     *deploy.Locations
	err     error
	started bool
	cursor  int
	filter  filter
	spinner spinner.Model
	ctx     context.Context
	cancel  context.CancelFunc
}

func newLocations(r *genRun, d *deploy.Draft) *locations {
	l := &locations{run: r, draft: d, spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	l.ctx, l.cancel = context.WithCancel(context.Background())
	return l
}

func (l *locations) title() string { return "Ubicación" }
func (l *locations) close()        { l.cancel() }

func (l *locations) keys() []key.Binding {
	if l.loc == nil {
		return nil
	}
	return []key.Binding{upKey, downKey, filterKey, pickKey}
}

func (l *locations) captures(k tea.KeyPressMsg) bool { return l.filter.captures(k) }

func (l *locations) init() tea.Cmd {
	if l.started {
		return nil
	}
	l.started = true
	s, ctx := l.run.session, l.ctx
	return tea.Batch(l.spinner.Tick, func() tea.Msg {
		loc, err := s.Locations(ctx)
		return locationsMsg{owner: l, loc: loc, err: err}
	})
}

// docs are the locations that match the filter, in the destination's order.
func (l *locations) docs() []provider.DocRef {
	var out []provider.DocRef
	for _, d := range l.loc.Docs {
		if l.filter.matches(d.Title) {
			out = append(out, d)
		}
	}
	return out
}

func (l *locations) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case locationsMsg:
		if msg.owner != l || l.ctx.Err() != nil {
			return l, nil
		}
		l.loc, l.err = msg.loc, msg.err
	case spinner.TickMsg:
		if l.loc != nil || l.err != nil {
			return l, nil
		}
		var cmd tea.Cmd
		l.spinner, cmd = l.spinner.Update(msg)
		return l, cmd
	case tea.KeyPressMsg:
		if l.loc == nil {
			return l, nil
		}
		if l.filter.handle(msg) {
			l.cursor = 0
			return l, nil
		}
		docs := l.docs()
		switch {
		case key.Matches(msg, upKey):
			l.cursor = max(l.cursor-1, 0)
		case key.Matches(msg, downKey):
			l.cursor = min(l.cursor+1, max(len(docs)-1, 0))
		case key.Matches(msg, pickKey) && len(docs) > 0:
			d := docs[min(l.cursor, len(docs)-1)]
			return l, open(newPublish(l.run, l.draft, &d))
		}
	}
	return l, nil
}

func (l *locations) view(width, height int) string {
	switch {
	case l.err != nil:
		return pad(errStyle.Render("✗ " + explainLocations(l.err).Error()))
	case l.loc == nil:
		return pad(l.spinner.View() + " Buscando documentos de despliegue recientes…")
	}
	lines := []string{faintStyle.Render("El documento nuevo se crea junto al que elijas (en la misma carpeta o página padre).")}
	if l.loc.OwnerUnknown {
		lines = append(lines, warnStyle.Render("⚠ No se pudo verificar tu usuario: se muestran documentos de todo el equipo."))
	} else if l.loc.TeamWide {
		lines = append(lines, warnStyle.Render("⚠ No tienes documentos propios aquí: se muestran los de todo el equipo como referencia."))
	}
	lines = append(lines, "")
	if f := l.filter.view(); f != "" {
		lines = append(lines, f)
	}
	docs := l.docs()
	if len(docs) == 0 {
		lines = append(lines, faintStyle.Render("Ningún documento coincide"))
	}
	lines = append(lines, docRows(docs, l.cursor, max((height-len(lines)-2)/2, 1), width-2)...)
	return pad(lipgloss.NewStyle().Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

// explainLocations adds what to do when there is nowhere to create the document.
func explainLocations(err error) error {
	if errors.Is(err, deploy.ErrNoLocations) {
		return fmt.Errorf("%w (o usa un perfil con destino Markdown, que no lo necesita)", err)
	}
	return err
}

// docRows draws up to rows documents around the cursor: title and URL, each
// cut to width.
func docRows(docs []provider.DocRef, cursor, rows, width int) []string {
	row := lipgloss.NewStyle().MaxWidth(width)
	var lines []string
	cur := min(cursor, max(len(docs)-1, 0))
	start := window(cur, len(docs), rows)
	for i := start; i < min(start+rows, len(docs)); i++ {
		mark, title := "  ", docs[i].Title
		if i == cur {
			mark, title = accentStyle.Render("▸ "), accentStyle.Bold(true).Render(title)
		}
		lines = append(lines, row.Render(mark+title), row.Render("    "+faintStyle.Render(docs[i].URL)))
	}
	return lines
}
