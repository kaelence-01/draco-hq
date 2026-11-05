package tui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/qa"
)

var (
	detectKey     = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "detectar desde Jira"))
	formKey       = key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "editar a mano"))
	saveSetupKey  = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "guardar"))
	pickFieldsKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "elegir los campos"))
	changeKey     = key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "cambiar"))
)

type qaSetupReadMsg struct {
	owner *qaSetup
	setup *qa.Setup
	err   error
}

// qaSetup fills qa_jira from what Jira lists: the project's statuses and the
// instance's fields, picked by name. It asks the project, reads, and shows
// what it proposes: enter saves it, or asks only the fields still missing; e
// opens the lists.
type qaSetup struct {
	parent     *settings
	project    string
	setup      *qa.Setup
	choice     qa.Choice
	reading    bool
	summary    bool
	fieldsOnly bool
	form       *huh.Form
	pages      pager
	err        string
	spinner    spinner.Model
	ctx        context.Context
	cancel     context.CancelFunc
}

func newQASetup(parent *settings, project string) *qaSetup {
	q := &qaSetup{parent: parent, project: project, spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	if q.project == "" {
		if file, err := parent.st.deps.LoadFile(); err == nil && file.QAJira != nil {
			q.project = file.QAJira.Project
		}
	}
	q.ctx, q.cancel = context.WithCancel(context.Background())
	q.form = q.build()
	return q
}

func (q *qaSetup) title() string { return "Detectar desde Jira" }
func (q *qaSetup) close()        { q.cancel() }

func (q *qaSetup) keys() []key.Binding {
	switch {
	case q.reading:
		return nil
	case q.summary && q.choice.Complete():
		return []key.Binding{saveSetupKey, changeKey}
	case q.summary:
		return []key.Binding{pickFieldsKey, changeKey}
	}
	return []key.Binding{nextFieldKey, prevFieldKey}
}

func (q *qaSetup) captures(k tea.KeyPressMsg) bool {
	return !q.reading && !q.summary && !key.Matches(k, backKey)
}

func (q *qaSetup) build() *huh.Form {
	if q.setup == nil {
		return newForm(huh.NewInput().Title("Proyecto de Jira").Description("La clave de la URL del tablero").
			Value(&q.project).Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errors.New("falta la clave del proyecto")
			}
			return nil
		}))
	}
	s := q.setup
	fields := newPage("Campos",
		fieldSelect("Campo de errores de codificación", s.FieldsOf(provider.FieldNumber), &q.choice.CodingErrors),
		fieldSelect("Campo de devoluciones a desarrollo", s.FieldsOf(provider.FieldNumber), &q.choice.DevReturns),
		fieldSelect("Campo del estado del PR", s.FieldsOf(provider.FieldPRStatus), &q.choice.DevStatus),
	)
	if q.fieldsOnly {
		form, pages := newPagedForm(fields)
		q.pages = pages
		return form
	}
	var names []huh.Option[string]
	var reached []huh.Option[string]
	for _, st := range s.Statuses {
		names = append(names, huh.NewOption(st.Name, st.Name))
		label := st.Name
		if st.Done {
			label += " · finalizado"
		}
		reached = append(reached, huh.NewOption(label, st.ID).Selected(slices.Contains(q.choice.Statuses, st.ID)))
	}
	form, pages := newPagedForm(
		newPage("Estados",
			huh.NewSelect[string]().Title("Estado con el que una tarea entra a QA").Description("Abre la ventana del modo Kanban").
				Options(names...).Value(&q.choice.Testing).Height(min(len(names)+2, 7)),
			huh.NewMultiSelect[string]().Title("Estados de una tarea que ya llegó a QA").
				Description("La de QA y las que siguen. espacio marca o desmarca").
				Options(reached...).Value(&q.choice.Statuses).Height(min(len(reached)+2, 10)).Validate(func(v []string) error {
				if len(v) == 0 {
					return errors.New("marca al menos un estado")
				}
				return nil
			}),
		),
		fields,
	)
	q.pages = pages
	return form
}

func fieldSelect(title string, fields []provider.Field, v *string) *huh.Select[string] {
	opts := make([]huh.Option[string], len(fields))
	for i, f := range fields {
		opts[i] = huh.NewOption(f.Name+"  ("+f.ID+")", f.ID)
	}
	return huh.NewSelect[string]().Title(title).Options(opts...).Value(v)
}

func (q *qaSetup) init() tea.Cmd {
	if q.reading || q.summary {
		return nil
	}
	if q.form.State != huh.StateNormal {
		q.form = q.build()
	}
	return q.form.Init()
}

// read opens the profile and reads the project from Jira, off the update loop.
func (q *qaSetup) read() tea.Cmd {
	q.reading, q.err = true, ""
	st, ctx, project := q.parent.st, q.ctx, strings.TrimSpace(q.project)
	return tea.Batch(q.spinner.Tick, func() tea.Msg {
		msg := qaSetupReadMsg{owner: q}
		bundle, err := st.deps.OpenProfile(st.cfg, profile.Options{AnyScope: true})
		if err != nil {
			msg.err = explainProfile(err)
			return msg
		}
		src, err := bundle.QASetup()
		if err != nil {
			msg.err = fmt.Errorf("%w: usa un perfil con tracker jira", err)
			return msg
		}
		msg.setup = qa.NewSetup(src)
		msg.err = msg.setup.Read(ctx, project)
		return msg
	})
}

