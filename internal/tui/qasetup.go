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

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/qa"
)

var detectKey = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "detectar desde Jira"))

type qaSetupReadMsg struct {
	owner *qaSetup
	setup *qa.Setup
	err   error
}

// qaSetup fills qa_jira from what Jira lists: the project's statuses and the
// instance's fields, picked by name. It asks the project, reads, then offers
// the lists with what the config has, or the suggestion, already chosen.
type qaSetup struct {
	parent  *settings
	project string
	setup   *qa.Setup
	choice  qa.Choice
	reading bool
	form    *huh.Form
	err     string
	spinner spinner.Model
	ctx     context.Context
	cancel  context.CancelFunc
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
	if q.reading {
		return nil
	}
	return []key.Binding{nextFieldKey, prevFieldKey}
}

func (q *qaSetup) captures(k tea.KeyPressMsg) bool { return !q.reading && !key.Matches(k, backKey) }

func (q *qaSetup) build() *huh.Form {
	if q.setup == nil {
		return newForm(huh.NewInput().Title("Proyecto de Jira").Description("La clave de la URL del tablero").
			Placeholder("OPS").Value(&q.project).Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errors.New("falta la clave del proyecto")
			}
			return nil
		}))
	}
	s := q.setup
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
	return newPagedForm(
		huh.NewGroup(
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
		huh.NewGroup(
			fieldSelect("Campo de errores de codificación", s.FieldsOf(provider.FieldNumber), &q.choice.CodingErrors),
			fieldSelect("Campo de devoluciones a desarrollo", s.FieldsOf(provider.FieldNumber), &q.choice.DevReturns),
			fieldSelect("Campo del estado del PR", s.FieldsOf(provider.FieldPRStatus), &q.choice.DevStatus),
		),
	)
}

func fieldSelect(title string, fields []provider.Field, v *string) *huh.Select[string] {
	opts := make([]huh.Option[string], len(fields))
	for i, f := range fields {
		opts[i] = huh.NewOption(f.Name+"  ("+f.ID+")", f.ID)
	}
	return huh.NewSelect[string]().Title(title).Options(opts...).Value(v)
}

func (q *qaSetup) init() tea.Cmd {
	if q.reading {
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
			q.setup, q.err = nil, err.Error()
		}
		q.form = q.build()
		return q, q.form.Init()
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
	m, cmd := q.form.Update(msg)
	q.form = m.(*huh.Form)
	if q.form.State != huh.StateCompleted {
		return q, cmd
	}
	if q.setup == nil {
		return q, q.read()
	}
	var saved *config.QAJiraConfig
	err := q.parent.st.save(func(c *config.Config) error {
		var err error
		saved, err = q.setup.Apply(c.QAJira, q.choice)
		c.QAJira = saved
		return err
	})
	if err != nil {
		q.err, q.form = "No se guardó: "+err.Error(), q.build()
		return q, q.form.Init()
	}
	q.parent.done(fmt.Sprintf("QA guardado: proyecto %s, %s", q.setup.Project, plural(len(q.choice.Statuses), "estado", "estados")), nil)
	return q, back
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
	if q.reading {
		return pad(q.spinner.View() + " Leyendo los estados de " + strings.TrimSpace(q.project) + " y los campos de Jira…")
	}
	return formView(q.form, q.err, width)
}
