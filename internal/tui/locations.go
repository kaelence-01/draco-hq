package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	loc   *docLocations
	err   error
}

// locations picks where a new document goes: next to one of the recent ones.
type locations struct {
	run     docRun
	draft   *docDraft
	loc     *docLocations
	err     error
	started bool
	cursor  int
	filter  filter
	spinner spinner.Model
	ctx     context.Context
	cancel  context.CancelFunc
}

func newLocations(r docRun, d *docDraft) *locations {
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
	r, ctx := l.run, l.ctx
	return tea.Batch(l.spinner.Tick, func() tea.Msg {
		loc, err := r.locations(ctx)
		return locationsMsg{owner: l, loc: loc, err: err}
	})
}

// docs are the locations that match the filter, in the destination's order.
func (l *locations) docs() []provider.DocRef {
	var out []provider.DocRef
	for _, d := range l.loc.docs {
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
			return l, open(newPublish(l.run, l.draft, l.selected()))
		}
	}
	return l, nil
}

func (l *locations) view(width, height int) string {
	switch {
	case l.err != nil:
		return pad(lipgloss.NewStyle().Width(width - 2).Render(errStyle.Render("✗ " + l.err.Error())))
	case l.loc == nil:
		return pad(l.spinner.View() + " Buscando " + l.run.about().recent + "…")
	}
	if width < panelMinWidth {
		return l.narrow(width, height)
	}
	return l.wide(width, height)
}

// wide is the list of places and, beside it, what choosing one will do.
func (l *locations) wide(width, height int) string {
	inner := width - 2
	top := titleLine(boldStyle.Render("¿Dónde se crea?"), l.run.about().steps, 3, inner)
	boxHeight := max(height-2, 6)
	listWidth, summaryWidth := columns(inner, 2, 6, 4)
	summary := append([]string{faintStyle.Render("RESUMEN"), ""}, l.summary(summaryWidth-4)...)
	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		stretch(strings.Join(l.list(listWidth-4, boxHeight-2), "\n"), listWidth, boxHeight), "  ",
		stretch(strings.Join(summary, "\n"), summaryWidth, boxHeight))
	return lipgloss.NewStyle().PaddingLeft(1).Render(lipgloss.JoinVertical(lipgloss.Left, top, "", cols))
}

// narrow is the list with the summary under it.
func (l *locations) narrow(width, height int) string {
	summary := l.summary(width - 2)
	lines := append(l.list(width-2, height-2-len(summary)-1), "")
	return pad(lipgloss.NewStyle().Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, append(lines, summary...)...)))
}

// list is the introduction, the filter and as many places as fit in height
// lines, two lines each.
func (l *locations) list(width, height int) []string {
	lines := []string{faintStyle.Render("El documento nuevo se crea junto al que elijas (en la misma carpeta o página padre).")}
	if l.loc.ownerUnknown {
		lines = append(lines, warnStyle.Render("⚠ No se pudo verificar tu usuario: se muestran documentos de todo el equipo."))
	} else if l.loc.teamWide {
		lines = append(lines, warnStyle.Render("⚠ No tienes documentos propios aquí: se muestran los de todo el equipo como referencia."))
	}
	lines = strings.Split(lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n")), "\n")
	lines = append(lines, "", l.filter.line())
	docs := l.docs()
	if len(docs) == 0 {
		lines = append(lines, faintStyle.Render("Ningún documento coincide"))
	}
	return append(lines, docRows(docs, l.cursor, max((height-len(lines))/2, 1), width)...)
}

// selected is the place under the cursor, if the filter leaves any.
func (l *locations) selected() *provider.DocRef {
	docs := l.docs()
	if len(docs) == 0 {
		return nil
	}
	d := docs[min(l.cursor, len(docs)-1)]
	return &d
}

// summary is what choosing the place under the cursor would do.
func (l *locations) summary(width int) []string {
	place, a := "—", l.run.about()
	if d := l.selected(); d != nil {
		place = d.Title
	}
	lines := []string{kv("Título", l.draft.title, width), kv("Destino", a.dest+" · "+place, width), kv("Página", "nueva, junto a «"+place+"»", width)}
	if a.issue != "" {
		lines = append(lines, kv("Después", "enlace remoto en "+a.issue+" ("+a.tracker+")", width))
	}
	return append(lines, "", lipgloss.NewStyle().Width(width).Render(faintStyle.Render("enter pasa a confirmar; nada se publica todavía.")))
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
