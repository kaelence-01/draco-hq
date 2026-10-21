package tui

import (
	"context"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

var (
	publishKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "publicar"))
	homeKey    = key.NewBinding(key.WithKeys("enter", "esc"), key.WithHelp("enter", "volver al inicio"))
)

type publishedMsg struct {
	owner *publish
	res   *docResult
	err   error
}

// publish confirms and publishes the document. Once publishing starts there
// is no going back: esc would leave a document without its link.
type publish struct {
	run        docRun
	draft      *docDraft
	sibling    *provider.DocRef // nil for an update
	publishing bool
	res        *docResult
	err        error
	spinner    spinner.Model
	ctx        context.Context
	cancel     context.CancelFunc
}

func newPublish(r docRun, d *docDraft, sibling *provider.DocRef) *publish {
	p := &publish{run: r, draft: d, sibling: sibling, spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p
}

func (p *publish) title() string { return "Publicar" }
func (p *publish) close()        { p.cancel() }

func (p *publish) keys() []key.Binding {
	switch {
	case p.res != nil:
		return []key.Binding{homeKey}
	case p.publishing:
		return nil
	}
	return []key.Binding{publishKey}
}

func (p *publish) captures(k tea.KeyPressMsg) bool {
	return (p.publishing || p.res != nil) && key.Matches(k, backKey)
}

func (p *publish) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case publishedMsg:
		if msg.owner != p || p.ctx.Err() != nil {
			return p, nil
		}
		p.publishing, p.res, p.err = false, msg.res, msg.err
	case spinner.TickMsg:
		if !p.publishing {
			return p, nil
		}
		var cmd tea.Cmd
		p.spinner, cmd = p.spinner.Update(msg)
		return p, cmd
	case tea.KeyPressMsg:
		switch {
		case p.res != nil && key.Matches(msg, homeKey):
			return p, toHome
		case p.res == nil && !p.publishing && key.Matches(msg, publishKey):
			p.publishing, p.err = true, nil
			r, ctx, sibling := p.run, p.ctx, p.sibling
			return p, tea.Batch(p.spinner.Tick, func() tea.Msg {
				res, err := r.publish(ctx, sibling)
				return publishedMsg{owner: p, res: res, err: err}
			})
		}
	}
	return p, nil
}

func (p *publish) view(width, height int) string {
	a := p.run.about()
	field := func(label, v string) string { return faintStyle.Render(label+"  ") + v }
	var action string
	if p.sibling != nil {
		action = "Crear junto a «" + p.sibling.Title + "»"
	} else {
		action = "Actualizar «" + p.draft.replaces.Title + "»"
	}
	lines := []string{field("Título ", p.draft.title), field("Destino", a.dest+" · "+action)}
	if a.issue != "" {
		lines = append(lines, field("Enlace ", "desde "+a.issue+" en "+a.tracker))
	}
	lines = append(lines, "")
	switch {
	case p.publishing:
		lines = append(lines, p.spinner.View()+" Publicando… (no se puede volver hasta que termine)")
	case p.err != nil:
		lines = append(lines, errStyle.Render("✗ "+p.err.Error()), faintStyle.Render("enter lo intenta de nuevo · esc vuelve"))
	case p.res != nil:
		lines = append(lines, p.result()...)
	default:
		lines = append(lines, warnStyle.Render("¿Publicar? enter confirma · esc vuelve sin publicar"))
	}
	lines = append(lines, "", faintStyle.Render("Lo mismo en modo comando:"), "$ "+a.command)
	return pad(lipgloss.NewStyle().Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

func (p *publish) result() []string {
	a := p.run.about()
	done := a.noun + " creado"
	if p.res.updated {
		done = a.noun + " actualizado"
	}
	out := []string{okStyle.Bold(true).Render("✓ " + done), "  " + accentStyle.Render(p.res.page.URL)}
	if p.res.linkErr != nil {
		out = append(out, "",
			warnStyle.Render("⚠ No se pudo enlazar el documento desde "+a.issue+": "+p.res.linkErr.Error()),
			warnStyle.Render("  Agrégalo a mano en la tarea; si no, el consolidado QA la marcará sin documentación."))
	}
	return out
}