func (q *qaSetup) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case qaSetupReadMsg:
		if msg.owner != q || q.ctx.Err() != nil {
			return q, nil
		}
		q.reading = false
		if msg.err != nil {
			q.err, q.form = msg.err.Error(), q.build()
			return q, q.form.Init()
		}
		q.setup = msg.setup
		var current *config.QAJiraConfig
		if file, err := q.parent.st.deps.LoadFile(); err == nil {
			current = file.QAJira
		}
		q.choice = q.setup.Propose(current)
		if err := q.missingFields(); err != nil {
			q.setup, q.err, q.form = nil, err.Error(), q.build()
			return q, q.form.Init()
		}
		q.summary = true
		return q, nil
	case spinner.TickMsg:
		if !q.reading {
			return q, nil
		}
		var cmd tea.Cmd
		q.spinner, cmd = q.spinner.Update(msg)
		return q, cmd
	}
	if q.reading {
		return q, nil
	}
	if q.summary {
		k, ok := msg.(tea.KeyPressMsg)
		switch {
		case !ok:
			return q, nil
		case key.Matches(k, saveSetupKey) && q.choice.Complete():
			return q, q.save()
		case key.Matches(k, pickFieldsKey):
			q.summary, q.fieldsOnly = false, true
		case key.Matches(k, changeKey):
			q.summary, q.fieldsOnly = false, false
		default:
			return q, nil
		}
		q.form = q.build()
		return q, q.form.Init()
	}
	m, cmd := q.form.Update(msg)
	q.form = m.(*huh.Form)
	if q.form.State != huh.StateCompleted {
		return q, cmd
	}
	if q.setup == nil {
		return q, q.read()
	}
	return q, q.save()
}

func (q *qaSetup) save() tea.Cmd {
	err := q.parent.st.save(func(c *config.Config) error {
		saved, err := q.setup.Apply(c.QAJira, q.choice)
		c.QAJira = saved
		return err
	})
	if err != nil {
		q.err, q.summary, q.form = "No se guardó: "+err.Error(), false, q.build()
		return q.form.Init()
	}
	note := fmt.Sprintf("QA guardado: proyecto %s, %s", q.setup.Project, plural(len(q.choice.Statuses), "estado", "estados"))
	if file, err := q.parent.st.deps.LoadFile(); err == nil && file.QAEmail == "" {
		note += ". Falta el email de la cuenta de QA: enter abre el formulario"
	}
	q.parent.done(note, nil)
	return back
}

// missingFields explains a Jira without the fields the report reads.
func (q *qaSetup) missingFields() error {
	switch {
	case len(q.setup.FieldsOf(provider.FieldNumber)) == 0:
		return errors.New("Jira no tiene campos numéricos personalizados: el consolidado los usa para los errores de codificación y las devoluciones")
	case len(q.setup.FieldsOf(provider.FieldPRStatus)) == 0:
		return errors.New("Jira no tiene el campo de desarrollo (Development), del que el consolidado lee el estado del PR")
	}
	return nil
}

func (q *qaSetup) view(width, height int) string {
	switch {
	case q.reading:
		return pad(q.spinner.View() + " Leyendo los estados de " + strings.TrimSpace(q.project) + " y los campos de Jira…")
	case q.summary:
		return pad(lipgloss.NewStyle().Width(width - 2).Render(q.summaryText(width - 2)))
	}
	return q.pages.formView(q.form, q.err, width, height)
}

// summaryText is what will be saved, by name, and where it came from.
func (q *qaSetup) summaryText(width int) string {
	s, c := q.setup, q.choice
	chosen, others := s.StatusNames(c.Statuses)
	row := func(label, value string) string {
		if value == "" {
			value = warnStyle.Render("falta elegirlo")
		}
		return hang(faintStyle.Render(padRight(label, 27)), value, width)
	}
	origin := "Propuesto desde tu configuración."
	if c.Suggested {
		origin = "Sugerido por draco: el estado " + config.DefaultTestingStatus + " y los finalizados. Revisa si falta alguno de los demás (por ejemplo, uno de aprobación)."
	}
	lines := []string{
		boldStyle.Render("Proyecto " + s.Project),
		lipgloss.NewStyle().Width(width).Render(faintStyle.Render(origin)),
		"",
		row("Entra a QA con", c.Testing),
		row(fmt.Sprintf("Ya llegó a QA (%d)", len(chosen)), strings.Join(chosen, ", ")),
		row("Los demás estados", strings.Join(others, ", ")),
		"",
		row("Errores de codificación", s.FieldName(c.CodingErrors)),
		row("Devoluciones a desarrollo", s.FieldName(c.DevReturns)),
		row("Estado del PR", s.FieldName(c.DevStatus)),
		"",
	}
	next := "enter guarda esto · e cambia los estados o los campos"
	if !c.Complete() {
		next = "enter elige los campos que faltan, con estos estados · e cambia también los estados"
	}
	return strings.Join(append(lines, lipgloss.NewStyle().Width(width).Render(faintStyle.Render(next))), "\n")
}
