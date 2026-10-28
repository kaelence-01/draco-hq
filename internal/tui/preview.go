package tui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	tabKey    = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "archivos/documento"))
	scrollKey = key.NewBinding(key.WithKeys("up", "down", "pgup", "pgdown"), key.WithHelp("↑↓", "desplazar"))
	nextKey   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continuar"))
	acceptKey = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "aceptar"))
	rawKey    = key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "Markdown/renderizado"))
)

type draftMsg struct {
	owner *preview
	draft *docDraft
	err   error
}

// preview shows the document before publishing; leaving it publishes nothing.
type preview struct {
	run      docRun
	draft    *docDraft
	err      error
	loading  bool
	started  bool
	showDoc  bool
	raw      bool // the Markdown as it is, not drawn
	accepted bool // the user accepted losing the sections that are not preserved
	note     string
	vp       viewport.Model
	spinner  spinner.Model
	ctx      context.Context
	cancel   context.CancelFunc

	// what the viewport holds, to draw the document again only when it changes
	filledWidth int
	filledRaw   bool
}

func newPreview(r docRun) *preview {
	p := &preview{run: r, showDoc: true, vp: viewport.New(), spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p
}

// fill puts the document in the viewport, drawn for width columns unless the
// user asked for the Markdown.
func (p *preview) fill(width int) {
	if p.filledWidth == width && p.filledRaw == p.raw {
		return
	}
	p.filledWidth, p.filledRaw = width, p.raw
	p.vp.SoftWrap = p.raw
	if p.raw {
		p.vp.SetContent(p.draft.markdown)
		return
	}
	p.vp.SetContent(renderMarkdown(p.draft.markdown, width))
}

func (p *preview) title() string { return "Vista previa" }
func (p *preview) close()        { p.cancel() }

func (p *preview) keys() []key.Binding {
	if p.draft == nil {
		return nil
	}
	keys := []key.Binding{scrollKey, nextKey}
	if p.draft.files != nil {
		keys = append([]key.Binding{tabKey}, keys...)
	}
	if p.mustAccept() {
		keys = append(keys, acceptKey)
	}
	return keys
}

func (p *preview) moreKeys() []key.Binding {
	if p.draft == nil || !p.showDoc {
		return nil
	}
	return []key.Binding{rawKey}
}

// mustAccept is an update that would lose a hand-written section.
func (p *preview) mustAccept() bool {
	return p.draft != nil && len(p.draft.notPreserved) > 0 && !p.accepted
}

func (p *preview) init() tea.Cmd {
	if p.started {
		return nil
	}
	p.started, p.loading = true, true
	r, ctx := p.run, p.ctx
	return tea.Batch(p.spinner.Tick, func() tea.Msg {
		d, err := r.draft(ctx)
		return draftMsg{owner: p, draft: d, err: err}
	})
}

func (p *preview) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case draftMsg:
		if msg.owner != p || p.ctx.Err() != nil {
			return p, nil
		}
		p.loading, p.draft, p.err = false, msg.draft, msg.err
		p.filledWidth = 0
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
		case key.Matches(msg, tabKey) && p.draft.files != nil:
			p.showDoc = !p.showDoc
		case key.Matches(msg, rawKey) && p.showDoc:
			p.raw = !p.raw
			p.vp.SetYOffset(0)
		case key.Matches(msg, acceptKey) && p.mustAccept():
			p.accepted = true
		case key.Matches(msg, nextKey) && p.mustAccept():
			p.note = "Pulsa s para continuar sin conservar " + quoted(p.draft.notPreserved) + ", o esc para volver"
		case key.Matches(msg, nextKey) && p.draft.replaces != nil:
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
	case p.loading && p.run.updating():
		return pad(p.spinner.View() + " Leyendo el documento actual para conservar lo escrito a mano…")
	case p.loading:
		return pad(p.spinner.View() + " Armando el documento…")
	case p.err != nil:
		return pad(lipgloss.NewStyle().Width(width - 2).Render(errStyle.Render("✗ " + p.err.Error())))
	}
	inner := width - 2
	a := p.run.about()
	title, steps := boldStyle.Render(p.draft.title), stepper(a.steps, 2)
	stepsOnTitle := width >= panelMinWidth && lipgloss.Width(title)+2+lipgloss.Width(steps) <= inner
	status := p.status(a)
	var head []string
	switch {
	case stepsOnTitle:
		head = []string{edges(inner, title, steps), status}
	case width >= panelMinWidth:
		head = []string{title, edges(inner, status, steps)}
	default:
		head = []string{title, status}
	}
	if p.note != "" {
		head = append(head, warnStyle.Render(p.note))
	}
	head = append(head, "", p.tabs(inner))
	headBlock := lipgloss.NewStyle().Width(inner).Render(lipgloss.JoinVertical(lipgloss.Left, head...))

	bodyHeight := max(height-lipgloss.Height(headBlock)-3, 3)
	var body string
	if p.showDoc {
		p.fill(max(width-6, 10))
		p.vp.SetWidth(max(width-6, 10))
		p.vp.SetHeight(bodyHeight)
		body = boxStyle.Render(p.vp.View())
	} else {
		body = lipgloss.NewStyle().MaxHeight(bodyHeight).Render(lipgloss.JoinVertical(lipgloss.Left, p.draft.files...))
	}
	return pad(lipgloss.JoinVertical(lipgloss.Left, headBlock, body))
}

// status says what publishing this draft would do.
func (p *preview) status(a docAbout) string {
	switch {
	case p.draft.replaces != nil && len(p.draft.notPreserved) > 0:
		return warnStyle.Render("⚠ El documento actual no tiene " + quoted(p.draft.notPreserved) + ": se publicaría como lo trae la plantilla.")
	case p.draft.replaces != nil:
		return okStyle.Render("✓ ") + "Lo escrito a mano en " + quoted(p.draft.preserve) + " se conserva."
	}
	return okStyle.Render("✓ ") + "Se creará un " + strings.ToLower(a.noun) + " nuevo."
}

// tabs are what the preview can show; the document says how it is drawn, and
// on its tab a reminder of how to change that.
func (p *preview) tabs(width int) string {
	doc, hint := "Documento (renderizado)", "m muestra el Markdown"
	if p.raw {
		doc, hint = "Documento (Markdown)", "m muestra el documento renderizado"
	}
	tabs := accentStyle.Bold(true).Render("▸ " + doc)
	if p.draft.files != nil {
		if p.showDoc {
			tabs = faintStyle.Render("  Archivos") + "   " + tabs
		} else {
			tabs = accentStyle.Bold(true).Render("▸ Archivos") + "   " + faintStyle.Render("  "+doc)
		}
	}
	if !p.showDoc {
		return tabs
	}
	return edges(width, tabs, faintStyle.Render(hint))
}

func pad(s string) string { return lipgloss.NewStyle().Padding(1, 1, 0).Render(s) }

func quoted(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = "«" + n + "»"
	}
	return strings.Join(q, ", ")
}
