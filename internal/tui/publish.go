package tui

import (
	"context"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

var (
	publishKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "publicar"))
	homeKey    = key.NewBinding(key.WithKeys("enter", "esc"), key.WithHelp("enter", "volver al inicio"))
)

type publishedMsg struct {
	owner *publish
	res   *deploy.Result
	err   error
}

// publish confirms and publishes the document. Once publishing starts there
// is no going back: esc would leave a document without its link.
type publish struct {
	run        *genRun
	draft      *deploy.Draft
	sibling    *provider.DocRef // nil for an update
	publishing bool
	res        *deploy.Result
	err        error
	spinner    spinner.Model
	ctx        context.Context
	cancel     context.CancelFunc
}

func newPublish(r *genRun, d *deploy.Draft, sibling *provider.DocRef) *publish {
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
			s, ctx, d, sibling := p.run.session, p.ctx, p.draft, p.sibling
			return p, tea.Batch(p.spinner.Tick, func() tea.Msg {
				res, err := s.Publish(ctx, d, sibling)
				return publishedMsg{owner: p, res: res, err: err}
			})
		}
	}
	return p, nil
}

func (p *publish) view(width, height int) string {
	s := p.run.session
	field := func(label, v string) string { return faintStyle.Render(label+"  ") + v }
	var action string
	if p.sibling != nil {
		action = "Crear junto a «" + p.sibling.Title + "»"
	} else {
		action = "Actualizar «" + p.draft.Replaces.Title + "»"
	}
	lines := []string{
		field("Título ", p.draft.Title),
		field("Destino", p.run.bundle.Profile.Docs.Provider+" · "+action),
		field("Enlace ", "desde "+s.Target.Issue+" en "+p.run.bundle.Profile.Tracker.Provider),
		"",
	}
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
	lines = append(lines, "", faintStyle.Render("Lo mismo en modo comando:"), "$ "+p.run.command)
	return pad(lipgloss.NewStyle().Width(width - 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

func (p *publish) result() []string {
	done := "Documento creado"
	if p.res.Updated {
		done = "Documento actualizado"
	}
	out := []string{okStyle.Bold(true).Render("✓ " + done), "  " + accentStyle.Render(p.res.Page.URL)}
	if p.res.LinkErr != nil {
		out = append(out, "",
			warnStyle.Render("⚠ No se pudo enlazar el documento desde "+p.run.session.Target.Issue+": "+p.res.LinkErr.Error()),
			warnStyle.Render("  Agrégalo a mano en la tarea; si no, el consolidado QA la marcará sin documentación."))
	}
	return out
}
