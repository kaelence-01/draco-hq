package tui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/profile"
)

// genRun is what the screens of Generar share after the progress.
type genRun struct {
	st      *state
	bundle  *profile.Bundle
	session *deploy.Session
	replace bool   // update the existing document
	command string // the same run in the command mode
}

var (
	tabKey    = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "archivos/documento"))
	scrollKey = key.NewBinding(key.WithKeys("up", "down", "pgup", "pgdown"), key.WithHelp("↑↓", "desplazar"))
	nextKey   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continuar"))
	acceptKey = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "aceptar"))
)

type draftMsg struct {
	owner *preview
	draft *deploy.Draft
	err   error
}

// preview shows the document before publishing; leaving it publishes nothing.
type preview struct {
	run      *genRun
	draft    *deploy.Draft
	err      error
	loading  bool
	started  bool
	showDoc  bool
	accepted bool // the user accepted losing the sections that are not preserved
	note     string
	vp       viewport.Model
	spinner  spinner.Model
	ctx      context.Context
	cancel   context.CancelFunc
}

func newPreview(r *genRun) *preview {
	p := &preview{run: r, showDoc: true, vp: viewport.New(), spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	p.vp.SoftWrap = true
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p
}

func (p *preview) title() string { return "Vista previa" }
func (p *preview) close()        { p.cancel() }

func (p *preview) keys() []key.Binding {
	if p.draft == nil {
		return nil
	}
	keys := []key.Binding{tabKey, scrollKey, nextKey}
	if p.mustAccept() {
		keys = append(keys, acceptKey)
	}
	return keys
}

// mustAccept is an update that would lose a hand-written section.
func (p *preview) mustAccept() bool {
	return p.draft != nil && len(p.draft.NotPreserved) > 0 && !p.accepted
}

func (p *preview) init() tea.Cmd {
	if p.started {
		return nil
	}
	p.started = true
	if !p.run.replace {
		p.draft, p.err = p.run.session.DraftCreate()
		p.fill()
		return nil
	}
	p.loading = true
	s, ctx := p.run.session, p.ctx
	return tea.Batch(p.spinner.Tick, func() tea.Msg {
		d, err := s.DraftUpdate(ctx)
		return draftMsg{owner: p, draft: d, err: err}
	})
}

func (p *preview) fill() {
	if p.draft != nil {
		p.vp.SetContent(p.draft.Markdown)
	}
}

func (p *preview) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case draftMsg:
		if msg.owner != p || p.ctx.Err() != nil {
			return p, nil
		}
		p.loading, p.draft, p.err = false, msg.draft, msg.err
		p.fill()
	case spinner.TickMsg:
		if !p.loading {
			return p, nil
		}
		var cmd tea.Cmd
		p.spinner, cmd = p.spinner.Update(msg)
		return p, cmd
	case tea.KeyPressMsg:
		if p.draft == nil {
			return p, nil
		}
		p.note = ""
		switch {
		case key.Matches(msg, tabKey):
			p.showDoc = !p.showDoc
		case key.Matches(msg, acceptKey) && p.mustAccept():
			p.accepted = true
		case key.Matches(msg, nextKey) && p.mustAccept():
			p.note = "Pulsa s para continuar sin conservar " + quoted(p.draft.NotPreserved) + ", o esc para volver"
		case key.Matches(msg, nextKey) && p.draft.Replaces != nil:
			return p, open(newPublish(p.run, p.draft, nil))
		case key.Matches(msg, nextKey):
			return p, open(newLocations(p.run, p.draft))
		default:
			var cmd tea.Cmd
			p.vp, cmd = p.vp.Update(msg)
			return p, cmd
		}
	}
	return p, nil
}

func (p *preview) view(width, height int) string {
	switch {
	case p.loading:
		return pad(p.spinner.View() + " Leyendo el documento actual para conservar lo escrito a mano…")
	case p.err != nil:
		return pad(lipgloss.NewStyle().Width(width - 2).Render(errStyle.Render("✗ " + p.err.Error())))
	}
	head := []string{boldStyle.Render(p.draft.Title)}
	switch {
	case p.draft.Replaces != nil && len(p.draft.NotPreserved) > 0:
		head = append(head, warnStyle.Render("⚠ El documento actual no tiene "+quoted(p.draft.NotPreserved)+": se publicaría como lo trae la plantilla."))
	case p.draft.Replaces != nil:
		head = append(head, okStyle.Render("✓ ")+"Lo escrito a mano en "+quoted(p.run.session.Target.Template.Manifest.Preserve)+" se conserva.")
	}
	if p.note != "" {
		head = append(head, warnStyle.Render(p.note))
	}
	tabs := []string{faintStyle.Render("  Archivos"), accentStyle.Bold(true).Render("▸ Documento")}
	if !p.showDoc {
		tabs = []string{accentStyle.Bold(true).Render("▸ Archivos"), faintStyle.Render("  Documento")}
	}
	head = append(head, "", strings.Join(tabs, "   "))

	bodyHeight := max(height-len(head)-3, 3)
	var body string
	if p.showDoc {
		p.vp.SetWidth(max(width-6, 10))
		p.vp.SetHeight(bodyHeight)
		body = boxStyle.Render(p.vp.View())
	} else {
		body = lipgloss.NewStyle().MaxHeight(bodyHeight).Render(lipgloss.JoinVertical(lipgloss.Left, readFiles(p.run.session)...))
	}
	return pad(lipgloss.JoinVertical(lipgloss.Left, append(head, body)...))
}

func pad(s string) string { return lipgloss.NewStyle().Padding(1, 1, 0).Render(s) }

func quoted(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = "«" + n + "»"
	}
	return strings.Join(q, ", ")
}
