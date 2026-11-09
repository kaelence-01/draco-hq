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

	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/history"
	"github.com/kaelence-01/draco-hq/internal/profile"
)

var doneKey = key.NewBinding(key.WithKeys("enter", "esc"), key.WithHelp("enter", "volver"))

type removedMsg struct {
	owner      *removeDoc
	res        *deploy.Removal
	historyErr error
	err        error
}

// removeDoc deletes a document of the history once the user types its task.
// Once deleting starts there is no going back, as when publishing.
type removeDoc struct {
	st       *state
	entry    history.Entry
	typed    string
	form     *huh.Form
	removing bool
	res      *deploy.Removal
	histErr  error
	err      error
	spinner  spinner.Model
	ctx      context.Context
	cancel   context.CancelFunc
}

func newRemoveDoc(st *state, e history.Entry) *removeDoc {
	r := &removeDoc{st: st, entry: e, spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.form = r.build()
	return r
}

func (r *removeDoc) build() *huh.Form {
	r.typed = ""
	return newForm(huh.NewInput().Title("Escribe " + r.entry.Issue + " para confirmar").Value(&r.typed).Validate(func(s string) error {
		if !strings.EqualFold(strings.TrimSpace(s), r.entry.Issue) {
			return fmt.Errorf("no coincide con %s: esc vuelve sin borrar", r.entry.Issue)
		}
		return nil
	}))
}

func (r *removeDoc) title() string { return "Borrar " + r.entry.Issue }
func (r *removeDoc) close()        { r.cancel() }
func (r *removeDoc) init() tea.Cmd { return r.form.Init() }

func (r *removeDoc) keys() []key.Binding {
	switch {
	case r.res != nil:
		return []key.Binding{doneKey}
	case r.removing:
		return nil
	}
	return []key.Binding{nextFieldKey}
}

// captures keeps every key while the task is typed, and esc while deleting
// or after it.
func (r *removeDoc) captures(k tea.KeyPressMsg) bool {
	if r.removing || r.res != nil {
		return key.Matches(k, backKey)
	}
	return !key.Matches(k, backKey)
}

func (r *removeDoc) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case removedMsg:
		if msg.owner != r || r.ctx.Err() != nil {
			return r, nil
		}
		r.removing, r.res, r.histErr, r.err = false, msg.res, msg.historyErr, msg.err
		if r.err != nil {
			r.form = r.build()
			return r, r.form.Init()
		}
		return r, report(r.summary())
	case spinner.TickMsg:
		if !r.removing {
			return r, nil
		}
		var cmd tea.Cmd
		r.spinner, cmd = r.spinner.Update(msg)
		return r, cmd
	case tea.KeyPressMsg:
		if r.res != nil {
			if key.Matches(msg, doneKey) {
				return r, back
			}
			return r, nil
		}
		if r.removing {
			return r, nil
		}
	}
	m, cmd := r.form.Update(msg)
	r.form = m.(*huh.Form)
	if r.form.State != huh.StateCompleted {
		return r, cmd
	}
	r.removing, r.err = true, nil
	return r, tea.Batch(r.spinner.Tick, r.remove())
}

// remove runs the deletion in the background: the profile of the document,
// the flow, and then the history.
func (r *removeDoc) remove() tea.Cmd {
	st, e, ctx := r.st, r.entry, r.ctx
	return func() tea.Msg {
		if st.cfg == nil {
			return removedMsg{owner: r, err: errors.New(st.notice)}
		}
		b, err := st.deps.OpenProfile(st.cfg, profile.Options{Name: e.Profile, AnyScope: true})
		if err != nil {
			return removedMsg{owner: r, err: explainProfile(err)}
		}
		if err := e.CheckDest(b.Profile.Docs.Provider); err != nil {
			return removedMsg{owner: r, err: err}
		}
		res, err := deploy.Remove(ctx, b.Tracker, b.Docs, e.Issue, e.Ref())
		if err != nil {
			return removedMsg{owner: r, err: err}
		}
		return removedMsg{owner: r, res: res, historyErr: st.forget(e)}
	}
}

// summary is the line the frame keeps; it warns when something stayed.
func (r *removeDoc) summary() (string, bool) {
	text := r.entry.Issue + ": documento borrado"
	if r.res.Missing {
		text = r.entry.Issue + ": el documento ya no existía"
	}
	if r.res.LinkErr != nil {
		return text + "; el enlace sigue en la tarea", true
	}
	return text, r.res.Missing || r.histErr != nil
}

func (r *removeDoc) view(width, height int) string {
	e := r.entry
	inner := min(width-2, 96) - 4
	wrap := lipgloss.NewStyle().Width(inner)
	field := func(label, v string) string { return hang(faintStyle.Render(padRight(label, 10)), v, inner) }
	what := "va a la papelera de Confluence: se puede restaurar desde ahí"
	if e.Dest != "confluence" {
		what = "se borra el archivo: no hay papelera"
	}
	lines := []string{
		field("Título", e.Title),
		field("Dirección", accentStyle.Render(e.URL)),
		field("Destino", e.Dest+": "+what),
		field("Enlace", "se quita desde la tarea "+e.Issue+", para que el consolidado QA no la dé por documentada"),
		"",
	}
	switch {
	case r.removing:
		lines = append(lines, r.spinner.View()+" Borrando… (no se puede volver hasta que termine)")
	case r.res != nil:
		lines = append(lines, r.result(wrap)...)
	default:
		if r.err != nil {
			lines = append(lines, wrap.Render(errStyle.Render("✗ "+r.err.Error())), "")
		}
		r.form.WithWidth(min(formMaxWidth, inner))
		lines = append(lines, formBody(r.form, "", inner, 0))
	}
	body := strings.Join(lines, "\n")
	if width < panelMinWidth {
		return lipgloss.NewStyle().Padding(1, 1, 0).Render(body)
	}
	box := boxStyle.Width(min(width-2, 96)).Render(body)
	return lipgloss.NewStyle().Padding(1, 1, 0).Render(lipgloss.JoinVertical(lipgloss.Left, warnStyle.Bold(true).Render("Borrar el documento de "+e.Issue), "", box))
}

func (r *removeDoc) result(wrap lipgloss.Style) []string {
	var out []string
	if r.res.Missing {
		out = append(out, warnStyle.Render("⚠ El documento ya no existía"))
	} else {
		out = append(out, okStyle.Bold(true).Render("✓ Documento borrado"))
	}
	if r.res.LinkErr != nil {
		out = append(out,
			wrap.Render(warnStyle.Render("⚠ No se pudo quitar el enlace desde "+r.entry.Issue+": "+r.res.LinkErr.Error())),
			wrap.Render(warnStyle.Render("  Quítalo a mano en la tarea; si no, el consolidado QA la dará por documentada.")))
	} else {
		out = append(out, okStyle.Render("✓ Enlace quitado desde "+r.entry.Issue))
	}
	if r.histErr != nil {
		out = append(out, wrap.Render(warnStyle.Render("⚠ Sigue en el historial: "+r.histErr.Error())))
	}
	return append(out, "", faintStyle.Render("enter vuelve"))
}
